package provider

import (
	"strings"
	"testing"

	"github.com/RodolfoBonis/go-otel-agent/config"
)

func TestCreateSampler_Always_ReturnsAlwaysSample(t *testing.T) {
	cfg := config.SamplingConfig{
		Type: "always",
		Rate: 1.0,
	}

	sampler := createSampler(cfg)
	desc := sampler.Description()

	if desc != "AlwaysOnSampler" {
		t.Errorf("sampler.Description() = %q, want %q", desc, "AlwaysOnSampler")
	}
}

func TestCreateSampler_AlwaysOn_ReturnsAlwaysSample(t *testing.T) {
	cfg := config.SamplingConfig{
		Type: "always_on",
		Rate: 1.0,
	}

	sampler := createSampler(cfg)
	desc := sampler.Description()

	if desc != "AlwaysOnSampler" {
		t.Errorf("sampler.Description() = %q, want %q", desc, "AlwaysOnSampler")
	}
}

func TestCreateSampler_Never_ReturnsNeverSample(t *testing.T) {
	cfg := config.SamplingConfig{
		Type: "never",
		Rate: 0.0,
	}

	sampler := createSampler(cfg)
	desc := sampler.Description()

	if desc != "AlwaysOffSampler" {
		t.Errorf("sampler.Description() = %q, want %q", desc, "AlwaysOffSampler")
	}
}

func TestCreateSampler_AlwaysOff_ReturnsNeverSample(t *testing.T) {
	cfg := config.SamplingConfig{
		Type: "always_off",
		Rate: 0.0,
	}

	sampler := createSampler(cfg)
	desc := sampler.Description()

	if desc != "AlwaysOffSampler" {
		t.Errorf("sampler.Description() = %q, want %q", desc, "AlwaysOffSampler")
	}
}

func TestCreateSampler_Ratio_WrapsInParentBased(t *testing.T) {
	cfg := config.SamplingConfig{
		Type: "ratio",
		Rate: 0.5,
	}

	sampler := createSampler(cfg)
	desc := sampler.Description()

	if !strings.HasPrefix(desc, "ParentBased") {
		t.Errorf("sampler.Description() = %q, want prefix %q", desc, "ParentBased")
	}
	if !strings.Contains(desc, "TraceIDRatioBased") {
		t.Errorf("sampler.Description() = %q, want to contain %q", desc, "TraceIDRatioBased")
	}
}

func TestCreateSampler_TraceIDRatio_WrapsInParentBased(t *testing.T) {
	cfg := config.SamplingConfig{
		Type: "traceidratio",
		Rate: 0.25,
	}

	sampler := createSampler(cfg)
	desc := sampler.Description()

	if !strings.HasPrefix(desc, "ParentBased") {
		t.Errorf("sampler.Description() = %q, want prefix %q", desc, "ParentBased")
	}
}

func TestCreateSampler_Default_WrapsInParentBased(t *testing.T) {
	cfg := config.SamplingConfig{
		Type: "",
		Rate: 0.75,
	}

	sampler := createSampler(cfg)
	desc := sampler.Description()

	if !strings.HasPrefix(desc, "ParentBased") {
		t.Errorf("sampler.Description() = %q, want prefix %q", desc, "ParentBased")
	}
	if !strings.Contains(desc, "TraceIDRatioBased") {
		t.Errorf("sampler.Description() = %q, want to contain %q", desc, "TraceIDRatioBased")
	}
}

func TestCreateSampler_UnknownType_WrapsInParentBased(t *testing.T) {
	cfg := config.SamplingConfig{
		Type: "unknown_type",
		Rate: 0.5,
	}

	sampler := createSampler(cfg)
	desc := sampler.Description()

	if !strings.HasPrefix(desc, "ParentBased") {
		t.Errorf("sampler.Description() = %q, want prefix %q", desc, "ParentBased")
	}
}
