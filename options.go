package otelagent

import (
	"github.com/RodolfoBonis/go-otel-agent/logger"
)

// Option configures the Agent.
type Option func(*Agent)

// WithConfig overrides the entire configuration.
func WithConfig(cfg *Config) Option {
	return func(a *Agent) {
		a.config = cfg
	}
}

// WithLogger sets a custom logger.
func WithLogger(l logger.Logger) Option {
	return func(a *Agent) {
		a.logger = l
	}
}

// WithServiceName sets the service name.
func WithServiceName(name string) Option {
	return func(a *Agent) {
		a.config.ServiceName = name
	}
}

// WithServiceNamespace sets the service namespace.
func WithServiceNamespace(ns string) Option {
	return func(a *Agent) {
		a.config.Namespace = ns
	}
}

// WithServiceVersion sets the service version.
func WithServiceVersion(version string) Option {
	return func(a *Agent) {
		a.config.Version = version
	}
}

// WithEndpoint sets the OTLP collector endpoint.
func WithEndpoint(endpoint string) Option {
	return func(a *Agent) {
		a.config.Endpoint = endpoint
	}
}

// WithSamplingRate sets the trace sampling rate (0.0 to 1.0).
func WithSamplingRate(rate float64) Option {
	return func(a *Agent) {
		a.config.Traces.Sampling.Rate = rate
	}
}

// WithDisabledSignals disables specific telemetry signals.
func WithDisabledSignals(signals ...Signal) Option {
	return func(a *Agent) {
		for _, s := range signals {
			switch s {
			case SignalTraces:
				a.config.Traces.Enabled = false
			case SignalMetrics:
				a.config.Metrics.Enabled = false
			case SignalLogs:
				a.config.Logs.Enabled = false
			}
		}
	}
}

// WithAutoInstrumentation enables/disables auto-instrumentation per component.
func WithAutoInstrumentation(http, database, redis, amqp bool) Option {
	return func(a *Agent) {
		a.config.Features.AutoHTTP = http
		a.config.Features.AutoDatabase = database
		a.config.Features.AutoRedis = redis
		a.config.Features.AutoAMQP = amqp
	}
}

// WithRouteExclusions sets route exclusion configuration.
func WithRouteExclusions(cfg RouteExclusionConfig) Option {
	return func(a *Agent) {
		a.config.RouteExclusion = cfg
	}
}

// WithEnvironment sets the deployment environment.
func WithEnvironment(env string) Option {
	return func(a *Agent) {
		a.config.Environment = env
	}
}

// WithInsecure sets whether to use insecure connection.
func WithInsecure(insecure bool) Option {
	return func(a *Agent) {
		a.config.Insecure = insecure
	}
}

// WithEnabled sets whether observability is enabled.
func WithEnabled(enabled bool) Option {
	return func(a *Agent) {
		a.config.Enabled = enabled
	}
}

// WithAuthHeaders sets authentication headers for the OTLP exporter.
func WithAuthHeaders(headers map[string]string) Option {
	return func(a *Agent) {
		a.config.Auth.Headers = headers
	}
}

// WithDebugMode enables debug mode.
func WithDebugMode(debug bool) Option {
	return func(a *Agent) {
		a.config.Features.DebugMode = debug
	}
}
