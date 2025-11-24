package upload

import (
	"bytes"
	"context"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/disintegration/imaging"
)

const (
	StaticsFsPath = "/uploads/"
)

type LocalUploader struct {
	uploadDirPath string
}

func NewLocalUploader(opts *Config) (*LocalUploader, error) {
	return &LocalUploader{
		uploadDirPath: opts.LocalDir,
	}, nil
}

func (u *LocalUploader) saveFile(src io.Reader, dst string) error {
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, src)
	return err
}

func (u *LocalUploader) Upload(files []*File, subPath string) ([]*UploadedFileInfo, error) {
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
				fileInfo.Provider = Local
				fileInfo.StoragePath = path.Join(u.uploadDirPath, subPath, generateFileName(file.Name, hash))

				// Save file to disk
				if err := u.saveFile(bytes.NewReader(file.Content), fileInfo.StoragePath); err != nil {
					errCh <- err
					cancel()
					return
				}
				fileInfo.URL = path.Join(StaticsFsPath, generateFileName(file.Name, hash))

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
					fileInfo.ThumbnailStoragePath = path.Join(u.uploadDirPath, subPath, thumbnailName)
					err = imaging.Save(thumbnail, fileInfo.ThumbnailStoragePath)
					if err != nil {
						errCh <- err
						cancel()
						return
					}

					fileInfo.ThumbnailURL = path.Join(StaticsFsPath, thumbnailName)
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

func (u *LocalUploader) Remove(fileInfos []*UploadedFileInfo) error {
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Limit the number of concurrent goroutines
	maxWorkers := runtime.GOMAXPROCS(0)
	semaphore := make(chan struct{}, maxWorkers)

	errCh := make(chan error, len(fileInfos))

	for _, fileInfo := range fileInfos {
		wg.Add(1)
		semaphore <- struct{}{} // Acquire semaphore

		go func(fileInfo *UploadedFileInfo) {
			defer func() {
				<-semaphore // Release semaphore
				wg.Done()
			}()

			select {
			case <-ctx.Done():
				// Context cancelled, do not proceed
				return
			default:

				if err := os.Remove(fileInfo.StoragePath); err != nil {
					errCh <- err
					cancel()
					return
				}

				if fileInfo.ThumbnailStoragePath != "" {
					if err := os.Remove(fileInfo.ThumbnailStoragePath); err != nil {
						errCh <- err
						cancel()
						return
					}
				}
			}
		}(fileInfo)
	}

	wg.Wait()

	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}

func (u *LocalUploader) GenerateGetPresignURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", nil
}
