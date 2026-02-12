package collector

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// BusinessCollector collects application-specific business metrics.
type BusinessCollector struct {
	interval         time.Duration
	meter            metric.Meter
	activeUsers      metric.Int64Gauge
	requestRate      metric.Float64Gauge
	errorRate        metric.Float64Gauge
	responseTime     metric.Float64Histogram
	featureUsage     metric.Int64Counter
	conversionRate   metric.Float64Gauge
	retentionRate    metric.Float64Gauge
	customCounters   map[string]metric.Int64Counter
	customGauges     map[string]metric.Int64Gauge
	customHistograms map[string]metric.Float64Histogram
	mu               sync.RWMutex
}

// NewBusinessCollector creates a new business metrics collector.
func NewBusinessCollector(meter metric.Meter, interval time.Duration) (*BusinessCollector, error) {
	bc := &BusinessCollector{
		interval:         interval,
		meter:            meter,
		customCounters:   make(map[string]metric.Int64Counter),
		customGauges:     make(map[string]metric.Int64Gauge),
		customHistograms: make(map[string]metric.Float64Histogram),
	}

	var err error

	bc.activeUsers, err = meter.Int64Gauge("active_users",
		metric.WithDescription("Current number of active users"))
	if err != nil {
		return nil, err
	}

	bc.requestRate, err = meter.Float64Gauge("request_rate",
		metric.WithDescription("Current request rate per second"), metric.WithUnit("1/s"))
	if err != nil {
		return nil, err
	}

	bc.errorRate, err = meter.Float64Gauge("error_rate",
		metric.WithDescription("Current error rate percentage"), metric.WithUnit("%"))
	if err != nil {
		return nil, err
	}

	bc.responseTime, err = meter.Float64Histogram("response_time_seconds",
		metric.WithDescription("Response time distribution"), metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}

	bc.featureUsage, err = meter.Int64Counter("feature_usage_total",
		metric.WithDescription("Total feature usage count"))
	if err != nil {
		return nil, err
	}

	bc.conversionRate, err = meter.Float64Gauge("conversion_rate",
		metric.WithDescription("Current conversion rate percentage"), metric.WithUnit("%"))
	if err != nil {
		return nil, err
	}

	bc.retentionRate, err = meter.Float64Gauge("retention_rate",
		metric.WithDescription("Current retention rate percentage"), metric.WithUnit("%"))
	if err != nil {
		return nil, err
	}

	return bc, nil
}

// RecordFeatureUsage records usage of a specific feature.
// Fix: removed user_id from metric attributes (cardinality bomb). Keep on spans only.
func (bc *BusinessCollector) RecordFeatureUsage(ctx context.Context, feature string) {
	if bc.featureUsage != nil {
		bc.featureUsage.Add(ctx, 1, metric.WithAttributes(
			attribute.String("feature", feature),
		))
	}
}

// CreateCustomCounter creates or retrieves a custom business counter.
func (bc *BusinessCollector) CreateCustomCounter(name, description string) (metric.Int64Counter, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if counter, exists := bc.customCounters[name]; exists {
		return counter, nil
	}

	counter, err := bc.meter.Int64Counter(name, metric.WithDescription(description))
	if err != nil {
		return nil, err
	}

	bc.customCounters[name] = counter
	return counter, nil
}

// CreateCustomGauge creates or retrieves a custom business gauge.
func (bc *BusinessCollector) CreateCustomGauge(name, description string) (metric.Int64Gauge, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if gauge, exists := bc.customGauges[name]; exists {
		return gauge, nil
	}

	gauge, err := bc.meter.Int64Gauge(name, metric.WithDescription(description))
	if err != nil {
		return nil, err
	}

	bc.customGauges[name] = gauge
	return gauge, nil
}

// CreateCustomHistogram creates or retrieves a custom business histogram.
func (bc *BusinessCollector) CreateCustomHistogram(name, description string) (metric.Float64Histogram, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if histogram, exists := bc.customHistograms[name]; exists {
		return histogram, nil
	}

	histogram, err := bc.meter.Float64Histogram(name, metric.WithDescription(description))
	if err != nil {
		return nil, err
	}

	bc.customHistograms[name] = histogram
	return histogram, nil
}

// Collect runs the business metric collection loop.
func (bc *BusinessCollector) Collect(ctx context.Context, stop <-chan struct{}) {
	ticker := time.NewTicker(bc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-stop:
			return
		case <-ticker.C:
			// Placeholder: business metrics are typically populated by app code
		}
	}
}
