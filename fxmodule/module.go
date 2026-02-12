package fxmodule

import (
	"context"

	otelagent "github.com/RodolfoBonis/go-otel-agent"
	"github.com/RodolfoBonis/go-otel-agent/helper"
	"github.com/RodolfoBonis/go-otel-agent/instrumentor"
	"github.com/RodolfoBonis/go-otel-agent/logger"
	"go.uber.org/fx"
)

// Module provides the full observability stack via FX dependency injection.
// It provides: *otelagent.Agent, *instrumentor.Instrumentor, logger.Logger
var Module = fx.Module("go-otel-agent",
	fx.Provide(
		provideAgent,
		provideInstrumentor,
		provideLogger,
	),
	fx.Invoke(registerLifecycle),
)

func provideLogger() logger.Logger {
	return logger.NewLogger("")
}

func provideAgent(lc fx.Lifecycle, log logger.Logger) (*otelagent.Agent, error) {
	agent := otelagent.NewAgent(otelagent.WithLogger(log))

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return agent.Init(ctx)
		},
		OnStop: func(ctx context.Context) error {
			return agent.Shutdown(ctx)
		},
	})

	return agent, nil
}

func provideInstrumentor(agent *otelagent.Agent) *instrumentor.Instrumentor {
	return agent.Instrumentor()
}

func registerLifecycle(lc fx.Lifecycle, agent *otelagent.Agent) {
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			helper.SetGlobalProvider(agent)
			return nil
		},
	})
}

// ProvideWithConfiguration creates a module with custom agent options.
func ProvideWithConfiguration(opts ...otelagent.Option) fx.Option {
	return fx.Options(
		fx.Provide(func() logger.Logger {
			return logger.NewLogger("")
		}),
		fx.Provide(func(lc fx.Lifecycle, log logger.Logger) (*otelagent.Agent, error) {
			allOpts := append([]otelagent.Option{otelagent.WithLogger(log)}, opts...)
			agent := otelagent.NewAgent(allOpts...)

			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					return agent.Init(ctx)
				},
				OnStop: func(ctx context.Context) error {
					return agent.Shutdown(ctx)
				},
			})

			return agent, nil
		}),
		fx.Provide(func(agent *otelagent.Agent) *instrumentor.Instrumentor {
			return agent.Instrumentor()
		}),
		fx.Invoke(func(lc fx.Lifecycle, agent *otelagent.Agent) {
			lc.Append(fx.Hook{
				OnStart: func(_ context.Context) error {
					helper.SetGlobalProvider(agent)
					return nil
				},
			})
		}),
	)
}

// ProvideForTesting provides a disabled agent for testing.
func ProvideForTesting() fx.Option {
	return ProvideWithConfiguration(
		otelagent.WithEnabled(false),
	)
}

// TracingOnlyModule provides only tracing (no metrics or logs).
func TracingOnlyModule(opts ...otelagent.Option) fx.Option {
	allOpts := append(opts,
		otelagent.WithDisabledSignals(otelagent.SignalMetrics, otelagent.SignalLogs),
	)
	return ProvideWithConfiguration(allOpts...)
}

// MetricsOnlyModule provides only metrics (no traces or logs).
func MetricsOnlyModule(opts ...otelagent.Option) fx.Option {
	allOpts := append(opts,
		otelagent.WithDisabledSignals(otelagent.SignalTraces, otelagent.SignalLogs),
	)
	return ProvideWithConfiguration(allOpts...)
}

// LogsOnlyModule provides only logs (no traces or metrics).
func LogsOnlyModule(opts ...otelagent.Option) fx.Option {
	allOpts := append(opts,
		otelagent.WithDisabledSignals(otelagent.SignalTraces, otelagent.SignalMetrics),
	)
	return ProvideWithConfiguration(allOpts...)
}
