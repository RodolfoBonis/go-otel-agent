package gormplugin

import (
	"context"
	"fmt"

	otelagent "github.com/RodolfoBonis/go-otel-agent"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
	"gorm.io/gorm"
	"gorm.io/plugin/opentelemetry/tracing"
)

// lazyTracerProvider defers tracer resolution to request time.
// This fixes the FX lifecycle ordering issue where gormplugin.Instrument()
// runs during fx.Invoke (step 2) but agent.Init() sets the global
// TracerProvider in OnStart (step 3). Without this, the GORM plugin
// captures a noop tracer eagerly and never picks up the real provider.
type lazyTracerProvider struct {
	embedded.TracerProvider
}

func (p *lazyTracerProvider) Tracer(name string, opts ...trace.TracerOption) trace.Tracer {
	return &lazyTracer{name: name, opts: opts}
}

// lazyTracer resolves the real global TracerProvider on every Start() call.
type lazyTracer struct {
	embedded.Tracer
	name string
	opts []trace.TracerOption
}

func (t *lazyTracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return otel.GetTracerProvider().Tracer(t.name, t.opts...).Start(ctx, spanName, opts...)
}

// Instrument adds OpenTelemetry instrumentation to a GORM database instance.
// Uses a lazy TracerProvider so spans are linked to the real provider
// regardless of initialization order.
func Instrument(db *gorm.DB, agent *otelagent.Agent) error {
	if agent == nil || !agent.IsEnabled() || !agent.Config().Features.AutoDatabase {
		return nil
	}

	pluginOpts := []tracing.Option{
		tracing.WithTracerProvider(&lazyTracerProvider{}),
		tracing.WithRecordStackTrace(),
	}

	// Apply SQL truncation when configured
	if agent.Config().Scrub.DBStatementMaxLength > 0 {
		maxLen := agent.Config().Scrub.DBStatementMaxLength
		pluginOpts = append(pluginOpts, tracing.WithQueryFormatter(func(query string) string {
			if len(query) > maxLen {
				return query[:maxLen] + "..."
			}
			return query
		}))
	}

	if err := db.Use(tracing.NewPlugin(pluginOpts...)); err != nil {
		return fmt.Errorf("failed to add GORM OpenTelemetry plugin: %w", err)
	}

	return nil
}
