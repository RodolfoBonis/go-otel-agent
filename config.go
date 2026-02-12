package otelagent

import (
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/config"
)

// Re-export config types so consumers can use otelagent.Config, etc.
type Config = config.Config
type AuthConfig = config.AuthConfig
type TLSConfig = config.TLSConfig
type ResourceConfig = config.ResourceConfig
type TracesConfig = config.TracesConfig
type SamplingConfig = config.SamplingConfig
type MetricsConfig = config.MetricsConfig
type CardinalityConfig = config.CardinalityConfig
type LogsConfig = config.LogsConfig
type PerformanceConfig = config.PerformanceConfig
type FeaturesConfig = config.FeaturesConfig
type RouteExclusionConfig = config.RouteExclusionConfig
type ScrubConfig = config.ScrubConfig
type HTTPConfig = config.HTTPConfig

// LoadConfigFromEnv loads configuration from environment variables with smart defaults.
func LoadConfigFromEnv() *Config {
	env := getStringEnv("development", "ENV", "DEPLOYMENT_ENVIRONMENT")

	return &Config{
		Enabled:     getBoolEnv(true, "SIGNOZ_ENABLED", "OTEL_ENABLED"),
		ServiceName: getStringEnv("", "OTEL_SERVICE_NAME"),
		Namespace:   getStringEnv("", "OTEL_SERVICE_NAMESPACE"),
		Version:     getStringEnv("0.0.0", "OTEL_SERVICE_VERSION", "VERSION"),
		Environment: env,

		Endpoint:         stripURLScheme(getStringEnv("signoz-otel-collector.signoz.svc.cluster.local:4317", "OTEL_EXPORTER_OTLP_ENDPOINT")),
		ExporterProtocol: getStringEnv("grpc", "OTEL_EXPORTER_OTLP_PROTOCOL"),
		Insecure:         getBoolEnv(true, "OTEL_EXPORTER_OTLP_INSECURE"),
		Timeout:          getDurationEnv("OTEL_EXPORTER_OTLP_TIMEOUT", 10*time.Second),
		Compression:      getStringEnv("gzip", "OTEL_EXPORTER_OTLP_COMPRESSION"),

		Auth: loadAuthConfig(),
		TLS:  loadTLSConfig(),

		Resource:       loadResourceConfig(env),
		Traces:         loadTracesConfig(env),
		Metrics:        loadMetricsConfig(),
		Logs:           loadLogsConfig(),
		Performance:    loadPerformanceConfig(),
		Features:       loadFeaturesConfig(env),
		RouteExclusion: loadRouteExclusionConfig(),
		Scrub:          loadScrubConfig(),
		HTTP:           loadHTTPConfig(),
	}
}

