package middleware

import (
	"go-clean-arch/domain"
	"go-clean-arch/pkg/cache"
	"go-clean-arch/pkg/log"

	"github.com/gin-gonic/gin"
)

// Middlewares defines all available middleware methods
type Middlewares interface {
	// Rate limiting middlewares
	RateLimit(config ...RateLimitConfig) gin.HandlerFunc
	RateLimitWithLogger(config ...RateLimitConfig) gin.HandlerFunc
	AuthRateLimits() gin.HandlerFunc
	APIRateLimits() gin.HandlerFunc
	AdminRateLimits() gin.HandlerFunc
	BurstProtection() []gin.HandlerFunc
	DifferentLimitsForEndpoints() gin.HandlerFunc

	// Logging middlewares
	LoggingMiddleware(config ...LoggerConfig) gin.HandlerFunc
	DetailedLoggingMiddleware(config LoggerConfig) gin.HandlerFunc
	RequestIDMiddleware() gin.HandlerFunc

	// CORS middlewares
	CORS(config ...CORSConfig) gin.HandlerFunc
	SimpleAllowAllCORS() gin.HandlerFunc
	CORSWithLogger(config ...CORSConfig) gin.HandlerFunc

	// Authentication middlewares
	Authenticator() gin.HandlerFunc
	RequireAnyRoles(roleIDs ...domain.RoleID) gin.HandlerFunc
}

// Dependencies holds all dependencies needed by middlewares
type Dependencies struct {
	Cache       cache.Client
	Logger      log.Logger
	JwtProvider JwtProvider
	SessionRepo SessionRepository
	UserRepo    UserRepository
}

// NewMiddlewares creates a new instance of middlewares with dependencies
func NewMiddlewares(deps Dependencies) Middlewares {
	return &middlewares{
		cache:       deps.Cache,
		logger:      deps.Logger,
		jwtProvider: deps.JwtProvider,
		sessionRepo: deps.SessionRepo,
		userRepo:    deps.UserRepo,
	}
}

// middlewares is the concrete implementation of Middlewares interface
type middlewares struct {
	cache       cache.Client
	logger      log.Logger
	jwtProvider JwtProvider
	sessionRepo SessionRepository
	userRepo    UserRepository
}
