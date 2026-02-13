package ginmiddleware

import (
	"net/http"

	otelagent "github.com/RodolfoBonis/go-otel-agent"
	"github.com/gin-gonic/gin"
)

// HealthHandler returns a Gin handler for the health check endpoint.
func HealthHandler(agent *otelagent.Agent) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := agent.HealthCheck()
		code := http.StatusOK
		if status.Status == "unhealthy" {
			code = http.StatusServiceUnavailable
		} else if status.Status == "degraded" {
			code = http.StatusOK // Still serving, but degraded
		}
		c.JSON(code, status)
	}
}

// ReadinessHandler returns a Gin handler for the readiness probe.
func ReadinessHandler(agent *otelagent.Agent) gin.HandlerFunc {
	return func(c *gin.Context) {
		if agent.ReadinessCheck() {
			c.JSON(http.StatusOK, gin.H{"ready": true})
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{"ready": false})
		}
	}
}

// DiagnosticsHandler returns a Gin handler that exposes runtime config
// for debugging telemetry issues (missing traces, wrong sampling, etc.).
func DiagnosticsHandler(agent *otelagent.Agent) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, agent.Diagnostics())
	}
}
