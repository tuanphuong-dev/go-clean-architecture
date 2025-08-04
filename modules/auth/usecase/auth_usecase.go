package usecase

import (
	"context"
	"fmt"
	"go-clean-arch/common"
	"go-clean-arch/domain"
	"time"
)

type Hasher interface {
	Hash(password string) (string, error)
	Compare(hashed, password string) bool
}

type JWTProvider interface {
	Generate(tokenType domain.TokenType, userID string, sessionID string) (string, error)
	Verify(tokenType domain.TokenType, tokenStr string) (*domain.JwtClaims, error)
}

type UserSessionRepository interface {
	Create(ctx context.Context, session *domain.UserSession) error
	FindByID(ctx context.Context, sessionID string, option *domain.FindOneOption) (*domain.UserSession, error)
	FindByRefreshToken(ctx context.Context, refreshToken string, option *domain.FindOneOption) (*domain.UserSession, error)
	FindOne(ctx context.Context, filter *domain.UserSessionFilter, option *domain.FindOneOption) (*domain.UserSession, error)
	FindMany(ctx context.Context, filter *domain.UserSessionFilter, option *domain.FindManyOption) ([]*domain.UserSession, error)
	FindPage(ctx context.Context, filter *domain.UserSessionFilter, option *domain.FindPageOption) ([]*domain.UserSession, *domain.Pagination, error)
	Update(ctx context.Context, session *domain.UserSession) error
	InvalidateRefreshToken(ctx context.Context, sessionID string) error
	Delete(ctx context.Context, sessionID string) error
	Count(ctx context.Context, filter *domain.UserSessionFilter) (int64, error)
}

type UserClient interface {
	Create(ctx context.Context, req *domain.UserCreateRequest) (*domain.User, error)
	FindOne(ctx context.Context, filter *domain.UserFilter, option *domain.FindOneOption) (*domain.User, error)
}

type EmailClient interface {
	SendEmailWithTemplate(ctx context.Context, req *domain.SendEmailWithTemplateRequest) (*domain.EmailLog, error)
}

type authUsecase struct {
	sessionRepo    UserSessionRepository
	userClient     UserClient
	emailRPCClient EmailClient
	jwtProvider    JWTProvider
	hasher         Hasher
}

func NewAuthUsecase(
	sessionRepo UserSessionRepository,
	userClient UserClient,
	emailRPCClient EmailClient,
	jwtProvider JWTProvider,
	hasher Hasher,
) domain.AuthUsecase {
	return &authUsecase{
		sessionRepo:    sessionRepo,
		userClient:     userClient,
		emailRPCClient: emailRPCClient,
		jwtProvider:    jwtProvider,
		hasher:         hasher,
	}
}

