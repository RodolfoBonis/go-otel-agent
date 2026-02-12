package otelagent

import (
	"context"
	"testing"
)

func TestNoopTracer_ReturnsValidTracer(t *testing.T) {
	tracer := noopTracer("test")
	if tracer == nil {
		t.Fatal("noopTracer returned nil")
	}

	// Verify the tracer can start a span without panicking.
	ctx, span := tracer.Start(context.Background(), "test-span")
	if ctx == nil {
		t.Fatal("expected non-nil context from noop tracer span")
	}
	span.End()
}

func TestNoopMeter_ReturnsValidMeter(t *testing.T) {
	meter := noopMeter("test")
	if meter == nil {
		t.Fatal("noopMeter returned nil")
	}

	// Verify the meter can create an instrument without panicking.
	counter, err := meter.Int64Counter("test_counter")
	if err != nil {
		t.Fatalf("expected no error creating counter on noop meter, got: %v", err)
	}
	if counter == nil {
		t.Fatal("expected non-nil counter from noop meter")
	}
}

func TestNoopTracerProvider_ReturnsValidProvider(t *testing.T) {
	provider := noopTracerProvider()
	if provider == nil {
		t.Fatal("noopTracerProvider returned nil")
	}

	tracer := provider.Tracer("test")
	if tracer == nil {
		t.Fatal("expected non-nil tracer from noop tracer provider")
	}
}

func TestNoopMeterProvider_ReturnsValidProvider(t *testing.T) {
	provider := noopMeterProvider()
	if provider == nil {
		t.Fatal("noopMeterProvider returned nil")
	}

	meter := provider.Meter("test")
	if meter == nil {
		t.Fatal("expected non-nil meter from noop meter provider")
	}
}
