package instrumentor

import (
	"fmt"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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

	client.Transport = otelhttp.NewTransport(
		transport,
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return fmt.Sprintf("HTTP %s %s", r.Method, r.URL.Host)
		}),
	)

	return client
}
