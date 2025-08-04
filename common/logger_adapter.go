package common

import (
	"fmt"
	"go-clean-arch/pkg/log"
)

type Logger interface {
	Info(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Debug(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Printf(format string, args ...interface{})
	Println(args ...interface{})
}

// LoggerAdapter adapts pkg/log.Logger to common.Logger interface
type LoggerAdapter struct {
	logger log.Logger
}

// NewLoggerAdapter creates a new adapter that wraps pkg/log.Logger
func NewLoggerAdapter(logger log.Logger) Logger {
	return &LoggerAdapter{logger: logger}
}

// Error implements common.Logger interface
func (a *LoggerAdapter) Error(msg string, fields ...interface{}) {
	// Convert interface{} fields to log.Field format
	logFields := make([]log.Field, 0, len(fields)/2)

	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			key := fmt.Sprintf("%v", fields[i])
			value := fields[i+1]
			logFields = append(logFields, log.Any(key, value))
		}
	}

	a.logger.Error(msg, logFields...)
}

// Printf implements common.Logger interface
func (a *LoggerAdapter) Printf(format string, args ...interface{}) {
	a.logger.Printf(format, args...)
}

// Println implements common.Logger interface
func (a *LoggerAdapter) Println(args ...interface{}) {
	a.logger.Println(args...)
}

// Info implements common.Logger interface
func (a *LoggerAdapter) Info(msg string, fields ...interface{}) {
	// Convert interface{} fields to log.Field format
	logFields := make([]log.Field, 0, len(fields)/2)

	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			key := fmt.Sprintf("%v", fields[i])
			value := fields[i+1]
			logFields = append(logFields, log.Any(key, value))
		}
	}

	a.logger.Info(msg, logFields...)
}

// Debug implements common.Logger interface
func (a *LoggerAdapter) Debug(msg string, fields ...interface{}) {
	// Convert interface{} fields to log.Field format
	logFields := make([]log.Field, 0, len(fields)/2)

	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			key := fmt.Sprintf("%v", fields[i])
			value := fields[i+1]
			logFields = append(logFields, log.Any(key, value))
		}
	}

	a.logger.Debug(msg, logFields...)
}

// Debug implements common.Logger interface
func (a *LoggerAdapter) Warn(msg string, fields ...interface{}) {
	// Convert interface{} fields to log.Field format
	logFields := make([]log.Field, 0, len(fields)/2)

	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			key := fmt.Sprintf("%v", fields[i])
			value := fields[i+1]
			logFields = append(logFields, log.Any(key, value))
		}
	}

	a.logger.Warn(msg, logFields...)
}

// Infof implements common.Logger interface
func (a *LoggerAdapter) Infof(format string, args ...interface{}) {
	a.logger.Infof(format, args...)
}

// Errorf implements common.Logger interface
func (a *LoggerAdapter) Errorf(format string, args ...interface{}) {
	a.logger.Errorf(format, args...)
}

// Debugf implements common.Logger interface
func (a *LoggerAdapter) Debugf(format string, args ...interface{}) {
	a.logger.Debugf(format, args...)
}

// Warnf implements common.Logger interface
func (a *LoggerAdapter) Warnf(format string, args ...interface{}) {
	a.logger.Warnf(format, args...)
}
