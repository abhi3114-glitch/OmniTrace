package storage

import (
	"sync"
	"time"

	"github.com/omnitrace/omnitrace/internal/models"
)

// SpanStore implements in-memory storage for spans
type SpanStore struct {
	spans        map[string][]models.Span // TraceID -> Spans
	serviceSpans map[string][]string      // Service -> TraceIDs
	mu           sync.RWMutex
	maxSpans     int
	ttl          time.Duration
}

// NewSpanStore creates a new span store
func NewSpanStore(maxSpans int, ttl time.Duration) *SpanStore {
	store := &SpanStore{
		spans:        make(map[string][]models.Span),
		serviceSpans: make(map[string][]string),
		maxSpans:     maxSpans,
		ttl:          ttl,
	}

	// Start cleanup loop
	go store.cleanupLoop()

	return store
}

// Store adds a span to storage
func (s *SpanStore) Store(span models.Span) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Store by TraceID
	s.spans[span.TraceID] = append(s.spans[span.TraceID], span)

	// Index by Service (if root span or just simpler indexing)
	// For simplicity, we just track trace IDs per service here
	// In a real DB, this would be an index
	s.serviceSpans[span.ServiceName] = append(s.serviceSpans[span.ServiceName], span.TraceID)

	return nil
}

// GetTrace retrieves a full trace by ID
func (s *SpanStore) GetTrace(traceID string) (*models.Trace, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	spans, ok := s.spans[traceID]
	if !ok {
		return nil, nil
	}

	// Return a copy to avoid race conditions
	spansCopy := make([]models.Span, len(spans))
	copy(spansCopy, spans)

	return models.BuildTrace(spansCopy), nil
}

// QueryTraces searches for traces matching criteria
func (s *SpanStore) QueryTraces(query models.TraceQuery) ([]models.TraceSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var summaries []models.TraceSummary

	// This is a naive O(N) scan.
	// In production, we would use proper indexes (B-Trees, Inverted Index).
	// Since we grouped by TraceID, we iterate over traces.

	count := 0
	skipped := 0

	for _, spans := range s.spans {
		// Fast check: service filter
		if query.Service != "" {
			found := false
			for _, span := range spans {
				if span.ServiceName == query.Service {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		trace := models.BuildTrace(spans)
		if trace == nil {
			continue
		}

		// Time range filter
		if !query.StartTime.IsZero() && trace.StartTime.Before(query.StartTime) {
			continue
		}
		if !query.EndTime.IsZero() && trace.EndTime.After(query.EndTime) {
			continue
		}

		// Duration filter
		if query.MinDuration > 0 && trace.Duration < query.MinDuration {
			continue
		}
		if query.MaxDuration > 0 && trace.Duration > query.MaxDuration {
			continue
		}

		// Error filter
		if query.HasError != nil {
			if *query.HasError != trace.HasError {
				continue
			}
		}

		// Operation filter (root span)
		if query.Operation != "" && trace.RootSpan != nil {
			if trace.RootSpan.OperationName != query.Operation {
				continue
			}
		}

		// Apply offset/limit
		if skipped < query.Offset {
			skipped++
			continue
		}

		summaries = append(summaries, trace.ToSummary())
		count++

		if query.Limit > 0 && count >= query.Limit {
			break
		}
	}

	return summaries, nil
}

// cleanupLoop periodically removes old traces
func (s *SpanStore) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		s.cleanup()
	}
}

func (s *SpanStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-s.ttl)

	for traceID, spans := range s.spans {
		if len(spans) > 0 {
			// Check if the trace is too old
			// We check the first span's start time (simplification)
			if spans[0].StartTime.Before(cutoff) {
				delete(s.spans, traceID)
			}
		}
	}
}
