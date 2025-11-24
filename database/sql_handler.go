package database

import (
	"context"
	"errors"
	"fmt"
	"go-clean-arch/domain"
	"go-clean-arch/pkg/utils"
	"strings"

	"gorm.io/gorm"
)

type SQLHandler[T any, V any] struct {
	db          *gorm.DB
	applyFilter func(*gorm.DB, *V) *gorm.DB
}

func NewSQLHandler[T any, V any](
	db *gorm.DB,
	applyFilter func(*gorm.DB, *V) *gorm.DB,

) *SQLHandler[T, V] {
	return &SQLHandler[T, V]{applyFilter: applyFilter, db: db}
}

type DBOption func(*gorm.DB) *gorm.DB

func WithOmit(fields ...string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Omit(fields...)
	}
}

func WithTx(tx *gorm.DB) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		if tx != nil {
			return tx
		}
		return db
	}
}

func (h *SQLHandler[T, V]) applyDBOptions(opts ...DBOption) *gorm.DB {
	qb := h.db
	for _, opt := range opts {
		qb = opt(qb)
	}
	return qb
}

func (h *SQLHandler[T, V]) Create(ctx context.Context, entity *T, opts ...DBOption) error {
	execDB := h.applyDBOptions(opts...)
	return execDB.WithContext(ctx).Create(&entity).Error
}

func (h *SQLHandler[T, V]) CreateMany(ctx context.Context, entities []*T, opts ...DBOption) error {
	execDB := h.applyDBOptions(opts...)
	return execDB.WithContext(ctx).Create(&entities).Error
}

func (h *SQLHandler[T, V]) FindByID(ctx context.Context, id any, option *domain.FindOneOption, opts ...DBOption) (*T, error) {
	var entity T
	execDB := h.applyDBOptions(opts...)
	err := execDB.WithContext(ctx).Where("id = ?", id).First(&entity).Error
	return &entity, err
}

func (h *SQLHandler[T, V]) FindOne(ctx context.Context, filter *V, option *domain.FindOneOption, opts ...DBOption) (*T, error) {
	execDB := h.applyDBOptions(opts...)
	execDB = h.applyFilter(execDB, filter)

	var entity T
	err := execDB.WithContext(ctx).First(&entity).Error
	if err == nil {
		return &entity, nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrRecordNotFound
	}
	return nil, err
}

func (h *SQLHandler[T, V]) applyFindManyOption(db *gorm.DB, option *domain.FindManyOption) *gorm.DB {
	if option == nil {
		return db
	}

	for _, sortField := range option.Sort {
		db = db.Order(sortField)
	}

	if option.Limit != nil {
		db = db.Limit(*option.Limit)
	}

	if option.Offset != nil {
		db = db.Offset(*option.Offset)
	}

	for _, field := range option.Preloads {
		db = db.Preload(field)
	}

	for _, field := range option.Joins {
		db = db.Joins(field)
	}
	return db
}

func (h *SQLHandler[T, V]) FindMany(ctx context.Context, filter *V, option *domain.FindManyOption, opts ...DBOption) ([]*T, error) {
	execDB := h.applyDBOptions(opts...)
	execDB = h.applyFilter(execDB, filter)
	execDB = h.applyFindManyOption(execDB, option)

	var entities []*T
	err := execDB.WithContext(ctx).Find(&entities).Error
	if err != nil {
		return nil, err
	}

	return entities, nil
}

func (h *SQLHandler[T, V]) applyFindPageOption(db *gorm.DB, option *domain.FindPageOption) (outDB *gorm.DB, page, perPage int) {
	outDB = db
	page = 1
	perPage = 10
	if option != nil {
		if len(option.Sort) > 0 {
			for _, sortField := range option.Sort {
				outDB = outDB.Order(sortField)
			}
		}
		if option.Page > 0 {
			page = option.Page
		}

		if option.PerPage > 0 {
			perPage = option.PerPage
		}

		for _, field := range option.Preloads {
			db = db.Preload(field)
		}
	}
	offset := (page - 1) * perPage
	outDB = outDB.Offset(offset).Limit(perPage)
	return
}

func (h *SQLHandler[T, V]) FindPage(ctx context.Context, filter *V, option *domain.FindPageOption, opts ...DBOption) ([]*T, *domain.Pagination, error) {
	execDB := h.applyDBOptions(opts...)
	execDB = h.applyFilter(execDB, filter)

	var totalItems int64
	countDB := execDB.Session(&gorm.Session{}) // clone for count
	err := countDB.WithContext(ctx).Model(new(T)).Count(&totalItems).Error
	if err != nil {
		return nil, nil, err
	}

	execDB, page, perPage := h.applyFindPageOption(execDB, option)

	var entities []*T
	err = execDB.WithContext(ctx).Find(&entities).Error
	if err != nil {
		return nil, nil, err
	}

	return entities, domain.NewPagination(page, perPage, totalItems), nil
}

