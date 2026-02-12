package helper

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestGetTraceID_ReturnsEmptyWhenNoSpan(t *testing.T) {
	ctx := context.Background()
	traceID := GetTraceID(ctx)

	if traceID != "" {
		t.Fatalf("expected empty trace ID, got %q", traceID)
	}
}

func TestGetSpanID_ReturnsEmptyWhenNoSpan(t *testing.T) {
	ctx := context.Background()
	spanID := GetSpanID(ctx)

	if spanID != "" {
		t.Fatalf("expected empty span ID, got %q", spanID)
	}
}

func TestIsTracing_ReturnsFalseWhenNoSpan(t *testing.T) {
	ctx := context.Background()
	result := IsTracing(ctx)

	if result {
		t.Fatal("expected IsTracing to return false with no span in context")
	}
}

func TestGetTraceID_ReturnsEmptyForNoopSpan(t *testing.T) {
	tp := noop.NewTracerProvider()
	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	traceID := GetTraceID(ctx)

	// Noop tracer produces invalid span context, so trace ID should be empty
	if traceID != "" {
		t.Fatalf("expected empty trace ID for noop span, got %q", traceID)
	}
}

func TestGetSpanID_ReturnsEmptyForNoopSpan(t *testing.T) {
	tp := noop.NewTracerProvider()
	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	spanID := GetSpanID(ctx)

	// Noop tracer produces invalid span context, so span ID should be empty
	if spanID != "" {
		t.Fatalf("expected empty span ID for noop span, got %q", spanID)
	}
}

func TestIsTracing_ReturnsFalseForNoopSpan(t *testing.T) {
	tp := noop.NewTracerProvider()
	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	result := IsTracing(ctx)

	// Noop tracer produces invalid span context
	if result {
		t.Fatal("expected IsTracing to return false for noop span")
	}
}

func TestGetTraceID_ReturnsValueForValidSpan(t *testing.T) {
	traceID := trace.TraceID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}
	spanID := trace.SpanID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})

	ctx := trace.ContextWithSpanContext(context.Background(), sc)
	result := GetTraceID(ctx)

	expected := traceID.String()
	if result != expected {
		t.Fatalf("expected trace ID %q, got %q", expected, result)
	}
}

func TestGetSpanID_ReturnsValueForValidSpan(t *testing.T) {
	traceID := trace.TraceID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}
	spanID := trace.SpanID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})

	ctx := trace.ContextWithSpanContext(context.Background(), sc)
	result := GetSpanID(ctx)

	expected := spanID.String()
	if result != expected {
		t.Fatalf("expected span ID %q, got %q", expected, result)
	}
}

func TestIsTracing_ReturnsTrueForValidSpan(t *testing.T) {
	traceID := trace.TraceID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}
	spanID := trace.SpanID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})

	ctx := trace.ContextWithSpanContext(context.Background(), sc)
	result := IsTracing(ctx)

	if !result {
		t.Fatal("expected IsTracing to return true for valid span context")
	}
}
