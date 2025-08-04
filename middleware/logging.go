package middleware

import (
	"bytes"
	"go-clean-arch/pkg/log"
	"io"
	"time"

	"github.com/gin-gonic/gin"
)

type LoggerConfig struct {
	// SkipPaths is an url path array which logs are not written.
	// Optional.
	SkipPaths []string

	// SkipPathRegexps is an url path regexp array which logs are not written.
	// Optional.
	SkipPathRegexps []string

	// EnableRequestBody enables logging of request body.
	// Optional. Default value is false.
	EnableRequestBody bool

	// EnableResponseBody enables logging of response body.
	// Optional. Default value is false.
	EnableResponseBody bool

	// MaxBodySize sets the maximum size of request/response body to log.
	// Optional. Default value is 4096 bytes.
	MaxBodySize int
}

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// LoggingMiddleware returns a gin.HandlerFunc (middleware) that logs requests using the provided logger.
func (m *middlewares) LoggingMiddleware(config ...LoggerConfig) gin.HandlerFunc {
	var conf LoggerConfig
	if len(config) > 0 {
		conf = config[0]
	}

	// Set default values
	if conf.MaxBodySize == 0 {
		conf.MaxBodySize = 4096
	}

	// Create skip path map for faster lookup
	skipPaths := make(map[string]bool)
	for _, path := range conf.SkipPaths {
		skipPaths[path] = true
	}

	return gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: func(param gin.LogFormatterParams) string {
			// Skip logging for specified paths
			if skipPaths[param.Path] {
				return ""
			}

			// Calculate response time
			latency := param.Latency
			if latency > time.Minute {
				latency = latency.Truncate(time.Second)
			}

			// Prepare log fields
			fields := []log.Field{
				log.String("method", param.Method),
				log.String("path", param.Path),
				log.String("protocol", param.Request.Proto),
				log.Int("status_code", param.StatusCode),
				log.Duration("latency", latency),
				log.String("client_ip", param.ClientIP),
				log.String("user_agent", param.Request.UserAgent()),
			}

			// Add request ID if available
			if requestID := param.Request.Header.Get("X-Request-ID"); requestID != "" {
				fields = append(fields, log.String("request_id", requestID))
			}

			// Add error if present
			if param.ErrorMessage != "" {
				fields = append(fields, log.String("error", param.ErrorMessage))
			}

			// Determine log level based on status code
			m.logger.Info("HTTP Request", fields...)

			return "" // Return empty string since we're handling logging ourselves
		},
		Output: io.Discard, // Discard default gin output since we're using our logger
	})
}

// DetailedLoggingMiddleware provides more detailed logging including request/response bodies
func (m *middlewares) DetailedLoggingMiddleware(config LoggerConfig) gin.HandlerFunc {
	// Set default values
	if config.MaxBodySize == 0 {
		config.MaxBodySize = 4096
	}

	// Create skip path map for faster lookup
	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return func(c *gin.Context) {
		// Skip logging for specified paths
		if skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Read request body if enabled
		var requestBody []byte
		if config.EnableRequestBody && c.Request.Body != nil {
			requestBody, _ = io.ReadAll(io.LimitReader(c.Request.Body, int64(config.MaxBodySize)))
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// Wrap response writer to capture response body
		var responseBody *bytes.Buffer
		if config.EnableResponseBody {
			responseBody = &bytes.Buffer{}
			c.Writer = &bodyLogWriter{
				ResponseWriter: c.Writer,
				body:           responseBody,
			}
		}

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)
		if latency > time.Minute {
			latency = latency.Truncate(time.Second)
		}

		// Build full path
		if raw != "" {
			path = path + "?" + raw
		}

		// Prepare log fields
		fields := []log.Field{
			log.String("method", c.Request.Method),
			log.String("path", path),
			log.String("protocol", c.Request.Proto),
			log.Int("status_code", c.Writer.Status()),
			log.Duration("latency", latency),
			log.String("client_ip", c.ClientIP()),
			log.String("user_agent", c.Request.UserAgent()),
			log.Int("response_size", c.Writer.Size()),
		}

		// Add request ID if available
		if requestID := c.GetHeader("X-Request-ID"); requestID != "" {
			fields = append(fields, log.String("request_id", requestID))
		}

		// Add user ID if available in context
		if userID, exists := c.Get("user_id"); exists {
			fields = append(fields, log.String("user_id", userID.(string)))
		}

		// Add request body if enabled and present
		if config.EnableRequestBody && len(requestBody) > 0 {
			fields = append(fields, log.String("request_body", string(requestBody)))
		}

		// Add response body if enabled and present
		if config.EnableResponseBody && responseBody != nil {
			body := responseBody.String()
			if len(body) > config.MaxBodySize {
				body = body[:config.MaxBodySize] + "...[truncated]"
			}
			fields = append(fields, log.String("response_body", body))
		}

		// Add errors if present
		if len(c.Errors) > 0 {
			errorMsgs := make([]string, len(c.Errors))
			for i, err := range c.Errors {
				errorMsgs[i] = err.Error()
			}
			fields = append(fields, log.Any("errors", errorMsgs))
		}

		// Determine log level and message based on status code
		statusCode := c.Writer.Status()
		message := "HTTP Request Completed"

		if statusCode >= 500 {
			m.logger.Error(message, fields...)
		} else if statusCode >= 400 {
			m.logger.Warn(message, fields...)
		} else {
			m.logger.Info(message, fields...)
		}
	}
}

// RequestIDMiddleware adds a request ID to each request
func (m *middlewares) RequestIDMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		// Add request ID if not present
		if c.GetHeader("X-Request-ID") == "" {
			// Generate a simple request ID (in production, use a proper UUID generator)
			requestID := generateRequestID()
			c.Header("X-Request-ID", requestID)
			c.Set("request_id", requestID)
		}
		c.AbortWithStatus(500)
	})
}

// Simple request ID generator (replace with proper UUID in production)
func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + "req"
}
