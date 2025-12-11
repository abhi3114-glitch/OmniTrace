package sdk

import (
	"fmt"
	"runtime"
	"strings"
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity string

const (
	SeverityDebug    ErrorSeverity = "debug"
	SeverityInfo     ErrorSeverity = "info"
	SeverityWarning  ErrorSeverity = "warning"
	SeverityError    ErrorSeverity = "error"
	SeverityCritical ErrorSeverity = "critical"
)

// ErrorRecord represents a captured error with context
type ErrorRecord struct {
	Message    string
	Type       string
	Severity   ErrorSeverity
	StackTrace []StackFrame
	Tags       map[string]string
}

// StackFrame represents a single frame in a stack trace
type StackFrame struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
}

// CaptureError captures an error with stack trace
func CaptureError(err error) *ErrorRecord {
	return &ErrorRecord{
		Message:    err.Error(),
		Type:       fmt.Sprintf("%T", err),
		Severity:   SeverityError,
		StackTrace: captureStackTrace(2),
		Tags:       make(map[string]string),
	}
}

// CaptureErrorWithSeverity captures an error with a specific severity
func CaptureErrorWithSeverity(err error, severity ErrorSeverity) *ErrorRecord {
	record := CaptureError(err)
	record.Severity = severity
	return record
}

// WithTag adds a tag to the error record
func (e *ErrorRecord) WithTag(key, value string) *ErrorRecord {
	e.Tags[key] = value
	return e
}

// AttachToSpan attaches the error to a span
func (e *ErrorRecord) AttachToSpan(span *SpanBuilder) {
	if span == nil {
		return
	}

	// Convert stack frames to strings
	stackStrings := make([]string, len(e.StackTrace))
	for i, frame := range e.StackTrace {
		stackStrings[i] = fmt.Sprintf("%s (%s:%d)", frame.Function, frame.File, frame.Line)
	}

	span.SetErrorWithStack(fmt.Errorf("%s", e.Message), stackStrings)
	span.SetTag("error.type", e.Type)
	span.SetTag("error.severity", string(e.Severity))

	for k, v := range e.Tags {
		span.SetTag(k, v)
	}
}

// StackTraceStrings returns the stack trace as a slice of strings
func (e *ErrorRecord) StackTraceStrings() []string {
	result := make([]string, len(e.StackTrace))
	for i, frame := range e.StackTrace {
		result[i] = fmt.Sprintf("%s (%s:%d)", frame.Function, frame.File, frame.Line)
	}
	return result
}

// captureStackTrace captures the current stack trace
func captureStackTrace(skip int) []StackFrame {
	const maxDepth = 32
	var pcs [maxDepth]uintptr
	n := runtime.Callers(skip+1, pcs[:])

	frames := make([]StackFrame, 0, n)
	callersFrames := runtime.CallersFrames(pcs[:n])

	for {
		frame, more := callersFrames.Next()

		// Skip runtime internal frames
		if strings.Contains(frame.Function, "runtime.") {
			if !more {
				break
			}
			continue
		}

		frames = append(frames, StackFrame{
			Function: frame.Function,
			File:     frame.File,
			Line:     frame.Line,
		})

		if !more {
			break
		}
	}

	return frames
}

// GetCurrentStackTrace returns the current stack trace as strings
func GetCurrentStackTrace(skip int) []string {
	frames := captureStackTrace(skip + 1)
	result := make([]string, len(frames))
	for i, frame := range frames {
		result[i] = fmt.Sprintf("%s (%s:%d)", frame.Function, frame.File, frame.Line)
	}
	return result
}

// RecoverWithSpan recovers from a panic and records it on the span
func RecoverWithSpan(span *SpanBuilder) {
	if r := recover(); r != nil {
		err := fmt.Errorf("panic: %v", r)
		record := &ErrorRecord{
			Message:    err.Error(),
			Type:       "panic",
			Severity:   SeverityCritical,
			StackTrace: captureStackTrace(2),
		}
		record.AttachToSpan(span)
		span.Finish()
		panic(r) // Re-panic after recording
	}
}

// SafeGo runs a function in a goroutine with panic recovery
func SafeGo(span *SpanBuilder, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("panic in goroutine: %v", r)
				record := CaptureErrorWithSeverity(err, SeverityCritical)
				record.AttachToSpan(span)
			}
		}()
		fn()
	}()
}

// ErrorLogger provides structured error logging
type ErrorLogger struct {
	exporter *Exporter
	service  string
}

// NewErrorLogger creates a new error logger
func NewErrorLogger(exporter *Exporter, service string) *ErrorLogger {
	return &ErrorLogger{
		exporter: exporter,
		service:  service,
	}
}

// LogError logs an error without a span context
func (l *ErrorLogger) LogError(err error, tags map[string]string) {
	record := CaptureError(err)
	for k, v := range tags {
		record.Tags[k] = v
	}
	record.Tags["service"] = l.service

	// Could be sent as a special error event
	// For now, we just capture it with the stack
}

// LogErrorWithSeverity logs an error with a specific severity
func (l *ErrorLogger) LogErrorWithSeverity(err error, severity ErrorSeverity, tags map[string]string) {
	record := CaptureErrorWithSeverity(err, severity)
	for k, v := range tags {
		record.Tags[k] = v
	}
	record.Tags["service"] = l.service
}
