package upload

import (
	"bytes"
	"context"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"path"
	"runtime"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	smithyEndpoints "github.com/aws/smithy-go/endpoints"
	"github.com/disintegration/imaging"
)

type S3Uploader struct {
	s3Client        *s3.Client
	s3PresignClient *s3.PresignClient
	uploader        *manager.Uploader
	bucketName      string
	pathPrefix      string
	region          string
}

type ResolverV2 struct{}

func (*ResolverV2) ResolveEndpoint(ctx context.Context, params s3.EndpointParameters) (
	smithyEndpoints.Endpoint, error,
) {
	return s3.NewDefaultEndpointResolverV2().ResolveEndpoint(ctx, params)
}

func NewS3Provider(opts *Config) (*S3Uploader, error) {
	creds := credentials.NewStaticCredentialsProvider(opts.S3AccessKey, opts.S3SecretKey, "")

	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithCredentialsProvider(creds))
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(opts.S3EndpointURL)
		o.Region = opts.S3Region
		o.EndpointResolverV2 = &ResolverV2{}
	})

	presignClient := s3.NewPresignClient(client)
	uploader := manager.NewUploader(client)
	return &S3Uploader{
		uploader:        uploader,
		s3Client:        client,
		s3PresignClient: presignClient,
		bucketName:      opts.S3BucketName,
		pathPrefix:      opts.S3PathPrefix,
		region:          opts.S3Region,
	}, nil
}

func (u *S3Uploader) uploadFileToS3(r io.Reader, objectKey, contentType string) (*manager.UploadOutput, error) {
	return u.uploader.Upload(context.Background(), &s3.PutObjectInput{
		Bucket:      aws.String(u.bucketName),
		Key:         aws.String(objectKey),
		ContentType: aws.String(contentType),
		Body:        r,
		ACL:         types.ObjectCannedACLPrivate,
		//ACL:       types.ObjectCannedACLPublicRead,
	})
}

func (u *S3Uploader) Upload(files []*File, subPath string) ([]*UploadedFileInfo, error) {
	var fileInfos []*UploadedFileInfo
	var wg sync.WaitGroup
	var mu sync.Mutex

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	maxWorkers := runtime.GOMAXPROCS(0)
	semaphore := make(chan struct{}, maxWorkers)

	errCh := make(chan error, len(files))

	for _, file := range files {
		wg.Add(1)
		semaphore <- struct{}{} // Acquire semaphore
		go func(file *File) {
			defer func() {
				<-semaphore // Release semaphore
				wg.Done()
			}()
			select {
			case <-ctx.Done():
				// Context cancelled, do not proceed
				return
			default:
				var fileInfo UploadedFileInfo
				hash := generateHash()
				fileInfo.Name = file.Name
				fileInfo.Mime = file.Mime
				fileInfo.Ext = getExt(file.Name)
				fileInfo.Size = int64(len(file.Content))
				fileInfo.Provider = S3
				fileInfo.StoragePath = path.Join(u.pathPrefix, subPath, generateFileName(file.Name, hash))

				// Upload file to S3
				s3Output, err := u.uploadFileToS3(bytes.NewReader(file.Content), fileInfo.StoragePath, fileInfo.Mime)
				if err != nil {
					errCh <- err
					cancel()
					return
				}
				fileInfo.URL = s3Output.Location

				// If file is not an image, skip to extract its thumbnail, width, and height
				if !file.IsImage() {
					mu.Lock()
					fileInfos = append(fileInfos, &fileInfo)
					mu.Unlock()
					return
				}

				// Get image dimension and thumbnail
				r := bytes.NewReader(file.Content)
				width, height, thumbnail, err := DecodeImgAndGenThumbnail(r, DefaultThumbnailWidthInPx, DefaultThumbnailHeightInPx)
				if err == nil {
					thumbnailName := generateThumbnailName(file.Name, hash)
					fileInfo.Width = width
					fileInfo.Height = height
					fileInfo.ThumbnailStoragePath = path.Join(u.pathPrefix, subPath, thumbnailName)
					thumbnailBuffer := new(bytes.Buffer)
					if err = imaging.Encode(thumbnailBuffer, thumbnail, imaging.PNG); err != nil {
						errCh <- err
						cancel()
						return
					}

					thumbS3Output, tErr := u.uploadFileToS3(bytes.NewReader(file.Content), fileInfo.ThumbnailStoragePath, fileInfo.Mime)
					if tErr != nil {
						errCh <- tErr
						cancel()
						return
					}

					fileInfo.ThumbnailURL = thumbS3Output.Location
				}

				mu.Lock()
				fileInfos = append(fileInfos, &fileInfo)
				mu.Unlock()
			}
		}(file)
	}

	wg.Wait()
	select {
	case err := <-errCh:
		return nil, err
	default:
		return fileInfos, nil
	}
}

func (u *S3Uploader) Remove(fileInfos []*UploadedFileInfo) error {
	if len(fileInfos) == 0 {
		return nil
	}

	var objectIds []types.ObjectIdentifier
	for _, fileInfo := range fileInfos {
		objectIds = append(objectIds, types.ObjectIdentifier{Key: aws.String(fileInfo.StoragePath)})
		if fileInfo.ThumbnailStoragePath != "" {
			objectIds = append(objectIds, types.ObjectIdentifier{Key: aws.String(fileInfo.ThumbnailStoragePath)})
		}
	}

	_, err := u.s3Client.DeleteObjects(context.Background(), &s3.DeleteObjectsInput{
		Bucket: aws.String(u.bucketName),
		Delete: &types.Delete{Objects: objectIds},
	})
	return err
}

func (u *S3Uploader) GenerateGetPresignURL(ctx context.Context, objectKey string, ttl time.Duration) (string, error) {
	presignReq, err := u.s3PresignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Key:    &objectKey,
		Bucket: &u.bucketName,
	}, func(opts *s3.PresignOptions) {
		opts.Expires = ttl
	})
	if err != nil {
		return "", err
	}

	return presignReq.URL, nil
}
