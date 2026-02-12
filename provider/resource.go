package provider

import (
	"context"
	"runtime"

	"github.com/RodolfoBonis/go-otel-agent/config"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// BuildResource creates an OTel Resource from the agent config.
func BuildResource(cfg *config.Config) (*resource.Resource, error) {
	attrs := []attribute.KeyValue{
		semconv.ServiceName(cfg.ServiceName),
		semconv.ServiceVersion(cfg.Version),
		attribute.String("service.namespace", cfg.Namespace),
		attribute.String("environment", cfg.Environment),
		attribute.String("service.instance.id", cfg.Resource.ServiceInstance),
	}

	if cfg.Resource.DeploymentEnvironment != "" {
		attrs = append(attrs, attribute.String("deployment.environment", cfg.Resource.DeploymentEnvironment))
	}

	// K8s attributes
	if cfg.Resource.K8sPodName != "" {
		attrs = append(attrs, semconv.K8SPodName(cfg.Resource.K8sPodName))
	}
	if cfg.Resource.K8sPodIP != "" {
		attrs = append(attrs, attribute.String("k8s.pod.ip", cfg.Resource.K8sPodIP))
	}
	if cfg.Resource.K8sNamespace != "" {
		attrs = append(attrs, semconv.K8SNamespaceName(cfg.Resource.K8sNamespace))
	}
	if cfg.Resource.K8sNodeName != "" {
		attrs = append(attrs, semconv.K8SNodeName(cfg.Resource.K8sNodeName))
	}
	if cfg.Resource.K8sClusterName != "" {
		attrs = append(attrs, semconv.K8SClusterName(cfg.Resource.K8sClusterName))
	}

	// Container attributes
	if cfg.Resource.ContainerName != "" {
		attrs = append(attrs, semconv.ContainerName(cfg.Resource.ContainerName))
	}
	if cfg.Resource.ContainerID != "" {
		attrs = append(attrs, semconv.ContainerID(cfg.Resource.ContainerID))
	}

	// Custom attributes
	for key, value := range cfg.Resource.CustomAttributes {
		attrs = append(attrs, attribute.String(key, value))
	}

	// Runtime information
	attrs = append(attrs,
		semconv.ProcessRuntimeName("go"),
		semconv.ProcessRuntimeVersion(runtime.Version()),
		semconv.ProcessRuntimeDescription("Go runtime"),
	)

	return resource.New(context.Background(),
		resource.WithAttributes(attrs...),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithProcess(),
		resource.WithOS(),
	)
}
