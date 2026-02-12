package ginmiddleware

import (
	"fmt"
	"net/http"
	"time"

	otelagent "github.com/RodolfoBonis/go-otel-agent"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// MiddlewareOption configures the Gin middleware.
type MiddlewareOption func(*middlewareConfig)

type middlewareConfig struct {
	customFilter func(*http.Request) bool
}

// WithFilter adds a custom filter function. Return false to skip instrumentation.
func WithFilter(fn func(*http.Request) bool) MiddlewareOption {
	return func(cfg *middlewareConfig) {
		cfg.customFilter = fn
	}
}

// New creates a consolidated otelgin-based middleware with custom enrichment.
// This replaces both InstrumentHTTPServer and TracingMiddleware from SpoolIQ,
// consolidating duplicate HTTP instrumentation into a single middleware.
func New(agent *otelagent.Agent, serviceName string, opts ...MiddlewareOption) gin.HandlerFunc {
	if agent == nil || !agent.IsEnabled() {
		return func(c *gin.Context) { c.Next() }
	}

	mCfg := &middlewareConfig{}
	for _, opt := range opts {
		opt(mCfg)
	}

	// Pre-create metric instruments (cached)
	meter := agent.GetMeter("github.com/RodolfoBonis/go-otel-agent/integration/ginmiddleware")
	httpDuration, _ := meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithDescription("HTTP server request duration"),
		metric.WithUnit("s"),
	)
	requestCounter, _ := meter.Int64Counter(
		"http.server.request.total",
		metric.WithDescription("Total HTTP server requests"),
	)
	errorCounter, _ := meter.Int64Counter(
		"http.server.errors.total",
		metric.WithDescription("Total HTTP server errors"),
	)

	// Combined filter: route matcher + custom filter
	filterFn := func(r *http.Request) bool {
		if agent.RouteMatcher().ShouldExclude(r.URL.Path) {
			return false
		}
		if mCfg.customFilter != nil {
			return mCfg.customFilter(r)
		}
		return true
	}

	// Use otelgin as the base middleware (correct semconv, maintained by OTel community)
	otelMiddleware := otelgin.Middleware(serviceName,
		otelgin.WithTracerProvider(agent.TracerProvider()),
		otelgin.WithFilter(filterFn),
		otelgin.WithSpanNameFormatter(func(c *gin.Context) string {
			return fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path)
		}),
	)

	return func(c *gin.Context) {
		// Check exclusion before any work
		if agent.RouteMatcher().ShouldExclude(c.Request.URL.Path) {
			c.Next()
			return
		}

		start := time.Now()

		// Run otelgin middleware (creates span with correct semconv attributes)
		otelMiddleware(c)

		duration := time.Since(start)

		// Enrich span with custom attributes (after otelgin has created it)
		span := trace.SpanFromContext(c.Request.Context())
		if span.SpanContext().IsValid() {
			span.SetAttributes(
				attribute.String("http.client_ip", c.ClientIP()),
				attribute.String("http.request.id", c.GetString("requestID")),
			)

			// Add user context if available (spans only, not metrics - cardinality fix)
			if userID, exists := c.Get("user_id"); exists {
				span.SetAttributes(attribute.String("user.id", fmt.Sprintf("%v", userID)))
			}
			if userRole, exists := c.Get("user_role"); exists {
				span.SetAttributes(attribute.String("user.role", fmt.Sprintf("%v", userRole)))
			}

			// Set trace ID response header for debugging
			c.Header("X-Trace-Id", span.SpanContext().TraceID().String())

			// Set span status based on HTTP status
			if c.Writer.Status() >= 500 {
				span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", c.Writer.Status()))
			}
		}

		// Record metrics (with bounded cardinality - no user_id, no error_message)
		route := c.FullPath()
		if route == "" {
			route = "unknown"
		}

		metricAttrs := []attribute.KeyValue{
			attribute.String("http.request.method", c.Request.Method),
			attribute.String("http.route", route),
			attribute.Int("http.response.status_code", c.Writer.Status()),
		}

		if httpDuration != nil {
			httpDuration.Record(c.Request.Context(), duration.Seconds(), metric.WithAttributes(metricAttrs...))
		}
		if requestCounter != nil {
			requestCounter.Add(c.Request.Context(), 1, metric.WithAttributes(metricAttrs...))
		}
		if c.Writer.Status() >= 400 && errorCounter != nil {
			errorCounter.Add(c.Request.Context(), 1, metric.WithAttributes(metricAttrs...))
		}
	}
}
