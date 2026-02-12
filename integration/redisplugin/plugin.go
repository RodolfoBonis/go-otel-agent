package redisplugin

import (
	otelagent "github.com/RodolfoBonis/go-otel-agent"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

// Instrument adds OpenTelemetry instrumentation to a Redis client.
func Instrument(client *redis.Client, agent *otelagent.Agent) error {
	if agent == nil || !agent.IsEnabled() || !agent.Config().Features.AutoRedis {
		return nil
	}

	if err := redisotel.InstrumentTracing(client); err != nil {
		return err
	}

	if err := redisotel.InstrumentMetrics(client); err != nil {
		return err
	}

	return nil
}
