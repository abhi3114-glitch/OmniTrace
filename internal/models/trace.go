package models

import (
	"sort"
	"time"
)

// Trace represents a complete distributed trace
type Trace struct {
	TraceID   string        `json:"trace_id"`
	RootSpan  *Span         `json:"root_span"`
	Spans     []Span        `json:"spans"`
	Services  []string      `json:"services"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`
	SpanCount int           `json:"span_count"`
	HasError  bool          `json:"has_error"`
}

// ServiceNode represents a node in the service dependency graph
type ServiceNode struct {
	Name        string   `json:"name"`
	SpanCount   int      `json:"span_count"`
	ErrorCount  int      `json:"error_count"`
	AvgDuration float64  `json:"avg_duration_ms"`
	Connections []string `json:"connections"`
}

// ServiceGraph represents the service dependency graph
type ServiceGraph struct {
	Nodes []ServiceNode `json:"nodes"`
	Edges []ServiceEdge `json:"edges"`
}

// ServiceEdge represents an edge in the service graph
type ServiceEdge struct {
	Source     string  `json:"source"`
	Target     string  `json:"target"`
	CallCount  int     `json:"call_count"`
	ErrorRate  float64 `json:"error_rate"`
	AvgLatency float64 `json:"avg_latency_ms"`
}

// TraceSummary provides a summary of a trace
type TraceSummary struct {
	TraceID       string        `json:"trace_id"`
	RootOperation string        `json:"root_operation"`
	RootService   string        `json:"root_service"`
	StartTime     time.Time     `json:"start_time"`
	Duration      time.Duration `json:"duration"`
	SpanCount     int           `json:"span_count"`
	ServiceCount  int           `json:"service_count"`
	HasError      bool          `json:"has_error"`
}

// TraceQuery represents a query for traces
type TraceQuery struct {
	Service     string        `json:"service,omitempty"`
	Operation   string        `json:"operation,omitempty"`
	MinDuration time.Duration `json:"min_duration,omitempty"`
	MaxDuration time.Duration `json:"max_duration,omitempty"`
	StartTime   time.Time     `json:"start_time"`
	EndTime     time.Time     `json:"end_time"`
	HasError    *bool         `json:"has_error,omitempty"`
	Limit       int           `json:"limit"`
	Offset      int           `json:"offset"`
}

// BuildTrace constructs a Trace from a slice of spans
func BuildTrace(spans []Span) *Trace {
	if len(spans) == 0 {
		return nil
	}

	trace := &Trace{
		TraceID:   spans[0].TraceID,
		Spans:     spans,
		SpanCount: len(spans),
	}

	// Sort spans by start time
	sort.Slice(trace.Spans, func(i, j int) bool {
		return trace.Spans[i].StartTime.Before(trace.Spans[j].StartTime)
	})

	// Find root span and collect unique services
	serviceMap := make(map[string]bool)
	for i := range trace.Spans {
		span := &trace.Spans[i]
		serviceMap[span.ServiceName] = true

		if span.ParentSpanID == "" {
			trace.RootSpan = span
		}

		if span.Status == SpanStatusError {
			trace.HasError = true
		}
	}

	// Extract unique services
	for service := range serviceMap {
		trace.Services = append(trace.Services, service)
	}
	sort.Strings(trace.Services)

	// Calculate trace timing
	if len(trace.Spans) > 0 {
		trace.StartTime = trace.Spans[0].StartTime
		trace.EndTime = trace.Spans[0].EndTime

		for _, span := range trace.Spans {
			if span.StartTime.Before(trace.StartTime) {
				trace.StartTime = span.StartTime
			}
			if span.EndTime.After(trace.EndTime) {
				trace.EndTime = span.EndTime
			}
		}
		trace.Duration = trace.EndTime.Sub(trace.StartTime)
	}

	return trace
}

// ToSummary creates a TraceSummary from a Trace
func (t *Trace) ToSummary() TraceSummary {
	summary := TraceSummary{
		TraceID:      t.TraceID,
		StartTime:    t.StartTime,
		Duration:     t.Duration,
		SpanCount:    t.SpanCount,
		ServiceCount: len(t.Services),
		HasError:     t.HasError,
	}

	if t.RootSpan != nil {
		summary.RootOperation = t.RootSpan.OperationName
		summary.RootService = t.RootSpan.ServiceName
	}

	return summary
}
