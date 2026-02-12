package config

import (
	"os"
	"time"
)

// Config holds comprehensive observability configuration.
type Config struct {
	// General settings
	Enabled     bool   `json:"enabled"`
	ServiceName string `json:"service_name"`
	Namespace   string `json:"namespace"`
	Version     string `json:"version"`
	Environment string `json:"environment"`

	// Export settings
	Endpoint         string        `json:"endpoint"`
	ExporterProtocol string        `json:"exporter_protocol"`
	Insecure         bool          `json:"insecure"`
	Timeout          time.Duration `json:"timeout"`
	Compression      string        `json:"compression"`

	// Auth for SigNoz Cloud / secured collectors
	Auth AuthConfig `json:"auth"`

	// TLS configuration
	TLS TLSConfig `json:"tls"`

	// Resource attributes
	Resource ResourceConfig `json:"resource"`

	// Component-specific settings
	Traces  TracesConfig  `json:"traces"`
	Metrics MetricsConfig `json:"metrics"`
	Logs    LogsConfig    `json:"logs"`

	// Performance settings
	Performance PerformanceConfig `json:"performance"`

	// Features
	Features FeaturesConfig `json:"features"`

	// Route exclusions
	RouteExclusion RouteExclusionConfig `json:"route_exclusion"`

	// PII scrubbing
	Scrub ScrubConfig `json:"scrub"`

	// HTTP capture settings
	HTTP HTTPConfig `json:"http"`
}

// AuthConfig holds authentication headers for OTLP exporters.
type AuthConfig struct {
	Headers        map[string]string `json:"headers"`
	HeadersFromEnv map[string]string `json:"headers_from_env"`
}

// TLSConfig holds TLS settings for OTLP exporters.
type TLSConfig struct {
	Insecure           bool   `json:"insecure"`
	CAFile             string `json:"ca_file"`
	CertFile           string `json:"cert_file"`
	KeyFile            string `json:"key_file"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify"`
	MinVersion         string `json:"min_version"`
}

// ResourceConfig defines resource attributes.
type ResourceConfig struct {
	ServiceNamespace      string `json:"service_namespace"`
	ServiceInstance       string `json:"service_instance"`
	DeploymentEnvironment string `json:"deployment_environment"`

	// K8s attributes (auto-detected)
	K8sPodName     string `json:"k8s_pod_name"`
	K8sPodIP       string `json:"k8s_pod_ip"`
	K8sNamespace   string `json:"k8s_namespace"`
	K8sNodeName    string `json:"k8s_node_name"`
	K8sClusterName string `json:"k8s_cluster_name"`

	// Container attributes
	ContainerName string `json:"container_name"`
	ContainerID   string `json:"container_id"`

	// Custom attributes
	CustomAttributes map[string]string `json:"custom_attributes"`
}

// TracesConfig configures tracing behavior.
type TracesConfig struct {
	Enabled bool `json:"enabled"`

	// Sampling configuration
	Sampling SamplingConfig `json:"sampling"`

	// Span limits
	MaxAttributesPerSpan int `json:"max_attributes_per_span"`
	MaxEventsPerSpan     int `json:"max_events_per_span"`
	MaxLinksPerSpan      int `json:"max_links_per_span"`

	// Span processors
	BatchTimeout   time.Duration `json:"batch_timeout"`
	BatchSize      int           `json:"batch_size"`
	QueueSize      int           `json:"queue_size"`
	MaxExportBatch int           `json:"max_export_batch"`

	// Filtering
	ExcludedPaths []string `json:"excluded_paths"`
}

// SamplingConfig defines sampling strategies.
type SamplingConfig struct {
	Type     string             `json:"type"`
	Rate     float64            `json:"rate"`
	PerRoute map[string]float64 `json:"per_route"` // route -> rate
}

// MetricsConfig configures metrics behavior.
type MetricsConfig struct {
	Enabled bool `json:"enabled"`

	// Collection intervals
	DefaultInterval time.Duration `json:"default_interval"`
	RuntimeInterval time.Duration `json:"runtime_interval"`

	// Metric types to collect
	HTTP     bool `json:"http"`
	Database bool `json:"database"`
	Redis    bool `json:"redis"`
	AMQP     bool `json:"amqp"`
	Runtime  bool `json:"runtime"`
	Business bool `json:"business"`

	// Resource metrics
	CPU    bool `json:"cpu"`
	Memory bool `json:"memory"`
	Disk   bool `json:"disk"`

	// Histogram boundaries
	HTTPLatencyBoundaries []float64 `json:"http_latency_boundaries"`
	DBLatencyBoundaries   []float64 `json:"db_latency_boundaries"`

	// Cardinality control
	Cardinality CardinalityConfig `json:"cardinality"`
}

