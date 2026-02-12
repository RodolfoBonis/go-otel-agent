package helper

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var (
	globalProvider TracerMeterProvider
	globalMu      sync.RWMutex
)

// SetGlobalProvider sets the global TracerMeterProvider for convenience functions.
func SetGlobalProvider(p TracerMeterProvider) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalProvider = p
}

// GlobalProvider returns the global TracerMeterProvider.
func GlobalProvider() TracerMeterProvider {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalProvider
}

// Trace starts a new span using the global provider.
func Trace(ctx context.Context, name string, opts *SpanOptions) (context.Context, trace.Span) {
	p := GlobalProvider()
	if p == nil {
		return ctx, trace.SpanFromContext(ctx)
	}
	return StartSpan(ctx, p, name, opts)
}

// Measure records a duration metric using the global provider.
func Measure(ctx context.Context, name string, duration time.Duration, opts *MetricOptions) {
	p := GlobalProvider()
	if p == nil {
		return
	}
	RecordDuration(ctx, p, name, duration, opts)
}

// Count increments a counter metric using the global provider.
func Count(ctx context.Context, name string, value int64, opts *MetricOptions) {
	p := GlobalProvider()
	if p == nil {
		return
	}
	IncrementCounter(ctx, p, name, value, opts)
}

// Event adds an event to the current span.
func Event(ctx context.Context, name string, attributes ...attribute.KeyValue) {
	AddSpanEvent(ctx, name, attributes...)
}

// Error records an error on the current span.
func Error(ctx context.Context, err error, attributes ...attribute.KeyValue) {
	RecordSpanError(ctx, err, attributes...)
}
