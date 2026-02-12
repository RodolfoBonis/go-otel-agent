package otelagent

import (
	"go.opentelemetry.io/otel/metric"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
)

// noopTracer returns a no-op tracer. Always safe to use.
func noopTracer(name string) trace.Tracer {
	return nooptrace.NewTracerProvider().Tracer(name)
}

// noopMeter returns a no-op meter. Always safe to use. Fixes the nil meter bug.
func noopMeter(name string) metric.Meter {
	return noopmetric.NewMeterProvider().Meter(name)
}

// noopTracerProvider returns a no-op tracer provider.
func noopTracerProvider() trace.TracerProvider {
	return nooptrace.NewTracerProvider()
}

// noopMeterProvider returns a no-op meter provider.
func noopMeterProvider() metric.MeterProvider {
	return noopmetric.NewMeterProvider()
}
