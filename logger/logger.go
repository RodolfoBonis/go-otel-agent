package logger

import (
	"context"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Fields represents structured log fields.
type Fields map[string]interface{}

// contextKey is a private type for context keys to avoid collisions.
type contextKey string

// RequestIDKey is the context key for request IDs.
const RequestIDKey contextKey = "requestID"

// Logger is a structured logger interface with automatic trace correlation.
type Logger interface {
	Debug(ctx context.Context, message string, fields ...Fields)
	Info(ctx context.Context, message string, fields ...Fields)
	Warning(ctx context.Context, message string, fields ...Fields)
	Error(ctx context.Context, message string, fields ...Fields)
	Fatal(ctx context.Context, message string, fields ...Fields)
	Panic(ctx context.Context, message string, fields ...Fields)
	With(fields Fields) Logger
	LogError(ctx context.Context, message string, err error)
}

// CustomLogger is a zap-based implementation of Logger with automatic trace correlation.
type CustomLogger struct {
	logger *zap.Logger
}

// NewLogger creates a new logger instance.
// environment should be "development" or "production".
// If empty, defaults to checking ENV environment variable, then "development".
func NewLogger(environment string) Logger {
	if environment == "" {
		environment = os.Getenv("ENV")
		if environment == "" {
			environment = "development"
		}
	}

	var cfg zap.Config
	if environment == "development" {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
	}

	// Dynamic log level from env
	if lvl := os.Getenv("LOG_LEVEL"); lvl != "" {
		if level, err := zapcore.ParseLevel(lvl); err == nil {
			cfg.Level = zap.NewAtomicLevelAt(level)
		}
	}

	cfg.EncoderConfig.TimeKey = "time"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zapLogger, _ := cfg.Build(
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.AddCaller(),
		zap.AddCallerSkip(1),
	)

	return &CustomLogger{logger: zapLogger}
}

// EnableOTelBridge adds an OTel log bridge core so zap entries are
// also exported as OTel log records via OTLP.
func (cl *CustomLogger) EnableOTelBridge(provider otellog.LoggerProvider) {
	otelCore := otelzap.NewCore("go-otel-agent",
		otelzap.WithLoggerProvider(provider),
	)
	cl.logger = cl.logger.WithOptions(
		zap.WrapCore(func(existing zapcore.Core) zapcore.Core {
			return zapcore.NewTee(existing, otelCore)
		}),
	)
}

func (cl *CustomLogger) Debug(ctx context.Context, message string, fields ...Fields) {
	cl.logger.Debug(message, cl.zapFields(ctx, fields...)...)
}

func (cl *CustomLogger) Info(ctx context.Context, message string, fields ...Fields) {
	cl.logger.Info(message, cl.zapFields(ctx, fields...)...)
}

func (cl *CustomLogger) Warning(ctx context.Context, message string, fields ...Fields) {
	cl.logger.Warn(message, cl.zapFields(ctx, fields...)...)
}

func (cl *CustomLogger) Error(ctx context.Context, message string, fields ...Fields) {
	cl.logger.Error(message, cl.zapFields(ctx, fields...)...)
}

func (cl *CustomLogger) Fatal(ctx context.Context, message string, fields ...Fields) {
	cl.logger.Fatal(message, cl.zapFields(ctx, fields...)...)
}

func (cl *CustomLogger) Panic(ctx context.Context, message string, fields ...Fields) {
	cl.logger.Panic(message, cl.zapFields(ctx, fields...)...)
}

func (cl *CustomLogger) With(fields Fields) Logger {
	return &CustomLogger{logger: cl.logger.With(cl.fieldsToZap(fields)...)}
}

func (cl *CustomLogger) LogError(ctx context.Context, message string, err error) {
	if err == nil {
		return
	}

	fields := Fields{"error": err.Error()}
	if appErr, ok := err.(interface{ ToLogFields() map[string]interface{} }); ok {
		fields = appErr.ToLogFields()
	}

	cl.Error(ctx, message, fields)
}

// zapFields merges context and custom fields, automatically injecting trace context.
func (cl *CustomLogger) zapFields(ctx context.Context, fields ...Fields) []zap.Field {
	allFields := make(Fields)
	for _, f := range fields {
		for k, v := range f {
			allFields[k] = v
		}
	}

	// Auto trace injection - always check for span context
	if ctx != nil {
		span := trace.SpanFromContext(ctx)
		if span.SpanContext().IsValid() {
			allFields["trace_id"] = span.SpanContext().TraceID().String()
			allFields["span_id"] = span.SpanContext().SpanID().String()
		}

		// Add requestID from context if present
		if reqID, ok := ctx.Value(RequestIDKey).(string); ok && reqID != "" {
			allFields["requestID"] = reqID
		}
	}

	zfs := cl.fieldsToZap(allFields)

	// Pass context to otelzap bridge for native OTel trace correlation.
	// SkipType is invisible to stdout (AddTo is no-op) but otelzap checks
	// field.Interface for context.Context before AddTo.
	if ctx != nil {
		zfs = append(zfs, zap.Field{Key: "", Type: zapcore.SkipType, Interface: ctx})
	}

	return zfs
}

func (cl *CustomLogger) fieldsToZap(fields Fields) []zap.Field {
	zfs := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zfs = append(zfs, zap.Any(k, v))
	}
	return zfs
}
