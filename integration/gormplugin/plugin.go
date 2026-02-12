package gormplugin

import (
	"fmt"

	otelagent "github.com/RodolfoBonis/go-otel-agent"
	"gorm.io/gorm"
	"gorm.io/plugin/opentelemetry/tracing"
)

// Instrument adds OpenTelemetry instrumentation to a GORM database instance.
func Instrument(db *gorm.DB, agent *otelagent.Agent) error {
	if agent == nil || !agent.IsEnabled() || !agent.Config().Features.AutoDatabase {
		return nil
	}

	if err := db.Use(tracing.NewPlugin()); err != nil {
		return fmt.Errorf("failed to add GORM OpenTelemetry plugin: %w", err)
	}

	return nil
}
