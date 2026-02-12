package collector

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/metric"
)

// SystemCollector collects system-level metrics.
type SystemCollector struct {
	interval         time.Duration
	dbConnections    metric.Int64Gauge
	redisConnections metric.Int64Gauge
	httpConnections  metric.Int64Gauge
	queueDepth       metric.Int64Gauge
	queueRate        metric.Float64Gauge
	healthScore      metric.Float64Gauge
	uptime           metric.Int64Gauge
}

// NewSystemCollector creates a new system metrics collector.
func NewSystemCollector(meter metric.Meter, interval time.Duration) (*SystemCollector, error) {
	sc := &SystemCollector{interval: interval}
	var err error

	sc.dbConnections, err = meter.Int64Gauge("database_connections_active",
		metric.WithDescription("Current active database connections"))
	if err != nil {
		return nil, err
	}

	sc.redisConnections, err = meter.Int64Gauge("redis_connections_active",
		metric.WithDescription("Current active Redis connections"))
	if err != nil {
		return nil, err
	}

	sc.httpConnections, err = meter.Int64Gauge("http_connections_active",
		metric.WithDescription("Current active HTTP connections"))
	if err != nil {
		return nil, err
	}

	sc.queueDepth, err = meter.Int64Gauge("queue_depth",
		metric.WithDescription("Current queue depth"))
	if err != nil {
		return nil, err
	}

	sc.queueRate, err = meter.Float64Gauge("queue_processing_rate",
		metric.WithDescription("Current queue processing rate"), metric.WithUnit("1/s"))
	if err != nil {
		return nil, err
	}

	sc.healthScore, err = meter.Float64Gauge("health_score",
		metric.WithDescription("Current health score (0-1)"))
	if err != nil {
		return nil, err
	}

	sc.uptime, err = meter.Int64Gauge("uptime_seconds",
		metric.WithDescription("Application uptime in seconds"), metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}

	return sc, nil
}

// Collect runs the system metric collection loop.
func (sc *SystemCollector) Collect(ctx context.Context, stop <-chan struct{}) {
	ticker := time.NewTicker(sc.interval)
	defer ticker.Stop()

	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case <-stop:
			return
		case <-ticker.C:
			sc.uptime.Record(ctx, int64(time.Since(startTime).Seconds()))
		}
	}
}
