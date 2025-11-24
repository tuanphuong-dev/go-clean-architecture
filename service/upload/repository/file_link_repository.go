package repository

import (
	"context"
	"go-clean-arch/common"
	"go-clean-arch/database"
	"go-clean-arch/domain"
	"go-clean-arch/pkg/upload"
	"time"

	"github.com/samber/lo"
	"gorm.io/gorm"
)

type FileRepository interface {
	FindMany(ctx context.Context, filter *domain.FileFilter, option *domain.FindManyOption) ([]*domain.File, error)
}

type FileLinkPgRepository struct {
	sqlHandler   *database.SQLHandler[domain.FileLink, domain.FileLinkFilter]
	baseURL      string
	presignTTL   time.Duration
	uploadClient upload.Client
	fileRepo     FileRepository
}

func NewFileLinkPgRepository(
	db *gorm.DB,
	srvCfg ServerConfig,
	uploadCfg UploadConfig,
	client upload.Client,
	fileRepo FileRepository,

) *FileLinkPgRepository {
	return &FileLinkPgRepository{
		sqlHandler:   database.NewSQLHandler[domain.FileLink](db, applyFileLinkFilter),
		baseURL:      srvCfg.Domain(),
		presignTTL:   uploadCfg.S3PresignUrlTTL(),
		uploadClient: client,
		fileRepo:     fileRepo,
	}
}

func applyFileLinkFilter(db *gorm.DB, filter *domain.FileLinkFilter) *gorm.DB {
	if filter == nil {
		return db
	}
	if filter.FileID != nil {
		db = db.Where("file_id = ?", *filter.FileID)
	}
	if filter.RelatedID != nil {
		db = db.Where("related_id = ?", *filter.RelatedID)
	}
	if len(filter.RelatedIDIn) > 0 {
		db = db.Where("related_id IN ?", filter.RelatedIDIn)
	}
	if filter.RelatedType != nil {
		db = db.Where("related_type = ?", *filter.RelatedType)
	}
	if filter.Field != nil {
		db = db.Where("field = ?", *filter.Field)
	}
	if len(filter.FieldIn) > 0 {
		db = db.Where("field IN ?", filter.FieldIn)
	}
	return db
}

func (r *FileLinkPgRepository) Create(ctx context.Context, relation *domain.FileLink) error {
	return r.sqlHandler.Create(ctx, relation)
}

func (r *FileLinkPgRepository) FindOne(ctx context.Context, filter *domain.FileLinkFilter, option *domain.FindOneOption) (*domain.FileLink, error) {
	relation, err := r.sqlHandler.FindOne(ctx, filter, option)
	if err != nil {
		return relation, err
	}
	if relation != nil && relation.File != nil {
		if err := prepareFileURLs(relation.File, r.baseURL, r.presignTTL, r.uploadClient); err != nil {
			return nil, err
		}
	}
	return relation, nil
}

func (r *FileLinkPgRepository) FindMany(ctx context.Context, filter *domain.FileLinkFilter, option *domain.FindManyOption) ([]*domain.FileLink, error) {
	relations, err := r.sqlHandler.FindMany(ctx, filter, option)
	if err != nil {
		return relations, err
	}
	for _, relation := range relations {
		if relation != nil && relation.File != nil {
			if err := prepareFileURLs(relation.File, r.baseURL, r.presignTTL, r.uploadClient); err != nil {
				return nil, err
			}
		}
	}
	return relations, nil
}

func (r *FileLinkPgRepository) FindPage(ctx context.Context, filter *domain.FileLinkFilter, option *domain.FindPageOption) ([]*domain.FileLink, *domain.Pagination, error) {
	relations, pagination, err := r.sqlHandler.FindPage(ctx, filter, option)
	if err != nil {
		return relations, pagination, err
	}
	for _, relation := range relations {
		if relation != nil && relation.File != nil {
			if err := prepareFileURLs(relation.File, r.baseURL, r.presignTTL, r.uploadClient); err != nil {
				return nil, nil, err
			}
		}
	}
	return relations, pagination, nil
}

