package repository

import (
	"context"
	"go-clean-arch/database"
	"go-clean-arch/domain"

	"gorm.io/gorm"
)

type UserRepository struct {
	sqlHandler *database.SQLHandler[domain.User, domain.UserFilter]
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	sqlHandler := database.NewSQLHandler[domain.User](db, applyFilter)
	return &UserRepository{
		sqlHandler: sqlHandler,
	}
}

func applyFilter(qb *gorm.DB, filter *domain.UserFilter) *gorm.DB {
	if filter == nil {
		return qb
	}

	if filter.ID != nil {
		qb = qb.Where("id = ?", *filter.ID)
	}
	if filter.IDNe != nil {
		qb = qb.Where("id != ?", *filter.IDNe)
	}
	if len(filter.IDIn) > 0 {
		qb = qb.Where("id IN (?)", filter.IDIn)
	}
	if filter.Email != nil {
		qb = qb.Where("email = ?", *filter.Email)
	}
	if filter.Username != nil {
		qb = qb.Where("username = ?", *filter.Username)
	}
	if filter.Active != nil {
		if *filter.Active {
			qb = qb.Where("status = ?", domain.UserSTTActive)
		} else {
			qb = qb.Where("status != ?", domain.UserSTTActive)
		}
	}
	if filter.Blocked != nil {
		if *filter.Blocked {
			qb = qb.Where("status = ?", domain.UserSTTBanned)
		} else {
			qb = qb.Where("status != ?", domain.UserSTTBanned)
		}
	}
	if filter.SearchTerm != nil && *filter.SearchTerm != "" {
		searchFields := filter.SearchFields
		if len(searchFields) == 0 {
			searchFields = []string{"username", "email", "first_name", "last_name"}
		}

		searchQuery := ""
		searchValues := []interface{}{}
		for i, field := range searchFields {
			if i > 0 {
				searchQuery += " OR "
			}
			searchQuery += field + " ILIKE ?"
			searchValues = append(searchValues, "%"+*filter.SearchTerm+"%")
		}
		qb = qb.Where(searchQuery, searchValues...)
	}
	if filter.IncludeDeleted == nil || !*filter.IncludeDeleted {
		qb = qb.Where("deleted_at = 0")
	}

	return qb
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	return r.sqlHandler.Create(ctx, user)
}

func (r *UserRepository) FindByID(ctx context.Context, userID string, option *domain.FindOneOption) (*domain.User, error) {
	return r.sqlHandler.FindByID(ctx, userID, option)
}

func (r *UserRepository) FindOne(ctx context.Context, filter *domain.UserFilter, option *domain.FindOneOption) (*domain.User, error) {
	return r.sqlHandler.FindOne(ctx, filter, option)
}

func (r *UserRepository) FindMany(ctx context.Context, filter *domain.UserFilter, option *domain.FindManyOption) ([]*domain.User, error) {
	return r.sqlHandler.FindMany(ctx, filter, option)
}

func (r *UserRepository) FindPage(ctx context.Context, filter *domain.UserFilter, option *domain.FindPageOption) ([]*domain.User, *domain.Pagination, error) {
	return r.sqlHandler.FindPage(ctx, filter, option)
}

func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	return r.sqlHandler.Update(ctx, user)
}

func (r *UserRepository) UpdatePassword(ctx context.Context, userID string, newPassword string) error {
	return r.sqlHandler.UpdateFields(ctx, userID, map[string]any{
		"password": newPassword,
	})
}

func (r *UserRepository) Delete(ctx context.Context, userID string) error {
	return r.sqlHandler.DeleteByID(ctx, userID)
}

func (r *UserRepository) Count(ctx context.Context, filter *domain.UserFilter) (int64, error) {
	return r.sqlHandler.Count(ctx, filter)
}
