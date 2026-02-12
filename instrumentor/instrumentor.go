package instrumentor

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/helper"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Instrumentor provides automatic instrumentation capabilities.
type Instrumentor struct {
	provider helper.TracerMeterProvider
	enabled  bool
}

// New creates a new Instrumentor.
func New(provider helper.TracerMeterProvider) *Instrumentor {
	return &Instrumentor{
		provider: provider,
		enabled:  provider != nil && provider.IsEnabled(),
	}
}

// StartSpan starts a new span with automatic attribute enrichment.
func (i *Instrumentor) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if !i.enabled {
		return ctx, trace.SpanFromContext(ctx)
	}
	tracer := i.provider.GetTracer("github.com/RodolfoBonis/go-otel-agent/instrumentor")
	return tracer.Start(ctx, name, opts...)
}

// TraceFunction automatically instruments a function with tracing.
func (i *Instrumentor) TraceFunction(ctx context.Context, fn interface{}, args ...interface{}) ([]interface{}, error) {
	if !i.enabled {
		return callFunction(fn, args...)
	}

	fnValue := reflect.ValueOf(fn)
	fnName := runtime.FuncForPC(fnValue.Pointer()).Name()

	tracer := i.provider.GetTracer("github.com/RodolfoBonis/go-otel-agent/instrumentor")
	ctx, span := tracer.Start(ctx, fnName)
	defer span.End()

	span.SetAttributes(
		attribute.String("function.name", fnName),
		attribute.Int("function.args_count", len(args)),
	)

	start := time.Now()
	results, err := callFunction(fn, args...)
	duration := time.Since(start)

	span.SetAttributes(attribute.Float64("function.duration_ms", float64(duration.Nanoseconds())/1e6))

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return results, err
}

// IsEnabled returns whether instrumentation is active.
func (i *Instrumentor) IsEnabled() bool {
	return i.enabled
}

// GetTracer returns a tracer from the underlying provider.
func (i *Instrumentor) GetTracer(name string) trace.Tracer {
	return i.provider.GetTracer(name)
}

// GetMeter returns a meter from the underlying provider.
func (i *Instrumentor) GetMeter(name string) metric.Meter {
	return i.provider.GetMeter(name)
}

func callFunction(fn interface{}, args ...interface{}) ([]interface{}, error) {
	fnValue := reflect.ValueOf(fn)
	if fnValue.Kind() != reflect.Func {
		return nil, fmt.Errorf("provided value is not a function")
	}

	reflectArgs := make([]reflect.Value, len(args))
	for i, arg := range args {
		reflectArgs[i] = reflect.ValueOf(arg)
	}

	results := fnValue.Call(reflectArgs)

	interfaceResults := make([]interface{}, len(results))
	var err error

	for i, result := range results {
		interfaceResults[i] = result.Interface()

		if i == len(results)-1 && result.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			if !result.IsNil() {
				err = result.Interface().(error)
			}
		}
	}

	return interfaceResults, err
}
