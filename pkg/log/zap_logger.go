package log

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type ZapLogger struct {
	logger *zap.Logger
	config Config
}

func NewZapLogger(config Config) (Logger, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid logger config: %w", err)
	}

	level, err := zapcore.ParseLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level '%s': %w", config.Level, err)
	}

	var encoderConfig zapcore.EncoderConfig
	if config.Environment == "production" {
		encoderConfig = zap.NewProductionEncoderConfig()
	} else {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
	}

	encoderConfig.TimeKey = "timestamp"
	encoderConfig.LevelKey = "level"
	encoderConfig.NameKey = "logger"
	encoderConfig.CallerKey = "caller"
	encoderConfig.MessageKey = "message"
	encoderConfig.StacktraceKey = "stacktrace"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	encoderConfig.EncodeDuration = zapcore.StringDurationEncoder

	var encoder zapcore.Encoder
	if strings.ToLower(config.Format) == "console" {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	writeSyncer, err := createWriteSyncer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create write syncer: %w", err)
	}

	core := zapcore.NewCore(encoder, writeSyncer, level)

	var options []zap.Option

	if !config.DisableCaller {
		options = append(options, zap.AddCaller(), zap.AddCallerSkip(1))
	}

	if !config.DisableStacktrace {
		options = append(options, zap.AddStacktrace(zapcore.ErrorLevel))
	}

	if config.SamplingConfig != nil {
		tick := config.SamplingConfig.Tick
		if tick == 0 {
			tick = time.Second
		}
		core = zapcore.NewSamplerWithOptions(
			core,
			tick,
			config.SamplingConfig.Initial,
			config.SamplingConfig.Thereafter,
		)
	}

	if len(config.InitialFields) > 0 {
		var fields []zap.Field
		for key, value := range config.InitialFields {
			fields = append(fields, zap.Any(key, value))
		}
		fields = append(fields,
			zap.String("service", config.ServiceName),
			//zap.String("version", config.Version),
			//zap.String("environment", config.Environment),
		)
		options = append(options, zap.Fields(fields...))
	} else {
		options = append(options, zap.Fields(
			zap.String("service", config.ServiceName),
			//zap.String("version", config.Version),
			//zap.String("environment", config.Environment),
		))
	}

	logger := zap.New(core, options...)

	return &ZapLogger{
		logger: logger,
		config: config,
	}, nil
}
func createWriteSyncer(config Config) (zapcore.WriteSyncer, error) {
	outputsMap := make(map[string]zapcore.WriteSyncer)

	// Add output syncer
	outputSyncer, err := getSyncer(config.OutputPath, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create output syncer: %w", err)
	}
	outputsMap[config.OutputPath] = outputSyncer

	outputs := make([]zapcore.WriteSyncer, 0, len(outputsMap))
	for _, syncer := range outputsMap {
		outputs = append(outputs, syncer)
	}

	if len(outputs) == 1 {
		return outputs[0], nil
	}
	return zapcore.NewMultiWriteSyncer(outputs...), nil
}

// Helper function to create a syncer based on path
func getSyncer(path string, config Config) (zapcore.WriteSyncer, error) {
	switch path {
	case "stdout":
		return zapcore.AddSync(os.Stdout), nil
	case "stderr":
		return zapcore.AddSync(os.Stderr), nil
	default:
		fileWriter, err := createRotatingFileWriter(path, config)
		if err != nil {
			return nil, err
		}
		return zapcore.AddSync(fileWriter), nil
	}
}

// createRotatingFileWriter creates a lumberjack writer for file rotation
func createRotatingFileWriter(filePath string, config Config) (*lumberjack.Logger, error) {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory '%s': %w", dir, err)
	}

	return &lumberjack.Logger{
		Filename:   filePath,
		MaxSize:    config.FileMaxSizeInMB,
		MaxAge:     config.FileMaxAgeInDays,
		MaxBackups: config.FileMaxBackups,
		Compress:   config.CompressRotated,
	}, nil
}

func NewDevelopmentLogger() (Logger, error) {
	config := DevelopmentConfig()
	return NewZapLogger(config)
}

func NewProductionLogger(serviceName, version string) (Logger, error) {
	config := ProductionConfig(serviceName, version)
	return NewZapLogger(config)
}

