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
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
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
// Uses sync.Once for lazy initialization to ensure providers are real (not noop)
// when running inside FX lifecycle where agent.Init() happens in OnStart.
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
		otelMiddleware gin.HandlerFunc
		httpDuration   metric.Float64Histogram
		requestCounter metric.Int64Counter
		errorCounter   metric.Int64Counter
		scrubber       *provider.HTTPScrubber
	)

	lazyInit := func() {
		initOnce.Do(func() {
			httpCfg := agent.Config().HTTP
			scrubCfg := agent.Config().Scrub
			scrubber = provider.NewHTTPScrubber(httpCfg, scrubCfg)

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

			// Use otelgin with the real TracerProvider (now initialized)
			otelMiddleware = otelgin.Middleware(serviceName,
				otelgin.WithTracerProvider(otel.GetTracerProvider()),
				otelgin.WithFilter(filterFn),
				otelgin.WithSpanNameFormatter(func(c *gin.Context) string {
					return fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path)
				}),
			)

			// Create metric instruments from the real MeterProvider
			meter := agent.GetMeter("github.com/RodolfoBonis/go-otel-agent/integration/ginmiddleware")
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

		// Lazy init on first request (after agent.Init() has completed)
		lazyInit()

		httpCfg := agent.Config().HTTP
		start := time.Now()

		// Capture request body BEFORE handler runs (if enabled)
		var reqBody string
		if httpCfg.CaptureRequestBody && scrubber.IsAllowedContentType(c.ContentType()) {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil && len(bodyBytes) > 0 {
				reqBody = string(bodyBytes)
				// Restore the body so the handler can read it
				c.Request.Body = io.NopCloser(strings.NewReader(reqBody))
			}
		}

		// Wrap response writer for body capture (if enabled)
		var blw *BodyLogWriter
		if httpCfg.CaptureResponseBody {
			blw = NewBodyLogWriter(c.Writer)
			c.Writer = blw
		}

		// Run otelgin middleware (creates HTTP span)
		otelMiddleware(c)

		duration := time.Since(start)
		statusCode := c.Writer.Status()

		// Enrich span with additional attributes
		span := trace.SpanFromContext(c.Request.Context())
		if span.SpanContext().IsValid() {
			enrichSpan(c, span, httpCfg, scrubber, reqBody, blw, statusCode)
		}

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

	// Span status and error events
	if statusCode >= 500 {
		span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", statusCode))
	}

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
