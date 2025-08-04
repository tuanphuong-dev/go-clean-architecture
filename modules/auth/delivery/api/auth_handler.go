package api

import (
	"go-clean-arch/common"
	"go-clean-arch/domain"
	"go-clean-arch/middleware"
	"time"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	usecase     domain.AuthUsecase
	middlewares middleware.Middlewares
}

func NewAuthHandler(
	usecase domain.AuthUsecase,
	middlewares middleware.Middlewares,
) *AuthHandler {
	return &AuthHandler{
		usecase:     usecase,
		middlewares: middlewares,
	}
}

func (h *AuthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")

	// Public routes
	auth.POST("/register", h.Register)
	auth.POST("/login", h.Login)

	// Refresh token with specific rate limiting
	auth.POST("/refresh-token", h.refreshTokenRateLimit(), h.RefreshToken)

	auth.POST("/verify-email", h.VerifyEmail)

	// Protected routes (authentication required)
	protected := auth.Group("")
	protected.Use(h.middlewares.Authenticator())
	{
		protected.POST("/logout", h.Logout)
		protected.POST("/send-verification-email", h.SendVerificationEmail)
	}
}

// refreshTokenRateLimit creates specific rate limiting for refresh token endpoint
func (h *AuthHandler) refreshTokenRateLimit() gin.HandlerFunc {
	return h.middlewares.RateLimitWithLogger(middleware.RateLimitConfig{
		WindowSize:  1 * time.Minute, // 1 minute window
		MaxRequests: 1,               // Max 1 refresh attempt per minute
		KeyPrefix:   "refresh_token:",
		KeyGenerator: func(c *gin.Context) string {
			// Rate limit by IP address
			return c.ClientIP()
		},
		HeaderRemainingRequests: "X-RateLimit-Remaining",
		HeaderRetryAfter:        "X-RateLimit-Retry-After",
		HeaderRateLimit:         "X-RateLimit-Limit",
	})
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req domain.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ResponseBadRequest(c, err.Error())
		return
	}
	common.PopulateClientInfo(c, &req.IPAddress, &req.UserAgent)

	resp, err := h.usecase.Register(c.Request.Context(), &req)
	if err != nil {
		common.ResponseError(c, err)
		return
	}
	common.ResponseCreated(c, resp, "Register successful")
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ResponseBadRequest(c, err.Error())
		return
	}
	common.PopulateClientInfo(c, &req.IPAddress, &req.UserAgent)

	resp, err := h.usecase.Login(c.Request.Context(), &req)
	if err != nil {
		common.ResponseError(c, err)
		return
	}
	common.ResponseOK(c, resp, "Login successful")
}

func (h *AuthHandler) Logout(c *gin.Context) {
	sessionID := common.GetSessionIDFromCtx(c)
	if sessionID == "" {
		common.ResponseError(c, domain.ErrUnauthorized)
		return
	}

	if err := h.usecase.Logout(c.Request.Context(), sessionID); err != nil {
		common.ResponseError(c, err)
		return
	}
	common.ResponseOK(c, true, "Logout successful")
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req domain.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ResponseBadRequest(c, err.Error())
		return
	}
	common.PopulateClientInfo(c, &req.IPAddress, &req.UserAgent)

	resp, err := h.usecase.RefreshToken(c.Request.Context(), &req)
	if err != nil {
		common.ResponseError(c, err)
		return
	}
	common.ResponseOK(c, resp, "Token refreshed")
}

func (h *AuthHandler) SendVerificationEmail(c *gin.Context) {
	var req domain.SendVerificationEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ResponseBadRequest(c, err.Error())
		return
	}
	if err := h.usecase.SendVerificationEmail(c.Request.Context(), &req); err != nil {
		common.ResponseBadRequest(c, err.Error())
		return
	}
	common.ResponseNoContent(c, "Verification email sent")
}

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req domain.VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ResponseBadRequest(c, err.Error())
		return
	}
	if err := h.usecase.VerifyEmail(c.Request.Context(), &req); err != nil {
		common.ResponseBadRequest(c, err.Error())
		return
	}
	common.ResponseNoContent(c, "Email verified")
}
