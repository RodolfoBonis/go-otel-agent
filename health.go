package otelagent

import (
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
