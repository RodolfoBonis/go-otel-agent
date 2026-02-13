package otelagent

import (
	"os"
	"testing"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/config"
)

// ---------------------------------------------------------------------------
// LoadConfigFromEnv
// ---------------------------------------------------------------------------

func TestLoadConfigFromEnv_Defaults(t *testing.T) {
	// Clear env vars that LoadConfigFromEnv reads so we test pure defaults.
	envVars := []string{
		"ENV", "DEPLOYMENT_ENVIRONMENT",
		"SIGNOZ_ENABLED", "OTEL_ENABLED",
		"OTEL_SERVICE_NAME",
		"OTEL_SERVICE_NAMESPACE",
		"OTEL_SERVICE_VERSION", "VERSION",
		"OTEL_EXPORTER_OTLP_ENDPOINT",
		"OTEL_EXPORTER_OTLP_PROTOCOL",
		"OTEL_EXPORTER_OTLP_INSECURE",
		"OTEL_EXPORTER_OTLP_TIMEOUT",
		"OTEL_EXPORTER_OTLP_COMPRESSION",
	}
	for _, key := range envVars {
		t.Setenv(key, "")
	}

	cfg := LoadConfigFromEnv()

	if !cfg.Enabled {
		t.Error("expected Enabled to default to true")
	}
	if cfg.ServiceName != "" {
		t.Errorf("expected ServiceName to be empty, got %q", cfg.ServiceName)
	}
	if cfg.Version != "0.0.0" {
		t.Errorf("expected Version 0.0.0, got %q", cfg.Version)
	}
	if cfg.Environment != "development" {
		t.Errorf("expected Environment 'development', got %q", cfg.Environment)
	}
	if cfg.ExporterProtocol != "grpc" {
		t.Errorf("expected ExporterProtocol 'grpc', got %q", cfg.ExporterProtocol)
	}
	if !cfg.Insecure {
		t.Error("expected Insecure to default to true")
	}
	if cfg.Timeout != 10*time.Second {
		t.Errorf("expected Timeout 10s, got %v", cfg.Timeout)
	}
	if cfg.Compression != "gzip" {
		t.Errorf("expected Compression 'gzip', got %q", cfg.Compression)
	}
	// Default endpoint is the stripped form of the signoz cluster address.
	if cfg.Endpoint != "signoz-otel-collector.signoz.svc.cluster.local:4317" {
		t.Errorf("unexpected default Endpoint: %q", cfg.Endpoint)
	}
}

func TestLoadConfigFromEnv_CustomEnvVars(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "my-service")
	t.Setenv("OTEL_SERVICE_VERSION", "1.2.3")
	t.Setenv("ENV", "production")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "https://collector.example.com:4317")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http")
	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "false")
	t.Setenv("OTEL_EXPORTER_OTLP_TIMEOUT", "30s")
	t.Setenv("OTEL_EXPORTER_OTLP_COMPRESSION", "none")
	t.Setenv("SIGNOZ_ENABLED", "false")

	cfg := LoadConfigFromEnv()

	if cfg.Enabled {
		t.Error("expected Enabled=false when SIGNOZ_ENABLED=false")
	}
	if cfg.ServiceName != "my-service" {
		t.Errorf("expected ServiceName 'my-service', got %q", cfg.ServiceName)
	}
	if cfg.Version != "1.2.3" {
		t.Errorf("expected Version '1.2.3', got %q", cfg.Version)
	}
	if cfg.Environment != "production" {
		t.Errorf("expected Environment 'production', got %q", cfg.Environment)
	}
	// stripURLScheme should extract host from https://collector.example.com:4317
	if cfg.Endpoint != "collector.example.com:4317" {
		t.Errorf("expected stripped endpoint 'collector.example.com:4317', got %q", cfg.Endpoint)
	}
	if cfg.ExporterProtocol != "http" {
		t.Errorf("expected ExporterProtocol 'http', got %q", cfg.ExporterProtocol)
	}
	if cfg.Insecure {
		t.Error("expected Insecure=false")
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("expected Timeout 30s, got %v", cfg.Timeout)
	}
	if cfg.Compression != "none" {
		t.Errorf("expected Compression 'none', got %q", cfg.Compression)
	}
}

