package repository

import (
	"context"
	"go-clean-arch/database"
	"go-clean-arch/domain"

	"gorm.io/gorm"
)

type EmailTemplateRepository struct {
	sqlHandler *database.SQLHandler[domain.EmailTemplate, domain.EmailTemplateFilter]
}

func NewEmailTemplateRepository(db *gorm.DB) *EmailTemplateRepository {
	sqlHandler := database.NewSQLHandler[domain.EmailTemplate](db, applyEmailTemplateFilter)
	return &EmailTemplateRepository{
		sqlHandler: sqlHandler,
	}
}

func applyEmailTemplateFilter(qb *gorm.DB, filter *domain.EmailTemplateFilter) *gorm.DB {
	if filter == nil {
		return qb
	}

	if filter.ID != nil {
		qb = qb.Where("id = ?", *filter.ID)
	}

	if filter.Code != nil {
		qb = qb.Where("code = ?", *filter.Code)
	}

	if filter.Name != nil {
		qb = qb.Where("name = ?", *filter.Name)
	}

	if filter.Locale != nil {
		qb = qb.Where("locale = ?", *filter.Locale)
	}

	if filter.IsActive != nil {
		qb = qb.Where("is_active = ?", *filter.IsActive)
	}

	// Search functionality
	if filter.SearchTerm != nil && *filter.SearchTerm != "" {
		searchFields := filter.SearchFields
		if len(searchFields) == 0 {
			searchFields = []string{"name", "subject", "description"}
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

		if searchQuery != "" {
			qb = qb.Where("("+searchQuery+")", searchValues...)
		}
	}

	// Default: exclude soft deleted records
	if filter.IncludeDeleted == nil || !*filter.IncludeDeleted {
		qb = qb.Where("deleted_at = 0")
	}

	return qb
}

func (r *EmailTemplateRepository) Create(ctx context.Context, template *domain.EmailTemplate) error {
	return r.sqlHandler.Create(ctx, template)
}

func (r *EmailTemplateRepository) FindByID(ctx context.Context, templateID string, option *domain.FindOneOption) (*domain.EmailTemplate, error) {
	return r.sqlHandler.FindByID(ctx, templateID, option)
}

func (r *EmailTemplateRepository) FindByCodeAndLocale(ctx context.Context, code domain.EmailCode, locale string, option *domain.FindOneOption) (*domain.EmailTemplate, error) {
	filter := &domain.EmailTemplateFilter{
		Code:   &code,
		Locale: &locale,
	}
	return r.sqlHandler.FindOne(ctx, filter, option)
}

func (r *EmailTemplateRepository) FindOne(ctx context.Context, filter *domain.EmailTemplateFilter, option *domain.FindOneOption) (*domain.EmailTemplate, error) {
	return r.sqlHandler.FindOne(ctx, filter, option)
}

func (r *EmailTemplateRepository) FindMany(ctx context.Context, filter *domain.EmailTemplateFilter, option *domain.FindManyOption) ([]*domain.EmailTemplate, error) {
	return r.sqlHandler.FindMany(ctx, filter, option)
}

func (r *EmailTemplateRepository) FindPage(ctx context.Context, filter *domain.EmailTemplateFilter, option *domain.FindPageOption) ([]*domain.EmailTemplate, *domain.Pagination, error) {
	return r.sqlHandler.FindPage(ctx, filter, option)
}

func (r *EmailTemplateRepository) Update(ctx context.Context, template *domain.EmailTemplate) error {
	return r.sqlHandler.Update(ctx, template)
}

func (r *EmailTemplateRepository) UpdateFields(ctx context.Context, id string, fields map[string]any) error {
	return r.sqlHandler.UpdateFields(ctx, id, fields)
}

func (r *EmailTemplateRepository) Delete(ctx context.Context, templateID string) error {
	return r.sqlHandler.DeleteByID(ctx, templateID)
}

func (r *EmailTemplateRepository) Count(ctx context.Context, filter *domain.EmailTemplateFilter) (int64, error) {
	return r.sqlHandler.Count(ctx, filter)
}

// Template-specific methods
func (r *EmailTemplateRepository) ActivateTemplate(ctx context.Context, templateID string) error {
	return r.sqlHandler.UpdateFields(ctx, templateID, map[string]any{
		"is_active": true,
	})
}

func (r *EmailTemplateRepository) DeactivateTemplate(ctx context.Context, templateID string) error {
	return r.sqlHandler.UpdateFields(ctx, templateID, map[string]any{
		"is_active": false,
	})
}

func (r *EmailTemplateRepository) GetActiveTemplatesByCode(ctx context.Context, code domain.EmailCode) ([]*domain.EmailTemplate, error) {
	isActive := true
	filter := &domain.EmailTemplateFilter{
		Code:     &code,
		IsActive: &isActive,
	}
	return r.sqlHandler.FindMany(ctx, filter, nil)
}
