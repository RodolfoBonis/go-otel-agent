package logger

import (
	"context"
	"errors"
	"testing"
)

func TestNewLogger_Development(t *testing.T) {
	l := NewLogger("development")
	if l == nil {
		t.Fatal("expected non-nil logger for development environment")
	}

	if _, ok := l.(*CustomLogger); !ok {
		t.Fatalf("expected *CustomLogger, got %T", l)
	}
}

func TestNewLogger_Production(t *testing.T) {
	l := NewLogger("production")
	if l == nil {
		t.Fatal("expected non-nil logger for production environment")
	}

	if _, ok := l.(*CustomLogger); !ok {
		t.Fatalf("expected *CustomLogger, got %T", l)
	}
}

func TestNewLogger_EmptyStringDefaultsToDevelopment(t *testing.T) {
	l := NewLogger("")
	if l == nil {
		t.Fatal("expected non-nil logger when environment is empty string")
	}

	if _, ok := l.(*CustomLogger); !ok {
		t.Fatalf("expected *CustomLogger, got %T", l)
	}
}

func TestLogger_InfoDoesNotPanic(t *testing.T) {
	l := NewLogger("development")
	ctx := context.Background()

	// Should not panic with no fields
	l.Info(ctx, "info message")

	// Should not panic with fields
	l.Info(ctx, "info message with fields", Fields{"key": "value"})

	// Should not panic with multiple field maps
	l.Info(ctx, "info message with multiple fields",
		Fields{"key1": "value1"},
		Fields{"key2": "value2"},
	)
}

func TestLogger_ErrorDoesNotPanic(t *testing.T) {
	l := NewLogger("development")
	ctx := context.Background()

	l.Error(ctx, "error message")
	l.Error(ctx, "error message with fields", Fields{"error_code": 500})
}

func TestLogger_DebugDoesNotPanic(t *testing.T) {
	l := NewLogger("development")
	ctx := context.Background()

	l.Debug(ctx, "debug message")
	l.Debug(ctx, "debug message with fields", Fields{"detail": "some detail"})
}

func TestLogger_WarningDoesNotPanic(t *testing.T) {
	l := NewLogger("development")
	ctx := context.Background()

	l.Warning(ctx, "warning message")
	l.Warning(ctx, "warning message with fields", Fields{"severity": "medium"})
}

func TestLogger_NilContextDoesNotPanic(t *testing.T) {
	l := NewLogger("development")

	//nolint:staticcheck // intentionally passing nil context to test safety
	l.Info(nil, "message with nil context")
	//nolint:staticcheck
	l.Debug(nil, "debug with nil context")
	//nolint:staticcheck
	l.Warning(nil, "warning with nil context")
	//nolint:staticcheck
	l.Error(nil, "error with nil context")
}

func TestLogger_WithReturnsNewLogger(t *testing.T) {
	l := NewLogger("development")
	enriched := l.With(Fields{"service": "test"})

	if enriched == nil {
		t.Fatal("expected non-nil logger from With")
	}

	if enriched == l {
		t.Fatal("expected With to return a new logger instance")
	}

	// The enriched logger should still work without panicking
	enriched.Info(context.Background(), "enriched log message")
}

func TestLogger_LogError_WithError(t *testing.T) {
	l := NewLogger("development")
	ctx := context.Background()

	// Should not panic when err is non-nil
	l.LogError(ctx, "operation failed", errors.New("something went wrong"))
}

func TestLogger_LogError_WithNilError(t *testing.T) {
	l := NewLogger("development")
	ctx := context.Background()

	// Should return early without panicking when err is nil
	l.LogError(ctx, "operation succeeded", nil)
}

func TestLogger_LogError_WithCustomErrorFields(t *testing.T) {
	l := NewLogger("development")
	ctx := context.Background()

	// Create an error that implements ToLogFields
	customErr := &testAppError{
		msg:    "custom error",
		code:   "ERR_CUSTOM",
		detail: "extra detail",
	}

	// Should not panic and should use ToLogFields
	l.LogError(ctx, "custom error occurred", customErr)
}

func TestLogger_ContextWithRequestID(t *testing.T) {
	l := NewLogger("development")
	ctx := context.WithValue(context.Background(), RequestIDKey, "req-123")

	// Should not panic and should include requestID in fields
	l.Info(ctx, "message with request ID")
}

// testAppError implements both error and the ToLogFields interface
// used by LogError.
type testAppError struct {
	msg    string
	code   string
	detail string
}

func (e *testAppError) Error() string {
	return e.msg
}

func (e *testAppError) ToLogFields() map[string]interface{} {
	return map[string]interface{}{
		"error":  e.msg,
		"code":   e.code,
		"detail": e.detail,
	}
}
