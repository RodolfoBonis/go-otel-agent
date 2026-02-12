package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/config"
	"github.com/RodolfoBonis/go-otel-agent/logger"
	otlptracegrpc "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	otlptracehttp "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// NewTraceProvider creates a TracerProvider with OTLP exporter.
// Fixes: always wraps sampler in ParentBased, wires span limits and retry config.
func NewTraceProvider(cfg *config.Config, res *resource.Resource, log logger.Logger) (*sdktrace.TracerProvider, error) {
	ctx := context.Background()

	exporter, err := createTraceExporter(ctx, cfg, log)
	if err != nil {
		return nil, err
	}

	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(cfg.Traces.BatchTimeout),
			sdktrace.WithMaxExportBatchSize(cfg.Traces.MaxExportBatch),
			sdktrace.WithMaxQueueSize(cfg.Traces.QueueSize),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(createSampler(cfg.Traces.Sampling)),
	}

	// Wire span limits (fix: was configured but never applied)
	if cfg.Traces.MaxAttributesPerSpan > 0 || cfg.Traces.MaxEventsPerSpan > 0 || cfg.Traces.MaxLinksPerSpan > 0 {
		limits := sdktrace.SpanLimits{
			AttributeCountLimit:         cfg.Traces.MaxAttributesPerSpan,
			EventCountLimit:             cfg.Traces.MaxEventsPerSpan,
			LinkCountLimit:              cfg.Traces.MaxLinksPerSpan,
			AttributePerEventCountLimit: cfg.Traces.MaxAttributesPerSpan,
			AttributePerLinkCountLimit:  cfg.Traces.MaxAttributesPerSpan,
		}
		opts = append(opts, sdktrace.WithRawSpanLimits(limits))
	}

	// Add PII scrubbing processor if enabled
	if cfg.Scrub.Enabled {
		processor := NewScrubProcessor(cfg.Scrub)
		opts = append(opts, sdktrace.WithSpanProcessor(processor))
	}

	return sdktrace.NewTracerProvider(opts...), nil
}

func createTraceExporter(ctx context.Context, cfg *config.Config, log logger.Logger) (sdktrace.SpanExporter, error) {
	protocol := cfg.ExporterProtocol
	if protocol == "" {
		protocol = "grpc"
	}

	switch protocol {
	case "grpc":
		return createGRPCTraceExporter(ctx, cfg, log)
	case "http", "http/protobuf":
		return createHTTPTraceExporter(ctx, cfg, log)
	default:
		return nil, fmt.Errorf("unsupported OTLP protocol: %s (use 'grpc' or 'http')", protocol)
	}
}

func createGRPCTraceExporter(ctx context.Context, cfg *config.Config, log logger.Logger) (sdktrace.SpanExporter, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithTimeout(cfg.Timeout),
	}

	if cfg.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	if cfg.Compression != "" && cfg.Compression != "none" {
		opts = append(opts, otlptracegrpc.WithCompressor(cfg.Compression))
	}

	// Wire auth headers
	headers := cfg.ResolvedAuthHeaders()
	if len(headers) > 0 {
		opts = append(opts, otlptracegrpc.WithHeaders(headers))
	}

	// Wire retry config (fix: was configured but never wired)
	if cfg.Performance.RetryAttempts > 0 {
		opts = append(opts, otlptracegrpc.WithRetry(otlptracegrpc.RetryConfig{
			Enabled:         true,
			InitialInterval: cfg.Performance.RetryBackoff,
			MaxInterval:     cfg.Performance.RetryBackoff * 5,
			MaxElapsedTime:  cfg.Performance.RetryBackoff * time.Duration(cfg.Performance.RetryAttempts) * 5,
		}))
	}

	exporter, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP gRPC trace exporter: %w", err)
	}

	log.Info(ctx, "OTLP trace exporter initialized", logger.Fields{
		"protocol": "grpc", "endpoint": cfg.Endpoint,
	})

	return exporter, nil
}

func createHTTPTraceExporter(ctx context.Context, cfg *config.Config, log logger.Logger) (sdktrace.SpanExporter, error) {
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(cfg.Endpoint),
		otlptracehttp.WithTimeout(cfg.Timeout),
	}

	if cfg.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	if cfg.Compression != "" && cfg.Compression != "none" {
		opts = append(opts, otlptracehttp.WithCompression(otlptracehttp.GzipCompression))
	}

	headers := cfg.ResolvedAuthHeaders()
	if len(headers) > 0 {
		opts = append(opts, otlptracehttp.WithHeaders(headers))
	}

	if cfg.Performance.RetryAttempts > 0 {
		opts = append(opts, otlptracehttp.WithRetry(otlptracehttp.RetryConfig{
			Enabled:         true,
			InitialInterval: cfg.Performance.RetryBackoff,
			MaxInterval:     cfg.Performance.RetryBackoff * 5,
			MaxElapsedTime:  cfg.Performance.RetryBackoff * time.Duration(cfg.Performance.RetryAttempts) * 5,
		}))
	}

	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP HTTP trace exporter: %w", err)
	}

	log.Info(ctx, "OTLP trace exporter initialized", logger.Fields{
		"protocol": "http", "endpoint": cfg.Endpoint,
	})

	return exporter, nil
}

// createSampler creates a sampler based on configuration.
// Fix: ratio sampler is always wrapped in ParentBased for correct distributed tracing.
func createSampler(sampling config.SamplingConfig) sdktrace.Sampler {
	var rootSampler sdktrace.Sampler

	switch sampling.Type {
	case "always", "always_on":
		return sdktrace.AlwaysSample()
	case "never", "always_off":
		return sdktrace.NeverSample()
	case "ratio", "traceidratio":
		rootSampler = sdktrace.TraceIDRatioBased(sampling.Rate)
	default:
		// Default: parent_based with ratio
		rootSampler = sdktrace.TraceIDRatioBased(sampling.Rate)
	}

	// Always wrap in ParentBased (fix: ratio was not wrapped before)
	return sdktrace.ParentBased(rootSampler)
}
