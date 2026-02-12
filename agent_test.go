package otelagent

import (
	"context"
	"errors"
	"testing"
)

// newTestAgent creates an agent configured for fast test execution.
// It disables metrics and logs to avoid the 10-second shutdown timeout
// when no OTLP collector is running. Tests that need to verify full
// signal initialization should create agents directly.
func newTestAgent(name string) *Agent {
	return NewAgent(
		WithServiceName(name),
		WithInsecure(true),
		WithEndpoint("localhost:4317"),
		WithDisabledSignals(SignalMetrics, SignalLogs),
	)
}

func TestNewAgent_CreatesAgentWithDefaults(t *testing.T) {
	agent := NewAgent()
	if agent == nil {
		t.Fatal("NewAgent returned nil")
	}

	cfg := agent.Config()
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}

	// Default environment from LoadConfigFromEnv is "development" when ENV is unset.
	if cfg.Environment != "development" {
		t.Errorf("expected default environment %q, got %q", "development", cfg.Environment)
	}

	// Enabled defaults to true.
	if !cfg.Enabled {
		t.Error("expected agent to be enabled by default")
	}

	// Logger should be created automatically.
	if agent.Logger() == nil {
		t.Error("expected non-nil logger")
	}

	// RouteMatcher should be created.
	if agent.RouteMatcher() == nil {
		t.Error("expected non-nil route matcher")
	}

	// ExporterHealth should be created.
	if agent.ExporterHealth() == nil {
		t.Error("expected non-nil exporter health")
	}
}

func TestNewAgent_WithServiceName(t *testing.T) {
	agent := NewAgent(WithServiceName("my-service"))

	if agent.Config().ServiceName != "my-service" {
		t.Errorf("expected service name %q, got %q", "my-service", agent.Config().ServiceName)
	}
}

func TestNewAgent_WithEnabled(t *testing.T) {
	agent := NewAgent(WithEnabled(false))

	if agent.Config().Enabled {
		t.Error("expected agent to be disabled")
	}
}

func TestNewAgent_WithInsecure(t *testing.T) {
	agent := NewAgent(WithInsecure(true))

	if !agent.Config().Insecure {
		t.Error("expected insecure to be true")
	}
}

func TestNewAgent_WithEnvironment(t *testing.T) {
	agent := NewAgent(WithEnvironment("production"))

	if agent.Config().Environment != "production" {
		t.Errorf("expected environment %q, got %q", "production", agent.Config().Environment)
	}
}

func TestNewAgent_WithServiceVersion(t *testing.T) {
	agent := NewAgent(WithServiceVersion("1.2.3"))

	if agent.Config().Version != "1.2.3" {
		t.Errorf("expected version %q, got %q", "1.2.3", agent.Config().Version)
	}
}

func TestNewAgent_WithServiceNamespace(t *testing.T) {
	agent := NewAgent(WithServiceNamespace("platform"))

	if agent.Config().Namespace != "platform" {
		t.Errorf("expected namespace %q, got %q", "platform", agent.Config().Namespace)
	}
}

func TestNewAgent_WithEndpoint(t *testing.T) {
	agent := NewAgent(WithEndpoint("localhost:4317"))

	if agent.Config().Endpoint != "localhost:4317" {
		t.Errorf("expected endpoint %q, got %q", "localhost:4317", agent.Config().Endpoint)
	}
}

func TestNewAgent_WithSamplingRate(t *testing.T) {
	agent := NewAgent(WithSamplingRate(0.5))

	if agent.Config().Traces.Sampling.Rate != 0.5 {
		t.Errorf("expected sampling rate %v, got %v", 0.5, agent.Config().Traces.Sampling.Rate)
	}
}

func TestNewAgent_WithDisabledSignals(t *testing.T) {
	agent := NewAgent(WithDisabledSignals(SignalTraces, SignalLogs))

	if agent.Config().Traces.Enabled {
		t.Error("expected traces to be disabled")
	}
	if agent.Config().Logs.Enabled {
		t.Error("expected logs to be disabled")
	}
	if !agent.Config().Metrics.Enabled {
		t.Error("expected metrics to remain enabled")
	}
}

func TestNewAgent_WithDebugMode(t *testing.T) {
	agent := NewAgent(WithDebugMode(true))

	if !agent.Config().Features.DebugMode {
		t.Error("expected debug mode to be enabled")
	}
}

