package logger

import (
	"context"
	"errors"
	"testing"
)

func TestNoopLogger_ImplementsLoggerInterface(t *testing.T) {
	var l Logger = &NoopLogger{}
	_ = l // compile-time interface check
}

func TestNoopLogger_DebugDoesNotPanic(t *testing.T) {
	l := &NoopLogger{}
	l.Debug(context.Background(), "debug message")
	l.Debug(context.Background(), "debug with fields", Fields{"key": "value"})
	//nolint:staticcheck
	l.Debug(nil, "debug with nil context")
}

func TestNoopLogger_InfoDoesNotPanic(t *testing.T) {
	l := &NoopLogger{}
	l.Info(context.Background(), "info message")
	l.Info(context.Background(), "info with fields", Fields{"key": "value"})
	//nolint:staticcheck
	l.Info(nil, "info with nil context")
}

func TestNoopLogger_WarningDoesNotPanic(t *testing.T) {
	l := &NoopLogger{}
	l.Warning(context.Background(), "warning message")
	l.Warning(context.Background(), "warning with fields", Fields{"key": "value"})
	//nolint:staticcheck
	l.Warning(nil, "warning with nil context")
}

func TestNoopLogger_ErrorDoesNotPanic(t *testing.T) {
	l := &NoopLogger{}
	l.Error(context.Background(), "error message")
	l.Error(context.Background(), "error with fields", Fields{"key": "value"})
	//nolint:staticcheck
	l.Error(nil, "error with nil context")
}

func TestNoopLogger_FatalDoesNotPanic(t *testing.T) {
	l := &NoopLogger{}
	l.Fatal(context.Background(), "fatal message")
	l.Fatal(context.Background(), "fatal with fields", Fields{"key": "value"})
	//nolint:staticcheck
	l.Fatal(nil, "fatal with nil context")
}

func TestNoopLogger_PanicDoesNotPanic(t *testing.T) {
	l := &NoopLogger{}
	l.Panic(context.Background(), "panic message")
	l.Panic(context.Background(), "panic with fields", Fields{"key": "value"})
	//nolint:staticcheck
	l.Panic(nil, "panic with nil context")
}

func TestNoopLogger_WithReturnsSelf(t *testing.T) {
	l := &NoopLogger{}
	result := l.With(Fields{"service": "test"})

	if result == nil {
		t.Fatal("expected non-nil logger from With")
	}

	if result != l {
		t.Fatal("expected NoopLogger.With to return the same instance")
	}
}

func TestNoopLogger_LogErrorDoesNotPanic(t *testing.T) {
	l := &NoopLogger{}

	// With a real error
	l.LogError(context.Background(), "operation failed", errors.New("test error"))

	// With nil error
	l.LogError(context.Background(), "operation succeeded", nil)

	// With nil context
	//nolint:staticcheck
	l.LogError(nil, "nil context error", errors.New("test error"))
}

func TestNoopLogger_WithEmptyFields(t *testing.T) {
	l := &NoopLogger{}
	result := l.With(Fields{})

	if result == nil {
		t.Fatal("expected non-nil logger from With with empty fields")
	}
}

func TestNoopLogger_WithNilFields(t *testing.T) {
	l := &NoopLogger{}
	result := l.With(nil)

	if result == nil {
		t.Fatal("expected non-nil logger from With with nil fields")
	}
}
