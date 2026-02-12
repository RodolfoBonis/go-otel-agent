package collector

import (
	"context"
	"runtime"
	"time"

	"go.opentelemetry.io/otel/metric"
)

// RuntimeCollector collects Go runtime metrics.
type RuntimeCollector struct {
	interval      time.Duration
	memAlloc      metric.Int64Gauge
	memSys        metric.Int64Gauge
	memHeapAlloc  metric.Int64Gauge
	memHeapSys    metric.Int64Gauge
	memStack      metric.Int64Gauge
	memGCCount    metric.Int64Counter
	memGCPause    metric.Float64Histogram
	goroutines    metric.Int64Gauge
	gcCPUFraction metric.Float64Gauge
}

// NewRuntimeCollector creates a new runtime metrics collector.
func NewRuntimeCollector(meter metric.Meter, interval time.Duration) (*RuntimeCollector, error) {
	rc := &RuntimeCollector{interval: interval}
	var err error

	rc.memAlloc, err = meter.Int64Gauge("go_memory_alloc_bytes",
		metric.WithDescription("Current allocated memory in bytes"), metric.WithUnit("By"))
	if err != nil {
		return nil, err
	}

	rc.memSys, err = meter.Int64Gauge("go_memory_sys_bytes",
		metric.WithDescription("Total system memory in bytes"), metric.WithUnit("By"))
	if err != nil {
		return nil, err
	}

	rc.memHeapAlloc, err = meter.Int64Gauge("go_memory_heap_alloc_bytes",
		metric.WithDescription("Current heap allocated memory in bytes"), metric.WithUnit("By"))
	if err != nil {
		return nil, err
	}

	rc.memHeapSys, err = meter.Int64Gauge("go_memory_heap_sys_bytes",
		metric.WithDescription("Total heap system memory in bytes"), metric.WithUnit("By"))
	if err != nil {
		return nil, err
	}

	rc.memStack, err = meter.Int64Gauge("go_memory_stack_bytes",
		metric.WithDescription("Current stack memory in bytes"), metric.WithUnit("By"))
	if err != nil {
		return nil, err
	}

	rc.memGCCount, err = meter.Int64Counter("go_gc_collections_total",
		metric.WithDescription("Total number of GC collections"))
	if err != nil {
		return nil, err
	}

	rc.memGCPause, err = meter.Float64Histogram("go_gc_pause_seconds",
		metric.WithDescription("GC pause duration in seconds"), metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}

	rc.goroutines, err = meter.Int64Gauge("go_goroutines",
		metric.WithDescription("Current number of goroutines"))
	if err != nil {
		return nil, err
	}

	rc.gcCPUFraction, err = meter.Float64Gauge("go_gc_cpu_fraction",
		metric.WithDescription("Fraction of CPU time used by GC"))
	if err != nil {
		return nil, err
	}

	return rc, nil
}

// Collect runs the runtime metric collection loop.
func (rc *RuntimeCollector) Collect(ctx context.Context, stop <-chan struct{}) {
	ticker := time.NewTicker(rc.interval)
	defer ticker.Stop()

	var lastNumGC uint32
	var lastPauseTotal time.Duration

	for {
		select {
		case <-ctx.Done():
			return
		case <-stop:
			return
		case <-ticker.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			rc.memAlloc.Record(ctx, int64(m.Alloc))
			rc.memSys.Record(ctx, int64(m.Sys))
			rc.memHeapAlloc.Record(ctx, int64(m.HeapAlloc))
			rc.memHeapSys.Record(ctx, int64(m.HeapSys))
			rc.memStack.Record(ctx, int64(m.StackSys))
			rc.goroutines.Record(ctx, int64(runtime.NumGoroutine()))
			rc.gcCPUFraction.Record(ctx, m.GCCPUFraction)

			if m.NumGC > lastNumGC {
				rc.memGCCount.Add(ctx, int64(m.NumGC-lastNumGC))
				lastNumGC = m.NumGC
			}

			totalPauseNs := time.Duration(m.PauseTotalNs)
			if totalPauseNs > lastPauseTotal {
				pauseDiff := totalPauseNs - lastPauseTotal
				rc.memGCPause.Record(ctx, pauseDiff.Seconds())
				lastPauseTotal = totalPauseNs
			}
		}
	}
}
