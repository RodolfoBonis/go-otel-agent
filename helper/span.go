package helper

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// TracerMeterProvider provides access to tracers and meters.
// The Agent struct implements this interface.
type TracerMeterProvider interface {
	GetTracer(name string) trace.Tracer
	GetMeter(name string) metric.Meter
	IsEnabled() bool
}

// SpanOptions configure span creation.
type SpanOptions struct {
	Component  string
	Operation  string
	Attributes []attribute.KeyValue
	Kind       trace.SpanKind
}

// StartSpan starts a new span with simplified configuration.
func StartSpan(ctx context.Context, p TracerMeterProvider, name string, opts *SpanOptions) (context.Context, trace.Span) {
	if p == nil || !p.IsEnabled() {
		return ctx, trace.SpanFromContext(ctx)
	}

	component := "default"
	if opts != nil && opts.Component != "" {
		component = opts.Component
	}

	tracer := p.GetTracer(component)
	spanOpts := []trace.SpanStartOption{}

	if opts != nil {
		if opts.Kind != trace.SpanKindUnspecified {
			spanOpts = append(spanOpts, trace.WithSpanKind(opts.Kind))
		}
		if len(opts.Attributes) > 0 {
			spanOpts = append(spanOpts, trace.WithAttributes(opts.Attributes...))
		}
	}

	ctx, span := tracer.Start(ctx, name, spanOpts...)
	span.SetAttributes(attribute.String("component", component))

	if opts != nil && opts.Operation != "" {
		span.SetAttributes(attribute.String("operation", opts.Operation))
	}

	return ctx, span
}

// TraceFunction automatically traces a function execution.
func TraceFunction(ctx context.Context, p TracerMeterProvider, name string, fn func(context.Context) error, opts *SpanOptions) error {
	ctx, span := StartSpan(ctx, p, name, opts)
	defer span.End()

	start := time.Now()
	err := fn(ctx)
	duration := time.Since(start)

	span.SetAttributes(attribute.Int64("duration_ms", duration.Milliseconds()))

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return err
}

// TraceFunctionWithResult traces a function with return value.
func TraceFunctionWithResult[T any](ctx context.Context, p TracerMeterProvider, name string, fn func(context.Context) (T, error), opts *SpanOptions) (T, error) {
	ctx, span := StartSpan(ctx, p, name, opts)
	defer span.End()

	start := time.Now()
	result, err := fn(ctx)
	duration := time.Since(start)

	span.SetAttributes(attribute.Int64("duration_ms", duration.Milliseconds()))

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return result, err
}

// AddSpanEvent adds an event to the current span.
func AddSpanEvent(ctx context.Context, name string, attributes ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		span.AddEvent(name, trace.WithAttributes(attributes...))
	}
}

// SetSpanAttributes sets attributes on the current span.
func SetSpanAttributes(ctx context.Context, attributes ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		span.SetAttributes(attributes...)
	}
}

// RecordSpanError records an error on the current span.
func RecordSpanError(ctx context.Context, err error, attributes ...attribute.KeyValue) {
	if err == nil {
		return
	}
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		span.RecordError(err, trace.WithAttributes(attributes...))
		span.SetStatus(codes.Error, err.Error())
	}
}
