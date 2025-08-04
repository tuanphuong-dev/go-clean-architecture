package middleware

import (
	"context"
	"fmt"
	"go-clean-arch/common"
	"go-clean-arch/pkg/cache"
	"go-clean-arch/pkg/log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	// Rate limiting parameters
	WindowSize  time.Duration // Time window for rate limiting (e.g., 1 minute)
	MaxRequests int64         // Maximum requests allowed in the window

	// Key generation
	KeyPrefix    string                    // Prefix for cache keys
	KeyGenerator func(*gin.Context) string // Custom key generator function

	// Response configuration
	HeaderRemainingRequests string // Header name for remaining requests
	HeaderRetryAfter        string // Header name for retry after
	HeaderRateLimit         string // Header name for rate limit

	// Skip configuration
	SkipPaths     []string                // Paths to skip rate limiting
	SkipCondition func(*gin.Context) bool // Custom skip condition

	// Error handling
	OnLimitReached func(*gin.Context, RateLimitInfo) // Custom handler when limit is reached
}

// RateLimitInfo contains rate limit status information
type RateLimitInfo struct {
	Key        string
	Limit      int64
	Remaining  int64
	ResetTime  time.Time
	RetryAt    time.Time // Changed from int64 to time.Time
	WindowSize time.Duration
}

// DefaultRateLimitConfig returns a default rate limiting configuration
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		WindowSize:              time.Minute,
		MaxRequests:             100,
		KeyPrefix:               "rate_limit:",
		KeyGenerator:            defaultKeyGenerator,
		HeaderRemainingRequests: "X-RateLimit-Remaining",
		HeaderRetryAfter:        "X-RateLimit-Retry-After",
		HeaderRateLimit:         "X-RateLimit-Limit",
		SkipPaths:               []string{"/health", "/metrics"},
		OnLimitReached:          defaultOnLimitReached,
	}
}

// RateLimit returns a rate limiting middleware
func (m *middlewares) RateLimit(config ...RateLimitConfig) gin.HandlerFunc {
	var cfg RateLimitConfig
	if len(config) > 0 {
		cfg = config[0]
	} else {
		cfg = DefaultRateLimitConfig()
	}

	// Ensure all required fields are properly initialized
	if cfg.KeyGenerator == nil {
		cfg.KeyGenerator = defaultKeyGenerator
	}
	if cfg.OnLimitReached == nil {
		cfg.OnLimitReached = defaultOnLimitReached
	}
	if cfg.KeyPrefix == "" {
		cfg.KeyPrefix = "rate_limit:"
	}
	if cfg.WindowSize == 0 {
		cfg.WindowSize = time.Minute
	}
	if cfg.MaxRequests == 0 {
		cfg.MaxRequests = 100
	}

	// Create skip path map for faster lookup
	skipPaths := make(map[string]bool)
	for _, path := range cfg.SkipPaths {
		skipPaths[path] = true
	}

	return func(c *gin.Context) {
		// Skip rate limiting for specified paths
		if skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		// Skip if custom condition is met
		if cfg.SkipCondition != nil && cfg.SkipCondition(c) {
			c.Next()
			return
		}

		// Generate rate limit key
		key := cfg.KeyPrefix + cfg.KeyGenerator(c)

		// Check and update rate limit
		info, allowed := checkRateLimit(c.Request.Context(), m.cache, key, cfg)

		// Set rate limit headers
		setRateLimitHeaders(c, cfg, info)

		if !allowed {
			cfg.OnLimitReached(c, info)
			return
		}

		c.Next()
	}
}

// RateLimitWithLogger returns a rate limiting middleware with logging support
func (m *middlewares) RateLimitWithLogger(config ...RateLimitConfig) gin.HandlerFunc {
	var cfg RateLimitConfig
	if len(config) > 0 {
		cfg = config[0]
	} else {
		cfg = DefaultRateLimitConfig()
	}

	// Ensure all required fields are properly initialized
	if cfg.KeyGenerator == nil {
		cfg.KeyGenerator = defaultKeyGenerator
	}
	if cfg.OnLimitReached == nil {
		cfg.OnLimitReached = defaultOnLimitReached
	}
	if cfg.KeyPrefix == "" {
		cfg.KeyPrefix = "rate_limit:"
	}
	if cfg.WindowSize == 0 {
		cfg.WindowSize = time.Minute
	}
	if cfg.MaxRequests == 0 {
		cfg.MaxRequests = 100
	}

	// Wrap the OnLimitReached handler to include logging
	originalHandler := cfg.OnLimitReached
	cfg.OnLimitReached = func(c *gin.Context, info RateLimitInfo) {
		if m.logger != nil {
			m.logger.Warn("Rate limit exceeded",
				log.String("key", info.Key),
				log.Int64("limit", info.Limit),
				log.Int64("remaining", info.Remaining),
				log.String("client_ip", c.ClientIP()),
				log.String("path", c.Request.URL.Path),
			)
		}
		originalHandler(c, info)
	}

	return m.RateLimit(cfg)
}

