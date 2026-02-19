package otelagent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/collector"
	"github.com/RodolfoBonis/go-otel-agent/helper"
	"github.com/RodolfoBonis/go-otel-agent/instrumentor"
	"github.com/RodolfoBonis/go-otel-agent/internal/matcher"
	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/go-otel-agent/provider"
	"go.opentelemetry.io/otel"
	otellog "go.opentelemetry.io/otel/log"
	logglobal "go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
)

// Signal represents a telemetry signal type.
type Signal int

const (
	SignalTraces  Signal = iota
	SignalMetrics
	SignalLogs
)

// Agent is the central observability agent. It manages providers,
// instrumentors, collectors, and health probes.
//
// Create with NewAgent(opts...), then call Init(ctx) to start.
type Agent struct {
	config *Config
	logger logger.Logger

	// Providers (SDK types, unexported)
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	loggerProvider *sdklog.LoggerProvider

	// Cached tracers/meters
	tracers sync.Map // name -> trace.Tracer
	meters  sync.Map // name -> metric.Meter

	// Components
	instrumentor *instrumentor.Instrumentor
	collector    *collector.MetricCollector
	routeMatcher *matcher.RouteMatcher
	health       *provider.ExporterHealth

	// State
	mu          sync.RWMutex
	initialized bool
	running     bool
}

// NewAgent creates a new Agent with the given options.
// No I/O is performed — call Init(ctx) to start providers and collectors.
func NewAgent(opts ...Option) *Agent {
	cfg := LoadConfigFromEnv()

	a := &Agent{
		config: cfg,
		health: provider.NewExporterHealth(),
	}

	for _, opt := range opts {
		opt(a)
	}

	// Create logger if not provided
	if a.logger == nil {
		a.logger = logger.NewLogger(cfg.Environment)
	}

	// Build route matcher from config + options
	a.routeMatcher = matcher.NewRouteMatcher(matcher.RouteExclusionConfig{
		ExactPaths:  cfg.RouteExclusion.ExactPaths,
		PrefixPaths: cfg.RouteExclusion.PrefixPaths,
		Patterns:    cfg.RouteExclusion.Patterns,
	})

	return a
}

// Init initializes all providers, sets global OTel state, and starts collectors.
func (a *Agent) Init(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.initialized {
		return ErrAlreadyInitialized
	}

	if !a.config.Enabled {
		a.logger.Info(ctx, "Observability disabled by configuration")
		a.initialized = true
		return nil
	}

	if a.config.ServiceName == "" {
		return ErrMissingServiceName
	}

	// Build resource
	res, err := provider.BuildResource(a.config)
	if err != nil {
		return fmt.Errorf("failed to build resource: %w", err)
	}

	// Initialize trace provider
	if a.config.Traces.Enabled {
		a.tracerProvider, err = provider.NewTraceProvider(a.config, res, a.logger)
		if err != nil {
			return fmt.Errorf("failed to create trace provider: %w", err)
		}
		otel.SetTracerProvider(a.tracerProvider)
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		))
	}

	// Initialize metric provider
	if a.config.Metrics.Enabled {
		a.meterProvider, err = provider.NewMetricProvider(a.config, res, a.logger)
		if err != nil {
			return fmt.Errorf("failed to create metric provider: %w", err)
		}
		otel.SetMeterProvider(a.meterProvider)
	}

	// Initialize log provider
	if a.config.Logs.Enabled {
		a.loggerProvider, err = provider.NewLogProvider(a.config, res, a.logger)
		if err != nil {
			return fmt.Errorf("failed to create log provider: %w", err)
		}
		logglobal.SetLoggerProvider(a.loggerProvider)

		// Bridge zap logger to OTel LoggerProvider so log entries
		// are exported via OTLP alongside traces and metrics.
		if bridgeable, ok := a.logger.(interface {
			EnableOTelBridge(otellog.LoggerProvider)
		}); ok {
			bridgeable.EnableOTelBridge(a.loggerProvider)
		}
	}

	// Initialize instrumentor
	a.instrumentor = instrumentor.New(a)

	// Initialize collectors
	if a.config.Metrics.Enabled {
		if err := a.initCollectors(); err != nil {
			return fmt.Errorf("failed to initialize collectors: %w", err)
		}
	}

	// Set global helper provider
	helper.SetGlobalProvider(a)

	a.initialized = true
	a.running = true

	// Start collectors
	if a.collector != nil {
		if err := a.collector.Start(ctx); err != nil {
			a.logger.Error(ctx, "Failed to start metric collector", logger.Fields{"error": err.Error()})
		}
	}

	a.logger.Info(ctx, "Observability agent initialized", logger.Fields{
		"service":  a.config.ServiceName,
		"version":  a.config.Version,
		"endpoint": a.config.Endpoint,
		"traces":   a.config.Traces.Enabled,
		"metrics":  a.config.Metrics.Enabled,
		"logs":     a.config.Logs.Enabled,
	})

	return nil
}

