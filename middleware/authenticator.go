package middleware

import (
	"context"
	"strings"

	"go-clean-arch/common"
	"go-clean-arch/domain"

	"github.com/gin-gonic/gin"
)

type JwtProvider interface {
	Verify(tokenType domain.TokenType, tokenStr string) (*domain.JwtClaims, error)
}

type SessionRepository interface {
	FindByID(ctx context.Context, sessionID string, option *domain.FindOneOption) (*domain.UserSession, error)
}

type UserRepository interface {
	FindByID(ctx context.Context, userID string, option *domain.FindOneOption) (*domain.User, error)
}

type headerData struct {
	AccessToken string
}

func extractHeaderData(c *gin.Context) *headerData {
	hData := &headerData{}

	authHeader := c.GetHeader("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		hData.AccessToken = strings.TrimPrefix(authHeader, "Bearer ")
	}

	return hData
}

func (m *middlewares) Authenticator() gin.HandlerFunc {
	return func(c *gin.Context) {
		headerData := extractHeaderData(c)

		claims, err := m.jwtProvider.Verify(domain.TokenTypeAccess, headerData.AccessToken)
		if err != nil {
			common.ResponseError(c, domain.ErrInvalidToken.WithWrap(err))
			return
		}

		session, err := m.sessionRepo.FindByID(c.Request.Context(), claims.Sid, nil)
		if err != nil && !common.IsRecordNotFound(err) {
			common.ResponseError(c, err)
			return
		}
		if session == nil || !session.IsActive() {
			common.ResponseError(c, domain.ErrSessionExpired)
			return
		}

		user, err := m.userRepo.FindByID(c.Request.Context(), claims.Sub, &domain.FindOneOption{
			Preloads: []string{common.FieldRoles},
		})
		if err != nil && !common.IsRecordNotFound(err) {
			common.ResponseError(c, err)
		}
		if user == nil {
			common.ResponseError(c, domain.ErrUserNotFound)
			return
		}

		if user.IsBanned() {
			common.ResponseError(c, domain.ErrAccountBanned)
			return
		}

		c.Set(common.UserContextKey, user)
		c.Set(common.SessionIDContextKey, session.ID)
		c.Next()
	}
}

func (m *middlewares) RequireAnyRoles(roleIDs ...domain.RoleID) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, exists := c.Get(common.UserContextKey)
		if !exists {
			common.ResponseForbidden(c, "User context not found")
			return
		}

		user, ok := v.(*domain.User)
		if !ok {
			common.ResponseForbidden(c, "Invalid user context type")
			return
		}

		if !user.HasAnyRole(roleIDs...) {
			roleStrs := make([]string, len(roleIDs))
			for i, r := range roleIDs {
				roleStrs[i] = string(r)
			}
			common.ResponseForbidden(c, "User does not have required roles: "+strings.Join(roleStrs, ", "))
			return
		}

		c.Next()
	}
}
