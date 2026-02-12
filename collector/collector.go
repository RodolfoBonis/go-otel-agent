package collector

import (
	"context"
	"fmt"
	"sync"

	"github.com/RodolfoBonis/go-otel-agent/logger"
)

// MetricCollector orchestrates sub-collectors for runtime, business, performance, and system metrics.
type MetricCollector struct {
	logger      logger.Logger
	runtime     *RuntimeCollector
	business    *BusinessCollector
	performance *PerformanceCollector
	system      *SystemCollector

	mu       sync.RWMutex
	running  bool
	stopChan chan struct{}
}

// CollectorConfig holds configuration for the metric collector.
type CollectorConfig struct {
	RuntimeEnabled     bool
	BusinessEnabled    bool
	PerformanceEnabled bool
	SystemEnabled      bool
	RuntimeInterval    interface{} // time.Duration, kept as interface to avoid circular
	DefaultInterval    interface{} // time.Duration
}

// New creates a new MetricCollector.
func New(log logger.Logger, runtime *RuntimeCollector, business *BusinessCollector, performance *PerformanceCollector, system *SystemCollector) *MetricCollector {
	return &MetricCollector{
		logger:      log,
		runtime:     runtime,
		business:    business,
		performance: performance,
		system:      system,
		stopChan:    make(chan struct{}),
	}
}

// Start starts all sub-collectors.
func (mc *MetricCollector) Start(ctx context.Context) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.running {
		return fmt.Errorf("metric collector is already running")
	}
	mc.running = true

	if mc.runtime != nil {
		go mc.runtime.Collect(ctx, mc.stopChan)
	}
	if mc.business != nil {
		go mc.business.Collect(ctx, mc.stopChan)
	}
	if mc.performance != nil {
		go mc.performance.Collect(ctx, mc.stopChan)
	}
	if mc.system != nil {
		go mc.system.Collect(ctx, mc.stopChan)
	}

	mc.logger.Info(ctx, "Metric collector started")
	return nil
}

// Stop stops all sub-collectors.
func (mc *MetricCollector) Stop(ctx context.Context) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if !mc.running {
		return nil
	}

	close(mc.stopChan)
	mc.running = false

	mc.logger.Info(ctx, "Metric collector stopped")
	return nil
}

// GetBusinessCollector returns the business collector.
func (mc *MetricCollector) GetBusinessCollector() *BusinessCollector {
	return mc.business
}
