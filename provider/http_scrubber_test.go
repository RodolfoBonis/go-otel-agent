package provider

import (
	"testing"

	"github.com/RodolfoBonis/go-otel-agent/config"
)

func defaultHTTPConfig() config.HTTPConfig {
	return config.HTTPConfig{
		CaptureRequestHeaders:  true,
		CaptureResponseHeaders: true,
		CaptureQueryParams:     true,
		CaptureRequestBody:     false,
		CaptureResponseBody:    false,
		RequestBodyMaxSize:     8192,
		ResponseBodyMaxSize:    8192,
		BodyAllowedContentTypes: []string{
			"application/json", "application/xml", "text/plain",
		},
		RecordExceptionEvents: true,
		SensitiveHeaders: []string{
			"authorization", "cookie", "set-cookie", "x-api-key", "x-auth-token",
		},
	}
}

func defaultScrubConfig() config.ScrubConfig {
	return config.ScrubConfig{
		Enabled:           true,
		SensitiveKeys:     []string{"password", "token", "secret"},
		SensitivePatterns: []string{".*password.*", ".*token.*", ".*secret.*"},
		RedactedValue:     "[REDACTED]",
	}
}

func TestNewHTTPScrubber_CreatesValidInstance(t *testing.T) {
	s := NewHTTPScrubber(defaultHTTPConfig(), defaultScrubConfig())
	if s == nil {
		t.Fatal("NewHTTPScrubber returned nil")
	}
	if len(s.sensitiveHeaderSet) != 5 {
		t.Errorf("sensitiveHeaderSet count = %d, want 5", len(s.sensitiveHeaderSet))
	}
	if len(s.allowedContentSet) != 3 {
		t.Errorf("allowedContentSet count = %d, want 3", len(s.allowedContentSet))
	}
}

// --- ScrubHeaders ---

func TestScrubHeaders_RedactsSensitiveHeaders(t *testing.T) {
	s := NewHTTPScrubber(defaultHTTPConfig(), defaultScrubConfig())

	headers := map[string][]string{
		"Authorization": {"Bearer secret-token"},
		"Content-Type":  {"application/json"},
		"Cookie":        {"session=abc123"},
		"X-Api-Key":     {"my-key"},
	}

	result := s.ScrubHeaders(headers, nil)

	if result["authorization"] != "[REDACTED]" {
		t.Errorf("expected Authorization redacted, got %q", result["authorization"])
	}
	if result["cookie"] != "[REDACTED]" {
		t.Errorf("expected Cookie redacted, got %q", result["cookie"])
	}
	if result["x-api-key"] != "[REDACTED]" {
		t.Errorf("expected X-Api-Key redacted, got %q", result["x-api-key"])
	}
	if result["content-type"] != "application/json" {
		t.Errorf("expected Content-Type preserved, got %q", result["content-type"])
	}
}

func TestScrubHeaders_CaseInsensitive(t *testing.T) {
	s := NewHTTPScrubber(defaultHTTPConfig(), defaultScrubConfig())

	headers := map[string][]string{
		"AUTHORIZATION": {"Bearer token"},
		"X-API-KEY":     {"key-value"},
	}

	result := s.ScrubHeaders(headers, nil)

	if result["authorization"] != "[REDACTED]" {
		t.Errorf("expected AUTHORIZATION redacted, got %q", result["authorization"])
	}
	if result["x-api-key"] != "[REDACTED]" {
		t.Errorf("expected X-API-KEY redacted, got %q", result["x-api-key"])
	}
}

func TestScrubHeaders_AllowList(t *testing.T) {
	s := NewHTTPScrubber(defaultHTTPConfig(), defaultScrubConfig())

	headers := map[string][]string{
		"Content-Type":   {"application/json"},
		"Accept":         {"text/html"},
		"X-Custom":       {"value"},
		"Authorization":  {"Bearer token"},
	}

	result := s.ScrubHeaders(headers, []string{"Content-Type", "Authorization"})

	if _, ok := result["content-type"]; !ok {
		t.Error("expected Content-Type in result (in allow list)")
	}
	if _, ok := result["authorization"]; !ok {
		t.Error("expected Authorization in result (in allow list, but redacted)")
	}
	if result["authorization"] != "[REDACTED]" {
		t.Errorf("expected Authorization redacted even in allow list, got %q", result["authorization"])
	}
	if _, ok := result["accept"]; ok {
		t.Error("expected Accept NOT in result (not in allow list)")
	}
	if _, ok := result["x-custom"]; ok {
		t.Error("expected X-Custom NOT in result (not in allow list)")
	}
}

func TestScrubHeaders_MultipleValues(t *testing.T) {
	s := NewHTTPScrubber(defaultHTTPConfig(), defaultScrubConfig())

	headers := map[string][]string{
		"Accept": {"text/html", "application/json"},
	}

	result := s.ScrubHeaders(headers, nil)
	if result["accept"] != "text/html, application/json" {
		t.Errorf("expected joined values, got %q", result["accept"])
	}
}