func TestNewAgent_WithMultipleOptions(t *testing.T) {
	agent := NewAgent(
		WithServiceName("combined-svc"),
		WithEnvironment("staging"),
		WithInsecure(true),
		WithEnabled(true),
	)

	cfg := agent.Config()
	if cfg.ServiceName != "combined-svc" {
		t.Errorf("expected service name %q, got %q", "combined-svc", cfg.ServiceName)
	}
	if cfg.Environment != "staging" {
		t.Errorf("expected environment %q, got %q", "staging", cfg.Environment)
	}
	if !cfg.Insecure {
		t.Error("expected insecure to be true")
	}
	if !cfg.Enabled {
		t.Error("expected enabled to be true")
	}
}

func TestInit_DisabledAgent_SucceedsImmediately(t *testing.T) {
	agent := NewAgent(WithEnabled(false))

	err := agent.Init(context.Background())
	if err != nil {
		t.Fatalf("expected no error when disabled, got: %v", err)
	}
	defer func() { _ = agent.Shutdown(context.Background()) }()

	// Agent should not be running since it was disabled.
	if agent.IsRunning() {
		t.Error("expected agent not to be running when disabled")
	}
}

func TestInit_FailsWithoutServiceName(t *testing.T) {
	// Clear any env var that might set the service name.
	t.Setenv("OTEL_SERVICE_NAME", "")

	agent := NewAgent(WithServiceName(""))

	err := agent.Init(context.Background())
	if err == nil {
		defer func() { _ = agent.Shutdown(context.Background()) }()
		t.Fatal("expected error when service name is missing")
	}

	if !errors.Is(err, ErrMissingServiceName) {
		t.Errorf("expected ErrMissingServiceName, got: %v", err)
	}
}

func TestInit_SucceedsWithValidConfig(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "test-service")

	agent := newTestAgent("test-service")

	err := agent.Init(context.Background())
	if err != nil {
		t.Fatalf("expected no error with valid config, got: %v", err)
	}
	defer func() { _ = agent.Shutdown(context.Background()) }()

	if !agent.IsRunning() {
		t.Error("expected agent to be running after Init")
	}
}

func TestInit_SucceedsWithAllSignals(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "full-signal-test")

	agent := NewAgent(
		WithServiceName("full-signal-test"),
		WithInsecure(true),
		WithEndpoint("localhost:4317"),
	)

	err := agent.Init(context.Background())
	if err != nil {
		t.Fatalf("expected no error with all signals enabled, got: %v", err)
	}
	defer func() { _ = agent.Shutdown(context.Background()) }()

	cfg := agent.Config()
	if !cfg.Traces.Enabled {
		t.Error("expected traces to be enabled")
	}
	if !cfg.Metrics.Enabled {
		t.Error("expected metrics to be enabled")
	}
	if !cfg.Logs.Enabled {
		t.Error("expected logs to be enabled")
	}

	if !agent.IsRunning() {
		t.Error("expected agent to be running after Init")
	}
}

func TestInit_DoubleInit_ReturnsErrAlreadyInitialized(t *testing.T) {
	agent := NewAgent(WithEnabled(false))

	err := agent.Init(context.Background())
	if err != nil {
		t.Fatalf("first Init failed: %v", err)
	}
	defer func() { _ = agent.Shutdown(context.Background()) }()

	err = agent.Init(context.Background())
	if err == nil {
		t.Fatal("expected error on double Init")
	}

	if !errors.Is(err, ErrAlreadyInitialized) {
		t.Errorf("expected ErrAlreadyInitialized, got: %v", err)
	}
}

func TestShutdown_BeforeInit_IsNoOp(t *testing.T) {
	agent := NewAgent()

	err := agent.Shutdown(context.Background())
	if err != nil {
		t.Fatalf("expected no error on shutdown before init, got: %v", err)
	}
}

func TestGetTracer_ReturnsNoopWhenNotInitialized(t *testing.T) {
	agent := NewAgent()

	tracer := agent.GetTracer("test")
	if tracer == nil {
		t.Fatal("GetTracer returned nil on uninitialized agent")
	}

	// Verify the tracer is usable (noop).
	ctx, span := tracer.Start(context.Background(), "test-span")
	if ctx == nil {
		t.Fatal("expected non-nil context from noop tracer span")
	}
	span.End()
}

func TestGetMeter_ReturnsNoopWhenNotInitialized(t *testing.T) {
	agent := NewAgent()

	meter := agent.GetMeter("test")
	if meter == nil {
		t.Fatal("GetMeter returned nil on uninitialized agent (this was the nil bug)")
	}

	// Verify the meter is usable (noop) -- the original nil bug caused panics here.
	counter, err := meter.Int64Counter("test_counter")
	if err != nil {
		t.Fatalf("expected no error creating counter on noop meter, got: %v", err)
	}
	if counter == nil {
		t.Fatal("expected non-nil counter from noop meter")
	}
	// Should not panic when recording.
	counter.Add(context.Background(), 1)
}