func (r *FileLinkPgRepository) Update(ctx context.Context, relation *domain.FileLink) error {
	err := r.sqlHandler.Update(ctx, relation)
	if err != nil {
		return err
	}
	if relation != nil && relation.File != nil {
		if err := prepareFileURLs(relation.File, r.baseURL, r.presignTTL, r.uploadClient); err != nil {
			return err
		}
	}
	return nil
}

func (r *FileLinkPgRepository) DeleteMany(ctx context.Context, filter *domain.FileLinkFilter) (int64, error) {
	return r.sqlHandler.DeleteMany(ctx, filter)
}

func (r *FileLinkPgRepository) Count(ctx context.Context, filter *domain.FileLinkFilter) (int64, error) {
	return r.sqlHandler.Count(ctx, filter)
}

func (r *FileLinkPgRepository) AddFiles(
	ctx context.Context,
	entityType string,
	entityID string,
	fieldFiles map[string][]*domain.File,

) (map[string]*domain.File, error) {
	if len(fieldFiles) == 0 {
		return map[string]*domain.File{}, nil
	}

	var fieldNames []string
	for fieldName := range fieldFiles {
		fieldNames = append(fieldNames, fieldName)
	}

	existingFileLinks, err := r.sqlHandler.FindMany(ctx, &domain.FileLinkFilter{
		RelatedID:   common.New(entityID),
		RelatedType: common.New(entityType),
		FieldIn:     fieldNames,
	}, nil)
	if err != nil {
		return map[string]*domain.File{}, err
	}

	allFileIDs := make(map[string]struct{})
	for _, files := range fieldFiles {
		for _, file := range files {
			if file.ID != "" {
				allFileIDs[file.ID] = struct{}{}
			}
		}
	}

	var fileIDs []string
	for fileID := range allFileIDs {
		fileIDs = append(fileIDs, fileID)
	}

	var filesFromDB []*domain.File
	if len(fileIDs) > 0 {
		filesFromDB, err = r.fileRepo.FindMany(ctx, &domain.FileFilter{
			IDIn: fileIDs,
		}, &domain.FindManyOption{})
		if err != nil {
			return map[string]*domain.File{}, err
		}
	}

	filesMap := make(map[string]*domain.File)
	for _, file := range filesFromDB {
		filesMap[file.ID] = file
	}

	existingByField := make(map[string][]*domain.FileLink)
	for _, relation := range existingFileLinks {
		existingByField[relation.Field] = append(existingByField[relation.Field], relation)
	}

	var toCreate []*domain.FileLink
	var toUpdate []*domain.FileLink
	var toDelete []*domain.FileLink
	for fieldName, files := range fieldFiles {
		existingRelations := existingByField[fieldName]

		existingMap := make(map[string]*domain.FileLink)
		for _, ef := range existingRelations {
			existingMap[ef.FileID] = ef
		}

		for idx, file := range files {
			if file.ID == "" {
				continue
			}

			newOrder := idx + 1
			if existing, found := existingMap[file.ID]; found {
				if existing.Order != newOrder {
					existing.Order = newOrder
					toUpdate = append(toUpdate, existing)
				}
			} else {
				toCreate = append(toCreate, &domain.FileLink{
					FileID:      file.ID,
					RelatedID:   entityID,
					RelatedType: entityType,
					Field:       fieldName,
					Order:       newOrder,
				})
			}

			if fileFromDB, ok := filesMap[file.ID]; ok {
				*file = *fileFromDB
			} else {
				file = nil
			}
		}

		newFilesMap := make(map[string]struct{})
		for _, file := range files {
			newFilesMap[file.ID] = struct{}{}
		}

		for _, existing := range existingRelations {
			if _, found := newFilesMap[existing.FileID]; !found {
				toDelete = append(toDelete, existing)
			}
		}
	}

	if len(toCreate) > 0 {
		if err = r.sqlHandler.CreateMany(ctx, toCreate); err != nil {
			return map[string]*domain.File{}, err
		}
	}

	if len(toUpdate) > 0 {
		for _, relation := range toUpdate {
			if err = r.Update(ctx, relation); err != nil {
				return map[string]*domain.File{}, err
			}
		}
	}

	if len(toDelete) > 0 {
		for _, relation := range toDelete {
			delQuery := &domain.FileLinkFilter{
				FileID:    common.New(relation.FileID),
				RelatedID: common.New(relation.RelatedID),
				Field:     common.New(relation.Field),
			}
			if _, err = r.DeleteMany(ctx, delQuery); err != nil {
				return map[string]*domain.File{}, err
			}
		}
	}
	return filesMap, nil
}

