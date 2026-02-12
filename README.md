# go-otel-agent

A production-ready, batteries-included OpenTelemetry observability library for Go applications. Provides distributed tracing, metrics, and structured logging with smart defaults — get full observability with just 3 environment variables.

Built for [SigNoz](https://signoz.io), but works with any OpenTelemetry-compatible backend (Jaeger, Grafana Tempo, Datadog, etc.).

## Features

- **Zero-config startup** — Only 3 env vars required (`OTEL_SERVICE_NAME`, `OTEL_SERVICE_NAMESPACE`, `OTEL_SERVICE_VERSION`)
- **Full OpenTelemetry stack** — Traces, metrics, and logs via OTLP (gRPC/HTTP)
- **Smart defaults** — 35+ configuration values baked in as production-ready defaults
- **Gin middleware** — Single consolidated middleware with route exclusion, custom enrichment, and metrics
- **GORM plugin** — Automatic database query tracing
- **Redis plugin** — Automatic Redis operation tracing
- **AMQP plugin** — RabbitMQ trace context propagation
- **Uber FX integration** — Full DI lifecycle management
- **PII scrubbing** — Automatic redaction of sensitive span attributes
- **Health probes** — Built-in health and readiness endpoints
- **Route exclusion** — Three-layer matcher (exact, prefix, glob) for excluding paths from instrumentation
- **Noop safety** — All providers return noop implementations when disabled (never nil, never panics)
- **Metric cardinality control** — No `user_id` or `error_message` in metric attributes
- **Instrument caching** — Metric instruments cached via `sync.Map` (no recreation per call)
- **ParentBased sampling** — Ratio sampler always wrapped in `ParentBased` for correct distributed tracing
- **SigNoz Cloud support** — Auth headers and TLS configuration for secured collectors
- **Graceful degradation** — Exporter health tracking (healthy/degraded/unhealthy)
- **HTTP client instrumentation** — Automatic tracing for outgoing HTTP requests

## Installation

```bash
go get github.com/RodolfoBonis/go-otel-agent@latest
```

Requires Go 1.24+.

## Quick Start

### Minimal Setup (3 lines + 3 env vars)

```go
package main

import (
    "context"
    "log"

    otelagent "github.com/RodolfoBonis/go-otel-agent"
)

func main() {
    agent := otelagent.NewAgent(
        otelagent.WithServiceName("my-api"),
        otelagent.WithServiceNamespace("my-platform"),
        otelagent.WithServiceVersion("1.0.0"),
    )

    ctx := context.Background()
    if err := agent.Init(ctx); err != nil {
        log.Fatal(err)
    }
    defer agent.Shutdown(ctx)

    // Your application code here...
}
```

Or configure entirely via environment variables:

```bash
export OTEL_SERVICE_NAME=my-api
export OTEL_SERVICE_NAMESPACE=my-platform
export OTEL_SERVICE_VERSION=1.0.0
```

```go
agent := otelagent.NewAgent()
```

### With Gin HTTP Server

```go
package main

import (
    "context"
    "log"

    otelagent "github.com/RodolfoBonis/go-otel-agent"
    "github.com/RodolfoBonis/go-otel-agent/integration/ginmiddleware"
    "github.com/gin-gonic/gin"
)

func main() {
    agent := otelagent.NewAgent(
        otelagent.WithServiceName("my-api"),
        otelagent.WithServiceNamespace("my-platform"),
        otelagent.WithServiceVersion("1.0.0"),
    )

    ctx := context.Background()
    if err := agent.Init(ctx); err != nil {
        log.Fatal(err)
    }
    defer agent.Shutdown(ctx)

    r := gin.Default()

    // Single middleware for traces + metrics + route exclusion
    r.Use(ginmiddleware.New(agent, "my-api"))

    // Health endpoints (excluded from tracing by default)
    r.GET("/health", ginmiddleware.HealthHandler(agent))
    r.GET("/ready", ginmiddleware.ReadinessHandler(agent))

    r.GET("/api/v1/users", func(c *gin.Context) {
        // Automatically traced and measured
        c.JSON(200, gin.H{"users": []string{"alice", "bob"}})
    })

    r.Run(":8080")
}
```

### With Uber FX

```go
package main

import (
    otelagent "github.com/RodolfoBonis/go-otel-agent"
    "github.com/RodolfoBonis/go-otel-agent/fxmodule"
    "go.uber.org/fx"
)

func main() {
    app := fx.New(
        fxmodule.ProvideWithConfiguration(
            otelagent.WithServiceName("my-api"),
            otelagent.WithServiceVersion("1.0.0"),
        ),
        // ... your modules
    )
    app.Run()
}
```

## Package Structure

```
go-otel-agent/
├── agent.go                        # Agent lifecycle: NewAgent, Init, Shutdown, ForceFlush
├── config.go                       # Configuration with smart defaults + env var loading
├── options.go                      # Functional options: WithServiceName, WithEndpoint, etc.
├── errors.go                       # Sentinel errors
├── health.go                       # HealthCheck, ReadinessCheck
├── noop.go                         # Noop tracer/meter (never nil)
├── config/
│   └── types.go                    # All configuration struct definitions
├── logger/
│   ├── logger.go                   # Zap-based logger with auto trace correlation
│   └── noop.go                     # NoopLogger for testing
├── provider/
│   ├── resource.go                 # OTel Resource builder
│   ├── trace.go                    # TracerProvider with ParentBased sampling
│   ├── metric.go                   # MeterProvider with OTLP exporter
│   ├── log.go                      # LoggerProvider with OTLP exporter
│   ├── scrub.go                    # PII scrubbing SpanProcessor
│   └── exporter_health.go          # Exporter health tracking
├── helper/
│   ├── span.go                     # StartSpan, TraceFunction, TraceFunctionWithResult
│   ├── metric.go                   # RecordDuration, IncrementCounter, SetGauge (cached)
│   ├── baggage.go                  # SetBaggage, GetBaggage
│   ├── composite.go                # TraceAndMeasure (combined trace+metric)
│   ├── context.go                  # GetTraceID, GetSpanID, IsTracing
│   └── global.go                   # Trace, Measure, Count, Event, Error (global)
├── collector/
│   ├── collector.go                # MetricCollector orchestrator
│   ├── runtime.go                  # Go runtime metrics (memory, GC, goroutines)
│   ├── system.go                   # System metrics (connections, queues)
│   ├── performance.go              # Performance metrics (latency percentiles)
│   └── business.go                 # Business metrics (custom counters/gauges)
├── instrumentor/
│   ├── instrumentor.go             # Function tracing via reflection
│   ├── propagation.go              # W3C trace context propagation
│   └── httpclient.go               # HTTP client instrumentation
├── internal/
│   └── matcher/
│       └── route.go                # Three-layer route exclusion matcher
├── integration/
│   ├── ginmiddleware/
│   │   ├── middleware.go           # otelgin-based Gin middleware
│   │   ├── health.go               # Health/readiness Gin handlers
│   │   └── body.go                 # Response body capture
│   ├── gormplugin/
│   │   └── plugin.go               # GORM auto-instrumentation
│   ├── redisplugin/
│   │   └── plugin.go               # Redis auto-instrumentation
│   └── amqpplugin/
│       └── plugin.go               # AMQP trace context propagation
└── fxmodule/
    └── module.go                   # Uber FX module with lifecycle hooks
```

## Configuration

### Environment Variables

The library loads configuration from environment variables with smart defaults. Only 3 are required:

#### Required

| Variable | Description | Example |
|----------|-------------|---------|
| `OTEL_SERVICE_NAME` | Unique service identity | `my-api` |
| `OTEL_SERVICE_NAMESPACE` | Logical grouping | `my-platform` |
| `OTEL_SERVICE_VERSION` | Release version | `1.0.0` |

#### Infrastructure (with defaults)

| Variable | Default | Description |
|----------|---------|-------------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `signoz-otel-collector.signoz.svc.cluster.local:4317` | Collector endpoint |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | `grpc` | Transport protocol (`grpc`, `http`) |
| `OTEL_EXPORTER_OTLP_INSECURE` | `true` | Disable TLS (default for in-cluster) |
| `OTEL_EXPORTER_OTLP_COMPRESSION` | `gzip` | Compression algorithm |
| `OTEL_TRACES_SAMPLER_ARG` | `0.1` (prod) / `1.0` (dev) | Sampling rate (0.0-1.0) |
| `ENV` | `development` | Deployment environment |

#### Signals (all enabled by default)

| Variable | Default | Description |
|----------|---------|-------------|
| `OTEL_TRACES_ENABLED` | `true` | Enable distributed tracing |
| `OTEL_METRICS_ENABLED` | `true` | Enable metrics collection |
| `OTEL_LOGS_ENABLED` | `true` | Enable log export |

#### Route Exclusion

| Variable | Default | Description |
|----------|---------|-------------|
| `OTEL_TRACES_EXCLUDED_PATHS` | `/health,/healthz,/metrics,/ready,/live` | Exact path exclusions |
| `OTEL_TRACES_EXCLUDED_PREFIXES` | (none) | Prefix exclusions (e.g., `/debug/,/internal/`) |
| `OTEL_TRACES_EXCLUDED_PATTERNS` | (none) | Glob patterns (e.g., `/api/v*/health`) |

#### PII Scrubbing

| Variable | Default | Description |
|----------|---------|-------------|
| `OTEL_SCRUB_ENABLED` | `false` | Enable PII scrubbing |
| `OTEL_SCRUB_SENSITIVE_KEYS` | (none) | Comma-separated attribute keys to redact |
| `OTEL_SCRUB_SENSITIVE_PATTERNS` | (none) | Regex patterns for key matching |
| `OTEL_SCRUB_REDACTED_VALUE` | `[REDACTED]` | Replacement value |
| `OTEL_SCRUB_DB_STATEMENT_MAX_LENGTH` | `0` | Truncate db.statement (0=full, -1=redact) |

#### SigNoz Cloud Authentication

| Variable | Default | Description |
|----------|---------|-------------|
| `SIGNOZ_ACCESS_TOKEN` | (none) | SigNoz Cloud ingestion key |
| `OTEL_EXPORTER_OTLP_HEADERS` | (none) | Custom headers (key=value pairs) |

### Functional Options

Override any default via code:

```go
agent := otelagent.NewAgent(
    otelagent.WithServiceName("my-api"),
    otelagent.WithServiceNamespace("my-platform"),
    otelagent.WithServiceVersion("1.0.0"),
    otelagent.WithEndpoint("custom-collector:4317"),
    otelagent.WithSamplingRate(0.5),
    otelagent.WithInsecure(true),
    otelagent.WithEnvironment("production"),
    otelagent.WithEnabled(true),
    otelagent.WithDebugMode(false),
    otelagent.WithDisabledSignals(otelagent.SignalLogs),
    otelagent.WithAutoInstrumentation(true, true, true, true),
    otelagent.WithRouteExclusions(otelagent.RouteExclusionConfig{
        ExactPaths:  []string{"/health", "/metrics"},
        PrefixPaths: []string{"/debug/", "/internal/"},
        Patterns:    []string{"/api/v*/health"},
    }),
    otelagent.WithAuthHeaders(map[string]string{
        "signoz-access-token": "your-token",
    }),
    otelagent.WithLogger(customLogger),
    otelagent.WithConfig(customConfig),
)
```

## Usage Guide

### Tracing

#### Manual Spans

```go
import "github.com/RodolfoBonis/go-otel-agent/helper"

// Using global provider
ctx, span := helper.Trace(ctx, "my-operation", &helper.SpanOptions{
    Component: "my-service",
    Attributes: []attribute.KeyValue{
        attribute.String("key", "value"),
    },
})
defer span.End()
```

#### Function Tracing

```go
// Trace a function (void)
err := helper.TraceFunction(ctx, agent, "process-order", func(ctx context.Context) error {
    // your logic here
    return nil
}, &helper.SpanOptions{Component: "orders"})

// Trace a function with return value (generic)
result, err := helper.TraceFunctionWithResult(ctx, agent, "get-user",
    func(ctx context.Context) (*User, error) {
        return fetchUser(ctx, userID)
    },
    &helper.SpanOptions{Component: "users"},
)
```

#### Span Events and Errors

```go
// Add event to current span
helper.AddSpanEvent(ctx, "cache-miss", attribute.String("key", cacheKey))

// Set attributes on current span
helper.SetSpanAttributes(ctx, attribute.Int("items.count", len(items)))

// Record error on current span
helper.RecordSpanError(ctx, err, attribute.String("operation", "db-query"))
```

#### Context Inspection

```go
import "github.com/RodolfoBonis/go-otel-agent/helper"

traceID := helper.GetTraceID(ctx)  // "abc123..."
spanID := helper.GetSpanID(ctx)    // "def456..."
isTracing := helper.IsTracing(ctx) // true/false
```

### Metrics

```go
import "github.com/RodolfoBonis/go-otel-agent/helper"

// Increment counter (global)
helper.Count(ctx, "orders.created", 1, &helper.MetricOptions{
    Component: "orders",
    Attributes: []attribute.KeyValue{
        attribute.String("type", "standard"),
    },
})

// Record duration (global)
helper.Measure(ctx, "db.query.duration", queryDuration, &helper.MetricOptions{
    Component: "database",
})

// Using provider directly
helper.RecordDuration(ctx, agent, "http.request.duration", duration, opts)
helper.IncrementCounter(ctx, agent, "requests.total", 1, opts)
helper.SetGauge(ctx, agent, "connections.active", 42, opts)
```

### Combined Tracing + Metrics

```go
import "github.com/RodolfoBonis/go-otel-agent/helper"

// Traces the function AND records duration + counter + error metrics
err := helper.TraceAndMeasure(ctx, agent, "process-payment",
    func(ctx context.Context) error {
        return processPayment(ctx, order)
    },
    &helper.SpanOptions{Component: "payments"},
)

// With return value
result, err := helper.TraceAndMeasureWithResult(ctx, agent, "fetch-inventory",
    func(ctx context.Context) (*Inventory, error) {
        return getInventory(ctx, sku)
    },
    &helper.SpanOptions{Component: "inventory"},
)
```

### Logging

```go
import "github.com/RodolfoBonis/go-otel-agent/logger"

log := logger.NewLogger("production") // or "development"

// All log calls automatically inject trace_id and span_id from context
log.Info(ctx, "Order created", logger.Fields{
    "order_id": orderID,
    "amount":   total,
})

log.Error(ctx, "Payment failed", logger.Fields{
    "order_id": orderID,
    "error":    err.Error(),
})

// Create a child logger with additional fields
orderLog := log.With(logger.Fields{"order_id": orderID})
orderLog.Info(ctx, "Processing order")
```

### Baggage

```go
import "github.com/RodolfoBonis/go-otel-agent/helper"

// Set baggage (propagated across service boundaries)
ctx = helper.SetBaggage(ctx, "tenant.id", "org-123")

// Get baggage
tenantID := helper.GetBaggage(ctx, "tenant.id") // "org-123"
```

### Integration: Gin Middleware

```go
import "github.com/RodolfoBonis/go-otel-agent/integration/ginmiddleware"

// Basic usage
r.Use(ginmiddleware.New(agent, "my-api"))

// With custom filter
r.Use(ginmiddleware.New(agent, "my-api",
    ginmiddleware.WithFilter(func(r *http.Request) bool {
        return r.URL.Path != "/custom-exclude"
    }),
))

// Health handlers
r.GET("/health", ginmiddleware.HealthHandler(agent))
r.GET("/ready", ginmiddleware.ReadinessHandler(agent))
```

The middleware:
- Creates spans with correct OpenTelemetry semantic conventions
- Records `http.server.request.duration`, `http.server.request.total`, `http.server.errors.total` metrics
- Respects route exclusion configuration
- Adds `X-Trace-Id` response header for debugging
- Enriches spans with `http.client_ip`, `http.request.id`, `user.id`, `user.role`

### Integration: GORM Database

```go
import "github.com/RodolfoBonis/go-otel-agent/integration/gormplugin"

db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})
gormplugin.Instrument(db, agent)

// All database queries are now automatically traced
```

### Integration: Redis

```go
import "github.com/RodolfoBonis/go-otel-agent/integration/redisplugin"

rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
redisplugin.Instrument(rdb)

// All Redis operations are now automatically traced
```

### Integration: AMQP (RabbitMQ)

```go
import "github.com/RodolfoBonis/go-otel-agent/integration/amqpplugin"

// Publishing with trace context
headers := amqpplugin.InjectTraceContext(ctx)
ch.Publish("exchange", "key", false, false, amqp.Publishing{
    Headers: headers,
    Body:    payload,
})

// Consuming with trace context extraction
ctx = amqpplugin.ExtractTraceContext(context.Background(), msg.Headers, agent)
```

### Integration: HTTP Client

```go
import "github.com/RodolfoBonis/go-otel-agent/instrumentor"

client := &http.Client{}
instrumentor.InstrumentHTTPClient(client) // Wraps transport with otelhttp

resp, err := client.Get("https://api.example.com/data")
// Outgoing request is automatically traced with span: "HTTP GET api.example.com"
```

### Health Probes

```go
// Programmatic health check
status := agent.HealthCheck()
// HealthStatus{Status: "ok", Signals: {...}, Running: true, Enabled: true}

// Readiness check
ready := agent.ReadinessCheck() // true when initialized and running

// Gin handlers
r.GET("/health", ginmiddleware.HealthHandler(agent))
r.GET("/ready", ginmiddleware.ReadinessHandler(agent))
```

### Uber FX Module

```go
import "github.com/RodolfoBonis/go-otel-agent/fxmodule"

// Full module (default config from env vars)
fx.New(fxmodule.Module)

// With custom options
fx.New(fxmodule.ProvideWithConfiguration(
    otelagent.WithServiceName("my-api"),
))

// Tracing only
fx.New(fxmodule.TracingOnlyModule())

// Metrics only
fx.New(fxmodule.MetricsOnlyModule())

// Logs only
fx.New(fxmodule.LogsOnlyModule())

// Testing (disabled agent)
fx.New(fxmodule.ProvideForTesting())
```

The FX module provides:
- `*otelagent.Agent` — the observability agent
- `*instrumentor.Instrumentor` — function/HTTP instrumentation
- `logger.Logger` — structured logger with trace correlation

## Route Exclusion

The three-layer matcher excludes paths from both tracing and metrics:

```go
// Via environment variables
// OTEL_TRACES_EXCLUDED_PATHS=/health,/healthz,/metrics,/ready
// OTEL_TRACES_EXCLUDED_PREFIXES=/debug/,/internal/
// OTEL_TRACES_EXCLUDED_PATTERNS=/api/v*/health

// Via code
otelagent.WithRouteExclusions(otelagent.RouteExclusionConfig{
    ExactPaths:  []string{"/health", "/metrics"},     // O(1) map lookup
    PrefixPaths: []string{"/debug/", "/internal/"},   // strings.HasPrefix
    Patterns:    []string{"/api/v*/health"},           // path.Match glob
})
```

## PII Scrubbing

Automatically redact sensitive attributes from spans before export:

```go
// Via environment variables
// OTEL_SCRUB_ENABLED=true
// OTEL_SCRUB_SENSITIVE_KEYS=user.email,user.phone,db.statement
// OTEL_SCRUB_SENSITIVE_PATTERNS=.*password.*,.*token.*,.*secret.*
// OTEL_SCRUB_REDACTED_VALUE=[REDACTED]
// OTEL_SCRUB_DB_STATEMENT_MAX_LENGTH=256
```

The scrubber:
- Matches attribute keys by exact name or regex pattern
- Replaces values with `[REDACTED]` (configurable)
- Optionally truncates `db.statement` to a max length
- Runs as a SpanProcessor (before export)

## Bugs Fixed from Original Implementation

This library was extracted from a production codebase and fixes these issues:

| Bug | Fix |
|-----|-----|
| `GetMeter()` returns nil when disabled — panics consumers | Returns noop meter, never nil |
| `getStringEnv()` default value logic broken | Redesigned: `getStringEnv(defaultValue, keys...)` |
| `ratio` sampler not wrapped in `ParentBased` | Always wrapped in `ParentBased` for correct distributed tracing |
| Span limits configured but never applied | Wired to `sdktrace.WithSpanLimits()` |
| Retry config configured but never wired | Wired to `WithRetry()` on all OTLP exporters |
| `error_message` as metric attribute — unbounded cardinality | Removed from metrics, kept on spans only |
| `user_id` as metric attribute — cardinality bomb | Removed from metrics, kept on spans only |
| Metric instruments recreated on every call | Cached via `sync.Map` |
| Duplicate HTTP instrumentation (instrumentor + middleware) | Consolidated into single `otelgin`-based middleware |

## Examples

See the [`examples/`](./examples/) directory for complete, runnable examples:

- **[basic](./examples/basic/)** — Minimal setup with tracing and metrics
- **[gin-api](./examples/gin-api/)** — Gin HTTP API with middleware, health probes, and custom metrics
- **[fx-app](./examples/fx-app/)** — Uber FX dependency injection with full lifecycle management

## Testing

```bash
go test ./...              # Run all tests
go test -race ./...        # Run with race detector
go test -v ./...           # Verbose output
go test -cover ./...       # With coverage
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feat/my-feature`)
3. Commit your changes (`git commit -m 'feat: add my feature'`)
4. Push to the branch (`git push origin feat/my-feature`)
5. Open a Pull Request

## License

MIT License — see [LICENSE](LICENSE) for details.
