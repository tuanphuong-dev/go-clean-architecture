package middleware

import (
	"fmt"
	"go-clean-arch/pkg/log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCORSConfig returns a default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodHead,
			http.MethodOptions,
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Length",
			"Content-Type",
			"Authorization",
			"Accept",
			"Accept-Encoding",
			"Accept-Language",
			"Cache-Control",
			"Connection",
			"DNT",
			"Host",
			"If-Modified-Since",
			"Keep-Alive",
			"User-Agent",
			"X-Requested-With",
			"X-Request-ID",
			"X-Real-IP",
			"X-Forwarded-For",
			"X-Forwarded-Proto",
		},
		ExposeHeaders: []string{
			"Content-Length",
			"X-Request-ID",
		},
		AllowCredentials: true,
		MaxAge:           86400, // 24 hours
	}
}

// CORS returns a middleware that handles CORS
func (m *middlewares) CORS(config ...CORSConfig) gin.HandlerFunc {
	var cfg CORSConfig
	if len(config) > 0 {
		cfg = config[0]
	} else {
		cfg = DefaultCORSConfig()
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Set Access-Control-Allow-Origin
		if len(cfg.AllowOrigins) == 1 && cfg.AllowOrigins[0] == "*" {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if origin != "" && isOriginAllowed(origin, cfg.AllowOrigins) {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		// Set Access-Control-Allow-Methods
		if len(cfg.AllowMethods) > 0 {
			c.Header("Access-Control-Allow-Methods", joinHeaders(cfg.AllowMethods))
		}

		// Set Access-Control-Allow-Headers
		if len(cfg.AllowHeaders) > 0 {
			c.Header("Access-Control-Allow-Headers", joinHeaders(cfg.AllowHeaders))
		}

		// Set Access-Control-Expose-Headers
		if len(cfg.ExposeHeaders) > 0 {
			c.Header("Access-Control-Expose-Headers", joinHeaders(cfg.ExposeHeaders))
		}

		// Set Access-Control-Allow-Credentials
		if cfg.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// Set Access-Control-Max-Age
		if cfg.MaxAge > 0 {
			c.Header("Access-Control-Max-Age", fmt.Sprintf("%d", cfg.MaxAge))
		}

		// Handle preflight requests
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// SimpleAllowAllCORS returns a simple CORS middleware that allows all origins
func (m *middlewares) SimpleAllowAllCORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Expose-Headers", "Content-Length")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// CORSWithLogger returns a CORS middleware with logging support
func (m *middlewares) CORSWithLogger(config ...CORSConfig) gin.HandlerFunc {
	var cfg CORSConfig
	if len(config) > 0 {
		cfg = config[0]
	} else {
		cfg = DefaultCORSConfig()
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		method := c.Request.Method

		// Log CORS request
		if origin != "" {
			m.logger.Debug("CORS request",
				log.String("origin", origin),
				log.String("method", method),
				log.String("path", c.Request.URL.Path),
			)
		}

		// Set Access-Control-Allow-Origin
		if len(cfg.AllowOrigins) == 1 && cfg.AllowOrigins[0] == "*" {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if origin != "" && isOriginAllowed(origin, cfg.AllowOrigins) {
			c.Header("Access-Control-Allow-Origin", origin)
		} else if origin != "" {
			m.logger.Warnf("CORS request from disallowed origin: %s", origin)
		}

		// Set other CORS headers
		if len(cfg.AllowMethods) > 0 {
			c.Header("Access-Control-Allow-Methods", joinHeaders(cfg.AllowMethods))
		}

		if len(cfg.AllowHeaders) > 0 {
			c.Header("Access-Control-Allow-Headers", joinHeaders(cfg.AllowHeaders))
		}

		if len(cfg.ExposeHeaders) > 0 {
			c.Header("Access-Control-Expose-Headers", joinHeaders(cfg.ExposeHeaders))
		}

		if cfg.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if cfg.MaxAge > 0 {
			c.Header("Access-Control-Max-Age", fmt.Sprintf("%d", cfg.MaxAge))
		}

		// Handle preflight requests
		if method == http.MethodOptions {
			m.logger.Debug("CORS preflight request handled",
				log.String("origin", origin),
			)
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// Helper functions

// isOriginAllowed checks if the origin is in the allowed origins list
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowedOrigin := range allowedOrigins {
		if allowedOrigin == "*" || allowedOrigin == origin {
			return true
		}
		// Add wildcard matching if needed in the future
		// if matched, _ := filepath.Match(allowedOrigin, origin); matched {
		//     return true
		// }
	}
	return false
}

// joinHeaders joins a slice of strings with comma separator
func joinHeaders(headers []string) string {
	if len(headers) == 0 {
		return ""
	}

	result := headers[0]
	for i := 1; i < len(headers); i++ {
		result += ", " + headers[i]
	}
	return result
}
