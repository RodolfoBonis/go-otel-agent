package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/config"
	"github.com/RodolfoBonis/go-otel-agent/logger"
	otlpmetricgrpc "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	otlpmetrichttp "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

// NewMetricProvider creates a MeterProvider with OTLP exporter.
func NewMetricProvider(cfg *config.Config, res *resource.Resource, log logger.Logger) (*metric.MeterProvider, error) {
	ctx := context.Background()

	exporter, err := createMetricExporter(ctx, cfg, log)
	if err != nil {
		return nil, err
	}

	opts := []metric.Option{
		metric.WithReader(metric.NewPeriodicReader(exporter,
			metric.WithInterval(cfg.Metrics.DefaultInterval),
		)),
		metric.WithResource(res),
	}

	return metric.NewMeterProvider(opts...), nil
}

func createMetricExporter(ctx context.Context, cfg *config.Config, log logger.Logger) (metric.Exporter, error) {
	protocol := cfg.ExporterProtocol
	if protocol == "" {
		protocol = "grpc"
	}

	switch protocol {
	case "grpc":
		return createGRPCMetricExporter(ctx, cfg, log)
	case "http", "http/protobuf":
		return createHTTPMetricExporter(ctx, cfg, log)
	default:
		return nil, fmt.Errorf("unsupported OTLP protocol: %s (use 'grpc' or 'http')", protocol)
	}
}

func createGRPCMetricExporter(ctx context.Context, cfg *config.Config, log logger.Logger) (metric.Exporter, error) {
	opts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(cfg.Endpoint),
		otlpmetricgrpc.WithTimeout(cfg.Timeout),
	}

	if cfg.Insecure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	}
	if cfg.Compression != "" && cfg.Compression != "none" {
		opts = append(opts, otlpmetricgrpc.WithCompressor(cfg.Compression))
	}

	headers := cfg.ResolvedAuthHeaders()
	if len(headers) > 0 {
		opts = append(opts, otlpmetricgrpc.WithHeaders(headers))
	}

	if cfg.Performance.RetryAttempts > 0 {
		opts = append(opts, otlpmetricgrpc.WithRetry(otlpmetricgrpc.RetryConfig{
			Enabled:         true,
			InitialInterval: cfg.Performance.RetryBackoff,
			MaxInterval:     cfg.Performance.RetryBackoff * 5,
			MaxElapsedTime:  cfg.Performance.RetryBackoff * time.Duration(cfg.Performance.RetryAttempts) * 5,
		}))
	}

	exporter, err := otlpmetricgrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP gRPC metric exporter: %w", err)
	}

	log.Info(ctx, "OTLP metric exporter initialized", logger.Fields{
		"protocol": "grpc", "endpoint": cfg.Endpoint,
	})

	return exporter, nil
}

func createHTTPMetricExporter(ctx context.Context, cfg *config.Config, log logger.Logger) (metric.Exporter, error) {
	opts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(cfg.Endpoint),
		otlpmetrichttp.WithTimeout(cfg.Timeout),
	}

	if cfg.Insecure {
		opts = append(opts, otlpmetrichttp.WithInsecure())
	}
	if cfg.Compression != "" && cfg.Compression != "none" {
		opts = append(opts, otlpmetrichttp.WithCompression(otlpmetrichttp.GzipCompression))
	}

	headers := cfg.ResolvedAuthHeaders()
	if len(headers) > 0 {
		opts = append(opts, otlpmetrichttp.WithHeaders(headers))
	}

	if cfg.Performance.RetryAttempts > 0 {
		opts = append(opts, otlpmetrichttp.WithRetry(otlpmetrichttp.RetryConfig{
			Enabled:         true,
			InitialInterval: cfg.Performance.RetryBackoff,
			MaxInterval:     cfg.Performance.RetryBackoff * 5,
			MaxElapsedTime:  cfg.Performance.RetryBackoff * time.Duration(cfg.Performance.RetryAttempts) * 5,
		}))
	}

	exporter, err := otlpmetrichttp.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP HTTP metric exporter: %w", err)
	}

	log.Info(ctx, "OTLP metric exporter initialized", logger.Fields{
		"protocol": "http", "endpoint": cfg.Endpoint,
	})

	return exporter, nil
}
