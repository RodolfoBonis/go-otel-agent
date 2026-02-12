package provider

import (
	"testing"
)

func TestNewExporterHealth_CreatesValidTracker(t *testing.T) {
	h := NewExporterHealth()

	if h == nil {
		t.Fatal("NewExporterHealth returned nil")
	}
	if h.consecutiveFailures == nil {
		t.Error("consecutiveFailures map not initialized")
	}
	if h.lastFailure == nil {
		t.Error("lastFailure map not initialized")
	}
	if h.lastSuccess == nil {
		t.Error("lastSuccess map not initialized")
	}
	if h.degradedThreshold != 3 {
		t.Errorf("degradedThreshold = %d, want 3", h.degradedThreshold)
	}
	if h.unhealthyThreshold != 10 {
		t.Errorf("unhealthyThreshold = %d, want 10", h.unhealthyThreshold)
	}
}

func TestRecordSuccess_ResetsFailureCount(t *testing.T) {
	h := NewExporterHealth()

	h.RecordFailure("traces")
	h.RecordFailure("traces")
	h.RecordSuccess("traces")

	if h.consecutiveFailures["traces"] != 0 {
		t.Errorf("consecutiveFailures after RecordSuccess = %d, want 0", h.consecutiveFailures["traces"])
	}
}

func TestRecordFailure_IncrementsCount(t *testing.T) {
	h := NewExporterHealth()

	h.RecordFailure("metrics")
	if h.consecutiveFailures["metrics"] != 1 {
		t.Errorf("consecutiveFailures after 1 failure = %d, want 1", h.consecutiveFailures["metrics"])
	}

	h.RecordFailure("metrics")
	if h.consecutiveFailures["metrics"] != 2 {
		t.Errorf("consecutiveFailures after 2 failures = %d, want 2", h.consecutiveFailures["metrics"])
	}
}

func TestStatus_ReturnsHealthyWithZeroFailures(t *testing.T) {
	h := NewExporterHealth()

	// Record a success to register the signal, then check status
	h.RecordSuccess("traces")
	got := h.Status("traces")

	if got != ExporterHealthy {
		t.Errorf("Status = %v, want %v (ExporterHealthy)", got, ExporterHealthy)
	}
}

func TestStatus_ReturnsDegradedAfterThreshold(t *testing.T) {
	h := NewExporterHealth()

	// degradedThreshold is 3
	for i := 0; i < 3; i++ {
		h.RecordFailure("traces")
	}

	got := h.Status("traces")
	if got != ExporterDegraded {
		t.Errorf("Status after 3 failures = %v, want %v (ExporterDegraded)", got, ExporterDegraded)
	}
}

func TestStatus_ReturnsUnhealthyAfterThreshold(t *testing.T) {
	h := NewExporterHealth()

	// unhealthyThreshold is 10
	for i := 0; i < 10; i++ {
		h.RecordFailure("traces")
	}

	got := h.Status("traces")
	if got != ExporterUnhealthy {
		t.Errorf("Status after 10 failures = %v, want %v (ExporterUnhealthy)", got, ExporterUnhealthy)
	}
}

func TestOverallStatus_ReturnsWorstAcrossSignals(t *testing.T) {
	h := NewExporterHealth()

	// traces: healthy
	h.RecordSuccess("traces")

	// metrics: degraded (3 failures)
	for i := 0; i < 3; i++ {
		h.RecordFailure("metrics")
	}

	// logs: unhealthy (10 failures)
	for i := 0; i < 10; i++ {
		h.RecordFailure("logs")
	}

	got := h.OverallStatus()
	if got != ExporterUnhealthy {
		t.Errorf("OverallStatus = %v, want %v (ExporterUnhealthy)", got, ExporterUnhealthy)
	}
}

func TestRecordSuccess_AfterFailures_ResetsToHealthy(t *testing.T) {
	h := NewExporterHealth()

	// Push to unhealthy
	for i := 0; i < 10; i++ {
		h.RecordFailure("traces")
	}
	if h.Status("traces") != ExporterUnhealthy {
		t.Fatal("expected ExporterUnhealthy before recovery")
	}

	// Recover
	h.RecordSuccess("traces")

	got := h.Status("traces")
	if got != ExporterHealthy {
		t.Errorf("Status after recovery = %v, want %v (ExporterHealthy)", got, ExporterHealthy)
	}
}

func TestSignalStatuses_ReturnsAllTrackedSignals(t *testing.T) {
	h := NewExporterHealth()

	h.RecordSuccess("traces")
	h.RecordFailure("metrics")
	h.RecordFailure("logs")

	statuses := h.SignalStatuses()

	if len(statuses) != 3 {
		t.Errorf("SignalStatuses returned %d entries, want 3", len(statuses))
	}

	expected := map[string]ExporterStatus{
		"traces":  ExporterHealthy,
		"metrics": ExporterHealthy, // 1 failure is still healthy (threshold is 3)
		"logs":    ExporterHealthy, // 1 failure is still healthy
	}

	for signal, wantStatus := range expected {
		gotStatus, ok := statuses[signal]
		if !ok {
			t.Errorf("signal %q not found in SignalStatuses", signal)
			continue
		}
		if gotStatus != wantStatus {
			t.Errorf("SignalStatuses[%q] = %v, want %v", signal, gotStatus, wantStatus)
		}
	}
}

func TestExporterStatus_String(t *testing.T) {
	tests := []struct {
		status ExporterStatus
		want   string
	}{
		{ExporterHealthy, "healthy"},
		{ExporterDegraded, "degraded"},
		{ExporterUnhealthy, "unhealthy"},
		{ExporterStatus(99), "unknown"},
	}

	for _, tt := range tests {
		got := tt.status.String()
		if got != tt.want {
			t.Errorf("ExporterStatus(%d).String() = %q, want %q", tt.status, got, tt.want)
		}
	}
}