func (a *Agent) initCollectors() error {
	runtimeMeter := a.GetMeter("runtime")
	businessMeter := a.GetMeter("business")
	performanceMeter := a.GetMeter("performance")
	systemMeter := a.GetMeter("system")

	var runtimeC *collector.RuntimeCollector
	var businessC *collector.BusinessCollector
	var performanceC *collector.PerformanceCollector
	var systemC *collector.SystemCollector

	if a.config.Metrics.Runtime {
		var err error
		runtimeC, err = collector.NewRuntimeCollector(runtimeMeter, a.config.Metrics.RuntimeInterval)
		if err != nil {
			return fmt.Errorf("runtime collector: %w", err)
		}
	}

	if a.config.Metrics.Business {
		var err error
		businessC, err = collector.NewBusinessCollector(businessMeter, a.config.Metrics.DefaultInterval)
		if err != nil {
			return fmt.Errorf("business collector: %w", err)
		}
	}

	var err error
	performanceC, err = collector.NewPerformanceCollector(performanceMeter, a.config.Metrics.DefaultInterval)
	if err != nil {
		return fmt.Errorf("performance collector: %w", err)
	}

	systemC, err = collector.NewSystemCollector(systemMeter, a.config.Metrics.DefaultInterval)
	if err != nil {
		return fmt.Errorf("system collector: %w", err)
	}

	a.collector = collector.New(a.logger, runtimeC, businessC, performanceC, systemC)
	return nil
}

// Shutdown gracefully shuts down all providers with a 10s timeout enforcement.
func (a *Agent) Shutdown(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.initialized {
		return nil
	}

	// Enforce 10s timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	a.logger.Info(ctx, "Shutting down observability agent...")

	// Stop collectors
	if a.collector != nil {
		if err := a.collector.Stop(shutdownCtx); err != nil {
			a.logger.Error(ctx, "Failed to stop collector", logger.Fields{"error": err.Error()})
		}
	}

	// Shutdown providers
	if a.tracerProvider != nil {
		if err := a.tracerProvider.Shutdown(shutdownCtx); err != nil {
			a.logger.Error(ctx, "Failed to shutdown trace provider", logger.Fields{"error": err.Error()})
		}
	}

	if a.meterProvider != nil {
		if err := a.meterProvider.Shutdown(shutdownCtx); err != nil {
			a.logger.Error(ctx, "Failed to shutdown metric provider", logger.Fields{"error": err.Error()})
		}
	}

	if a.loggerProvider != nil {
		if err := a.loggerProvider.Shutdown(shutdownCtx); err != nil {
			a.logger.Error(ctx, "Failed to shutdown log provider", logger.Fields{"error": err.Error()})
		}
	}

	a.running = false
	a.logger.Info(ctx, "Observability agent shut down")
	return nil
}

// ForceFlush flushes all pending telemetry without shutting down.
func (a *Agent) ForceFlush(ctx context.Context) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.initialized {
		return nil
	}

	if a.tracerProvider != nil {
		if err := a.tracerProvider.ForceFlush(ctx); err != nil {
			return fmt.Errorf("trace flush: %w", err)
		}
	}

	if a.meterProvider != nil {
		if err := a.meterProvider.ForceFlush(ctx); err != nil {
			return fmt.Errorf("metric flush: %w", err)
		}
	}

	if a.loggerProvider != nil {
		if err := a.loggerProvider.ForceFlush(ctx); err != nil {
			return fmt.Errorf("log flush: %w", err)
		}
	}

	return nil
}

// --- TracerMeterProvider interface implementation ---

// GetTracer returns a tracer for the given name. Never returns nil.
func (a *Agent) GetTracer(name string) trace.Tracer {
	if a.tracerProvider == nil {
		return noopTracer(name)
	}

	if cached, ok := a.tracers.Load(name); ok {
		return cached.(trace.Tracer)
	}

	tracer := a.tracerProvider.Tracer(name)
	a.tracers.Store(name, tracer)
	return tracer
}

// GetMeter returns a meter for the given name. Never returns nil.
// Fix: original returned nil when disabled, causing panics in consumers.
func (a *Agent) GetMeter(name string) metric.Meter {
	if a.meterProvider == nil {
		return noopMeter(name)
	}

	if cached, ok := a.meters.Load(name); ok {
		return cached.(metric.Meter)
	}

	meter := a.meterProvider.Meter(name)
	a.meters.Store(name, meter)
	return meter
}

// IsEnabled returns whether observability is enabled.
func (a *Agent) IsEnabled() bool {
	return a.config.Enabled
}

// --- Accessors ---

// Config returns the agent configuration.
func (a *Agent) Config() *Config {
	return a.config
}

// Logger returns the agent logger.
func (a *Agent) Logger() logger.Logger {
	return a.logger
}

// Instrumentor returns the instrumentor.
func (a *Agent) Instrumentor() *instrumentor.Instrumentor {
	return a.instrumentor
}

// RouteMatcher returns the route exclusion matcher.
func (a *Agent) RouteMatcher() *matcher.RouteMatcher {
	return a.routeMatcher
}

// ExporterHealth returns the exporter health tracker.
func (a *Agent) ExporterHealth() *provider.ExporterHealth {
	return a.health
}

// TracerProvider returns the underlying trace.TracerProvider.
// Returns a noop provider if not initialized.
func (a *Agent) TracerProvider() trace.TracerProvider {
	if a.tracerProvider == nil {
		return nooptrace.NewTracerProvider()
	}
	return a.tracerProvider
}

// LoggerProvider returns the underlying sdklog.LoggerProvider.
// Returns nil if logs are disabled or not initialized.
func (a *Agent) LoggerProvider() *sdklog.LoggerProvider {
	return a.loggerProvider
}

// IsRunning returns whether the agent is currently running.
func (a *Agent) IsRunning() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.running
}
