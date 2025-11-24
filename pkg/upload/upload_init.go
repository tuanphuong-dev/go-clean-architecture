package upload

import (
	"context"
	"fmt"
	"time"
)

var defaultUploader Client

type Provider string

const (
	Local Provider = "local"
	S3    Provider = "s3"

	DefaultThumbnailWidthInPx  = 400
	DefaultThumbnailHeightInPx = 400
)

type Client interface {
	Upload(files []*File, subPath string) ([]*UploadedFileInfo, error)
	Remove(fileInfos []*UploadedFileInfo) error
	GenerateGetPresignURL(ctx context.Context, objectKey string, ttl time.Duration) (string, error)
}

type Config struct {
	LocalDir string

	S3AccessKey   string
	S3SecretKey   string
	S3EndpointURL string
	S3BucketName  string
	S3PathPrefix  string
	S3Region      string
}

func New(provider Provider, options *Config) (Client, error) {
	var uploader Client
	var err error
	switch provider {
	case Local:
		uploader, err = NewLocalUploader(options)

	case S3:
		uploader, err = NewS3Provider(options)

	default:
		err = fmt.Errorf("unsupported upload provider: %s", provider)
	}

	if err != nil {
		return nil, err
	}
	return uploader, nil
}

func Init(provider Provider, options *Config) error {
	uploader, err := New(provider, options)
	if err != nil {
		return err
	}

	defaultUploader = uploader
	return nil
}

func Upload(files []*File, subPath string) ([]*UploadedFileInfo, error) {
	return defaultUploader.Upload(files, subPath)
}

func Remove(fileInfos []*UploadedFileInfo) error {
	return defaultUploader.Remove(fileInfos)
}

func GenerateGetPresignURL(ctx context.Context, objectKey string, ttl time.Duration) (string, error) {
	return defaultUploader.GenerateGetPresignURL(ctx, objectKey, ttl)
}
