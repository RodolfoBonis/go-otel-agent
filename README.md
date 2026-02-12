# go-otel-agent

A production-ready, batteries-included OpenTelemetry observability library for Go applications. Provides distributed tracing, metrics, and structured logging with smart defaults — get full observability with just 3 environment variables.

Built for [SigNoz](https://signoz.io), but works with any OpenTelemetry-compatible backend (Jaeger, Grafana Tempo, Datadog, etc.).

## Features

- **Zero-config startup** — Only 3 env vars required (`OTEL_SERVICE_NAME`, `OTEL_SERVICE_NAMESPACE`, `OTEL_SERVICE_VERSION`)
- **Full OpenTelemetry stack** — Traces, metrics, and logs via OTLP (gRPC/HTTP)
- **Smart defaults** — 50+ configuration values baked in as production-ready defaults
- **Datadog-level HTTP enrichment** — Request/response headers, query params, body capture, exception events
- **Gin middleware** — Lazy-init middleware with route exclusion, HTTP enrichment, PII scrubbing, and metrics
- **GORM plugin** — Lazy tracer provider with full SQL query text, query variables, and stack traces on errors
- **Redis plugin** — Automatic Redis operation tracing
- **AMQP plugin** — RabbitMQ trace context propagation
- **Uber FX compatible** — Lazy initialization solves FX lifecycle ordering (works with `fx.Invoke` + `OnStart`)
- **PII scrubbing** — Automatic redaction of sensitive span attributes and HTTP data
- **HTTP PII scrubbing** — Sensitive headers always redacted, query params and body patterns scrubbed
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
│   ├── http_scrubber.go            # HTTP-specific PII scrubber (headers, query, body)
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
│   │   ├── middleware.go           # Lazy-init Gin middleware with HTTP enrichment
│   │   ├── health.go               # Health/readiness Gin handlers
│   │   └── body.go                 # Response body capture
│   ├── gormplugin/
│   │   └── plugin.go               # GORM with lazy TracerProvider + SQL truncation
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
| `OTEL_PII_SCRUB_ENABLED` | `false` | Enable PII scrubbing |
| `OTEL_PII_SENSITIVE_KEYS` | `password,token,secret,key,email` | Comma-separated attribute keys to redact |
| `OTEL_PII_SENSITIVE_PATTERNS` | `.*password.*,.*token.*,.*secret.*` | Regex patterns for key matching |
| `OTEL_PII_REDACTED_VALUE` | `[REDACTED]` | Replacement value |
| `OTEL_PII_DB_STATEMENT_MAX_LENGTH` | `2048` | Truncate db.statement (0=full, -1=redact) |

#### HTTP Capture

Control what HTTP data is captured in spans. Headers are captured by default; body capture is opt-in.

| Variable | Default | Description |
|----------|---------|-------------|
| `OTEL_HTTP_CAPTURE_REQUEST_HEADERS` | `true` | Capture request headers as span attributes |
| `OTEL_HTTP_CAPTURE_RESPONSE_HEADERS` | `true` | Capture response headers as span attributes |
| `OTEL_HTTP_CAPTURE_QUERY_PARAMS` | `true` | Capture URL query string |
| `OTEL_HTTP_CAPTURE_REQUEST_BODY` | `false` | Capture request body (opt-in, expensive) |
| `OTEL_HTTP_CAPTURE_RESPONSE_BODY` | `false` | Capture response body (opt-in, expensive) |
| `OTEL_HTTP_REQUEST_BODY_MAX_SIZE` | `8192` | Max request body bytes to capture |
| `OTEL_HTTP_RESPONSE_BODY_MAX_SIZE` | `8192` | Max response body bytes to capture |
| `OTEL_HTTP_BODY_ALLOWED_CONTENT_TYPES` | `application/json,application/xml,text/plain` | Content types eligible for body capture |
| `OTEL_HTTP_RECORD_EXCEPTION_EVENTS` | `true` | Add exception events for 4xx/5xx responses |
| `OTEL_HTTP_SENSITIVE_HEADERS` | `authorization,cookie,set-cookie,x-api-key,x-auth-token` | Headers always redacted (regardless of scrub config) |

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

The middleware uses **lazy initialization** (`sync.Once`) to resolve the real TracerProvider and MeterProvider on the first request. This solves the FX lifecycle ordering issue where `ginmiddleware.New()` runs during `fx.Invoke` but `agent.Init()` hasn't completed yet.

**Span attributes captured automatically:**

| Attribute | Source |
|---|---|
| `http.request.method`, `http.route`, `url.path`, `http.response.status_code` | otelgin (OpenTelemetry semconv) |
| `http.client_ip` | `c.ClientIP()` |
| `http.request.id` | Gin context `requestID` |
| `user.id`, `user.role` | Gin context (if set by auth middleware) |
| `http.request.header.<name>` | Request headers (sensitive ones redacted) |
| `http.response.header.<name>` | Response headers (sensitive ones redacted) |
| `url.query` | Query string (sensitive params redacted) |
| `http.request.body` | Request body (opt-in via `OTEL_HTTP_CAPTURE_REQUEST_BODY`) |
| `http.response.body` | Response body (opt-in via `OTEL_HTTP_CAPTURE_RESPONSE_BODY`) |
| `http.request.body.size` | Request content length |
| `http.response.body.size` | Response body length |

**Error handling:**
- 5xx responses set span status to `Error`
- 4xx/5xx responses record an `exception` event with type and message (configurable via `OTEL_HTTP_RECORD_EXCEPTION_EVENTS`)

**PII protection:**
- Headers in `OTEL_HTTP_SENSITIVE_HEADERS` are **always** redacted (default: `authorization`, `cookie`, `set-cookie`, `x-api-key`, `x-auth-token`)
- Query param values matching `OTEL_PII_SENSITIVE_PATTERNS` are redacted when scrubbing is enabled
- Body content matching sensitive patterns is redacted when scrubbing is enabled

**Metrics recorded:**
- `http.server.request.duration` (histogram, seconds)
- `http.server.request.total` (counter)
- `http.server.errors.total` (counter, 4xx/5xx)

### Integration: GORM Database

```go
import "github.com/RodolfoBonis/go-otel-agent/integration/gormplugin"

db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})
gormplugin.Instrument(db, agent)

// All database queries are now automatically traced with:
// - Full SQL query text with query variables
// - Stack traces on database errors
// - SQL truncation based on OTEL_PII_DB_STATEMENT_MAX_LENGTH (default: 2048)
```

The GORM plugin uses a **lazy TracerProvider** that resolves the real global TracerProvider on every query. This solves the FX lifecycle issue where `gormplugin.Instrument()` is called during `fx.Invoke` before `agent.Init()` sets the global provider.

DB spans appear as children of HTTP spans, creating a complete trace: `HTTP GET /api/v1/plans` -> `SELECT plans`.

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

Two layers of PII protection work together:

### Span Attribute Scrubbing

Automatically redact sensitive span attributes before export:

```bash
OTEL_PII_SCRUB_ENABLED=true
OTEL_PII_SENSITIVE_KEYS=password,token,secret,key,email
OTEL_PII_SENSITIVE_PATTERNS=.*password.*,.*token.*,.*secret.*
OTEL_PII_REDACTED_VALUE=[REDACTED]
OTEL_PII_DB_STATEMENT_MAX_LENGTH=2048
```

- Matches attribute keys by exact name or regex pattern
- Replaces values with `[REDACTED]` (configurable)
- Truncates `db.statement` to configurable max length (default: 2048 chars)
- Runs as a SpanProcessor (before export)

### HTTP Data Scrubbing

HTTP-specific scrubbing that protects sensitive data in request/response captures:

- **Sensitive headers** (e.g., `Authorization`, `Cookie`) are **always** redacted, regardless of whether PII scrubbing is enabled
- **Query param values** matching sensitive patterns are redacted when `OTEL_PII_SCRUB_ENABLED=true`
- **Body content** matching sensitive patterns is redacted when `OTEL_PII_SCRUB_ENABLED=true`
- **Body content** is truncated to `OTEL_HTTP_REQUEST_BODY_MAX_SIZE` / `OTEL_HTTP_RESPONSE_BODY_MAX_SIZE`
- Only `OTEL_HTTP_BODY_ALLOWED_CONTENT_TYPES` are eligible for body capture (binary data is never captured)

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
| FX lifecycle: Gin middleware captures noop TracerProvider | Lazy init via `sync.Once` — resolves real provider on first request |
| FX lifecycle: GORM plugin captures noop tracer eagerly | Lazy `TracerProvider`/`Tracer` wrappers resolve global provider per query |
| DB spans orphaned (no HTTP parent) | Both fixes together ensure correct parent-child span linking |
| DB spans missing query text | GORM plugin includes query variables by default + SQL truncation |
| No HTTP request/response details in spans | Full header, query param, and body capture with PII scrubbing |
| No error events for HTTP 4xx/5xx | Exception events recorded with status code and error message |

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
