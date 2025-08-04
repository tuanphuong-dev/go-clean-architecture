package domain

import (
	"context"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

/****************************
*        Auth errors        *
****************************/
var (
	ErrInvalidCredentials = &DetailedError{
		IDField:         "INVALID_CREDENTIALS",
		StatusDescField: http.StatusText(http.StatusUnauthorized),
		ErrorField:      "Invalid email or password",
		StatusCodeField: http.StatusUnauthorized,
	}
	ErrInvalidToken = &DetailedError{
		IDField:         "INVALID_TOKEN",
		StatusDescField: http.StatusText(http.StatusUnauthorized),
		ErrorField:      "Invalid or expired token",
		StatusCodeField: http.StatusUnauthorized,
	}
	ErrSessionExpired = &DetailedError{
		IDField:         "SESSION_EXPIRED",
		StatusDescField: http.StatusText(http.StatusUnauthorized),
		ErrorField:      "Session has expired",
		StatusCodeField: http.StatusUnauthorized,
	}
	ErrSessionFindFailed = &DetailedError{
		IDField:         "SESSION_FIND_FAILED",
		StatusDescField: http.StatusText(http.StatusInternalServerError),
		ErrorField:      "Failed to find session",
		StatusCodeField: http.StatusInternalServerError,
	}
	ErrEmailNotVerified = &DetailedError{
		IDField:         "EMAIL_NOT_VERIFIED",
		StatusDescField: http.StatusText(http.StatusForbidden),
		ErrorField:      "Email address is not verified",
		StatusCodeField: http.StatusForbidden,
	}
	ErrAccountBanned = &DetailedError{
		IDField:         "ACCOUNT_BANNED",
		StatusDescField: http.StatusText(http.StatusForbidden),
		ErrorField:      "Account has been banned",
		StatusCodeField: http.StatusForbidden,
	}
	ErrPasswordTooWeak = &DetailedError{
		IDField:         "PASSWORD_TOO_WEAK",
		StatusDescField: http.StatusText(http.StatusBadRequest),
		ErrorField:      "Password does not meet security requirements",
		StatusCodeField: http.StatusBadRequest,
	}
	ErrTokenExpired = &DetailedError{
		IDField:         "TOKEN_EXPIRED",
		StatusDescField: http.StatusText(http.StatusUnauthorized),
		ErrorField:      "Token has expired",
		StatusCodeField: http.StatusUnauthorized,
	}
	ErrCannotCreateSession = &DetailedError{
		IDField:         "CANNOT_CREATE_SESSION",
		StatusDescField: http.StatusText(http.StatusInternalServerError),
		ErrorField:      "Failed to create session",
		StatusCodeField: http.StatusInternalServerError,
	}
)

/***************************************
*       Auth entities and types       *
***************************************/

type TokenType int

const (
	TokenTypeAccess TokenType = iota
	TokenTypeRefresh
)

type JwtClaims struct {
	Sub string `json:"sub"` // User ID
	Sid string `json:"sid"` // Session ID
	jwt.RegisteredClaims
}

type UserSession struct {
	SQLModel
	UserID         string `json:"user_id" db:"user_id"`                   // Foreign key reference to User.ID
	RefreshToken   string `json:"refresh_token" db:"refresh_token"`       // One-time refresh token (random string)
	FCMToken       string `json:"fcm_token" db:"fcm_token"`               // Firebase Cloud Messaging token for push notifications
	IPAddress      string `json:"ip_address" db:"ip_address"`             // Client IP address (e.g., "192.168.1.1", "2001:db8::1")
	UserAgent      string `json:"user_agent" db:"user_agent"`             // HTTP User-Agent string from the client browser/app
	Active         bool   `json:"active" db:"active"`                     // Whether the session is currently active (not logged out)
	ExpiresAt      int64  `json:"expires_at" db:"expires_at"`             // When the session expires (absolute timestamp)
	LastActivityAt int64  `json:"last_activity_at" db:"last_activity_at"` // Last time the session was used for any request (timestamp)
}

func (s *UserSession) IsActive() bool {
	return s.Active && (s.ExpiresAt == 0 || s.ExpiresAt > time.Now().UnixMilli())
}

type UserSessionFilter struct {
	ID            *string `json:"id,omitempty"`             // Filter by specific session ID
	UserID        *string `json:"user_id,omitempty"`        // Filter by user ID (find all sessions for a user)
	RefreshToken  *string `json:"refresh_token,omitempty"`  // Filter by refresh token
	FCMToken      *string `json:"fcm_token,omitempty"`      // Filter by FCM token (exact match)
	IPAddress     *string `json:"ip_address,omitempty"`     // Filter by IP address (exact match)
	Active        *bool   `json:"is_active,omitempty"`      // Filter by active status (true=active, false=inactive)
	ExpiresAfter  *int64  `json:"expires_after,omitempty"`  // Find sessions that expire after this timestamp
	ExpiresBefore *int64  `json:"expires_before,omitempty"` // Find sessions that expire before this timestamp
	CreatedAfter  *int64  `json:"created_after,omitempty"`  // Find sessions created after this timestamp
	CreatedBefore *int64  `json:"created_before,omitempty"` // Find sessions created before this timestamp
}

/*************************************
*  Auth usecase interfaces and types *
**************************************/
type AuthUsecase interface {
	Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error)

	Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error)
	Logout(ctx context.Context, sessionID string) error
	RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*AuthResponse, error)

	SendVerificationEmail(ctx context.Context, req *SendVerificationEmailRequest) error
	VerifyEmail(ctx context.Context, req *VerifyEmailRequest) error
}

type RegisterRequest struct {
	Username  string `json:"username" validate:"required,min=3,max=50"`
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=6"`
	FirstName string `json:"first_name" validate:"required,min=1,max=50"`
	LastName  string `json:"last_name" validate:"required,min=1,max=50"`
	IPAddress string `json:"ip_address,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
}

type LoginRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=6"`
	IPAddress string `json:"ip_address,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
	IPAddress    string `json:"ip_address,omitempty"`
	UserAgent    string `json:"user_agent,omitempty"`
}

type AuthResponse struct {
	User         *User  `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type LogoutRequest struct {
	SessionID string `json:"session_id"`
}

type VerifyEmailRequest struct {
	Email string `json:"email" validate:"required,email"`
	Code  string `json:"code" validate:"required"`
}

type SendVerificationEmailRequest struct {
	UserID string
	Token  string // or other fields if needed
}