func (r *FileLinkPgRepository) GetFiles(
	ctx context.Context,
	entityType string,
	entityID string,
	fieldName string,

) ([]*domain.File, error) {
	filter := &domain.FileLinkFilter{
		RelatedID:   common.New(entityID),
		RelatedType: common.New(entityType),
		Field:       common.New(fieldName),
	}
	option := &domain.FindManyOption{
		Preloads: []string{"File"},
		Sort:     []string{`"order" ASC`},
	}
	relations, err := r.sqlHandler.FindMany(ctx, filter, option)
	if err != nil {
		return nil, err
	}

	return lo.Map(relations, func(relation *domain.FileLink, _ int) *domain.File {
		return relation.File
	}), nil
}

func (r *FileLinkPgRepository) GetFilesByEntitiesAndField(
	ctx context.Context,
	entityType string,
	entitiesID []string,
	fieldName string,

) (map[string][]*domain.File, error) {
	filter := &domain.FileLinkFilter{
		RelatedIDIn: lo.Uniq(entitiesID),
		Field:       common.New(fieldName),
		RelatedType: common.New(entityType),
	}
	option := &domain.FindManyOption{
		Preloads: []string{"File"},
		Sort:     []string{`"order" ASC`},
	}
	relations, err := r.sqlHandler.FindMany(ctx, filter, option)
	if err != nil {
		return nil, err
	}

	relationsByEntitiesIDs := map[string][]*domain.File{}
	for _, relation := range relations {
		if err := prepareFileURLs(relation.File, r.baseURL, r.presignTTL, r.uploadClient); err != nil {
			return nil, err
		}
		relationsByEntitiesIDs[relation.RelatedID] = append(relationsByEntitiesIDs[relation.RelatedID], relation.File)
	}
	return relationsByEntitiesIDs, nil
}

func (r *FileLinkPgRepository) GetFilesByEntities(
	ctx context.Context,
	entityType string,
	entitiesID []string,

) (map[string]map[string][]*domain.File, error) {
	filter := &domain.FileLinkFilter{
		RelatedIDIn: lo.Uniq(entitiesID),
		RelatedType: common.New(entityType),
	}
	option := &domain.FindManyOption{
		Joins: []string{common.FieldFile},
		Sort:  []string{common.SortOrderAsc},
	}
	relations, err := r.sqlHandler.FindMany(ctx, filter, option)
	if err != nil {
		return nil, err
	}

	filesByEntityAndField := map[string]map[string][]*domain.File{}
	for _, relation := range relations {
		if err := prepareFileURLs(relation.File, r.baseURL, r.presignTTL, r.uploadClient); err != nil {
			return nil, err
		}
		if _, exists := filesByEntityAndField[relation.RelatedID]; !exists {
			filesByEntityAndField[relation.RelatedID] = make(map[string][]*domain.File)
		}
		filesByEntityAndField[relation.RelatedID][relation.Field] = append(filesByEntityAndField[relation.RelatedID][relation.Field], relation.File)
	}
	return filesByEntityAndField, nil
}
