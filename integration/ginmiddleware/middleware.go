package ginmiddleware

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	otelagent "github.com/RodolfoBonis/go-otel-agent"
	"github.com/RodolfoBonis/go-otel-agent/provider"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const scopeName = "github.com/RodolfoBonis/go-otel-agent/integration/ginmiddleware"

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

// New creates a Gin middleware that manages HTTP spans directly, with full
// enrichment support. Uses sync.Once for lazy initialization to ensure
// providers are real (not noop) inside FX lifecycle.
//
// Previous versions delegated to otelgin.Middleware, but otelgin's
// defer span.End() + context restoration made post-handler enrichment
// a silent no-op. This version owns the full span lifecycle:
//
//	tracer.Start → c.Next() → enrichSpan → defer span.End()
func New(agent *otelagent.Agent, serviceName string, opts ...MiddlewareOption) gin.HandlerFunc {
	if agent == nil || !agent.IsEnabled() {
		return func(c *gin.Context) { c.Next() }
	}

	mCfg := &middlewareConfig{}
	for _, opt := range opts {
		opt(mCfg)
	}

	var (
		initOnce       sync.Once
		tracer         trace.Tracer
		httpDuration   metric.Float64Histogram
		requestCounter metric.Int64Counter
		errorCounter   metric.Int64Counter
		scrubber       *provider.HTTPScrubber
	)

	lazyInit := func() {
		initOnce.Do(func() {
			scrubber = provider.NewHTTPScrubber(agent.Config().HTTP, agent.Config().Scrub)
			tracer = otel.GetTracerProvider().Tracer(scopeName)

			meter := agent.GetMeter(scopeName)
			httpDuration, _ = meter.Float64Histogram(
				"http.server.request.duration",
				metric.WithDescription("HTTP server request duration"),
				metric.WithUnit("s"),
			)
			requestCounter, _ = meter.Int64Counter(
				"http.server.request.total",
				metric.WithDescription("Total HTTP server requests"),
			)
			errorCounter, _ = meter.Int64Counter(
				"http.server.errors.total",
				metric.WithDescription("Total HTTP server errors"),
			)
		})
	}

	return func(c *gin.Context) {
		// Check exclusion before any work
		if agent.RouteMatcher().ShouldExclude(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Custom filter
		if mCfg.customFilter != nil && !mCfg.customFilter(c.Request) {
			c.Next()
			return
		}

		// Lazy init on first request (after agent.Init() has completed)
		lazyInit()

		httpCfg := agent.Config().HTTP
		start := time.Now()

		// Extract propagation context from incoming headers (W3C traceparent, baggage)
		ctx := otel.GetTextMapPropagator().Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		spanName := fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path)

		// Start span with HTTP semconv request attributes
		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(requestAttrs(serviceName, c)...),
		)
		defer span.End()

		// Propagate trace context into the request so handlers and downstream
		// instrumentation (GORM, otelhttp clients) use the correct parent span.
		c.Request = c.Request.WithContext(ctx)

		// Capture request body BEFORE handler runs (if enabled)
		var reqBody string
		if httpCfg.CaptureRequestBody && scrubber.IsAllowedContentType(c.ContentType()) {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil && len(bodyBytes) > 0 {
				reqBody = string(bodyBytes)
				c.Request.Body = io.NopCloser(strings.NewReader(reqBody))
			}
		}

		// Wrap response writer for body capture (if enabled)
		var blw *BodyLogWriter
		if httpCfg.CaptureResponseBody {
			blw = NewBodyLogWriter(c.Writer)
			c.Writer = blw
		}

		// ---- Run handler chain ----
		c.Next()

		// ---- Post-handler: span is still open, enrichment works ----
		duration := time.Since(start)
		statusCode := c.Writer.Status()

		// Response attributes
		span.SetAttributes(
			attribute.Int("http.response.status_code", statusCode),
			attribute.Int("http.response.body.size", c.Writer.Size()),
		)

		if route := c.FullPath(); route != "" {
			span.SetAttributes(attribute.String("http.route", route))
			// Update span name to use the registered route pattern
			span.SetName(fmt.Sprintf("%s %s", c.Request.Method, route))
		}

		// Span status
		if statusCode >= 500 {
			span.SetStatus(codes.Error, "")
		}
		if len(c.Errors) > 0 {
			span.SetStatus(codes.Error, c.Errors.String())
			for _, err := range c.Errors {
				span.RecordError(err.Err)
			}
		}

		// Custom enrichment: headers, body, query params, user context
		enrichSpan(c, span, httpCfg, scrubber, reqBody, blw, statusCode)

		// Record metrics (bounded cardinality)
		route := c.FullPath()
		if route == "" {
			route = "unknown"
		}

		metricAttrs := []attribute.KeyValue{
			attribute.String("http.request.method", c.Request.Method),
			attribute.String("http.route", route),
			attribute.Int("http.response.status_code", statusCode),
		}

		if httpDuration != nil {
			httpDuration.Record(c.Request.Context(), duration.Seconds(), metric.WithAttributes(metricAttrs...))
		}
		if requestCounter != nil {
			requestCounter.Add(c.Request.Context(), 1, metric.WithAttributes(metricAttrs...))
		}
		if statusCode >= 400 && errorCounter != nil {
			errorCounter.Add(c.Request.Context(), 1, metric.WithAttributes(metricAttrs...))
		}
	}
}

