package repository

import (
	"go-clean-arch/domain"
	"go-clean-arch/pkg/upload"
	"time"
)

func prepareFileURLs(
	f *domain.File,
	baseURL string,
	presignTTL time.Duration,
	uploadClient upload.Client,

) error {
	switch upload.Provider(f.Props.Provider) {
	case upload.Local:
		f.URL = common.JoinURLPath(baseURL, f.URL)
		if f.ThumbnailURL != "" {
			f.ThumbnailURL = common.JoinURLPath(baseURL, f.ThumbnailURL)
		}

	case upload.S3:
		ctx := context.Background()
		ttl := presignTTL
		presignedURL, err := uploadClient.GenerateGetPresignURL(ctx, f.Props.StoragePath, ttl)
		if err != nil {
			return err
		}
		f.URL = presignedURL

		if f.ThumbnailURL != "" {
			presignedThumbURL, tErr := uploadClient.GenerateGetPresignURL(ctx, f.Props.ThumbStoragePath, ttl)
			if tErr != nil {
				return tErr
			}
			f.ThumbnailURL = presignedThumbURL
		}
	}
	return nil
}