func MustNewDevelopmentLogger() Logger {
	logger, err := NewDevelopmentLogger()
	if err != nil {
		panic(fmt.Sprintf("failed to create development logger: %v", err))
	}
	return logger
}

func MustNewProductionLogger(serviceName, version string) Logger {
	logger, err := NewProductionLogger(serviceName, version)
	if err != nil {
		panic(fmt.Sprintf("failed to create production logger: %v", err))
	}
	return logger
}

func (l *ZapLogger) Debug(msg string, fields ...Field) {
	l.logger.Debug(msg, fields...)
}

func (l *ZapLogger) Info(msg string, fields ...Field) {
	l.logger.Info(msg, fields...)
}

func (l *ZapLogger) Warn(msg string, fields ...Field) {
	l.logger.Warn(msg, fields...)
}

func (l *ZapLogger) Error(msg string, fields ...Field) {
	l.logger.Error(msg, fields...)
}

func (l *ZapLogger) Fatal(msg string, fields ...Field) {
	l.logger.Fatal(msg, fields...)
}

func (l *ZapLogger) With(fields ...Field) Logger {
	return &ZapLogger{
		logger: l.logger.With(fields...),
		config: l.config,
	}
}

func (l *ZapLogger) Debugf(format string, args ...interface{}) {
	l.logger.Debug(fmt.Sprintf(format, args...))
}

func (l *ZapLogger) Infof(format string, args ...interface{}) {
	l.logger.Info(fmt.Sprintf(format, args...))
}

func (l *ZapLogger) Warnf(format string, args ...interface{}) {
	l.logger.Warn(fmt.Sprintf(format, args...))
}

func (l *ZapLogger) Errorf(format string, args ...interface{}) {
	l.logger.Error(fmt.Sprintf(format, args...))
}

func (l *ZapLogger) Fatalf(format string, args ...interface{}) {
	l.logger.Fatal(fmt.Sprintf(format, args...))
}

func (l *ZapLogger) DebugContext(ctx context.Context, msg string, fields ...Field) {
	fields = append(fields, extractContextFields(ctx)...)
	l.logger.Debug(msg, fields...)
}

func (l *ZapLogger) InfoContext(ctx context.Context, msg string, fields ...Field) {
	fields = append(fields, extractContextFields(ctx)...)
	l.logger.Info(msg, fields...)
}

func (l *ZapLogger) WarnContext(ctx context.Context, msg string, fields ...Field) {
	fields = append(fields, extractContextFields(ctx)...)
	l.logger.Warn(msg, fields...)
}

func (l *ZapLogger) ErrorContext(ctx context.Context, msg string, fields ...Field) {
	fields = append(fields, extractContextFields(ctx)...)
	l.logger.Error(msg, fields...)
}

func (l *ZapLogger) WithContext(ctx context.Context) Logger {
	fields := l.extractContextFields(ctx)
	return &ZapLogger{
		logger: l.logger.With(fields...),
	}
}

func (l *ZapLogger) extractContextFields(ctx context.Context) []Field {
	var fields []Field

	if requestID := ctx.Value("request_id"); requestID != nil {
		fields = append(fields, Any("request_id", requestID))
	}

	if userID := ctx.Value("user_id"); userID != nil {
		fields = append(fields, Any("user_id", userID))
	}

	if traceID := ctx.Value("trace_id"); traceID != nil {
		fields = append(fields, Any("trace_id", traceID))
	}
	return fields
}

func extractContextFields(ctx context.Context) []Field {
	var fields []Field

	if requestID := ctx.Value("request_id"); requestID != nil {
		if id, ok := requestID.(string); ok {
			fields = append(fields, RequestID(id))
		}
	}

	if traceID := ctx.Value("trace_id"); traceID != nil {
		if id, ok := traceID.(string); ok {
			fields = append(fields, TraceID(id))
		}
	}

	if userID := ctx.Value("user_id"); userID != nil {
		if id, ok := userID.(string); ok {
			fields = append(fields, UserID(id))
		}
	}

	return fields
}

func (l *ZapLogger) Sync() error {
	return l.logger.Sync()
}

func (l *ZapLogger) Printf(format string, args ...interface{}) {
	l.logger.Info(fmt.Sprintf(format, args...))
}

func (l *ZapLogger) Println(args ...interface{}) {
	l.logger.Info(fmt.Sprint(args...))
}