func TestLoadConfigFromEnv_SamplingRateProductionDefault(t *testing.T) {
	t.Setenv("ENV", "production")
	// Make sure OTEL_TRACES_SAMPLER_ARG is not set so we get the default.
	t.Setenv("OTEL_TRACES_SAMPLER_ARG", "")

	cfg := LoadConfigFromEnv()

	if cfg.Traces.Sampling.Rate != 0.1 {
		t.Errorf("expected production sampling rate 0.1, got %f", cfg.Traces.Sampling.Rate)
	}
}

func TestLoadConfigFromEnv_DebugModeDefaultsByEnvironment(t *testing.T) {
	tests := []struct {
		env       string
		wantDebug bool
	}{
		{"development", true},
		{"production", false},
		{"staging", false},
	}
	for _, tc := range tests {
		t.Run(tc.env, func(t *testing.T) {
			t.Setenv("ENV", tc.env)
			t.Setenv("OTEL_DEBUG_MODE", "")
			cfg := LoadConfigFromEnv()
			if cfg.Features.DebugMode != tc.wantDebug {
				t.Errorf("env=%q: expected DebugMode=%v, got %v", tc.env, tc.wantDebug, cfg.Features.DebugMode)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// getStringEnv
// ---------------------------------------------------------------------------

func TestGetStringEnv_DefaultWhenNoVars(t *testing.T) {
	t.Setenv("KEY_A", "")
	t.Setenv("KEY_B", "")

	got := getStringEnv("fallback", "KEY_A", "KEY_B")
	if got != "fallback" {
		t.Errorf("expected 'fallback', got %q", got)
	}
}

func TestGetStringEnv_FirstNonEmptyWins(t *testing.T) {
	t.Setenv("KEY_A", "")
	t.Setenv("KEY_B", "value-b")
	t.Setenv("KEY_C", "value-c")

	got := getStringEnv("fallback", "KEY_A", "KEY_B", "KEY_C")
	if got != "value-b" {
		t.Errorf("expected 'value-b', got %q", got)
	}
}

func TestGetStringEnv_FirstKeyTakesPrecedence(t *testing.T) {
	t.Setenv("KEY_A", "value-a")
	t.Setenv("KEY_B", "value-b")

	got := getStringEnv("fallback", "KEY_A", "KEY_B")
	if got != "value-a" {
		t.Errorf("expected 'value-a', got %q", got)
	}
}

// ---------------------------------------------------------------------------
// getBoolEnv
// ---------------------------------------------------------------------------

func TestGetBoolEnv_DefaultWhenUnset(t *testing.T) {
	t.Setenv("BOOL_KEY", "")

	if got := getBoolEnv(true, "BOOL_KEY"); !got {
		t.Error("expected default true")
	}
	if got := getBoolEnv(false, "BOOL_KEY"); got {
		t.Error("expected default false")
	}
}

func TestGetBoolEnv_TruthyValues(t *testing.T) {
	truthy := []string{"true", "1", "yes"}
	for _, v := range truthy {
		t.Run(v, func(t *testing.T) {
			t.Setenv("BOOL_KEY", v)
			if got := getBoolEnv(false, "BOOL_KEY"); !got {
				t.Errorf("expected true for %q", v)
			}
		})
	}
}

func TestGetBoolEnv_FalsyValues(t *testing.T) {
	falsy := []string{"false", "0", "no", "random"}
	for _, v := range falsy {
		t.Run(v, func(t *testing.T) {
			t.Setenv("BOOL_KEY", v)
			if got := getBoolEnv(true, "BOOL_KEY"); got {
				t.Errorf("expected false for %q", v)
			}
		})
	}
}

func TestGetBoolEnv_MultipleKeys(t *testing.T) {
	t.Setenv("BOOL_A", "")
	t.Setenv("BOOL_B", "true")

	if got := getBoolEnv(false, "BOOL_A", "BOOL_B"); !got {
		t.Error("expected true from second key")
	}
}

// ---------------------------------------------------------------------------
// getIntEnv
// ---------------------------------------------------------------------------

func TestGetIntEnv_Default(t *testing.T) {
	t.Setenv("INT_KEY", "")
	if got := getIntEnv("INT_KEY", 42); got != 42 {
		t.Errorf("expected 42, got %d", got)
	}
}

func TestGetIntEnv_Valid(t *testing.T) {
	t.Setenv("INT_KEY", "100")
	if got := getIntEnv("INT_KEY", 42); got != 100 {
		t.Errorf("expected 100, got %d", got)
	}
}

func TestGetIntEnv_InvalidFallsBackToDefault(t *testing.T) {
	t.Setenv("INT_KEY", "not-a-number")
	if got := getIntEnv("INT_KEY", 42); got != 42 {
		t.Errorf("expected default 42 for invalid input, got %d", got)
	}
}

// ---------------------------------------------------------------------------
// getFloat64Env
// ---------------------------------------------------------------------------

func TestGetFloat64Env_Default(t *testing.T) {
	t.Setenv("FLOAT_KEY", "")
	if got := getFloat64Env("FLOAT_KEY", 3.14); got != 3.14 {
		t.Errorf("expected 3.14, got %f", got)
	}
}

func TestGetFloat64Env_Valid(t *testing.T) {
	t.Setenv("FLOAT_KEY", "0.75")
	if got := getFloat64Env("FLOAT_KEY", 0.0); got != 0.75 {
		t.Errorf("expected 0.75, got %f", got)
	}
}

func TestGetFloat64Env_InvalidFallsBackToDefault(t *testing.T) {
	t.Setenv("FLOAT_KEY", "abc")
	if got := getFloat64Env("FLOAT_KEY", 1.5); got != 1.5 {
		t.Errorf("expected default 1.5 for invalid input, got %f", got)
	}
}

// ---------------------------------------------------------------------------
// getDurationEnv
// ---------------------------------------------------------------------------

func TestGetDurationEnv_Default(t *testing.T) {
	t.Setenv("DUR_KEY", "")
	if got := getDurationEnv("DUR_KEY", 5*time.Second); got != 5*time.Second {
		t.Errorf("expected 5s, got %v", got)
	}
}

func TestGetDurationEnv_GoDurationString(t *testing.T) {
	t.Setenv("DUR_KEY", "30s")
	if got := getDurationEnv("DUR_KEY", 0); got != 30*time.Second {
		t.Errorf("expected 30s, got %v", got)
	}
}

func TestGetDurationEnv_MillisecondsInteger(t *testing.T) {
	t.Setenv("DUR_KEY", "500")
	if got := getDurationEnv("DUR_KEY", 0); got != 500*time.Millisecond {
		t.Errorf("expected 500ms, got %v", got)
	}
}

func TestGetDurationEnv_InvalidFallsBackToDefault(t *testing.T) {
	t.Setenv("DUR_KEY", "not-a-duration")
	if got := getDurationEnv("DUR_KEY", 10*time.Second); got != 10*time.Second {
		t.Errorf("expected default 10s for invalid input, got %v", got)
	}
}

// ---------------------------------------------------------------------------
// getStringSliceEnv
// ---------------------------------------------------------------------------

func TestGetStringSliceEnv_Default(t *testing.T) {
	t.Setenv("SLICE_KEY", "")
	def := []string{"a", "b"}
	got := getStringSliceEnv("SLICE_KEY", def)
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("expected default [a b], got %v", got)
	}
}

func TestGetStringSliceEnv_CommaSeparated(t *testing.T) {
	t.Setenv("SLICE_KEY", " foo , bar , baz ")
	got := getStringSliceEnv("SLICE_KEY", nil)
	if len(got) != 3 || got[0] != "foo" || got[1] != "bar" || got[2] != "baz" {
		t.Errorf("expected [foo bar baz], got %v", got)
	}
}

func TestGetStringSliceEnv_EmptyItemsFiltered(t *testing.T) {
	t.Setenv("SLICE_KEY", "a,,b, ,c")
	got := getStringSliceEnv("SLICE_KEY", nil)
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Errorf("expected [a b c], got %v", got)
	}
}

// ---------------------------------------------------------------------------
// getFloat64SliceEnv
// ---------------------------------------------------------------------------

func TestGetFloat64SliceEnv_Default(t *testing.T) {
	t.Setenv("FSLICE_KEY", "")
	def := []float64{1.0, 2.0}
	got := getFloat64SliceEnv("FSLICE_KEY", def)
	if len(got) != 2 || got[0] != 1.0 || got[1] != 2.0 {
		t.Errorf("expected default [1 2], got %v", got)
	}
}

func TestGetFloat64SliceEnv_CommaSeparated(t *testing.T) {
	t.Setenv("FSLICE_KEY", "0.5, 1.0, 2.5")
	got := getFloat64SliceEnv("FSLICE_KEY", nil)
	if len(got) != 3 || got[0] != 0.5 || got[1] != 1.0 || got[2] != 2.5 {
		t.Errorf("expected [0.5 1 2.5], got %v", got)
	}
}

func TestGetFloat64SliceEnv_InvalidEntriesSkipped(t *testing.T) {
	t.Setenv("FSLICE_KEY", "1.0, bad, 3.0")
	got := getFloat64SliceEnv("FSLICE_KEY", nil)
	if len(got) != 2 || got[0] != 1.0 || got[1] != 3.0 {
		t.Errorf("expected [1 3], got %v", got)
	}
}

// ---------------------------------------------------------------------------
// parsePerRouteSampling
// ---------------------------------------------------------------------------

func TestParsePerRouteSampling_Empty(t *testing.T) {
	got := parsePerRouteSampling("")
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

func TestParsePerRouteSampling_SingleRoute(t *testing.T) {
	got := parsePerRouteSampling("/api/health:0.01")
	if rate, ok := got["/api/health"]; !ok || rate != 0.01 {
		t.Errorf("expected /api/health -> 0.01, got %v", got)
	}
}

func TestParsePerRouteSampling_MultipleRoutes(t *testing.T) {
	got := parsePerRouteSampling("/health:0.01, /api:0.5")
	if len(got) != 2 {
		t.Errorf("expected 2 entries, got %d", len(got))
	}
	if rate := got["/health"]; rate != 0.01 {
		t.Errorf("expected /health -> 0.01, got %f", rate)
	}
	if rate := got["/api"]; rate != 0.5 {
		t.Errorf("expected /api -> 0.5, got %f", rate)
	}
}

func TestParsePerRouteSampling_InvalidRateSkipped(t *testing.T) {
	got := parsePerRouteSampling("/good:0.5,/bad:notanumber,/also-good:0.1")
	if len(got) != 2 {
		t.Errorf("expected 2 valid entries, got %d: %v", len(got), got)
	}
}

func TestParsePerRouteSampling_MalformedEntrySkipped(t *testing.T) {
	got := parsePerRouteSampling("/health:0.5,nocolon,/api:1.0")
	if len(got) != 2 {
		t.Errorf("expected 2 entries (malformed skipped), got %d", len(got))
	}
}

// ---------------------------------------------------------------------------
// parseKeyValuePairs
// ---------------------------------------------------------------------------

func TestParseKeyValuePairs_Empty(t *testing.T) {
	got := parseKeyValuePairs("")
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

func TestParseKeyValuePairs_SinglePair(t *testing.T) {
	got := parseKeyValuePairs("env=prod")
	if got["env"] != "prod" {
		t.Errorf("expected env=prod, got %v", got)
	}
}

func TestParseKeyValuePairs_MultiplePairs(t *testing.T) {
	got := parseKeyValuePairs("env=prod, region=us-east-1")
	if len(got) != 2 {
		t.Errorf("expected 2 pairs, got %d", len(got))
	}
	if got["env"] != "prod" {
		t.Errorf("expected env=prod, got %q", got["env"])
	}
	if got["region"] != "us-east-1" {
		t.Errorf("expected region=us-east-1, got %q", got["region"])
	}
}

func TestParseKeyValuePairs_ValueWithEquals(t *testing.T) {
	got := parseKeyValuePairs("url=https://host?a=b")
	if got["url"] != "https://host?a=b" {
		t.Errorf("expected value with equals preserved, got %q", got["url"])
	}
}

func TestParseKeyValuePairs_MalformedEntrySkipped(t *testing.T) {
	got := parseKeyValuePairs("good=value,noequals,also_good=ok")
	if len(got) != 2 {
		t.Errorf("expected 2 valid pairs, got %d: %v", len(got), got)
	}
}

// ---------------------------------------------------------------------------
// stripURLScheme
// ---------------------------------------------------------------------------

func TestStripURLScheme_EmptyString(t *testing.T) {
	got := stripURLScheme("")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestStripURLScheme_HTTPSUrl(t *testing.T) {
	got := stripURLScheme("https://collector.example.com:4317")
	if got != "collector.example.com:4317" {
		t.Errorf("expected 'collector.example.com:4317', got %q", got)
	}
}

func TestStripURLScheme_HTTPUrl(t *testing.T) {
	got := stripURLScheme("http://localhost:4317")
	if got != "localhost:4317" {
		t.Errorf("expected 'localhost:4317', got %q", got)
	}
}

func TestStripURLScheme_NoScheme(t *testing.T) {
	got := stripURLScheme("collector.example.com:4317")
	// url.Parse without scheme will not populate Host, so the original is returned.
	if got != "collector.example.com:4317" {
		t.Errorf("expected original value returned, got %q", got)
	}
}

func TestStripURLScheme_GRPCScheme(t *testing.T) {
	got := stripURLScheme("grpc://collector.example.com:4317")
	if got != "collector.example.com:4317" {
		t.Errorf("expected 'collector.example.com:4317', got %q", got)
	}
}

// ---------------------------------------------------------------------------
// defaultSamplingRate
// ---------------------------------------------------------------------------

func TestDefaultSamplingRate_Production(t *testing.T) {
	if got := defaultSamplingRate("production"); got != 0.1 {
		t.Errorf("expected 0.1, got %f", got)
	}
}

func TestDefaultSamplingRate_Staging(t *testing.T) {
	if got := defaultSamplingRate("staging"); got != 0.5 {
		t.Errorf("expected 0.5, got %f", got)
	}
}

func TestDefaultSamplingRate_Development(t *testing.T) {
	if got := defaultSamplingRate("development"); got != 1.0 {
		t.Errorf("expected 1.0, got %f", got)
	}
}

func TestDefaultSamplingRate_UnknownEnvironment(t *testing.T) {
	if got := defaultSamplingRate("custom-env"); got != 1.0 {
		t.Errorf("expected 1.0 for unknown env, got %f", got)
	}
}

// ---------------------------------------------------------------------------
// Config.ResolvedAuthHeaders (from config/types.go)
// ---------------------------------------------------------------------------

func TestResolvedAuthHeaders_StaticHeaders(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			Headers:        map[string]string{"x-api-key": "secret123"},
			HeadersFromEnv: map[string]string{},
		},
	}

	got := cfg.ResolvedAuthHeaders()
	if got["x-api-key"] != "secret123" {
		t.Errorf("expected x-api-key=secret123, got %q", got["x-api-key"])
	}
}

func TestResolvedAuthHeaders_FromEnv(t *testing.T) {
	t.Setenv("MY_AUTH_TOKEN", "env-token-value")

	cfg := &config.Config{
		Auth: config.AuthConfig{
			Headers:        map[string]string{},
			HeadersFromEnv: map[string]string{"authorization": "MY_AUTH_TOKEN"},
		},
	}

	got := cfg.ResolvedAuthHeaders()
	if got["authorization"] != "env-token-value" {
		t.Errorf("expected authorization=env-token-value, got %q", got["authorization"])
	}
}

func TestResolvedAuthHeaders_EnvVarEmptyIsSkipped(t *testing.T) {
	t.Setenv("EMPTY_TOKEN", "")

	cfg := &config.Config{
		Auth: config.AuthConfig{
			Headers:        map[string]string{"static": "val"},
			HeadersFromEnv: map[string]string{"dynamic": "EMPTY_TOKEN"},
		},
	}

	got := cfg.ResolvedAuthHeaders()
	if _, exists := got["dynamic"]; exists {
		t.Error("expected 'dynamic' key to be absent when env var is empty")
	}
	if got["static"] != "val" {
		t.Errorf("expected static=val, got %q", got["static"])
	}
}

func TestResolvedAuthHeaders_EnvOverridesStaticSameKey(t *testing.T) {
	t.Setenv("OVERRIDE_TOKEN", "from-env")

	cfg := &config.Config{
		Auth: config.AuthConfig{
			Headers:        map[string]string{"token": "static-value"},
			HeadersFromEnv: map[string]string{"token": "OVERRIDE_TOKEN"},
		},
	}

	got := cfg.ResolvedAuthHeaders()
	// The implementation copies static first, then env on top -- env wins.
	if got["token"] != "from-env" {
		t.Errorf("expected env to override static for same key, got %q", got["token"])
	}
}

// ---------------------------------------------------------------------------
// LoadConfigFromEnv - Auth headers via SIGNOZ_ACCESS_TOKEN
// ---------------------------------------------------------------------------

func TestLoadConfigFromEnv_SignozAccessToken(t *testing.T) {
	t.Setenv("SIGNOZ_ACCESS_TOKEN", "my-signoz-token")
	t.Setenv("OTEL_EXPORTER_OTLP_HEADERS", "")

	cfg := LoadConfigFromEnv()
	if cfg.Auth.Headers["signoz-access-token"] != "my-signoz-token" {
		t.Errorf("expected signoz-access-token header, got %v", cfg.Auth.Headers)
	}
}

func TestLoadConfigFromEnv_OTLPHeaders(t *testing.T) {
	t.Setenv("SIGNOZ_ACCESS_TOKEN", "")
	t.Setenv("OTEL_EXPORTER_OTLP_HEADERS", "x-api-key=abc123,x-org=my-org")

	cfg := LoadConfigFromEnv()
	if cfg.Auth.Headers["x-api-key"] != "abc123" {
		t.Errorf("expected x-api-key=abc123, got %v", cfg.Auth.Headers)
	}
	if cfg.Auth.Headers["x-org"] != "my-org" {
		t.Errorf("expected x-org=my-org, got %v", cfg.Auth.Headers)
	}
}

// ---------------------------------------------------------------------------
// LoadConfigFromEnv - Route exclusion defaults
// ---------------------------------------------------------------------------

func TestLoadConfigFromEnv_RouteExclusionDefaults(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXCLUDED_PATHS", "")
	t.Setenv("OTEL_TRACES_EXCLUDED_PREFIXES", "")
	t.Setenv("OTEL_TRACES_EXCLUDED_PATTERNS", "")

	cfg := LoadConfigFromEnv()

	defaultPaths := []string{"/health", "/healthz", "/health_check", "/metrics", "/ready", "/live"}
	if len(cfg.RouteExclusion.ExactPaths) != len(defaultPaths) {
		t.Errorf("expected %d default exact paths, got %d", len(defaultPaths), len(cfg.RouteExclusion.ExactPaths))
	}
	for i, expected := range defaultPaths {
		if i < len(cfg.RouteExclusion.ExactPaths) && cfg.RouteExclusion.ExactPaths[i] != expected {
			t.Errorf("expected ExactPaths[%d]=%q, got %q", i, expected, cfg.RouteExclusion.ExactPaths[i])
		}
	}

	defaultPatterns := []string{
		"/*/health", "/*/healthz", "/*/health_check",
		"/*/metrics", "/*/ready", "/*/live",
		"/*/*/health", "/*/*/healthz", "/*/*/health_check",
		"/*/*/metrics", "/*/*/ready", "/*/*/live",
	}
	if len(cfg.RouteExclusion.Patterns) != len(defaultPatterns) {
		t.Fatalf("expected %d default patterns, got %d: %v", len(defaultPatterns), len(cfg.RouteExclusion.Patterns), cfg.RouteExclusion.Patterns)
	}
	for i, expected := range defaultPatterns {
		if cfg.RouteExclusion.Patterns[i] != expected {
			t.Errorf("expected Patterns[%d]=%q, got %q", i, expected, cfg.RouteExclusion.Patterns[i])
		}
	}
}

// ---------------------------------------------------------------------------
// getInt64Env (additional coverage)
// ---------------------------------------------------------------------------

func TestGetInt64Env_Default(t *testing.T) {
	t.Setenv("I64_KEY", "")
	if got := getInt64Env("I64_KEY", 1024); got != 1024 {
		t.Errorf("expected 1024, got %d", got)
	}
}

func TestGetInt64Env_Valid(t *testing.T) {
	t.Setenv("I64_KEY", "999999999")
	if got := getInt64Env("I64_KEY", 0); got != 999999999 {
		t.Errorf("expected 999999999, got %d", got)
	}
}

func TestGetInt64Env_InvalidFallsBackToDefault(t *testing.T) {
	t.Setenv("I64_KEY", "xyz")
	if got := getInt64Env("I64_KEY", 512); got != 512 {
		t.Errorf("expected default 512, got %d", got)
	}
}

// ---------------------------------------------------------------------------
// LoadConfigFromEnv - Scrub config
// ---------------------------------------------------------------------------

func TestLoadConfigFromEnv_ScrubDefaults(t *testing.T) {
	t.Setenv("OTEL_PII_SCRUB_ENABLED", "")
	t.Setenv("OTEL_PII_SENSITIVE_KEYS", "")
	t.Setenv("OTEL_PII_REDACTED_VALUE", "")
	t.Setenv("OTEL_PII_DB_STATEMENT_MAX_LENGTH", "")

	cfg := LoadConfigFromEnv()
	if cfg.Scrub.Enabled {
		t.Error("expected Scrub.Enabled to default to false")
	}
	if cfg.Scrub.RedactedValue != "[REDACTED]" {
		t.Errorf("expected RedactedValue '[REDACTED]', got %q", cfg.Scrub.RedactedValue)
	}
	if cfg.Scrub.DBStatementMaxLength != 2048 {
		t.Errorf("expected DBStatementMaxLength 2048, got %d", cfg.Scrub.DBStatementMaxLength)
	}
	// Verify updated default sensitive keys
	expectedKeys := []string{"password", "token", "secret", "key", "email"}
	if len(cfg.Scrub.SensitiveKeys) != len(expectedKeys) {
		t.Errorf("expected %d sensitive keys, got %d: %v", len(expectedKeys), len(cfg.Scrub.SensitiveKeys), cfg.Scrub.SensitiveKeys)
	}
}

// ---------------------------------------------------------------------------
// LoadConfigFromEnv - HTTP config
// ---------------------------------------------------------------------------

func TestLoadConfigFromEnv_HTTPDefaults(t *testing.T) {
	// Clear all HTTP env vars
	httpEnvVars := []string{
		"OTEL_HTTP_CAPTURE_REQUEST_HEADERS",
		"OTEL_HTTP_CAPTURE_RESPONSE_HEADERS",
		"OTEL_HTTP_CAPTURE_QUERY_PARAMS",
		"OTEL_HTTP_CAPTURE_REQUEST_BODY",
		"OTEL_HTTP_CAPTURE_RESPONSE_BODY",
		"OTEL_HTTP_REQUEST_BODY_MAX_SIZE",
		"OTEL_HTTP_RESPONSE_BODY_MAX_SIZE",
		"OTEL_HTTP_BODY_ALLOWED_CONTENT_TYPES",
		"OTEL_HTTP_RECORD_EXCEPTION_EVENTS",
		"OTEL_HTTP_SENSITIVE_HEADERS",
	}
	for _, key := range httpEnvVars {
		t.Setenv(key, "")
	}

	cfg := LoadConfigFromEnv()

	if !cfg.HTTP.CaptureRequestHeaders {
		t.Error("expected CaptureRequestHeaders to default to true")
	}
	if !cfg.HTTP.CaptureResponseHeaders {
		t.Error("expected CaptureResponseHeaders to default to true")
	}
	if !cfg.HTTP.CaptureQueryParams {
		t.Error("expected CaptureQueryParams to default to true")
	}
	if cfg.HTTP.CaptureRequestBody {
		t.Error("expected CaptureRequestBody to default to false")
	}
	if cfg.HTTP.CaptureResponseBody {
		t.Error("expected CaptureResponseBody to default to false")
	}
	if cfg.HTTP.RequestBodyMaxSize != 8192 {
		t.Errorf("expected RequestBodyMaxSize 8192, got %d", cfg.HTTP.RequestBodyMaxSize)
	}
	if cfg.HTTP.ResponseBodyMaxSize != 8192 {
		t.Errorf("expected ResponseBodyMaxSize 8192, got %d", cfg.HTTP.ResponseBodyMaxSize)
	}
	if !cfg.HTTP.RecordExceptionEvents {
		t.Error("expected RecordExceptionEvents to default to true")
	}
	if len(cfg.HTTP.SensitiveHeaders) != 5 {
		t.Errorf("expected 5 default sensitive headers, got %d: %v", len(cfg.HTTP.SensitiveHeaders), cfg.HTTP.SensitiveHeaders)
	}
	if len(cfg.HTTP.BodyAllowedContentTypes) != 3 {
		t.Errorf("expected 3 default content types, got %d: %v", len(cfg.HTTP.BodyAllowedContentTypes), cfg.HTTP.BodyAllowedContentTypes)
	}
}

func TestLoadConfigFromEnv_HTTPCustomEnvVars(t *testing.T) {
	t.Setenv("OTEL_HTTP_CAPTURE_REQUEST_BODY", "true")
	t.Setenv("OTEL_HTTP_CAPTURE_RESPONSE_BODY", "true")
	t.Setenv("OTEL_HTTP_REQUEST_BODY_MAX_SIZE", "4096")
	t.Setenv("OTEL_HTTP_SENSITIVE_HEADERS", "authorization,x-custom-secret")

	cfg := LoadConfigFromEnv()

	if !cfg.HTTP.CaptureRequestBody {
		t.Error("expected CaptureRequestBody=true")
	}
	if !cfg.HTTP.CaptureResponseBody {
		t.Error("expected CaptureResponseBody=true")
	}
	if cfg.HTTP.RequestBodyMaxSize != 4096 {
		t.Errorf("expected RequestBodyMaxSize 4096, got %d", cfg.HTTP.RequestBodyMaxSize)
	}
	if len(cfg.HTTP.SensitiveHeaders) != 2 {
		t.Errorf("expected 2 sensitive headers, got %d: %v", len(cfg.HTTP.SensitiveHeaders), cfg.HTTP.SensitiveHeaders)
	}
}

// ---------------------------------------------------------------------------
// loadAuthConfig - SIGNOZ_ACCESS_TOKEN env var handling
// ---------------------------------------------------------------------------

func TestLoadAuthConfig_NoTokens(t *testing.T) {
	t.Setenv("SIGNOZ_ACCESS_TOKEN", "")
	t.Setenv("OTEL_EXPORTER_OTLP_HEADERS", "")

	ac := loadAuthConfig()
	// Should have empty but non-nil maps.
	if ac.Headers == nil {
		t.Fatal("expected non-nil Headers map")
	}
	if len(ac.Headers) != 0 {
		t.Errorf("expected empty headers, got %v", ac.Headers)
	}
}

// ---------------------------------------------------------------------------
// Ensure VERSION env var fallback works for Version field
// ---------------------------------------------------------------------------

func TestLoadConfigFromEnv_VersionFallbackToVERSION(t *testing.T) {
	t.Setenv("OTEL_SERVICE_VERSION", "")
	t.Setenv("VERSION", "2.0.0-beta")

	cfg := LoadConfigFromEnv()
	if cfg.Version != "2.0.0-beta" {
		t.Errorf("expected Version from VERSION env, got %q", cfg.Version)
	}
}

// ---------------------------------------------------------------------------
// Ensure resource config picks K8s env vars
// ---------------------------------------------------------------------------

func TestLoadConfigFromEnv_ResourceK8sEnvVars(t *testing.T) {
	t.Setenv("POD_NAME", "my-pod-abc")
	t.Setenv("POD_IP", "10.0.0.5")
	t.Setenv("POD_NAMESPACE", "default")
	t.Setenv("NODE_NAME", "node-1")

	cfg := LoadConfigFromEnv()
	if cfg.Resource.K8sPodName != "my-pod-abc" {
		t.Errorf("expected K8sPodName 'my-pod-abc', got %q", cfg.Resource.K8sPodName)
	}
	if cfg.Resource.K8sPodIP != "10.0.0.5" {
		t.Errorf("expected K8sPodIP '10.0.0.5', got %q", cfg.Resource.K8sPodIP)
	}
	if cfg.Resource.K8sNamespace != "default" {
		t.Errorf("expected K8sNamespace 'default', got %q", cfg.Resource.K8sNamespace)
	}
	if cfg.Resource.K8sNodeName != "node-1" {
		t.Errorf("expected K8sNodeName 'node-1', got %q", cfg.Resource.K8sNodeName)
	}
}

// ---------------------------------------------------------------------------
// Edge case: OTEL_RESOURCE_ATTRIBUTES parsed as custom attributes
// ---------------------------------------------------------------------------

func TestLoadConfigFromEnv_CustomResourceAttributes(t *testing.T) {
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "team=backend,tier=critical")
	// Prevent os.Getenv from leaking other state.
	_ = os.Setenv("ENV", "development")

	cfg := LoadConfigFromEnv()
	if cfg.Resource.CustomAttributes["team"] != "backend" {
		t.Errorf("expected team=backend, got %v", cfg.Resource.CustomAttributes)
	}
	if cfg.Resource.CustomAttributes["tier"] != "critical" {
		t.Errorf("expected tier=critical, got %v", cfg.Resource.CustomAttributes)
	}
}
