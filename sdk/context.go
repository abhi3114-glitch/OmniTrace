package sdk

import (
	"context"
	"sync"
)

// SpanContext holds the trace context information
type SpanContext struct {
	TraceID  string
	SpanID   string
	ParentID string
	Sampled  bool
	Baggage  map[string]string
}

// contextKey is a private type for context keys
type contextKey struct{}

var (
	spanContextKey = contextKey{}
	spanBuilderKey = contextKey{}
)

// ContextWithSpan returns a new context with the span attached
func ContextWithSpan(ctx context.Context, span *SpanBuilder) context.Context {
	return context.WithValue(ctx, spanBuilderKey, span)
}

// SpanFromContext returns the span from the context, or nil if not present
func SpanFromContext(ctx context.Context) *SpanBuilder {
	if span, ok := ctx.Value(spanBuilderKey).(*SpanBuilder); ok {
		return span
	}
	return nil
}

// ContextWithSpanContext returns a new context with the span context attached
func ContextWithSpanContext(ctx context.Context, sc SpanContext) context.Context {
	return context.WithValue(ctx, spanContextKey, sc)
}

// SpanContextFromContext returns the span context from the context
func SpanContextFromContext(ctx context.Context) (SpanContext, bool) {
	if sc, ok := ctx.Value(spanContextKey).(SpanContext); ok {
		return sc, true
	}
	return SpanContext{}, false
}

// StartSpanFromContext creates a new span as a child of the span in the context
func StartSpanFromContext(ctx context.Context, operationName string, opts ...SpanOption) (*SpanBuilder, context.Context) {
	tracer := GlobalTracer()

	// Check for existing span in context
	if parentSpan := SpanFromContext(ctx); parentSpan != nil {
		opts = append([]SpanOption{WithParent(parentSpan)}, opts...)
	} else if sc, ok := SpanContextFromContext(ctx); ok {
		opts = append([]SpanOption{WithParentContext(sc)}, opts...)
	}

	span := tracer.StartSpan(operationName, opts...)
	newCtx := ContextWithSpan(ctx, span)
	newCtx = ContextWithSpanContext(newCtx, span.Context())

	return span, newCtx
}

// AsyncContext maintains trace context across goroutines
type AsyncContext struct {
	mu      sync.RWMutex
	spanCtx SpanContext
	baggage map[string]string
}

// NewAsyncContext creates a new async context from a span
func NewAsyncContext(span *SpanBuilder) *AsyncContext {
	return &AsyncContext{
		spanCtx: span.Context(),
		baggage: make(map[string]string),
	}
}

// NewAsyncContextFromSpanContext creates an async context from SpanContext
func NewAsyncContextFromSpanContext(sc SpanContext) *AsyncContext {
	ac := &AsyncContext{
		spanCtx: sc,
		baggage: make(map[string]string),
	}
	if sc.Baggage != nil {
		for k, v := range sc.Baggage {
			ac.baggage[k] = v
		}
	}
	return ac
}

// SpanContext returns the span context
func (ac *AsyncContext) SpanContext() SpanContext {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return ac.spanCtx
}

// SetBaggage sets a baggage item
func (ac *AsyncContext) SetBaggage(key, value string) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.baggage[key] = value
	ac.spanCtx.Baggage = ac.baggage
}

// GetBaggage gets a baggage item
func (ac *AsyncContext) GetBaggage(key string) (string, bool) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	v, ok := ac.baggage[key]
	return v, ok
}

// ToContext converts the async context to a Go context
func (ac *AsyncContext) ToContext(ctx context.Context) context.Context {
	return ContextWithSpanContext(ctx, ac.SpanContext())
}

// Go executes a function in a goroutine with trace context propagation
func (ac *AsyncContext) Go(fn func(ctx context.Context)) {
	go func() {
		ctx := ac.ToContext(context.Background())
		fn(ctx)
	}()
}

// GoWithSpan executes a function in a goroutine with a new child span
func (ac *AsyncContext) GoWithSpan(operationName string, fn func(ctx context.Context, span *SpanBuilder)) {
	go func() {
		ctx := ac.ToContext(context.Background())
		span, ctx := StartSpanFromContext(ctx, operationName)
		defer span.Finish()
		fn(ctx, span)
	}()
}

// WrapAsync wraps a function to preserve trace context
func WrapAsync(ctx context.Context, fn func(ctx context.Context)) func() {
	sc, _ := SpanContextFromContext(ctx)
	return func() {
		newCtx := ContextWithSpanContext(context.Background(), sc)
		fn(newCtx)
	}
}