func loadAuthConfig() AuthConfig {
	cfg := AuthConfig{
		Headers:        make(map[string]string),
		HeadersFromEnv: make(map[string]string),
	}

	if token := os.Getenv("SIGNOZ_ACCESS_TOKEN"); token != "" {
		cfg.Headers["signoz-access-token"] = token
	}

	if headerStr := os.Getenv("OTEL_EXPORTER_OTLP_HEADERS"); headerStr != "" {
		pairs := strings.Split(headerStr, ",")
		for _, pair := range pairs {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 {
				cfg.Headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}

	return cfg
}

func loadTLSConfig() TLSConfig {
	return TLSConfig{
		Insecure:           getBoolEnv(true, "OTEL_EXPORTER_OTLP_INSECURE"),
		CAFile:             getStringEnv("", "OTEL_EXPORTER_OTLP_CERTIFICATE"),
		CertFile:           getStringEnv("", "OTEL_EXPORTER_OTLP_CLIENT_CERTIFICATE"),
		KeyFile:            getStringEnv("", "OTEL_EXPORTER_OTLP_CLIENT_KEY"),
		InsecureSkipVerify: getBoolEnv(false, "OTEL_EXPORTER_OTLP_TLS_SKIP_VERIFY"),
		MinVersion:         getStringEnv("1.2", "OTEL_EXPORTER_OTLP_TLS_MIN_VERSION"),
	}
}

func loadResourceConfig(env string) ResourceConfig {
	return ResourceConfig{
		ServiceNamespace:      getStringEnv("", "OTEL_SERVICE_NAMESPACE"),
		ServiceInstance:       getStringEnv(getHostname(), "OTEL_SERVICE_INSTANCE"),
		DeploymentEnvironment: env,

		K8sPodName:     getStringEnv("", "POD_NAME", "K8S_POD_NAME"),
		K8sPodIP:       getStringEnv("", "POD_IP", "K8S_POD_IP"),
		K8sNamespace:   getStringEnv("", "POD_NAMESPACE", "K8S_NAMESPACE"),
		K8sNodeName:    getStringEnv("", "NODE_NAME", "K8S_NODE_NAME"),
		K8sClusterName: getStringEnv("", "K8S_CLUSTER_NAME"),

		ContainerName: getStringEnv("", "CONTAINER_NAME"),
		ContainerID:   getStringEnv("", "CONTAINER_ID"),

		CustomAttributes: parseKeyValuePairs(os.Getenv("OTEL_RESOURCE_ATTRIBUTES")),
	}
}

func loadTracesConfig(env string) TracesConfig {
	return TracesConfig{
		Enabled: getBoolEnv(true, "OTEL_TRACES_ENABLED"),

		Sampling: SamplingConfig{
			Type:     getStringEnv("parent_based", "OTEL_TRACES_SAMPLER"),
			Rate:     getFloat64Env("OTEL_TRACES_SAMPLER_ARG", defaultSamplingRate(env)),
			PerRoute: parsePerRouteSampling(os.Getenv("OTEL_TRACES_SAMPLING_ROUTES")),
		},

		MaxAttributesPerSpan: getIntEnv("OTEL_SPAN_ATTRIBUTE_COUNT_LIMIT", 128),
		MaxEventsPerSpan:     getIntEnv("OTEL_SPAN_EVENT_COUNT_LIMIT", 128),
		MaxLinksPerSpan:      getIntEnv("OTEL_SPAN_LINK_COUNT_LIMIT", 128),

		BatchTimeout:   getDurationEnv("OTEL_BSP_SCHEDULE_DELAY", 5*time.Second),
		BatchSize:      getIntEnv("OTEL_BSP_MAX_EXPORT_BATCH_SIZE", 512),
		QueueSize:      getIntEnv("OTEL_BSP_MAX_QUEUE_SIZE", 2048),
		MaxExportBatch: getIntEnv("OTEL_BSP_EXPORT_BATCH_SIZE", 512),

		ExcludedPaths: getStringSliceEnv("OTEL_TRACES_EXCLUDED_PATHS", []string{
			"/health", "/healthz", "/health_check", "/metrics", "/ready", "/live",
		}),
	}
}

func loadMetricsConfig() MetricsConfig {
	return MetricsConfig{
		Enabled: getBoolEnv(true, "OTEL_METRICS_ENABLED"),

		DefaultInterval: getDurationEnv("OTEL_METRIC_EXPORT_INTERVAL", 30*time.Second),
		RuntimeInterval: getDurationEnv("OTEL_RUNTIME_METRIC_INTERVAL", 10*time.Second),

		HTTP:     getBoolEnv(true, "OTEL_METRICS_HTTP_ENABLED"),
		Database: getBoolEnv(true, "OTEL_METRICS_DATABASE_ENABLED"),
		Redis:    getBoolEnv(true, "OTEL_METRICS_REDIS_ENABLED"),
		AMQP:     getBoolEnv(true, "OTEL_METRICS_AMQP_ENABLED"),
		Runtime:  getBoolEnv(true, "OTEL_METRICS_RUNTIME_ENABLED"),
		Business: getBoolEnv(true, "OTEL_METRICS_BUSINESS_ENABLED"),

		CPU:    getBoolEnv(true, "OTEL_METRICS_CPU_ENABLED"),
		Memory: getBoolEnv(true, "OTEL_METRICS_MEMORY_ENABLED"),
		Disk:   getBoolEnv(false, "OTEL_METRICS_DISK_ENABLED"),

		HTTPLatencyBoundaries: getFloat64SliceEnv("OTEL_HTTP_LATENCY_BOUNDARIES",
			[]float64{0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1.0, 2.5, 5.0, 7.5, 10.0}),
		DBLatencyBoundaries: getFloat64SliceEnv("OTEL_DB_LATENCY_BOUNDARIES",
			[]float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0}),

		Cardinality: CardinalityConfig{
			DropAttributes:     getStringSliceEnv("OTEL_METRICS_DROP_ATTRIBUTES", []string{"error_message", "user_id"}),
			MaxAttributeLength: getIntEnv("OTEL_METRICS_MAX_ATTR_LENGTH", 256),
			UseExponentialHist: getBoolEnv(false, "OTEL_METRICS_EXPONENTIAL_HIST"),
		},
	}
}

func loadLogsConfig() LogsConfig {
	return LogsConfig{
		Enabled: getBoolEnv(true, "OTEL_LOGS_ENABLED"),

		TraceCorrelation: getBoolEnv(true, "OTEL_LOGS_TRACE_CORRELATION"),
		SpanCorrelation:  getBoolEnv(true, "OTEL_LOGS_SPAN_CORRELATION"),
		ExportLevels:     getStringSliceEnv("OTEL_LOGS_EXPORT_LEVELS", []string{"info", "warn", "error"}),

		BatchTimeout: getDurationEnv("OTEL_BLRP_SCHEDULE_DELAY", 5*time.Second),
		BatchSize:    getIntEnv("OTEL_BLRP_MAX_EXPORT_BATCH_SIZE", 512),
		QueueSize:    getIntEnv("OTEL_BLRP_MAX_QUEUE_SIZE", 2048),

		StructuredFields: getBoolEnv(true, "OTEL_LOGS_STRUCTURED"),
		CustomFields:     parseKeyValuePairs(os.Getenv("OTEL_LOGS_CUSTOM_FIELDS")),
	}
}

func loadPerformanceConfig() PerformanceConfig {
	return PerformanceConfig{
		MaxMemoryUsage:     getInt64Env("OTEL_MAX_MEMORY_USAGE", 128*1024*1024),
		MemoryLimitPercent: getIntEnv("OTEL_MEMORY_LIMIT_PERCENT", 10),
		MaxCPUUsage:        getFloat64Env("OTEL_MAX_CPU_USAGE", 0.1),
		WorkerPoolSize:     getIntEnv("OTEL_WORKER_POOL_SIZE", 4),
		QueueBufferSize:    getIntEnv("OTEL_QUEUE_BUFFER_SIZE", 1000),

		MaxBatchSize:   getIntEnv("OTEL_MAX_BATCH_SIZE", 1000),
		FlushTimeout:   getDurationEnv("OTEL_FLUSH_TIMEOUT", 5*time.Second),
		RetryAttempts:  getIntEnv("OTEL_RETRY_ATTEMPTS", 3),
		RetryBackoff:   getDurationEnv("OTEL_RETRY_BACKOFF", 1*time.Second),
		ConnectionPool: getIntEnv("OTEL_CONNECTION_POOL", 5),

		AdaptiveSampling:   getBoolEnv(true, "OTEL_ADAPTIVE_SAMPLING"),
		ErrorSamplingBoost: getFloat64Env("OTEL_ERROR_SAMPLING_BOOST", 5.0),
	}
}

func loadFeaturesConfig(env string) FeaturesConfig {
	return FeaturesConfig{
		AutoHTTP:     getBoolEnv(true, "OTEL_AUTO_HTTP"),
		AutoDatabase: getBoolEnv(true, "OTEL_AUTO_DATABASE"),
		AutoRedis:    getBoolEnv(true, "OTEL_AUTO_REDIS"),
		AutoAMQP:     getBoolEnv(true, "OTEL_AUTO_AMQP"),

		DistributedTracing: getBoolEnv(true, "OTEL_DISTRIBUTED_TRACING"),
		ErrorTracking:      getBoolEnv(true, "OTEL_ERROR_TRACKING"),
		PerformanceMonitor: getBoolEnv(true, "OTEL_PERFORMANCE_MONITOR"),
		BusinessMetrics:    getBoolEnv(true, "OTEL_BUSINESS_METRICS"),

		HealthChecks:    getBoolEnv(true, "OTEL_HEALTH_CHECKS"),
		ReadinessProbes: getBoolEnv(true, "OTEL_READINESS_PROBES"),
		LivenessProbes:  getBoolEnv(true, "OTEL_LIVENESS_PROBES"),

		DebugMode: getBoolEnv(env == "development", "OTEL_DEBUG_MODE"),
		DryRun:    getBoolEnv(false, "OTEL_DRY_RUN"),
	}
}

func loadRouteExclusionConfig() RouteExclusionConfig {
	return RouteExclusionConfig{
		ExactPaths: getStringSliceEnv("OTEL_TRACES_EXCLUDED_PATHS", []string{
			"/health", "/healthz", "/health_check", "/metrics", "/ready", "/live",
		}),
		PrefixPaths: getStringSliceEnv("OTEL_TRACES_EXCLUDED_PREFIXES", nil),
		Patterns:    getStringSliceEnv("OTEL_TRACES_EXCLUDED_PATTERNS", nil),
	}
}

func loadScrubConfig() ScrubConfig {
	return ScrubConfig{
		Enabled:              getBoolEnv(false, "OTEL_PII_SCRUB_ENABLED"),
		SensitiveKeys:        getStringSliceEnv("OTEL_PII_SENSITIVE_KEYS", []string{"password", "token", "secret", "key", "email"}),
		SensitivePatterns:    getStringSliceEnv("OTEL_PII_SENSITIVE_PATTERNS", []string{".*password.*", ".*token.*", ".*secret.*"}),
		RedactedValue:        getStringEnv("[REDACTED]", "OTEL_PII_REDACTED_VALUE"),
		DBStatementMaxLength: getIntEnv("OTEL_PII_DB_STATEMENT_MAX_LENGTH", 2048),
	}
}

func loadHTTPConfig() config.HTTPConfig {
	return config.HTTPConfig{
		CaptureRequestHeaders:  getBoolEnv(true, "OTEL_HTTP_CAPTURE_REQUEST_HEADERS"),
		CaptureResponseHeaders: getBoolEnv(true, "OTEL_HTTP_CAPTURE_RESPONSE_HEADERS"),
		AllowedRequestHeaders:  getStringSliceEnv("OTEL_HTTP_ALLOWED_REQUEST_HEADERS", nil),
		AllowedResponseHeaders: getStringSliceEnv("OTEL_HTTP_ALLOWED_RESPONSE_HEADERS", nil),
		CaptureQueryParams:     getBoolEnv(true, "OTEL_HTTP_CAPTURE_QUERY_PARAMS"),
		CaptureRequestBody:     getBoolEnv(false, "OTEL_HTTP_CAPTURE_REQUEST_BODY"),
		CaptureResponseBody:    getBoolEnv(false, "OTEL_HTTP_CAPTURE_RESPONSE_BODY"),
		RequestBodyMaxSize:     getIntEnv("OTEL_HTTP_REQUEST_BODY_MAX_SIZE", 8192),
		ResponseBodyMaxSize:    getIntEnv("OTEL_HTTP_RESPONSE_BODY_MAX_SIZE", 8192),
		BodyAllowedContentTypes: getStringSliceEnv("OTEL_HTTP_BODY_ALLOWED_CONTENT_TYPES", []string{
			"application/json", "application/xml", "text/plain",
		}),
		RecordExceptionEvents: getBoolEnv(true, "OTEL_HTTP_RECORD_EXCEPTION_EVENTS"),
		SensitiveHeaders: getStringSliceEnv("OTEL_HTTP_SENSITIVE_HEADERS", []string{
			"authorization", "cookie", "set-cookie", "x-api-key", "x-auth-token",
		}),
	}
}

// --- Helper functions for env var parsing (FIXED) ---

// getStringEnv returns the value of the first non-empty env var, or defaultValue.
// FIXED: default value is now the first parameter, not mixed with keys.
func getStringEnv(defaultValue string, keys ...string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return defaultValue
}

func getBoolEnv(defaultValue bool, keys ...string) bool {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value == "true" || value == "1" || value == "yes"
		}
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getInt64Env(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getFloat64Env(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
		if ms, err := strconv.ParseInt(value, 10, 64); err == nil {
			return time.Duration(ms) * time.Millisecond
		}
	}
	return defaultValue
}

func getStringSliceEnv(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		parts := strings.Split(value, ",")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultValue
}

func getFloat64SliceEnv(key string, defaultValue []float64) []float64 {
	if value := os.Getenv(key); value != "" {
		parts := strings.Split(value, ",")
		result := make([]float64, 0, len(parts))
		for _, part := range parts {
			if f, err := strconv.ParseFloat(strings.TrimSpace(part), 64); err == nil {
				result = append(result, f)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultValue
}

func defaultSamplingRate(env string) float64 {
	switch env {
	case "production":
		return 0.1
	case "staging":
		return 0.5
	default:
		return 1.0
	}
}

func getHostname() string {
	if hostname, err := os.Hostname(); err == nil {
		return hostname
	}
	return "unknown"
}

func parseKeyValuePairs(value string) map[string]string {
	result := make(map[string]string)
	if value == "" {
		return result
	}
	pairs := strings.Split(value, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return result
}

func parsePerRouteSampling(value string) map[string]float64 {
	result := make(map[string]float64)
	if value == "" {
		return result
	}
	pairs := strings.Split(value, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) == 2 {
			if rate, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
				result[strings.TrimSpace(parts[0])] = rate
			}
		}
	}
	return result
}

func stripURLScheme(endpoint string) string {
	if endpoint == "" {
		return endpoint
	}
	if parsedURL, err := url.Parse(endpoint); err == nil && parsedURL.Host != "" {
		return parsedURL.Host
	}
	return endpoint
}
