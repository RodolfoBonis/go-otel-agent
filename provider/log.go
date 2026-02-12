package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/config"
	"github.com/RodolfoBonis/go-otel-agent/logger"
	otlploggrpc "go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	otlploghttp "go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
)

// NewLogProvider creates a LoggerProvider with OTLP exporter.
func NewLogProvider(cfg *config.Config, res *resource.Resource, lgr logger.Logger) (*log.LoggerProvider, error) {
	ctx := context.Background()

	exporter, err := createLogExporter(ctx, cfg, lgr)
	if err != nil {
		return nil, err
	}

	provider := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(exporter,
			log.WithExportTimeout(cfg.Logs.BatchTimeout),
			log.WithExportMaxBatchSize(cfg.Logs.BatchSize),
			log.WithExportInterval(5*time.Second),
		)),
		log.WithResource(res),
	)

	return provider, nil
}

func createLogExporter(ctx context.Context, cfg *config.Config, lgr logger.Logger) (log.Exporter, error) {
	protocol := cfg.ExporterProtocol
	if protocol == "" {
		protocol = "grpc"
	}

	switch protocol {
	case "grpc":
		return createGRPCLogExporter(ctx, cfg, lgr)
	case "http", "http/protobuf":
		return createHTTPLogExporter(ctx, cfg, lgr)
	default:
		return nil, fmt.Errorf("unsupported OTLP protocol: %s (use 'grpc' or 'http')", protocol)
	}
}

func createGRPCLogExporter(ctx context.Context, cfg *config.Config, lgr logger.Logger) (log.Exporter, error) {
	opts := []otlploggrpc.Option{
		otlploggrpc.WithEndpoint(cfg.Endpoint),
		otlploggrpc.WithTimeout(cfg.Timeout),
	}

	if cfg.Insecure {
		opts = append(opts, otlploggrpc.WithInsecure())
	}
	if cfg.Compression != "" && cfg.Compression != "none" {
		opts = append(opts, otlploggrpc.WithCompressor(cfg.Compression))
	}

	headers := cfg.ResolvedAuthHeaders()
	if len(headers) > 0 {
		opts = append(opts, otlploggrpc.WithHeaders(headers))
	}

	exporter, err := otlploggrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP gRPC log exporter: %w", err)
	}

	lgr.Info(ctx, "OTLP log exporter initialized", logger.Fields{
		"protocol": "grpc", "endpoint": cfg.Endpoint,
	})

	return exporter, nil
}

func createHTTPLogExporter(ctx context.Context, cfg *config.Config, lgr logger.Logger) (log.Exporter, error) {
	opts := []otlploghttp.Option{
		otlploghttp.WithEndpoint(cfg.Endpoint),
		otlploghttp.WithTimeout(cfg.Timeout),
	}

	if cfg.Insecure {
		opts = append(opts, otlploghttp.WithInsecure())
	}
	if cfg.Compression != "" && cfg.Compression != "none" {
		opts = append(opts, otlploghttp.WithCompression(otlploghttp.GzipCompression))
	}

	headers := cfg.ResolvedAuthHeaders()
	if len(headers) > 0 {
		opts = append(opts, otlploghttp.WithHeaders(headers))
	}

	exporter, err := otlploghttp.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP HTTP log exporter: %w", err)
	}

	lgr.Info(ctx, "OTLP log exporter initialized", logger.Fields{
		"protocol": "http", "endpoint": cfg.Endpoint,
	})

	return exporter, nil
}
