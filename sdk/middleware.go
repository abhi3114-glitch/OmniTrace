package sdk

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/omnitrace/omnitrace/internal/models"
)

// Middleware provides HTTP middleware for automatic instrumentation
type Middleware struct {
	tracer *Tracer
	config MiddlewareConfig
}

// MiddlewareConfig configures the middleware behavior
type MiddlewareConfig struct {
	SkipPaths      []string
	OperationNamer func(r *http.Request) string
	SpanFilter     func(r *http.Request) bool
	ErrorHandler   func(w http.ResponseWriter, r *http.Request, span *SpanBuilder, err interface{})
}

// NewMiddleware creates a new middleware instance
func NewMiddleware(tracer *Tracer, config ...MiddlewareConfig) *Middleware {
	m := &Middleware{
		tracer: tracer,
		config: MiddlewareConfig{
			OperationNamer: defaultOperationNamer,
			SpanFilter:     func(r *http.Request) bool { return true },
		},
	}
	if len(config) > 0 {
		m.config = config[0]
		if m.config.OperationNamer == nil {
			m.config.OperationNamer = defaultOperationNamer
		}
		if m.config.SpanFilter == nil {
			m.config.SpanFilter = func(r *http.Request) bool { return true }
		}
	}
	return m
}

func defaultOperationNamer(r *http.Request) string {
	return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
}

// Handler wraps an http.Handler with tracing
func (m *Middleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check skip paths
		for _, path := range m.config.SkipPaths {
			if strings.HasPrefix(r.URL.Path, path) {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Check span filter
		if !m.config.SpanFilter(r) {
			next.ServeHTTP(w, r)
			return
		}

		// Extract trace context from headers
		spanCtx := extractSpanContext(r)

		// Create span options
		opts := []SpanOption{
			WithKind(models.SpanKindServer),
			WithTag("http.method", r.Method),
			WithTag("http.url", r.URL.String()),
			WithTag("http.host", r.Host),
			WithTag("http.user_agent", r.UserAgent()),
		}

		if spanCtx.TraceID != "" {
			opts = append(opts, WithParentContext(spanCtx))
		}

		// Start span
		operationName := m.config.OperationNamer(r)
		span := m.tracer.StartSpan(operationName, opts...)

		// Add span to request context
		ctx := ContextWithSpan(r.Context(), span)
		ctx = ContextWithSpanContext(ctx, span.Context())
		r = r.WithContext(ctx)

		// Wrap response writer to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Handle panics
		defer func() {
			if err := recover(); err != nil {
				span.SetTag("error", "true")
				span.SetTag("error.type", "panic")
				span.LogFields(map[string]string{
					"event":   "panic",
					"message": fmt.Sprintf("%v", err),
				})
				span.span.Status = models.SpanStatusError
				span.span.StatusMessage = fmt.Sprintf("panic: %v", err)
				span.Finish()

				if m.config.ErrorHandler != nil {
					m.config.ErrorHandler(w, r, span, err)
				} else {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}
		}()

		// Execute handler
		next.ServeHTTP(rw, r)

		// Record response
		span.SetTag("http.status_code", fmt.Sprintf("%d", rw.statusCode))

		if rw.statusCode >= 400 {
			span.SetTag("error", "true")
			span.span.Status = models.SpanStatusError
			span.span.StatusMessage = fmt.Sprintf("HTTP %d", rw.statusCode)
		}

		span.Finish()
	})
}

// HandlerFunc wraps an http.HandlerFunc with tracing
func (m *Middleware) HandlerFunc(next http.HandlerFunc) http.HandlerFunc {
	return m.Handler(next).ServeHTTP
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// W3C Trace Context header names
const (
	TraceparentHeader = "traceparent"
	TracestateHeader  = "tracestate"
)

// extractSpanContext extracts trace context from HTTP headers (W3C Trace Context)
func extractSpanContext(r *http.Request) SpanContext {
	sc := SpanContext{}

	// Parse traceparent header: version-trace_id-parent_id-trace_flags
	traceparent := r.Header.Get(TraceparentHeader)
	if traceparent != "" {
		parts := strings.Split(traceparent, "-")
		if len(parts) == 4 {
			sc.TraceID = parts[1]
			sc.SpanID = parts[2]
			sc.Sampled = parts[3] == "01"
		}
	}

	return sc
}

// InjectSpanContext injects trace context into HTTP headers
func InjectSpanContext(r *http.Request, sc SpanContext) {
	traceparent := fmt.Sprintf("00-%s-%s-01", sc.TraceID, sc.SpanID)
	r.Header.Set(TraceparentHeader, traceparent)
}

// RequestTimer provides simple request timing without full tracing
type RequestTimer struct {
	startTime time.Time
	operation string
	tags      map[string]string
}

// NewRequestTimer creates a new request timer
func NewRequestTimer(operation string) *RequestTimer {
	return &RequestTimer{
		startTime: time.Now(),
		operation: operation,
		tags:      make(map[string]string),
	}
}

// SetTag adds a tag to the timer
func (rt *RequestTimer) SetTag(key, value string) *RequestTimer {
	rt.tags[key] = value
	return rt
}

// Duration returns the elapsed time
func (rt *RequestTimer) Duration() time.Duration {
	return time.Since(rt.startTime)
}

// Record records the timing as a metric
func (rt *RequestTimer) Record(exporter *Exporter) {
	if exporter == nil {
		return
	}

	metric := models.NewGauge(
		"request_duration_ms",
		float64(rt.Duration().Milliseconds()),
		rt.tags["service"],
	)
	metric.Labels = rt.tags
	exporter.ExportMetric(*metric)
}
