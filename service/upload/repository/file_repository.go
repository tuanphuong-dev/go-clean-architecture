package repository

import (
	"context"
	"go-clean-arch/database"
	"go-clean-arch/domain"
	"go-clean-arch/pkg/upload"
	"time"

	"gorm.io/gorm"
)

type FilePgRepository struct {
	handler      *database.SQLHandler[domain.File, domain.FileFilter]
	baseURL      string
	presignTTL   time.Duration
	uploadClient upload.Client
}

type ServerConfig interface {
	Domain() string
}

type UploadConfig interface {
	S3PresignUrlTTL() time.Duration
}

func NewFilePgRepository(db *gorm.DB, srvCfg ServerConfig, uploadCfg UploadConfig, client upload.Client) *FilePgRepository {
	return &FilePgRepository{
		handler:      database.NewSQLHandler[domain.File](db, applyFileFilter),
		baseURL:      srvCfg.Domain(),
		presignTTL:   uploadCfg.S3PresignUrlTTL(),
		uploadClient: client,
	}
}

func applyFileFilter(db *gorm.DB, filter *domain.FileFilter) *gorm.DB {
	if filter == nil {
		return db
	}
	if filter.ID != nil {
		db = db.Where("id = ?", *filter.ID)
	}
	if len(filter.IDIn) > 0 {
		db = db.Where("id IN ?", filter.IDIn)
	}
	if filter.Ext != nil {
		db = db.Where("ext = ?", *filter.Ext)
	}
	if filter.IncludeDeleted != nil && *filter.IncludeDeleted {
		// Do not filter out deleted records
	} else {
		// Only include non-deleted records (assuming DeletedAt == 0 means not deleted)
		db = db.Where("deleted_at = 0")
	}
	return db
}

func (r *FilePgRepository) Create(ctx context.Context, file *domain.File) error {
	err := r.handler.Create(ctx, file)
	if err != nil {
		return err
	}
	return prepareFileURLs(file, r.baseURL, r.presignTTL, r.uploadClient)
}

func (r *FilePgRepository) CreateMany(ctx context.Context, files []*domain.File) error {
	err := r.handler.CreateMany(ctx, files)
	if err != nil {
		return err
	}
	for _, f := range files {
		if fErr := prepareFileURLs(f, r.baseURL, r.presignTTL, r.uploadClient); fErr != nil {
			return fErr
		}
	}
	return nil
}

func (r *FilePgRepository) FindByID(ctx context.Context, id string, option *domain.FindOneOption) (*domain.File, error) {
	file, err := r.handler.FindByID(ctx, id, option)
	if err != nil {
		return file, err
	}
	if file != nil {
		if fErr := prepareFileURLs(file, r.baseURL, r.presignTTL, r.uploadClient); fErr != nil {
			return nil, fErr
		}
	}
	return file, nil
}

func (r *FilePgRepository) FindOne(ctx context.Context, filter *domain.FileFilter, option *domain.FindOneOption) (*domain.File, error) {
	file, err := r.handler.FindOne(ctx, filter, option)
	if err != nil {
		return file, err
	}
	if file != nil {
		if fErr := prepareFileURLs(file, r.baseURL, r.presignTTL, r.uploadClient); fErr != nil {
			return nil, fErr
		}
	}
	return file, nil
}

func (r *FilePgRepository) FindMany(ctx context.Context, filter *domain.FileFilter, option *domain.FindManyOption) ([]*domain.File, error) {
	files, err := r.handler.FindMany(ctx, filter, option)
	if err != nil {
		return files, err
	}
	for _, f := range files {
		if fErr := prepareFileURLs(f, r.baseURL, r.presignTTL, r.uploadClient); fErr != nil {
			return nil, fErr
		}
	}
	return files, nil
}

func (r *FilePgRepository) FindPage(ctx context.Context, filter *domain.FileFilter, option *domain.FindPageOption) ([]*domain.File, *domain.Pagination, error) {
	files, pagination, err := r.handler.FindPage(ctx, filter, option)
	if err != nil {
		return files, pagination, err
	}
	for _, f := range files {
		if fErr := prepareFileURLs(f, r.baseURL, r.presignTTL, r.uploadClient); fErr != nil {
			return nil, nil, fErr
		}
	}
	return files, pagination, nil
}

func (r *FilePgRepository) Update(ctx context.Context, file *domain.File) error {
	err := r.handler.Update(ctx, file)
	if err != nil {
		return err
	}
	return prepareFileURLs(file, r.baseURL, r.presignTTL, r.uploadClient)
}

func (r *FilePgRepository) DeleteByID(ctx context.Context, id string) error {
	return r.handler.DeleteByID(ctx, id)
}

func (r *FilePgRepository) DeleteMany(ctx context.Context, filter *domain.FileFilter) (int64, error) {
	return r.handler.DeleteMany(ctx, filter)
}

func (r *FilePgRepository) Count(ctx context.Context, filter *domain.FileFilter) (int64, error) {
	return r.handler.Count(ctx, filter)
}
