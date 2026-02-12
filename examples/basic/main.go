package main

import (
	"context"
	"log"
	"time"

	otelagent "github.com/RodolfoBonis/go-otel-agent"
	"github.com/RodolfoBonis/go-otel-agent/helper"
	"go.opentelemetry.io/otel/attribute"
)

func main() {
	agent := otelagent.NewAgent(
		otelagent.WithServiceName("example-basic"),
		otelagent.WithServiceNamespace("examples"),
		otelagent.WithServiceVersion("1.0.0"),
		otelagent.WithInsecure(true),
	)

	ctx := context.Background()
	if err := agent.Init(ctx); err != nil {
		log.Fatal(err)
	}
	defer agent.Shutdown(ctx)

	// Create a span
	ctx, span := helper.Trace(ctx, "my-operation", &helper.SpanOptions{
		Component: "example",
		Attributes: []attribute.KeyValue{
			attribute.String("example.key", "value"),
		},
	})
	defer span.End()

	// Simulate work
	time.Sleep(100 * time.Millisecond)

	// Record a metric
	helper.Count(ctx, "example.operations", 1, &helper.MetricOptions{
		Component: "example",
	})

	// Trace a function
	err := helper.TraceFunction(ctx, agent, "process-data", func(ctx context.Context) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	}, &helper.SpanOptions{Component: "example"})
	if err != nil {
		log.Printf("Error: %v", err)
	}

	log.Println("Done! Check your SigNoz/collector for traces and metrics.")
}