func (h *SQLHandler[T, V]) Update(ctx context.Context, entity *T, opts ...DBOption) error {
	execDB := h.applyDBOptions(opts...)
	return execDB.WithContext(ctx).Save(&entity).Error
}

func (h *SQLHandler[T, V]) UpdateFields(ctx context.Context, id any, fields map[string]any, opts ...DBOption) error {
	execDB := h.applyDBOptions(opts...)
	var entity T
	return execDB.WithContext(ctx).Model(&entity).Where("id = ?", id).Updates(fields).Error
}

func (h *SQLHandler[T, V]) DeleteByID(ctx context.Context, id any, opts ...DBOption) error {
	execDB := h.applyDBOptions(opts...)
	var entity T
	return execDB.WithContext(ctx).
		Model(&entity).
		Where("id = ? AND deleted_at = 0", id).
		Updates(map[string]any{
			"deleted_at": utils.NowUnixMillis(),
		}).Error
}

func (h *SQLHandler[T, V]) DeleteMany(ctx context.Context, filter *V, opts ...DBOption) (int64, error) {
	execDB := h.applyDBOptions(opts...)
	execDB = h.applyFilter(execDB, filter)
	var entity T
	err := execDB.WithContext(ctx).Delete(&entity).Error
	if err != nil {
		return 0, err
	}
	return execDB.RowsAffected, nil
}

func (h *SQLHandler[T, V]) Count(ctx context.Context, filter *V, opts ...DBOption) (int64, error) {
	var count int64
	execDB := h.applyDBOptions(opts...)
	execDB = h.applyFilter(execDB, filter)
	err := execDB.WithContext(ctx).Model(new(T)).Count(&count).Error
	return count, err
}

// ApplySearch applies full-text search and partial match search to the given gorm.DB instance.
// searchableFields is map[alias]dbField
// Example: map[string]string{"name": "users.name", "email": "users.email"}
func ApplySearch(
	db *gorm.DB,
	searchTerm string,
	searchFields []string,
	searchableFields map[string]string,

) *gorm.DB {
	if searchTerm == "" {
		return db
	}

	fieldsToSearch := getValidSearchFields(searchFields, searchableFields)
	if len(fieldsToSearch) == 0 {
		return db
	}

	ftsQuery := buildFullTextSearchQuery(fieldsToSearch, searchTerm)
	ilikeQuery := buildPartialMatchQuery(fieldsToSearch, searchTerm)

	combinedQuery := combineSearchQueries(ftsQuery, ilikeQuery)

	return db.Where(combinedQuery.condition, combinedQuery.args...)
}

// getValidSearchFields returns database field names that are valid for searching
func getValidSearchFields(requestedFields []string, searchableFields map[string]string) []string {
	if len(requestedFields) == 0 {
		return getAllSearchableFields(searchableFields)
	}
	return filterValidFields(requestedFields, searchableFields)
}

// getAllSearchableFields returns all available searchable database fields
func getAllSearchableFields(searchableFields map[string]string) []string {
	fields := make([]string, 0, len(searchableFields))
	for _, dbField := range searchableFields {
		fields = append(fields, dbField)
	}
	return fields
}

// filterValidFields returns only the requested fields that exist in searchableFields
func filterValidFields(requestedFields []string, searchableFields map[string]string) []string {
	var validFields []string
	for _, field := range requestedFields {
		if dbField, exists := searchableFields[field]; exists {
			validFields = append(validFields, dbField)
		}
	}
	return validFields
}

type searchQuery struct {
	condition string
	args      []any
}

func buildFullTextSearchQuery(fields []string, searchTerm string) searchQuery {
	conditions := make([]string, len(fields))
	args := make([]any, len(fields))

	for i, field := range fields {
		conditions[i] = fmt.Sprintf("to_tsvector_vietnamese(COALESCE(%s, '')) @@ plainto_tsquery('vietnamese', ?)", field)
		args[i] = searchTerm
	}

	return searchQuery{
		condition: strings.Join(conditions, " OR "),
		args:      args,
	}
}

// buildPartialMatchQuery creates ILIKE conditions for partial string matching
func buildPartialMatchQuery(fields []string, searchTerm string) searchQuery {
	conditions := make([]string, len(fields))
	args := make([]any, len(fields))
	searchPattern := "%" + searchTerm + "%"

	for i, field := range fields {
		conditions[i] = fmt.Sprintf("%s ILIKE ?", field)
		args[i] = searchPattern
	}

	return searchQuery{
		condition: strings.Join(conditions, " OR "),
		args:      args,
	}
}

// combineSearchQueries merges FTS and ILIKE queries with OR logic
func combineSearchQueries(ftsQuery, ilikeQuery searchQuery) searchQuery {
	combinedCondition := fmt.Sprintf("(%s) OR (%s)", ftsQuery.condition, ilikeQuery.condition)
	combinedArgs := append(ftsQuery.args, ilikeQuery.args...)

	return searchQuery{
		condition: combinedCondition,
		args:      combinedArgs,
	}
}
