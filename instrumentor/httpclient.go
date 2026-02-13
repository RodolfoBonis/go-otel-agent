package instrumentor

import (
	"fmt"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentHTTPClient wraps an HTTP client's transport with OTel instrumentation.
func InstrumentHTTPClient(client *http.Client) *http.Client {
	if client == nil {
		client = &http.Client{}
	}

	transport := client.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	// legacySemconvTransport is the inner transport: otelhttp wraps it,
	// so by the time RoundTrip runs, the span is already in req.Context().
	client.Transport = otelhttp.NewTransport(
		&legacySemconvTransport{base: transport},
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return fmt.Sprintf("HTTP %s %s", r.Method, r.URL.Host)
		}),
	)

	return client
}

// legacySemconvTransport injects legacy semantic convention attributes
// (net.peer.name, http.url, http.method, http.status_code) that SigNoz
// External Call dashboard uses for hostname grouping. otelhttp v0.65.0
// only emits new semconv (server.address, url.full, http.request.method).
type legacySemconvTransport struct {
	base http.RoundTripper
}

func (t *legacySemconvTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)

	span := trace.SpanFromContext(req.Context())
	if span.SpanContext().IsValid() {
		span.SetAttributes(
			attribute.String("net.peer.name", req.URL.Hostname()),
			attribute.String("http.url", req.URL.String()),
			attribute.String("http.method", req.Method),
		)
		if resp != nil {
			span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))
		}
	}

	return resp, err
}