func TestGetMeter_ReturnsNoopWhenDisabled(t *testing.T) {
	agent := NewAgent(WithEnabled(false))

	err := agent.Init(context.Background())
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer func() { _ = agent.Shutdown(context.Background()) }()

	meter := agent.GetMeter("test")
	if meter == nil {
		t.Fatal("GetMeter returned nil on disabled agent")
	}

	histogram, err := meter.Float64Histogram("test_histogram")
	if err != nil {
		t.Fatalf("expected no error creating histogram on noop meter, got: %v", err)
	}
	if histogram == nil {
		t.Fatal("expected non-nil histogram from noop meter")
	}
}

func TestIsEnabled(t *testing.T) {
	enabledAgent := NewAgent(WithEnabled(true))
	if !enabledAgent.IsEnabled() {
		t.Error("expected IsEnabled to return true")
	}

	disabledAgent := NewAgent(WithEnabled(false))
	if disabledAgent.IsEnabled() {
		t.Error("expected IsEnabled to return false")
	}
}

func TestIsRunning_BeforeInit(t *testing.T) {
	agent := NewAgent()

	if agent.IsRunning() {
		t.Error("expected IsRunning to return false before Init")
	}
}

func TestIsRunning_AfterInit(t *testing.T) {
	agent := newTestAgent("test-svc")

	err := agent.Init(context.Background())
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer func() { _ = agent.Shutdown(context.Background()) }()

	if !agent.IsRunning() {
		t.Error("expected IsRunning to return true after Init")
	}
}

func TestIsRunning_AfterShutdown(t *testing.T) {
	agent := newTestAgent("test-svc")

	err := agent.Init(context.Background())
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	err = agent.Shutdown(context.Background())
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	if agent.IsRunning() {
		t.Error("expected IsRunning to return false after Shutdown")
	}
}

func TestConfig_Accessor(t *testing.T) {
	agent := NewAgent(WithServiceName("accessor-test"))

	cfg := agent.Config()
	if cfg == nil {
		t.Fatal("Config returned nil")
	}
	if cfg.ServiceName != "accessor-test" {
		t.Errorf("expected service name %q, got %q", "accessor-test", cfg.ServiceName)
	}
}

func TestTracerProvider_ReturnsNoopWhenNotInitialized(t *testing.T) {
	agent := NewAgent()

	tp := agent.TracerProvider()
	if tp == nil {
		t.Fatal("TracerProvider returned nil on uninitialized agent")
	}

	// Verify the provider is usable.
	tracer := tp.Tracer("test")
	if tracer == nil {
		t.Fatal("expected non-nil tracer from noop provider")
	}
}

func TestGetTracer_CachesTracers(t *testing.T) {
	agent := newTestAgent("cache-test")

	err := agent.Init(context.Background())
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer func() { _ = agent.Shutdown(context.Background()) }()

	tracer1 := agent.GetTracer("my-tracer")
	tracer2 := agent.GetTracer("my-tracer")

	// Same name should return the same cached tracer instance.
	if tracer1 != tracer2 {
		t.Error("expected GetTracer to return the same cached tracer for the same name")
	}
}

func TestGetMeter_CachesMeters(t *testing.T) {
	// Use an agent with metrics enabled to test real meter caching.
	agent := NewAgent(
		WithServiceName("cache-test"),
		WithInsecure(true),
		WithEndpoint("localhost:4317"),
		WithDisabledSignals(SignalTraces, SignalLogs),
	)

	err := agent.Init(context.Background())
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer func() { _ = agent.Shutdown(context.Background()) }()

	meter1 := agent.GetMeter("my-meter")
	meter2 := agent.GetMeter("my-meter")

	// Same name should return the same cached meter instance.
	if meter1 != meter2 {
		t.Error("expected GetMeter to return the same cached meter for the same name")
	}
}

func TestGetTracer_DifferentNames_ReturnDifferentTracers(t *testing.T) {
	agent := newTestAgent("diff-test")

	err := agent.Init(context.Background())
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer func() { _ = agent.Shutdown(context.Background()) }()

	tracer1 := agent.GetTracer("tracer-a")
	tracer2 := agent.GetTracer("tracer-b")

	if tracer1 == tracer2 {
		t.Error("expected different tracers for different names")
	}
}

func TestShutdown_AfterDisabledInit(t *testing.T) {
	agent := NewAgent(WithEnabled(false))

	err := agent.Init(context.Background())
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	err = agent.Shutdown(context.Background())
	if err != nil {
		t.Fatalf("expected no error on shutdown of disabled agent, got: %v", err)
	}
}
