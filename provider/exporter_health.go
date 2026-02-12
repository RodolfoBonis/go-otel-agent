package provider

import (
	"sync"
	"time"
)

// ExporterStatus represents the health status of an exporter.
type ExporterStatus int

const (
	ExporterHealthy ExporterStatus = iota
	ExporterDegraded
	ExporterUnhealthy
)

func (s ExporterStatus) String() string {
	switch s {
	case ExporterHealthy:
		return "healthy"
	case ExporterDegraded:
		return "degraded"
	case ExporterUnhealthy:
		return "unhealthy"
	default:
		return "unknown"
	}
}

// ExporterHealth tracks the health of OTLP exporters.
type ExporterHealth struct {
	mu                  sync.RWMutex
	consecutiveFailures map[string]int
	lastFailure         map[string]time.Time
	lastSuccess         map[string]time.Time
	degradedThreshold   int
	unhealthyThreshold  int
}

// NewExporterHealth creates a new exporter health tracker.
func NewExporterHealth() *ExporterHealth {
	return &ExporterHealth{
		consecutiveFailures: make(map[string]int),
		lastFailure:         make(map[string]time.Time),
		lastSuccess:         make(map[string]time.Time),
		degradedThreshold:   3,
		unhealthyThreshold:  10,
	}
}

// RecordSuccess records a successful export for the given signal.
func (h *ExporterHealth) RecordSuccess(signal string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.consecutiveFailures[signal] = 0
	h.lastSuccess[signal] = time.Now()
}

// RecordFailure records a failed export for the given signal.
func (h *ExporterHealth) RecordFailure(signal string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.consecutiveFailures[signal]++
	h.lastFailure[signal] = time.Now()
}

// Status returns the health status for the given signal.
func (h *ExporterHealth) Status(signal string) ExporterStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()

	failures := h.consecutiveFailures[signal]
	if failures >= h.unhealthyThreshold {
		return ExporterUnhealthy
	}
	if failures >= h.degradedThreshold {
		return ExporterDegraded
	}
	return ExporterHealthy
}

// OverallStatus returns the worst health status across all signals.
func (h *ExporterHealth) OverallStatus() ExporterStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()

	worst := ExporterHealthy
	for _, failures := range h.consecutiveFailures {
		var status ExporterStatus
		if failures >= h.unhealthyThreshold {
			status = ExporterUnhealthy
		} else if failures >= h.degradedThreshold {
			status = ExporterDegraded
		}
		if status > worst {
			worst = status
		}
	}
	return worst
}

// SignalStatuses returns the status of each signal.
func (h *ExporterHealth) SignalStatuses() map[string]ExporterStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()

	statuses := make(map[string]ExporterStatus)
	for signal := range h.consecutiveFailures {
		statuses[signal] = h.Status(signal)
	}
	return statuses
}
