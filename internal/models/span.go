package models

import (
	"time"
)

// SpanKind represents the type of span
type SpanKind string

const (
	SpanKindInternal SpanKind = "internal"
	SpanKindServer   SpanKind = "server"
	SpanKindClient   SpanKind = "client"
	SpanKindProducer SpanKind = "producer"
	SpanKindConsumer SpanKind = "consumer"
)

// SpanStatus represents the status of a span
type SpanStatus string

const (
	SpanStatusUnset SpanStatus = "unset"
	SpanStatusOK    SpanStatus = "ok"
	SpanStatusError SpanStatus = "error"
)

// Span represents a single unit of work in a distributed trace
type Span struct {
	TraceID      string            `json:"trace_id"`
	SpanID       string            `json:"span_id"`
	ParentSpanID string            `json:"parent_span_id,omitempty"`
	OperationName string           `json:"operation_name"`
	ServiceName  string            `json:"service_name"`
	Kind         SpanKind          `json:"kind"`
	StartTime    time.Time         `json:"start_time"`
	EndTime      time.Time         `json:"end_time"`
	Duration     time.Duration     `json:"duration"`
	Status       SpanStatus        `json:"status"`
	StatusMessage string           `json:"status_message,omitempty"`
	Tags         map[string]string `json:"tags,omitempty"`
	Logs         []SpanLog         `json:"logs,omitempty"`
	ErrorInfo    *ErrorInfo        `json:"error_info,omitempty"`
}

// SpanLog represents a log entry within a span
type SpanLog struct {
	Timestamp time.Time         `json:"timestamp"`
	Fields    map[string]string `json:"fields"`
}

// ErrorInfo contains detailed error information
type ErrorInfo struct {
	Message    string   `json:"message"`
	Type       string   `json:"type"`
	StackTrace []string `json:"stack_trace,omitempty"`
}

// SpanBatch represents a batch of spans for ingestion
type SpanBatch struct {
	Spans []Span `json:"spans"`
}

// CalculateDuration sets the duration based on start and end times
func (s *Span) CalculateDuration() {
	s.Duration = s.EndTime.Sub(s.StartTime)
}

// AddTag adds a tag to the span
func (s *Span) AddTag(key, value string) {
	if s.Tags == nil {
		s.Tags = make(map[string]string)
	}
	s.Tags[key] = value
}

// AddLog adds a log entry to the span
func (s *Span) AddLog(fields map[string]string) {
	s.Logs = append(s.Logs, SpanLog{
		Timestamp: time.Now(),
		Fields:    fields,
	})
}

// SetError marks the span as errored with details
func (s *Span) SetError(err error, stackTrace []string) {
	s.Status = SpanStatusError
	s.ErrorInfo = &ErrorInfo{
		Message:    err.Error(),
		Type:       "error",
		StackTrace: stackTrace,
	}
}
