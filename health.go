package otelagent

import (
	"fmt"

	"github.com/RodolfoBonis/go-otel-agent/provider"
)

// HealthStatus represents the overall health of the agent.
type HealthStatus struct {
	Status   string                              `json:"status"` // "ok", "degraded", "unhealthy"
	Signals  map[string]provider.ExporterStatus   `json:"signals,omitempty"`
	Running  bool                                 `json:"running"`
	Enabled  bool                                 `json:"enabled"`
}

// HealthCheck returns the current health status of the agent.
func (a *Agent) HealthCheck() HealthStatus {
	if !a.config.Enabled {
		return HealthStatus{
			Status:  "ok",
			Running: false,
			Enabled: false,
		}
	}

	overall := a.health.OverallStatus()
	var status string
	switch overall {
	case provider.ExporterHealthy:
		status = "ok"
	case provider.ExporterDegraded:
		status = "degraded"
	case provider.ExporterUnhealthy:
		status = "unhealthy"
	default:
		status = "unknown"
	}

	return HealthStatus{
		Status:  status,
		Signals: a.health.SignalStatuses(),
		Running: a.IsRunning(),
		Enabled: a.config.Enabled,
	}
}

// ReadinessCheck returns true when the agent is initialized and running.
func (a *Agent) ReadinessCheck() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.initialized && a.running
}

// DiagnosticsInfo surfaces runtime configuration for debugging telemetry issues.
type DiagnosticsInfo struct {
	Enabled      bool    `json:"enabled"`
	Running      bool    `json:"running"`
	Environment  string  `json:"environment"`
	ServiceName  string  `json:"service_name"`
	Namespace    string  `json:"namespace"`
	Version      string  `json:"version"`
	Endpoint     string  `json:"endpoint"`
	SamplingRate float64 `json:"sampling_rate"`
	TracerType   string  `json:"tracer_type"`
	LoggerType   string  `json:"logger_type"`
	Features     any     `json:"features"`
}

// Diagnostics returns runtime configuration details for debugging.
func (a *Agent) Diagnostics() DiagnosticsInfo {
	tracerType := "noop"
	if a.tracerProvider != nil {
		tracerType = fmt.Sprintf("%T", a.tracerProvider)
	}

	loggerType := "noop"
	if a.loggerProvider != nil {
		loggerType = fmt.Sprintf("%T", a.loggerProvider)
	}

	return DiagnosticsInfo{
		Enabled:      a.config.Enabled,
		Running:      a.IsRunning(),
		Environment:  a.config.Environment,
		ServiceName:  a.config.ServiceName,
		Namespace:    a.config.Namespace,
		Version:      a.config.Version,
		Endpoint:     a.config.Endpoint,
		SamplingRate: a.config.Traces.Sampling.Rate,
		TracerType:   tracerType,
		LoggerType:   loggerType,
		Features:     a.config.Features,
	}
}
