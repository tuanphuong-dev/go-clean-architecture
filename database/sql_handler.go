package database

import (
	"context"
	"errors"
	"go-clean-arch/domain"

	"gorm.io/gorm"
)

type SQLHandler[T any, V any] struct {
	db          *gorm.DB
	applyFilter func(*gorm.DB, *V) *gorm.DB
}

func NewSQLHandler[T any, V any](db *gorm.DB, applyFilter func(*gorm.DB, *V) *gorm.DB) *SQLHandler[T, V] {
	return &SQLHandler[T, V]{applyFilter: applyFilter, db: db}
}

type DBOption func(*gorm.DB) *gorm.DB

func WithTx(tx *gorm.DB) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		if tx != nil {
			return tx
		}
		return db
	}
}

func (h *SQLHandler[T, V]) applyOptions(opts ...DBOption) *gorm.DB {
	qb := h.db
	for _, opt := range opts {
		qb = opt(qb)
	}
	return qb
}

func (h *SQLHandler[T, V]) Create(ctx context.Context, entity *T, opts ...DBOption) error {
	execDB := h.applyOptions(opts...)
	return execDB.WithContext(ctx).Create(&entity).Error
}

func (h *SQLHandler[T, V]) FindByID(ctx context.Context, id any, option *domain.FindOneOption, opts ...DBOption) (*T, error) {
	var entity T
	execDB := h.applyOptions(opts...)
	err := execDB.WithContext(ctx).Where("id = ?", id).First(&entity).Error
	return &entity, err
}

func (h *SQLHandler[T, V]) FindOne(ctx context.Context, filter *V, option *domain.FindOneOption, opts ...DBOption) (*T, error) {
	var entity T
	execDB := h.applyOptions(opts...)
	execDB = h.applyFilter(execDB, filter)
	if err := execDB.WithContext(ctx).First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			println("Debug info: record not found for filter", filter)
			return nil, domain.ErrRecordNotFound
		}
		return nil, err
	}
	return &entity, nil
}

func (h *SQLHandler[T, V]) FindMany(ctx context.Context, filter *V, option *domain.FindManyOption, opts ...DBOption) ([]*T, error) {
	// TODO implement me
	panic("implement me")
}

func (h *SQLHandler[T, V]) FindPage(ctx context.Context, filter *V, option *domain.FindPageOption, opts ...DBOption) ([]*T, *domain.Pagination, error) {
	// TODO implement me
	panic("implement me")
}

func (h *SQLHandler[T, V]) Update(ctx context.Context, entity *T, opts ...DBOption) error {
	execDB := h.applyOptions(opts...)
	return execDB.WithContext(ctx).Save(&entity).Error
}

func (h *SQLHandler[T, V]) UpdateFields(ctx context.Context, id any, fields map[string]any, opts ...DBOption) error {
	execDB := h.applyOptions(opts...)
	var entity T
	return execDB.WithContext(ctx).Model(&entity).Where("id = ?", id).Updates(fields).Error
}

func (h *SQLHandler[T, V]) DeleteByID(ctx context.Context, id any, opts ...DBOption) error {
	execDB := h.applyOptions(opts...)
	var entity T
	return execDB.WithContext(ctx).Delete(&entity).Where("id = ?", id).Error
}

func (h *SQLHandler[T, V]) Count(ctx context.Context, filter *V, opts ...DBOption) (int64, error) {
	var count int64
	execDB := h.applyOptions(opts...)
	execDB = h.applyFilter(execDB, filter)
	err := execDB.WithContext(ctx).Model(new(T)).Count(&count).Error
	return count, err
}
