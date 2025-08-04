package log

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type Field = zap.Field

type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	Printf(format string, args ...interface{})
	Println(args ...interface{})
	With(fields ...Field) Logger
	DebugContext(ctx context.Context, msg string, fields ...Field)
	InfoContext(ctx context.Context, msg string, fields ...Field)
	WarnContext(ctx context.Context, msg string, fields ...Field)
	ErrorContext(ctx context.Context, msg string, fields ...Field)
	WithContext(ctx context.Context) Logger
	Sync() error
}

type ContextLogger interface {
	DebugContext(ctx context.Context, msg string, fields ...Field)
	InfoContext(ctx context.Context, msg string, fields ...Field)
	WarnContext(ctx context.Context, msg string, fields ...Field)
	ErrorContext(ctx context.Context, msg string, fields ...Field)
}

func String(key, value string) Field                 { return zap.String(key, value) }
func Int(key string, value int) Field                { return zap.Int(key, value) }
func Int64(key string, value int64) Field            { return zap.Int64(key, value) }
func Float64(key string, value float64) Field        { return zap.Float64(key, value) }
func Bool(key string, value bool) Field              { return zap.Bool(key, value) }
func Time(key string, value time.Time) Field         { return zap.Time(key, value) }
func Duration(key string, value time.Duration) Field { return zap.Duration(key, value) }
func Error(err error) Field                          { return zap.Error(err) }
func Any(key string, value interface{}) Field        { return zap.Any(key, value) }
func UserID(value string) Field                      { return zap.String("user_id", value) }
func RequestID(value string) Field                   { return zap.String("request_id", value) }
func TraceID(value string) Field                     { return zap.String("trace_id", value) }
func Method(value string) Field                      { return zap.String("method", value) }
func URL(value string) Field                         { return zap.String("url", value) }
func StatusCode(value int) Field                     { return zap.Int("status_code", value) }
func ResponseTime(value time.Duration) Field         { return zap.Duration("response_time", value) }

// Global logger instance for convenience
var defaultLogger Logger

// SetDefaultLogger sets the global default logger
func SetDefaultLogger(logger Logger) {
	defaultLogger = logger
}

// GetDefaultLogger returns the global default logger
func GetDefaultLogger() Logger {
	if defaultLogger == nil {
		defaultLogger = MustNewDevelopmentLogger()
	}
	return defaultLogger
}

// Global Printf functions
func Printf(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Printf(format, args...)
	}
}

func Println(args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Println(args...)
	}
}
