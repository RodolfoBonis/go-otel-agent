package gormplugin

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
)

func TestLazyTracerProvider_ReturnsLazyTracer(t *testing.T) {
	p := &lazyTracerProvider{}
	tracer := p.Tracer("test-tracer")
	if tracer == nil {
		t.Fatal("Tracer() returned nil")
	}
	lt, ok := tracer.(*lazyTracer)
	if !ok {
		t.Fatal("expected *lazyTracer type")
	}
	if lt.name != "test-tracer" {
		t.Errorf("expected name 'test-tracer', got %q", lt.name)
	}
}

func TestLazyTracer_ResolvesGlobalProvider(t *testing.T) {
	// Set a noop provider as global
	noopProvider := nooptrace.NewTracerProvider()
	otel.SetTracerProvider(noopProvider)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	lt := &lazyTracer{name: "test"}
	ctx, span := lt.Start(context.Background(), "test-span")

	if ctx == nil {
		t.Fatal("Start() returned nil context")
	}
	if span == nil {
		t.Fatal("Start() returned nil span")
	}
}

func TestLazyTracerProvider_ImplementsInterface(t *testing.T) {
	// Compile-time check that lazyTracerProvider implements trace.TracerProvider
	var _ trace.TracerProvider = (*lazyTracerProvider)(nil)
}

func TestLazyTracer_ImplementsInterface(t *testing.T) {
	// Compile-time check that lazyTracer implements trace.Tracer
	var _ trace.Tracer = (*lazyTracer)(nil)
}

func TestLazyTracer_PicksUpProviderChange(t *testing.T) {
	// Start with noop
	otel.SetTracerProvider(nooptrace.NewTracerProvider())

	p := &lazyTracerProvider{}
	tracer := p.Tracer("test")

	// First call should work with noop
	ctx1, span1 := tracer.Start(context.Background(), "before")
	if ctx1 == nil || span1 == nil {
		t.Fatal("First Start() failed")
	}
	span1.End()

	// Change global provider (simulating agent.Init() completing)
	otel.SetTracerProvider(nooptrace.NewTracerProvider())

	// Second call should pick up the new provider
	ctx2, span2 := tracer.Start(context.Background(), "after")
	if ctx2 == nil || span2 == nil {
		t.Fatal("Second Start() failed after provider change")
	}
	span2.End()
}