// CardinalityConfig controls metric cardinality.
type CardinalityConfig struct {
	DropAttributes     []string `json:"drop_attributes"`
	MaxAttributeLength int      `json:"max_attribute_length"`
	UseExponentialHist bool     `json:"use_exponential_hist"`
}

// LogsConfig configures logging behavior.
type LogsConfig struct {
	Enabled bool `json:"enabled"`

	TraceCorrelation bool     `json:"trace_correlation"`
	SpanCorrelation  bool     `json:"span_correlation"`
	ExportLevels     []string `json:"export_levels"`

	BatchTimeout time.Duration `json:"batch_timeout"`
	BatchSize    int           `json:"batch_size"`
	QueueSize    int           `json:"queue_size"`

	StructuredFields bool              `json:"structured_fields"`
	CustomFields     map[string]string `json:"custom_fields"`
}

// PerformanceConfig optimizes performance.
type PerformanceConfig struct {
	MaxMemoryUsage     int64   `json:"max_memory_usage"`
	MemoryLimitPercent int     `json:"memory_limit_percent"`
	MaxCPUUsage        float64 `json:"max_cpu_usage"`
	WorkerPoolSize     int     `json:"worker_pool_size"`
	QueueBufferSize    int     `json:"queue_buffer_size"`

	MaxBatchSize   int           `json:"max_batch_size"`
	FlushTimeout   time.Duration `json:"flush_timeout"`
	RetryAttempts  int           `json:"retry_attempts"`
	RetryBackoff   time.Duration `json:"retry_backoff"`
	ConnectionPool int           `json:"connection_pool"`

	AdaptiveSampling   bool    `json:"adaptive_sampling"`
	ErrorSamplingBoost float64 `json:"error_sampling_boost"`
}

// FeaturesConfig enables/disables specific features.
type FeaturesConfig struct {
	AutoHTTP     bool `json:"auto_http"`
	AutoDatabase bool `json:"auto_database"`
	AutoRedis    bool `json:"auto_redis"`
	AutoAMQP     bool `json:"auto_amqp"`

	DistributedTracing bool `json:"distributed_tracing"`
	ErrorTracking      bool `json:"error_tracking"`
	PerformanceMonitor bool `json:"performance_monitor"`
	BusinessMetrics    bool `json:"business_metrics"`

	HealthChecks    bool `json:"health_checks"`
	ReadinessProbes bool `json:"readiness_probes"`
	LivenessProbes  bool `json:"liveness_probes"`

	DebugMode bool `json:"debug_mode"`
	DryRun    bool `json:"dry_run"`
}

// RouteExclusionConfig configures route exclusions for tracing and metrics.
type RouteExclusionConfig struct {
	ExactPaths  []string `json:"exact_paths"`
	PrefixPaths []string `json:"prefix_paths"`
	Patterns    []string `json:"patterns"`
}

// ScrubConfig configures PII scrubbing.
type ScrubConfig struct {
	Enabled              bool     `json:"enabled"`
	SensitiveKeys        []string `json:"sensitive_keys"`
	SensitivePatterns    []string `json:"sensitive_patterns"`
	RedactedValue        string   `json:"redacted_value"`
	DBStatementMaxLength int      `json:"db_statement_max_length"`
}

// HTTPConfig configures HTTP request/response capture for spans.
type HTTPConfig struct {
	CaptureRequestHeaders  bool     `json:"capture_request_headers"`
	CaptureResponseHeaders bool     `json:"capture_response_headers"`
	AllowedRequestHeaders  []string `json:"allowed_request_headers"`
	AllowedResponseHeaders []string `json:"allowed_response_headers"`
	CaptureQueryParams     bool     `json:"capture_query_params"`
	CaptureRequestBody     bool     `json:"capture_request_body"`
	CaptureResponseBody    bool     `json:"capture_response_body"`
	RequestBodyMaxSize     int      `json:"request_body_max_size"`
	ResponseBodyMaxSize    int      `json:"response_body_max_size"`
	BodyAllowedContentTypes []string `json:"body_allowed_content_types"`
	RecordExceptionEvents  bool     `json:"record_exception_events"`
	SensitiveHeaders       []string `json:"sensitive_headers"`
}

// ResolvedAuthHeaders returns all auth headers with env vars resolved.
func (c *Config) ResolvedAuthHeaders() map[string]string {
	headers := make(map[string]string)
	for k, v := range c.Auth.Headers {
		headers[k] = v
	}
	for k, envKey := range c.Auth.HeadersFromEnv {
		if val := os.Getenv(envKey); val != "" {
			headers[k] = val
		}
	}
	return headers
}