// AuthRateLimits creates rate limits specifically for authentication endpoints
func (m *middlewares) AuthRateLimits() gin.HandlerFunc {
	return m.DifferentLimitsForEndpoints()
}

// APIRateLimits creates general API rate limits
func (m *middlewares) APIRateLimits() gin.HandlerFunc {
	return m.RateLimitWithLogger(RateLimitConfig{
		WindowSize:   time.Minute,
		MaxRequests:  60,
		KeyPrefix:    "api:",
		KeyGenerator: UserKeyGenerator,
		SkipPaths:    []string{"/health", "/metrics"},
	})
}

// AdminRateLimits creates rate limits for admin endpoints
func (m *middlewares) AdminRateLimits() gin.HandlerFunc {
	return m.RateLimitWithLogger(RateLimitConfig{
		WindowSize:   time.Minute,
		MaxRequests:  30,
		KeyPrefix:    "admin:",
		KeyGenerator: UserKeyGenerator,
		SkipCondition: func(c *gin.Context) bool {
			// Skip rate limiting for super admin users
			if userID, exists := c.Get("user_id"); exists {
				// Add logic to check if user is super admin
				_ = userID
				return false // For now, don't skip
			}
			return false
		},
	})
}

// BurstProtection creates burst protection middleware
func (m *middlewares) BurstProtection() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		// Burst rate limit: 10 requests per 10 seconds
		m.RateLimitWithLogger(RateLimitConfig{
			WindowSize:              10 * time.Second,
			MaxRequests:             10,
			KeyPrefix:               "burst:",
			KeyGenerator:            defaultKeyGenerator,
			HeaderRemainingRequests: "X-RateLimit-Burst-Remaining",
			HeaderRetryAfter:        "X-RateLimit-Burst-Retry-After",
			HeaderRateLimit:         "X-RateLimit-Burst-Limit",
			SkipPaths:               []string{"/health", "/metrics"},
			OnLimitReached: func(c *gin.Context, info RateLimitInfo) {
				retryAtISO := ""
				if !info.RetryAt.IsZero() {
					retryAtISO = info.RetryAt.Format(time.RFC3339)
				}

				c.JSON(http.StatusTooManyRequests, gin.H{
					"status":      http.StatusTooManyRequests,
					"code":        "BURST_RATE_LIMIT_EXCEEDED",
					"description": "Too many requests in a short time. Please slow down.",
					"data": gin.H{
						"retry_at":            retryAtISO,
						"retry_after_seconds": int64(time.Until(info.RetryAt).Seconds()),
						"limit":               info.Limit,
						"remaining":           info.Remaining,
					},
				})
				c.Abort()
			},
		}),
		// Sustained rate limit: 100 requests per minute
		m.RateLimitWithLogger(RateLimitConfig{
			WindowSize:              time.Minute,
			MaxRequests:             100,
			KeyPrefix:               "sustained:",
			KeyGenerator:            defaultKeyGenerator,
			HeaderRemainingRequests: "X-RateLimit-Sustained-Remaining",
			HeaderRetryAfter:        "X-RateLimit-Sustained-Retry-After",
			HeaderRateLimit:         "X-RateLimit-Sustained-Limit",
			SkipPaths:               []string{"/health", "/metrics"},
			OnLimitReached:          defaultOnLimitReached,
		}),
	}
}

// DifferentLimitsForEndpoints creates different rate limits for different endpoints
func (m *middlewares) DifferentLimitsForEndpoints() gin.HandlerFunc {
	return func(c *gin.Context) {
		var config RateLimitConfig

		switch {
		case c.Request.URL.Path == "/api/v1/auth/login":
			// Stricter limit for login attempts
			config = RateLimitConfig{
				WindowSize:              5 * time.Minute,
				MaxRequests:             5,
				KeyPrefix:               "login:",
				KeyGenerator:            defaultKeyGenerator,
				HeaderRemainingRequests: "X-RateLimit-Remaining",
				HeaderRetryAfter:        "X-RateLimit-Retry-After",
				HeaderRateLimit:         "X-RateLimit-Limit",
				OnLimitReached: func(c *gin.Context, info RateLimitInfo) {
					retryAtISO := ""
					if !info.RetryAt.IsZero() {
						retryAtISO = info.RetryAt.Format(time.RFC3339)
					}

					c.JSON(http.StatusTooManyRequests, gin.H{
						"status":      http.StatusTooManyRequests,
						"code":        "LOGIN_RATE_LIMIT_EXCEEDED",
						"description": "Too many login attempts. Please try again later.",
						"data": gin.H{
							"retry_at":            retryAtISO,
							"retry_after_seconds": int64(time.Until(info.RetryAt).Seconds()),
							"limit":               info.Limit,
							"remaining":           info.Remaining,
						},
					})
					c.Abort()
				},
			}
		case c.Request.URL.Path == "/api/v1/auth/register":
			// Moderate limit for registration
			config = RateLimitConfig{
				WindowSize:              time.Hour,
				MaxRequests:             3,
				KeyPrefix:               "register:",
				KeyGenerator:            defaultKeyGenerator,
				HeaderRemainingRequests: "X-RateLimit-Remaining",
				HeaderRetryAfter:        "X-RateLimit-Retry-After",
				HeaderRateLimit:         "X-RateLimit-Limit",
				// OnLimitReached is nil here, will use defaultOnLimitReached
			}
		default:
			// Default rate limit for other endpoints
			config = DefaultRateLimitConfig()
		}

		// Apply the rate limit
		rateLimitHandler := m.RateLimitWithLogger(config)
		rateLimitHandler(c)
	}
}

