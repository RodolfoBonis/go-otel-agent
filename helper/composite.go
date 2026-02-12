package helper

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
)

// TraceAndMeasure combines tracing and metrics for a function.
func TraceAndMeasure(ctx context.Context, p TracerMeterProvider, name string, fn func(context.Context) error, opts *SpanOptions) error {
	start := time.Now()

	err := TraceFunction(ctx, p, name, fn, opts)
	duration := time.Since(start)

	component := "default"
	if opts != nil && opts.Component != "" {
		component = opts.Component
	}

	RecordDuration(ctx, p, fmt.Sprintf("%s_duration_seconds", name), duration, &MetricOptions{
		Component: component,
		Attributes: []attribute.KeyValue{
			attribute.Bool("success", err == nil),
		},
	})

	IncrementCounter(ctx, p, fmt.Sprintf("%s_operations_total", component), 1, &MetricOptions{
		Component: component,
		Attributes: []attribute.KeyValue{
			attribute.String("operation", name),
			attribute.Bool("success", err == nil),
		},
	})

	if err != nil {
		// Record error without error_message (cardinality fix)
		IncrementCounter(ctx, p, "errors_total", 1, &MetricOptions{
			Component: component,
			Attributes: []attribute.KeyValue{
				attribute.String("operation", name),
				attribute.String("error_type", fmt.Sprintf("%T", err)),
			},
		})
	}

	return err
}

// TraceAndMeasureWithResult combines tracing and metrics for a function with result.
func TraceAndMeasureWithResult[T any](ctx context.Context, p TracerMeterProvider, name string, fn func(context.Context) (T, error), opts *SpanOptions) (T, error) {
	start := time.Now()

	result, err := TraceFunctionWithResult(ctx, p, name, fn, opts)
	duration := time.Since(start)

	component := "default"
	if opts != nil && opts.Component != "" {
		component = opts.Component
	}

	RecordDuration(ctx, p, fmt.Sprintf("%s_duration_seconds", name), duration, &MetricOptions{
		Component: component,
		Attributes: []attribute.KeyValue{
			attribute.Bool("success", err == nil),
		},
	})

	IncrementCounter(ctx, p, fmt.Sprintf("%s_operations_total", component), 1, &MetricOptions{
		Component: component,
		Attributes: []attribute.KeyValue{
			attribute.String("operation", name),
			attribute.Bool("success", err == nil),
		},
	})

	if err != nil {
		IncrementCounter(ctx, p, "errors_total", 1, &MetricOptions{
			Component: component,
			Attributes: []attribute.KeyValue{
				attribute.String("operation", name),
				attribute.String("error_type", fmt.Sprintf("%T", err)),
			},
		})
	}

	return result, err
}
