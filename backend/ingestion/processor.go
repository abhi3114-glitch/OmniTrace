package ingestion

import (
	"log"

	"github.com/omnitrace/omnitrace/backend/storage"
	"github.com/omnitrace/omnitrace/internal/models"
)

// Processor processes incoming data before storage
type Processor struct {
	spanStore   *storage.SpanStore
	metricStore *storage.MetricStore
}

// NewProcessor creates a new processor
func NewProcessor(spanStore *storage.SpanStore, metricStore *storage.MetricStore) *Processor {
	return &Processor{
		spanStore:   spanStore,
		metricStore: metricStore,
	}
}

// ProcessSpans normalizes and stores spans
func (p *Processor) ProcessSpans(spans []models.Span) {
	for _, span := range spans {
		// Basic validation could go here
		if span.TraceID == "" || span.SpanID == "" {
			continue
		}

		log.Printf("Storing span: %s", span.TraceID)

		// In a real system, we might enrich with geo-ip, etc.

		if err := p.spanStore.Store(span); err != nil {
			log.Printf("Failed to store span: %v", err)
		}
	}
}

// ProcessMetrics aggregates and stores metrics
func (p *Processor) ProcessMetrics(metrics []models.Metric) {
	for _, metric := range metrics {
		if metric.Name == "" {
			continue
		}

		if err := p.metricStore.Store(metric); err != nil {
			log.Printf("Failed to store metric: %v", err)
		}
	}
}