// checkRateLimit checks and updates the rate limit for a given key
func checkRateLimit(ctx context.Context, cache cache.Client, key string, cfg RateLimitConfig) (RateLimitInfo, bool) {
	now := time.Now()
	windowStart := now.Truncate(cfg.WindowSize)
	resetTime := windowStart.Add(cfg.WindowSize)

	// Try to increment the counter
	current, err := cache.Increment(ctx, key, 1, cfg.WindowSize)
	if err != nil {
		// If increment fails, allow the request but log the error
		return RateLimitInfo{
			Key:        key,
			Limit:      cfg.MaxRequests,
			Remaining:  cfg.MaxRequests,
			ResetTime:  resetTime,
			RetryAt:    time.Time{}, // Zero time for no retry restriction
			WindowSize: cfg.WindowSize,
		}, true
	}

	remaining := cfg.MaxRequests - current
	if remaining < 0 {
		remaining = 0
	}

	info := RateLimitInfo{
		Key:        key,
		Limit:      cfg.MaxRequests,
		Remaining:  remaining,
		ResetTime:  resetTime,
		RetryAt:    resetTime, // Use time.Time instead of Unix timestamp
		WindowSize: cfg.WindowSize,
	}

	allowed := current <= cfg.MaxRequests
	return info, allowed
}

// setRateLimitHeaders sets rate limit headers in the response
func setRateLimitHeaders(c *gin.Context, cfg RateLimitConfig, info RateLimitInfo) {
	if cfg.HeaderRateLimit != "" {
		c.Header(cfg.HeaderRateLimit, strconv.FormatInt(info.Limit, 10))
	}

	if cfg.HeaderRemainingRequests != "" {
		c.Header(cfg.HeaderRemainingRequests, strconv.FormatInt(info.Remaining, 10))
	}

	if cfg.HeaderRetryAfter != "" && info.Remaining == 0 && !info.RetryAt.IsZero() {
		retryAfterSeconds := int64(time.Until(info.RetryAt).Seconds())
		if retryAfterSeconds > 0 {
			c.Header(cfg.HeaderRetryAfter, strconv.FormatInt(retryAfterSeconds, 10))
		}
	}
}

// defaultKeyGenerator generates a rate limit key based on client IP
func defaultKeyGenerator(c *gin.Context) string {
	return c.ClientIP()
}

// defaultOnLimitReached handles when rate limit is reached
func defaultOnLimitReached(c *gin.Context, info RateLimitInfo) {
	message := fmt.Sprintf("Too many requests. Limit %d requests per %v", info.Limit, info.WindowSize)

	// Set rate limit headers
	c.Header("X-RateLimit-Limit", strconv.FormatInt(info.Limit, 10))
	c.Header("X-RateLimit-Remaining", "0")

	var retryAfter int64
	var retryAtISO string
	if !info.RetryAt.IsZero() {
		retryAfterDuration := time.Until(info.RetryAt)
		if retryAfterDuration > 0 {
			retryAfter = int64(retryAfterDuration.Seconds())
			c.Header("X-RateLimit-Retry-After", strconv.FormatInt(retryAfter, 10))
		}
		retryAtISO = info.RetryAt.Format(time.RFC3339)
	}

	common.Response(c, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", gin.H{
		"retry_after_seconds": retryAfter,
		"retry_at":            retryAtISO,
		"limit":               info.Limit,
		"remaining":           info.Remaining,
		"window_size":         info.WindowSize.String(),
	}, message)
}

// Per-user rate limiting key generator
func UserKeyGenerator(c *gin.Context) string {
	// Try to get user ID from context (set by auth middleware)
	if userID, exists := c.Get("user_id"); exists {
		return fmt.Sprintf("user:%v", userID)
	}
	// Fallback to IP-based rate limiting
	return fmt.Sprintf("ip:%s", c.ClientIP())
}

// Per-API endpoint rate limiting key generator
func EndpointKeyGenerator(c *gin.Context) string {
	return fmt.Sprintf("endpoint:%s:%s:%s", c.Request.Method, c.FullPath(), c.ClientIP())
}
