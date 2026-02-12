package amqpplugin

import (
	"context"

	otelagent "github.com/RodolfoBonis/go-otel-agent"
	"github.com/RodolfoBonis/go-otel-agent/instrumentor"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// amqpHeaderCarrier adapts AMQP message headers for OTel propagation.
type amqpHeaderCarrier amqp.Table

func (c amqpHeaderCarrier) Get(key string) string {
	if v, ok := c[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (c amqpHeaderCarrier) Set(key, value string) {
	c[key] = value
}

func (c amqpHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// InjectContext injects trace context into AMQP message headers.
func InjectContext(ctx context.Context, headers amqp.Table) amqp.Table {
	if headers == nil {
		headers = amqp.Table{}
	}
	instrumentor.InjectContext(ctx, amqpHeaderCarrier(headers))
	return headers
}

// ExtractContext extracts trace context from AMQP message headers.
func ExtractContext(ctx context.Context, headers amqp.Table) context.Context {
	if headers == nil {
		return ctx
	}
	return instrumentor.ExtractContext(ctx, amqpHeaderCarrier(headers))
}

// PublishWithTrace publishes an AMQP message with trace context propagation.
func PublishWithTrace(ctx context.Context, agent *otelagent.Agent, ch *amqp.Channel, exchange, routingKey string, msg amqp.Publishing) error {
	if agent == nil || !agent.IsEnabled() || !agent.Config().Features.AutoAMQP {
		return ch.PublishWithContext(ctx, exchange, routingKey, false, false, msg)
	}

	tracer := agent.GetTracer("github.com/RodolfoBonis/go-otel-agent/integration/amqpplugin")
	ctx, span := tracer.Start(ctx, "amqp.publish "+exchange,
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.system", "rabbitmq"),
			attribute.String("messaging.destination.name", exchange),
			attribute.String("messaging.rabbitmq.destination.routing_key", routingKey),
			attribute.String("messaging.operation.type", "publish"),
		),
	)
	defer span.End()

	// Inject trace context into message headers
	if msg.Headers == nil {
		msg.Headers = amqp.Table{}
	}
	instrumentor.InjectContext(ctx, amqpHeaderCarrier(msg.Headers))

	err := ch.PublishWithContext(ctx, exchange, routingKey, false, false, msg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

// StartConsumeSpan starts a span for consuming an AMQP message.
// Returns the enriched context and span. Caller must call span.End().
func StartConsumeSpan(ctx context.Context, agent *otelagent.Agent, delivery amqp.Delivery, queue string) (context.Context, trace.Span) {
	if agent == nil || !agent.IsEnabled() || !agent.Config().Features.AutoAMQP {
		return ctx, trace.SpanFromContext(ctx)
	}

	// Extract parent context from message headers
	ctx = ExtractContext(ctx, delivery.Headers)

	tracer := agent.GetTracer("github.com/RodolfoBonis/go-otel-agent/integration/amqpplugin")
	ctx, span := tracer.Start(ctx, "amqp.consume "+queue,
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.system", "rabbitmq"),
			attribute.String("messaging.destination.name", delivery.Exchange),
			attribute.String("messaging.rabbitmq.destination.routing_key", delivery.RoutingKey),
			attribute.String("messaging.operation.type", "receive"),
			attribute.String("messaging.consumer.group.name", queue),
		),
	)

	return ctx, span
}

// Ensure amqpHeaderCarrier implements propagation.TextMapCarrier.
var _ propagation.TextMapCarrier = amqpHeaderCarrier(nil)