func (a *authUsecase) Register(ctx context.Context, req *domain.RegisterRequest) (*domain.AuthResponse, error) {
	user, err := a.userClient.Create(ctx, &domain.UserCreateRequest{
		Username:  req.Username,
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	})
	if err != nil {
		if de, ok := common.IsDetailError(err); ok {
			return nil, de
		}
		return nil, domain.ErrUserCreationFailed.WithWrap(err)
	}

	// Generate refresh token
	refreshToken, err := a.jwtProvider.Generate(domain.TokenTypeRefresh, "", "")
	if err != nil {
		return nil, domain.ErrInternalServerError.WithWrap(err)
	}

	session := &domain.UserSession{
		UserID:       user.ID,
		RefreshToken: refreshToken,
		IPAddress:    req.IPAddress,
		UserAgent:    req.UserAgent,
		Active:       true,
	}
	if err := a.sessionRepo.Create(ctx, session); err != nil {
		return nil, domain.ErrCannotCreateSession.WithWrap(err)
	}

	accessToken, err := a.jwtProvider.Generate(domain.TokenTypeAccess, user.ID, session.ID)
	if err != nil {
		return nil, domain.ErrInternalServerError.WithWrap(err)
	}

	// Send verification email after successful registration
	verificationToken, err := a.jwtProvider.Generate(domain.TokenTypeAccess, user.ID, "")
	if err != nil {
		return &domain.AuthResponse{
			User:         user,
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		}, nil
	}

	// Prepare template data for verification email
	templateData := map[string]any{
		"app_name":         "Go Clean Arch",
		"user_name":        user.FirstName + " " + user.LastName,
		"verification_url": "https://your-domain.com/verify-email?token=" + verificationToken,
		"user_email":       user.Email,
		"current_time":     time.Now().Format("2006-01-02 15:04:05"),
	}

	// Send verification email using email RPC client
	emailReq := &domain.SendEmailWithTemplateRequest{
		To:           []string{user.Email},
		TemplateCode: domain.EmailCodeVerification,
		Locale:       "en", // Default locale
		Data:         templateData,
		RequestID:    common.GenerateUUID(),
	}

	// Send email asynchronously - don't block registration if email fails
	go func() {
		if _, err := a.emailRPCClient.SendEmailWithTemplate(context.Background(), emailReq); err != nil {
			// Log error but don't fail the registration process
			// In a production environment, you might want to queue this for retry
		}
	}()

	return &domain.AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (a *authUsecase) Login(ctx context.Context, req *domain.LoginRequest) (*domain.AuthResponse, error) {
	user, err := a.userClient.FindOne(ctx, &domain.UserFilter{
		Email: &req.Email,
	}, &domain.FindOneOption{})
	if err != nil || user == nil {
		return nil, domain.ErrInvalidCredentials
	}

	if !a.hasher.Compare(user.Password, req.Password) {
		return nil, domain.ErrInvalidCredentials
	}

	if user.Status != domain.UserSTTActive {
		return nil, domain.ErrUserInactive
	}

	// Generate refresh token
	refreshToken, err := a.jwtProvider.Generate(domain.TokenTypeRefresh, "", "")
	if err != nil {
		return nil, domain.ErrInternalServerError.WithWrap(err)
	}

	session := &domain.UserSession{
		UserID:       user.ID,
		RefreshToken: refreshToken,
		IPAddress:    req.IPAddress,
		UserAgent:    req.UserAgent,
		Active:       true,
	}
	if err := a.sessionRepo.Create(ctx, session); err != nil {
		return nil, domain.ErrCannotCreateSession.WithWrap(err)
	}

	accessToken, err := a.jwtProvider.Generate(domain.TokenTypeAccess, user.ID, session.ID)
	if err != nil {
		return nil, domain.ErrInternalServerError.WithWrap(err)
	}

	return &domain.AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (a *authUsecase) Logout(ctx context.Context, sessionID string) error {
	session, err := a.sessionRepo.FindByID(ctx, sessionID, nil)
	if err != nil || session == nil {
		return domain.ErrSessionExpired.WithWrap(err)
	}
	if !session.Active {
		return nil // Already logged out
	}
	session.Active = false
	if err := a.sessionRepo.Update(ctx, session); err != nil {
		return domain.ErrInternalServerError.WithWrap(err)
	}
	return nil
}

func (a *authUsecase) RefreshToken(ctx context.Context, req *domain.RefreshTokenRequest) (*domain.AuthResponse, error) {
	// Find session by refresh token
	session, err := a.sessionRepo.FindByRefreshToken(ctx, req.RefreshToken, nil)
	if err != nil || session == nil {
		return nil, domain.ErrInvalidToken.WithError("invalid refresh token")
	}

	if !session.IsActive() {
		return nil, domain.ErrSessionExpired
	}

	// Invalidate the current refresh token (one-time use)
	if err := a.sessionRepo.InvalidateRefreshToken(ctx, session.ID); err != nil {
		return nil, domain.ErrInternalServerError.WithWrap(err)
	}

	// Get user information
	user, err := a.userClient.FindOne(ctx, &domain.UserFilter{
		ID: &session.UserID,
	}, &domain.FindOneOption{})
	if err != nil || user == nil {
		return nil, domain.ErrUserNotFound.WithWrap(err)
	}

	if !user.IsActive() {
		return nil, domain.ErrUserInactive
	}

	// Generate new refresh token
	newRefreshToken, err := a.jwtProvider.Generate(domain.TokenTypeRefresh, "", "")
	if err != nil {
		return nil, domain.ErrInternalServerError.WithWrap(err)
	}

	// Update session with new refresh token and client info
	session.RefreshToken = newRefreshToken
	if req.IPAddress != "" {
		session.IPAddress = req.IPAddress
	}
	if req.UserAgent != "" {
		session.UserAgent = req.UserAgent
	}

	if err := a.sessionRepo.Update(ctx, session); err != nil {
		return nil, domain.ErrInternalServerError.WithWrap(err)
	}

	// Generate new access token
	accessToken, err := a.jwtProvider.Generate(domain.TokenTypeAccess, user.ID, session.ID)
	if err != nil {
		return nil, domain.ErrInternalServerError.WithWrap(err)
	}

	return &domain.AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (a *authUsecase) SendVerificationEmail(ctx context.Context, req *domain.SendVerificationEmailRequest) error {
	// Get user information
	user, err := a.userClient.FindOne(ctx, &domain.UserFilter{
		ID: &req.UserID,
	}, &domain.FindOneOption{})
	if err != nil || user == nil {
		return domain.ErrUserNotFound.WithWrap(err)
	}

	// Prepare template data
	templateData := map[string]interface{}{
		"user_name":        user.FirstName + " " + user.LastName,
		"verification_url": "https://your-domain.com/verify-email?token=" + req.Token,
		"user_email":       user.Email,
		"current_time":     time.Now().Format("2006-01-02 15:04:05"),
	}

	// Send verification email using email template
	emailReq := &domain.SendEmailWithTemplateRequest{
		To:           []string{user.Email},
		TemplateCode: domain.EmailCodeVerification,
		Locale:       "en", // Default locale
		Data:         templateData,
		RequestID:    fmt.Sprintf("auth_verify_%s", req.UserID),
	}

	_, err = a.emailRPCClient.SendEmailWithTemplate(ctx, emailReq)
	if err != nil {
		return domain.ErrEmailSendFailed.WithError("failed to send verification email").WithWrap(err)
	}

	return nil
}

func (a *authUsecase) VerifyEmail(ctx context.Context, req *domain.VerifyEmailRequest) error {
	// TODO: Implement email verification logic
	// This would typically:
	// 1. Validate the verification code/token
	// 2. Update user's email_verified status
	// 3. Invalidate the verification token
	return domain.ErrNotImplemented
}