// requestAttrs returns HTTP semconv request attributes for the span start.
func requestAttrs(server string, c *gin.Context) []attribute.KeyValue {
	req := c.Request
	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}

	attrs := []attribute.KeyValue{
		attribute.String("http.request.method", req.Method),
		attribute.String("url.scheme", scheme),
		attribute.String("server.address", server),
	}

	if req.URL != nil && req.URL.Path != "" {
		attrs = append(attrs, attribute.String("url.path", req.URL.Path))
	}

	if clientIP := c.ClientIP(); clientIP != "" {
		attrs = append(attrs, attribute.String("client.address", clientIP))
	}

	if ua := req.UserAgent(); ua != "" {
		attrs = append(attrs, attribute.String("user_agent.original", ua))
	}

	if req.ContentLength > 0 {
		attrs = append(attrs, attribute.Int64("http.request.content_length", req.ContentLength))
	}

	return attrs
}

// enrichSpan adds HTTP headers, query params, body, user context, and error events to the span.
func enrichSpan(c *gin.Context, span trace.Span, httpCfg otelagent.HTTPConfig, scrubber *provider.HTTPScrubber, reqBody string, blw *BodyLogWriter, statusCode int) {
	// Client IP and request ID
	span.SetAttributes(
		attribute.String("http.client_ip", c.ClientIP()),
		attribute.String("http.request.id", c.GetString("requestID")),
	)

	// Request headers
	if httpCfg.CaptureRequestHeaders {
		headers := scrubber.ScrubHeaders(c.Request.Header, httpCfg.AllowedRequestHeaders)
		for k, v := range headers {
			span.SetAttributes(attribute.String("http.request.header."+k, v))
		}
	}

	// Response headers
	if httpCfg.CaptureResponseHeaders {
		headers := scrubber.ScrubHeaders(c.Writer.Header(), httpCfg.AllowedResponseHeaders)
		for k, v := range headers {
			span.SetAttributes(attribute.String("http.response.header."+k, v))
		}
	}

	// Query params
	if httpCfg.CaptureQueryParams && c.Request.URL.RawQuery != "" {
		scrubbed := scrubber.ScrubQueryString(c.Request.URL.RawQuery)
		span.SetAttributes(attribute.String("url.query", scrubbed))
	}

	// Request body
	if httpCfg.CaptureRequestBody && reqBody != "" {
		scrubbed := scrubber.ScrubBody(reqBody, httpCfg.RequestBodyMaxSize)
		span.SetAttributes(
			attribute.String("http.request.body", scrubbed),
			attribute.Int("http.request.body.size", len(reqBody)),
		)
	}

	// Response body
	if httpCfg.CaptureResponseBody && blw != nil && blw.Body.Len() > 0 {
		respBody := blw.Body.String()
		if scrubber.IsAllowedContentType(c.Writer.Header().Get("Content-Type")) {
			scrubbed := scrubber.ScrubBody(respBody, httpCfg.ResponseBodyMaxSize)
			span.SetAttributes(
				attribute.String("http.response.body", scrubbed),
				attribute.Int("http.response.body.size", len(respBody)),
			)
		}
	}

	// User context (spans only, not metrics - cardinality fix)
	if userID, exists := c.Get("user_id"); exists {
		span.SetAttributes(attribute.String("user.id", fmt.Sprintf("%v", userID)))
	}
	if userRole, exists := c.Get("user_role"); exists {
		span.SetAttributes(attribute.String("user.role", fmt.Sprintf("%v", userRole)))
	}

	// Trace ID response header for debugging
	c.Header("X-Trace-Id", span.SpanContext().TraceID().String())

	// Exception events for 4xx/5xx
	if httpCfg.RecordExceptionEvents && statusCode >= 400 {
		errMsg := http.StatusText(statusCode)
		if len(c.Errors) > 0 {
			errMsg = c.Errors.String()
		}
		span.AddEvent("exception", trace.WithAttributes(
			attribute.String("exception.type", fmt.Sprintf("HTTP %d", statusCode)),
			attribute.String("exception.message", errMsg),
		))
	}
}
