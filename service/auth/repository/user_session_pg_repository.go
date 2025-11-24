package repository

import (
	"context"
	"go-clean-arch/database"
	"go-clean-arch/domain"

	"gorm.io/gorm"
)

type UserSessionRepository struct {
	sqlHandler *database.SQLHandler[domain.UserSession, domain.UserSessionFilter]
}

func NewPgUserSessionRepo(db *gorm.DB) *UserSessionRepository {
	sqlHandler := database.NewSQLHandler[domain.UserSession](db, applyFilter)
	return &UserSessionRepository{
		sqlHandler: sqlHandler,
	}
}

func applyFilter(qb *gorm.DB, filter *domain.UserSessionFilter) *gorm.DB {
	if filter == nil {
		return qb
	}

	if filter.ID != nil {
		qb = qb.Where("id = ?", *filter.ID)
	}
	if filter.UserID != nil {
		qb = qb.Where("user_id = ?", *filter.UserID)
	}
	if filter.RefreshToken != nil {
		qb = qb.Where("refresh_token = ?", *filter.RefreshToken)
	}
	if filter.FCMToken != nil {
		qb = qb.Where("fcm_token = ?", *filter.FCMToken)
	}
	if filter.IPAddress != nil {
		qb = qb.Where("ip_address = ?", *filter.IPAddress)
	}
	if filter.Active != nil {
		qb = qb.Where("active = ?", *filter.Active)
	}
	if filter.ExpiresAfter != nil {
		qb = qb.Where("expires_at > ?", *filter.ExpiresAfter)
	}
	if filter.ExpiresBefore != nil {
		qb = qb.Where("expires_at < ?", *filter.ExpiresBefore)
	}
	if filter.CreatedAfter != nil {
		qb = qb.Where("created_at >= ?", *filter.CreatedAfter)
	}
	if filter.CreatedBefore != nil {
		qb = qb.Where("created_at <= ?", *filter.CreatedBefore)
	}

	return qb
}

func (r *UserSessionRepository) Create(ctx context.Context, session *domain.UserSession) error {
	return r.sqlHandler.Create(ctx, session)
}

func (r *UserSessionRepository) FindByID(ctx context.Context, sessionID string, option *domain.FindOneOption) (*domain.UserSession, error) {
	return r.sqlHandler.FindByID(ctx, sessionID, option)
}

func (r *UserSessionRepository) FindOne(ctx context.Context, filter *domain.UserSessionFilter, option *domain.FindOneOption) (*domain.UserSession, error) {
	return r.sqlHandler.FindOne(ctx, filter, option)
}

func (r *UserSessionRepository) FindMany(ctx context.Context, filter *domain.UserSessionFilter, option *domain.FindManyOption) ([]*domain.UserSession, error) {
	return r.sqlHandler.FindMany(ctx, filter, option)
}

func (r *UserSessionRepository) FindPage(ctx context.Context, filter *domain.UserSessionFilter, option *domain.FindPageOption) ([]*domain.UserSession, *domain.Pagination, error) {
	return r.sqlHandler.FindPage(ctx, filter, option)
}

func (r *UserSessionRepository) Update(ctx context.Context, session *domain.UserSession) error {
	return r.sqlHandler.Update(ctx, session)
}

func (r *UserSessionRepository) Delete(ctx context.Context, sessionID string) error {
	return r.sqlHandler.DeleteByID(ctx, sessionID)
}

func (r *UserSessionRepository) Count(ctx context.Context, filter *domain.UserSessionFilter) (int64, error) {
	return r.sqlHandler.Count(ctx, filter)
}

func (r *UserSessionRepository) FindByRefreshToken(ctx context.Context, refreshToken string, option *domain.FindOneOption) (*domain.UserSession, error) {
	return r.sqlHandler.FindOne(ctx, &domain.UserSessionFilter{
		RefreshToken: &refreshToken,
	}, option)
}

func (r *UserSessionRepository) InvalidateRefreshToken(ctx context.Context, sessionID string) error {
	return r.sqlHandler.UpdateFields(ctx, sessionID, map[string]any{
		"refresh_token": "",
	})
}
