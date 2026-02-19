package gormplugin

import (
	"context"
	"fmt"

	otelagent "github.com/RodolfoBonis/go-otel-agent"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
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
	ctx, span := otel.GetTracerProvider().Tracer(t.name, t.opts...).Start(ctx, spanName, opts...)
	bridged := &dbSemconvBridgeSpan{Span: span}
	return trace.ContextWithSpan(ctx, bridged), bridged
}

// dbSemconvBridgeSpan intercepts SetAttributes to duplicate db.query.text
// (new semconv emitted by GORM plugin v0.1.16) as db.statement (legacy
// semconv that SigNoz uses for displaying SQL queries).
type dbSemconvBridgeSpan struct {
	trace.Span
}

func (s *dbSemconvBridgeSpan) SetAttributes(attrs ...attribute.KeyValue) {
	var extra []attribute.KeyValue
	for _, a := range attrs {
		if a.Key == "db.query.text" {
			extra = append(extra, attribute.String("db.statement", a.Value.AsString()))
		}
	}
	if len(extra) > 0 {
		attrs = append(attrs, extra...)
	}
	s.Span.SetAttributes(attrs...)
}

// InstrumentOption configures additional attributes for GORM DB spans.
type InstrumentOption func(*instrumentConfig)

type instrumentConfig struct {
	dbName string
	dbUser string
}

// WithDBName adds the db.namespace attribute to every DB span.
func WithDBName(name string) InstrumentOption {
	return func(cfg *instrumentConfig) {
		cfg.dbName = name
	}
}

// WithDBUser adds the db.user attribute to every DB span.
func WithDBUser(user string) InstrumentOption {
	return func(cfg *instrumentConfig) {
		cfg.dbUser = user
	}
}

// Instrument adds OpenTelemetry instrumentation to a GORM database instance.
// Uses a lazy TracerProvider so spans are linked to the real provider
// regardless of initialization order.
func Instrument(db *gorm.DB, agent *otelagent.Agent, opts ...InstrumentOption) error {
	if agent == nil || !agent.IsEnabled() || !agent.Config().Features.AutoDatabase {
		return nil
	}

	var icfg instrumentConfig
	for _, opt := range opts {
		opt(&icfg)
	}

	pluginOpts := []tracing.Option{
		tracing.WithTracerProvider(&lazyTracerProvider{}),
		tracing.WithRecordStackTrace(),
	}

	// Add db.namespace and db.user as static attributes on every span
	var staticAttrs []attribute.KeyValue
	if icfg.dbName != "" {
		staticAttrs = append(staticAttrs, attribute.String("db.namespace", icfg.dbName))
	}
	if icfg.dbUser != "" {
		staticAttrs = append(staticAttrs, attribute.String("db.user", icfg.dbUser))
	}
	if len(staticAttrs) > 0 {
		pluginOpts = append(pluginOpts, tracing.WithAttributes(staticAttrs...))
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
