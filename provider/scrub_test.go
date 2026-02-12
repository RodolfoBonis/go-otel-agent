package provider

import (
	"context"
	"testing"

	"github.com/RodolfoBonis/go-otel-agent/config"
)

func TestNewScrubProcessor_CreatesValidProcessor(t *testing.T) {
	cfg := config.ScrubConfig{
		Enabled:       true,
		SensitiveKeys: []string{"password", "secret"},
		SensitivePatterns: []string{
			"(?i)token",
			"(?i)api.?key",
		},
		RedactedValue: "[SCRUBBED]",
	}

	sp := NewScrubProcessor(cfg)

	if sp == nil {
		t.Fatal("NewScrubProcessor returned nil")
	}
	if len(sp.sensitiveKeys) != 2 {
		t.Errorf("sensitiveKeys count = %d, want 2", len(sp.sensitiveKeys))
	}
	if _, ok := sp.sensitiveKeys["password"]; !ok {
		t.Error("sensitiveKeys missing 'password'")
	}
	if _, ok := sp.sensitiveKeys["secret"]; !ok {
		t.Error("sensitiveKeys missing 'secret'")
	}
	if len(sp.compiledPatterns) != 2 {
		t.Errorf("compiledPatterns count = %d, want 2", len(sp.compiledPatterns))
	}
}

func TestIsSensitive_ExactKeyMatch(t *testing.T) {
	cfg := config.ScrubConfig{
		Enabled:       true,
		SensitiveKeys: []string{"password", "authorization"},
	}
	sp := NewScrubProcessor(cfg)

	tests := []struct {
		key  string
		want bool
	}{
		{"password", true},
		{"authorization", true},
		{"username", false},
		{"Password", false}, // exact match is case-sensitive
	}

	for _, tt := range tests {
		got := sp.isSensitive(tt.key)
		if got != tt.want {
			t.Errorf("isSensitive(%q) = %v, want %v", tt.key, got, tt.want)
		}
	}
}

func TestIsSensitive_PatternMatch(t *testing.T) {
	cfg := config.ScrubConfig{
		Enabled: true,
		SensitivePatterns: []string{
			"(?i)token",
			"(?i)api.?key",
		},
	}
	sp := NewScrubProcessor(cfg)

	tests := []struct {
		key  string
		want bool
	}{
		{"auth_token", true},
		{"bearer_token", true},
		{"api_key", true},
		{"apikey", true},
		{"username", false},
		{"endpoint", false},
	}

	for _, tt := range tests {
		got := sp.isSensitive(tt.key)
		if got != tt.want {
			t.Errorf("isSensitive(%q) = %v, want %v", tt.key, got, tt.want)
		}
	}
}

func TestScrubProcessor_DisabledDoesNothing(t *testing.T) {
	cfg := config.ScrubConfig{
		Enabled:       false,
		SensitiveKeys: []string{"password"},
	}
	sp := NewScrubProcessor(cfg)

	// When disabled, OnStart should return early without processing.
	// We verify by calling OnStart with a nil span -- if the processor
	// tried to access the span it would panic, proving the early return.
	sp.OnStart(context.Background(), nil)
}

func TestScrubProcessor_Shutdown_ReturnsNil(t *testing.T) {
	cfg := config.ScrubConfig{}
	sp := NewScrubProcessor(cfg)

	err := sp.Shutdown(context.Background())
	if err != nil {
		t.Errorf("Shutdown returned error: %v", err)
	}
}

func TestScrubProcessor_ForceFlush_ReturnsNil(t *testing.T) {
	cfg := config.ScrubConfig{}
	sp := NewScrubProcessor(cfg)

	err := sp.ForceFlush(context.Background())
	if err != nil {
		t.Errorf("ForceFlush returned error: %v", err)
	}
}

func TestNewScrubProcessor_InvalidPatternSkipped(t *testing.T) {
	cfg := config.ScrubConfig{
		Enabled: true,
		SensitivePatterns: []string{
			"[invalid",  // invalid regex, should be skipped
			"(?i)token", // valid
		},
	}
	sp := NewScrubProcessor(cfg)

	if len(sp.compiledPatterns) != 1 {
		t.Errorf("compiledPatterns count = %d, want 1 (invalid pattern should be skipped)", len(sp.compiledPatterns))
	}
}
