package sdk

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/omnitrace/omnitrace/internal/models"
)

// HTTPClient is an instrumented HTTP client
type HTTPClient struct {
	client *http.Client
	tracer *Tracer
}

// NewHTTPClient creates a new instrumented HTTP client
func NewHTTPClient(tracer *Tracer, timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
		tracer: tracer,
	}
}

// NewHTTPClientWithClient wraps an existing http.Client
func NewHTTPClientWithClient(tracer *Tracer, client *http.Client) *HTTPClient {
	return &HTTPClient{
		client: client,
		tracer: tracer,
	}
}

// Do executes an HTTP request with tracing
func (c *HTTPClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	operationName := fmt.Sprintf("HTTP %s %s", req.Method, req.URL.Host)

	span, ctx := StartSpanFromContext(ctx, operationName,
		WithKind(models.SpanKindClient),
		WithTag("http.method", req.Method),
		WithTag("http.url", req.URL.String()),
		WithTag("http.host", req.URL.Host),
		WithTag("peer.service", req.URL.Host),
	)
	defer span.Finish()

	// Inject trace context into outgoing request
	if sc, ok := SpanContextFromContext(ctx); ok {
		InjectSpanContext(req, sc)
	} else {
		InjectSpanContext(req, span.Context())
	}

	req = req.WithContext(ctx)

	start := time.Now()
	resp, err := c.client.Do(req)
	duration := time.Since(start)

	span.SetTag("http.duration_ms", fmt.Sprintf("%d", duration.Milliseconds()))

	if err != nil {
		span.SetError(err)
		span.SetTag("error", "true")
		return nil, err
	}

	span.SetTag("http.status_code", fmt.Sprintf("%d", resp.StatusCode))

	if resp.StatusCode >= 400 {
		span.SetTag("error", "true")
		span.span.Status = models.SpanStatusError
		span.span.StatusMessage = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return resp, nil
}

// Get performs a GET request
func (c *HTTPClient) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(ctx, req)
}

// Post performs a POST request
func (c *HTTPClient) Post(ctx context.Context, url string, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return c.Do(ctx, req)
}

// RoundTripper is an http.RoundTripper that adds tracing
type RoundTripper struct {
	transport http.RoundTripper
	tracer    *Tracer
}

// NewRoundTripper creates a new tracing RoundTripper
func NewRoundTripper(tracer *Tracer, transport http.RoundTripper) *RoundTripper {
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &RoundTripper{
		transport: transport,
		tracer:    tracer,
	}
}

// RoundTrip implements http.RoundTripper
func (rt *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	operationName := fmt.Sprintf("HTTP %s %s", req.Method, req.URL.Host)

	span, ctx := StartSpanFromContext(ctx, operationName,
		WithKind(models.SpanKindClient),
		WithTag("http.method", req.Method),
		WithTag("http.url", req.URL.String()),
		WithTag("http.host", req.URL.Host),
	)
	defer span.Finish()

	// Inject trace context
	if sc, ok := SpanContextFromContext(ctx); ok {
		InjectSpanContext(req, sc)
	} else {
		InjectSpanContext(req, span.Context())
	}

	req = req.WithContext(ctx)

	start := time.Now()
	resp, err := rt.transport.RoundTrip(req)
	duration := time.Since(start)

	span.SetTag("http.duration_ms", fmt.Sprintf("%d", duration.Milliseconds()))

	if err != nil {
		span.SetError(err)
		return nil, err
	}

	span.SetTag("http.status_code", fmt.Sprintf("%d", resp.StatusCode))
	if resp.StatusCode >= 400 {
		span.span.Status = models.SpanStatusError
	}

	return resp, nil
}

// InstrumentedClient returns an http.Client with tracing instrumentation
func InstrumentedClient(tracer *Tracer, timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: NewRoundTripper(tracer, nil),
	}
}
