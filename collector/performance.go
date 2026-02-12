package collector

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/metric"
)

// PerformanceCollector collects performance metrics.
type PerformanceCollector struct {
	interval          time.Duration
	p50Latency        metric.Float64Gauge
	p90Latency        metric.Float64Gauge
	p95Latency        metric.Float64Gauge
	p99Latency        metric.Float64Gauge
	requestsPerSecond metric.Float64Gauge
	messagesPerSecond metric.Float64Gauge
	cpuUtilization    metric.Float64Gauge
	memoryUtilization metric.Float64Gauge
	cacheHitRate      metric.Float64Gauge
	cacheMissRate     metric.Float64Gauge
}

// NewPerformanceCollector creates a new performance metrics collector.
func NewPerformanceCollector(meter metric.Meter, interval time.Duration) (*PerformanceCollector, error) {
	pc := &PerformanceCollector{interval: interval}
	var err error

	pc.p50Latency, err = meter.Float64Gauge("latency_p50_seconds",
		metric.WithDescription("50th percentile latency"), metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}

	pc.p90Latency, err = meter.Float64Gauge("latency_p90_seconds",
		metric.WithDescription("90th percentile latency"), metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}

	pc.p95Latency, err = meter.Float64Gauge("latency_p95_seconds",
		metric.WithDescription("95th percentile latency"), metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}

	pc.p99Latency, err = meter.Float64Gauge("latency_p99_seconds",
		metric.WithDescription("99th percentile latency"), metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}

	pc.requestsPerSecond, err = meter.Float64Gauge("requests_per_second",
		metric.WithDescription("Current requests per second"), metric.WithUnit("1/s"))
	if err != nil {
		return nil, err
	}

	pc.messagesPerSecond, err = meter.Float64Gauge("messages_per_second",
		metric.WithDescription("Current messages per second"), metric.WithUnit("1/s"))
	if err != nil {
		return nil, err
	}

	pc.cpuUtilization, err = meter.Float64Gauge("cpu_utilization_percent",
		metric.WithDescription("Current CPU utilization percentage"), metric.WithUnit("%"))
	if err != nil {
		return nil, err
	}

	pc.memoryUtilization, err = meter.Float64Gauge("memory_utilization_percent",
		metric.WithDescription("Current memory utilization percentage"), metric.WithUnit("%"))
	if err != nil {
		return nil, err
	}

	pc.cacheHitRate, err = meter.Float64Gauge("cache_hit_rate_percent",
		metric.WithDescription("Current cache hit rate percentage"), metric.WithUnit("%"))
	if err != nil {
		return nil, err
	}

	pc.cacheMissRate, err = meter.Float64Gauge("cache_miss_rate_percent",
		metric.WithDescription("Current cache miss rate percentage"), metric.WithUnit("%"))
	if err != nil {
		return nil, err
	}

	return pc, nil
}

// Collect runs the performance metric collection loop.
func (pc *PerformanceCollector) Collect(ctx context.Context, stop <-chan struct{}) {
	ticker := time.NewTicker(pc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-stop:
			return
		case <-ticker.C:
			// Placeholder: performance metrics are typically populated by middleware/handlers
		}
	}
}
