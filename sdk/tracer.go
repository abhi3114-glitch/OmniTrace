package sdk

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/omnitrace/omnitrace/internal/models"
)

// Tracer is the main entry point for creating spans
type Tracer struct {
	serviceName string
	exporter    *Exporter
	sampler     Sampler
	mu          sync.RWMutex
	enabled     bool
}

// TracerOption is a function that configures a Tracer
type TracerOption func(*Tracer)

// Sampler determines whether a trace should be sampled
type Sampler interface {
	ShouldSample(traceID string) bool
}

// AlwaysSample always samples traces
type AlwaysSample struct{}

func (s AlwaysSample) ShouldSample(traceID string) bool { return true }

// ProbabilitySampler samples traces with a given probability
type ProbabilitySampler struct {
	rate float64
}

func NewProbabilitySampler(rate float64) *ProbabilitySampler {
	if rate < 0 {
		rate = 0
	}
	if rate > 1 {
		rate = 1
	}
	return &ProbabilitySampler{rate: rate}
}

func (s *ProbabilitySampler) ShouldSample(traceID string) bool {
	if s.rate >= 1.0 {
		return true
	}
	if s.rate <= 0.0 {
		return false
	}
	// Simple hash-based sampling
	if len(traceID) < 2 {
		return true
	}
	b, _ := hex.DecodeString(traceID[:2])
	if len(b) == 0 {
		return true
	}
	return float64(b[0])/255.0 < s.rate
}

// Global tracer instance
var globalTracer *Tracer
var globalTracerOnce sync.Once

// NewTracer creates a new Tracer
func NewTracer(serviceName string, opts ...TracerOption) *Tracer {
	t := &Tracer{
		serviceName: serviceName,
		sampler:     AlwaysSample{},
		enabled:     true,
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// WithExporter sets the exporter for the tracer
func WithExporter(e *Exporter) TracerOption {
	return func(t *Tracer) {
		t.exporter = e
	}
}

// WithSampler sets the sampler for the tracer
func WithSampler(s Sampler) TracerOption {
	return func(t *Tracer) {
		t.sampler = s
	}
}

// InitGlobalTracer initializes the global tracer
func InitGlobalTracer(serviceName string, opts ...TracerOption) {
	globalTracerOnce.Do(func() {
		globalTracer = NewTracer(serviceName, opts...)
	})
}

// GlobalTracer returns the global tracer
func GlobalTracer() *Tracer {
	if globalTracer == nil {
		globalTracer = NewTracer("default-service")
	}
	return globalTracer
}

// StartSpan creates a new span with the given operation name
func (t *Tracer) StartSpan(operationName string, opts ...SpanOption) *SpanBuilder {
	sb := &SpanBuilder{
		tracer: t,
		span: models.Span{
			TraceID:       generateTraceID(),
			SpanID:        generateSpanID(),
			OperationName: operationName,
			ServiceName:   t.serviceName,
			Kind:          models.SpanKindInternal,
			StartTime:     time.Now(),
			Status:        models.SpanStatusUnset,
			Tags:          make(map[string]string),
		},
	}
	for _, opt := range opts {
		opt(sb)
	}
	return sb
}

// SpanBuilder helps construct spans
type SpanBuilder struct {
	tracer *Tracer
	span   models.Span
}

// SpanOption is a function that configures a SpanBuilder
type SpanOption func(*SpanBuilder)

// WithParent sets the parent span
func WithParent(parent *SpanBuilder) SpanOption {
	return func(sb *SpanBuilder) {
		if parent != nil {
			sb.span.TraceID = parent.span.TraceID
			sb.span.ParentSpanID = parent.span.SpanID
		}
	}
}

// WithParentContext sets the parent from a SpanContext
func WithParentContext(ctx SpanContext) SpanOption {
	return func(sb *SpanBuilder) {
		if ctx.TraceID != "" {
			sb.span.TraceID = ctx.TraceID
		}
		if ctx.SpanID != "" {
			sb.span.ParentSpanID = ctx.SpanID
		}
	}
}

// WithKind sets the span kind
func WithKind(kind models.SpanKind) SpanOption {
	return func(sb *SpanBuilder) {
		sb.span.Kind = kind
	}
}

// WithTag adds a tag to the span
func WithTag(key, value string) SpanOption {
	return func(sb *SpanBuilder) {
		sb.span.Tags[key] = value
	}
}

// SetTag adds a tag to the span
func (sb *SpanBuilder) SetTag(key, value string) *SpanBuilder {
	sb.span.Tags[key] = value
	return sb
}

// SetOperationName changes the operation name
func (sb *SpanBuilder) SetOperationName(name string) *SpanBuilder {
	sb.span.OperationName = name
	return sb
}

// LogFields adds a log entry to the span
func (sb *SpanBuilder) LogFields(fields map[string]string) *SpanBuilder {
	sb.span.AddLog(fields)
	return sb
}

// SetError marks the span as errored
func (sb *SpanBuilder) SetError(err error) *SpanBuilder {
	sb.span.Status = models.SpanStatusError
	sb.span.StatusMessage = err.Error()
	sb.span.ErrorInfo = &models.ErrorInfo{
		Message: err.Error(),
		Type:    "error",
	}
	return sb
}

// SetErrorWithStack marks the span as errored with stack trace
func (sb *SpanBuilder) SetErrorWithStack(err error, stack []string) *SpanBuilder {
	sb.span.Status = models.SpanStatusError
	sb.span.StatusMessage = err.Error()
	sb.span.ErrorInfo = &models.ErrorInfo{
		Message:    err.Error(),
		Type:       "error",
		StackTrace: stack,
	}
	return sb
}

// Finish completes the span
func (sb *SpanBuilder) Finish() {
	sb.span.EndTime = time.Now()
	sb.span.CalculateDuration()

	if sb.span.Status == models.SpanStatusUnset {
		sb.span.Status = models.SpanStatusOK
	}

	// Export the span
	if sb.tracer.exporter != nil && sb.tracer.enabled {
		if sb.tracer.sampler.ShouldSample(sb.span.TraceID) {
			sb.tracer.exporter.Export(sb.span)
		}
	}
}

// Context returns the span context
func (sb *SpanBuilder) Context() SpanContext {
	return SpanContext{
		TraceID: sb.span.TraceID,
		SpanID:  sb.span.SpanID,
		Sampled: true,
	}
}

// Span returns the underlying span (for testing)
func (sb *SpanBuilder) Span() models.Span {
	return sb.span
}

// generateTraceID generates a random 128-bit trace ID
func generateTraceID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// generateSpanID generates a random 64-bit span ID
func generateSpanID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
