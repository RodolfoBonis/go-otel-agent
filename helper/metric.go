package helper

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// MetricOptions configure metric recording.
type MetricOptions struct {
	Component  string
	Attributes []attribute.KeyValue
}

// instrumentCache caches metric instruments to avoid recreation on every call.
// Fix: original code recreated instruments on every RecordDuration/IncrementCounter/SetGauge call.
var (
	histogramCache sync.Map // key -> metric.Float64Histogram
	counterCache   sync.Map // key -> metric.Int64Counter
	gaugeCache     sync.Map // key -> metric.Int64Gauge
)

// RecordDuration records a duration metric with cached instrument.
func RecordDuration(ctx context.Context, p TracerMeterProvider, name string, duration time.Duration, opts *MetricOptions) {
	if p == nil || !p.IsEnabled() {
		return
	}

	component := "default"
	if opts != nil && opts.Component != "" {
		component = opts.Component
	}

	cacheKey := component + ":" + name
	var histogram metric.Float64Histogram

	if cached, ok := histogramCache.Load(cacheKey); ok {
		histogram = cached.(metric.Float64Histogram)
	} else {
		meter := p.GetMeter(component)
		var err error
		histogram, err = meter.Float64Histogram(
			name,
			metric.WithDescription(fmt.Sprintf("Duration of %s operations", name)),
			metric.WithUnit("s"),
		)
		if err != nil {
			return
		}
		histogramCache.Store(cacheKey, histogram)
	}

	attrs := []attribute.KeyValue{
		attribute.String("component", component),
	}
	if opts != nil && len(opts.Attributes) > 0 {
		attrs = append(attrs, opts.Attributes...)
	}

	histogram.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// IncrementCounter increments a counter metric with cached instrument.
func IncrementCounter(ctx context.Context, p TracerMeterProvider, name string, value int64, opts *MetricOptions) {
	if p == nil || !p.IsEnabled() {
		return
	}

	component := "default"
	if opts != nil && opts.Component != "" {
		component = opts.Component
	}

	cacheKey := component + ":" + name
	var counter metric.Int64Counter

	if cached, ok := counterCache.Load(cacheKey); ok {
		counter = cached.(metric.Int64Counter)
	} else {
		meter := p.GetMeter(component)
		var err error
		counter, err = meter.Int64Counter(
			name,
			metric.WithDescription(fmt.Sprintf("Counter for %s events", name)),
		)
		if err != nil {
			return
		}
		counterCache.Store(cacheKey, counter)
	}

	attrs := []attribute.KeyValue{
		attribute.String("component", component),
	}
	if opts != nil && len(opts.Attributes) > 0 {
		attrs = append(attrs, opts.Attributes...)
	}

	counter.Add(ctx, value, metric.WithAttributes(attrs...))
}

// SetGauge sets a gauge metric value with cached instrument.
func SetGauge(ctx context.Context, p TracerMeterProvider, name string, value int64, opts *MetricOptions) {
	if p == nil || !p.IsEnabled() {
		return
	}

	component := "default"
	if opts != nil && opts.Component != "" {
		component = opts.Component
	}

	cacheKey := component + ":" + name
	var gauge metric.Int64Gauge

	if cached, ok := gaugeCache.Load(cacheKey); ok {
		gauge = cached.(metric.Int64Gauge)
	} else {
		meter := p.GetMeter(component)
		var err error
		gauge, err = meter.Int64Gauge(
			name,
			metric.WithDescription(fmt.Sprintf("Gauge for %s values", name)),
		)
		if err != nil {
			return
		}
		gaugeCache.Store(cacheKey, gauge)
	}

	attrs := []attribute.KeyValue{
		attribute.String("component", component),
	}
	if opts != nil && len(opts.Attributes) > 0 {
		attrs = append(attrs, opts.Attributes...)
	}

	gauge.Record(ctx, value, metric.WithAttributes(attrs...))
}