// --- ScrubQueryString ---

func TestScrubQueryString_RedactsSensitiveParams(t *testing.T) {
	s := NewHTTPScrubber(defaultHTTPConfig(), defaultScrubConfig())

	result := s.ScrubQueryString("name=john&password=secret123&page=1")

	if result != "name=john&password=[REDACTED]&page=1" {
		t.Errorf("unexpected result: %q", result)
	}
}

func TestScrubQueryString_EmptyString(t *testing.T) {
	s := NewHTTPScrubber(defaultHTTPConfig(), defaultScrubConfig())
	if got := s.ScrubQueryString(""); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestScrubQueryString_DisabledScrub(t *testing.T) {
	scrubCfg := defaultScrubConfig()
	scrubCfg.Enabled = false
	s := NewHTTPScrubber(defaultHTTPConfig(), scrubCfg)

	result := s.ScrubQueryString("password=secret123&page=1")
	if result != "password=secret123&page=1" {
		t.Errorf("expected unmodified query when scrub disabled, got %q", result)
	}
}

func TestScrubQueryString_NoSensitiveParams(t *testing.T) {
	s := NewHTTPScrubber(defaultHTTPConfig(), defaultScrubConfig())

	result := s.ScrubQueryString("page=1&limit=10")
	if result != "page=1&limit=10" {
		t.Errorf("expected unmodified query, got %q", result)
	}
}

// --- ScrubBody ---

func TestScrubBody_TruncatesLargeBody(t *testing.T) {
	s := NewHTTPScrubber(defaultHTTPConfig(), defaultScrubConfig())

	body := "a]body that is longer than the limit"
	result := s.ScrubBody(body, 10)

	if result != "a]body tha...[truncated]" {
		t.Errorf("unexpected truncated result: %q", result)
	}
}

func TestScrubBody_EmptyBody(t *testing.T) {
	s := NewHTTPScrubber(defaultHTTPConfig(), defaultScrubConfig())
	if got := s.ScrubBody("", 100); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestScrubBody_RedactsSensitivePatterns(t *testing.T) {
	s := NewHTTPScrubber(defaultHTTPConfig(), defaultScrubConfig())

	body := `{"user": "john", "password": "secret123", "token": "abc"}`
	result := s.ScrubBody(body, 8192)

	if result == body {
		t.Error("expected body to be modified by scrubbing")
	}
}

func TestScrubBody_DisabledScrub(t *testing.T) {
	scrubCfg := defaultScrubConfig()
	scrubCfg.Enabled = false
	s := NewHTTPScrubber(defaultHTTPConfig(), scrubCfg)

	body := `{"password": "secret123"}`
	result := s.ScrubBody(body, 8192)
	if result != body {
		t.Errorf("expected unmodified body when scrub disabled, got %q", result)
	}
}

// --- IsAllowedContentType ---

func TestIsAllowedContentType_JSONAllowed(t *testing.T) {
	s := NewHTTPScrubber(defaultHTTPConfig(), defaultScrubConfig())

	tests := []struct {
		ct   string
		want bool
	}{
		{"application/json", true},
		{"application/json; charset=utf-8", true},
		{"APPLICATION/JSON", true},
		{"application/xml", true},
		{"text/plain", true},
		{"image/png", false},
		{"multipart/form-data", false},
		{"application/octet-stream", false},
	}

	for _, tt := range tests {
		got := s.IsAllowedContentType(tt.ct)
		if got != tt.want {
			t.Errorf("IsAllowedContentType(%q) = %v, want %v", tt.ct, got, tt.want)
		}
	}
}

func TestIsAllowedContentType_EmptyAllowList(t *testing.T) {
	httpCfg := defaultHTTPConfig()
	httpCfg.BodyAllowedContentTypes = nil
	s := NewHTTPScrubber(httpCfg, defaultScrubConfig())

	// Empty allow list means all content types are allowed
	if !s.IsAllowedContentType("image/png") {
		t.Error("expected all content types allowed when list is empty")
	}
}

// --- Edge cases ---

func TestScrubHeaders_EmptyHeaders(t *testing.T) {
	s := NewHTTPScrubber(defaultHTTPConfig(), defaultScrubConfig())
	result := s.ScrubHeaders(map[string][]string{}, nil)
	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}

func TestScrubQueryString_MalformedPart(t *testing.T) {
	s := NewHTTPScrubber(defaultHTTPConfig(), defaultScrubConfig())

	// Part without = sign should be passed through
	result := s.ScrubQueryString("page=1&malformed&password=secret")
	if result != "page=1&malformed&password=[REDACTED]" {
		t.Errorf("unexpected result: %q", result)
	}
}
