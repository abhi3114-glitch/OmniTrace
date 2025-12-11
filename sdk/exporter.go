package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/omnitrace/omnitrace/internal/models"
)

// Exporter handles exporting spans and metrics to the collector
type Exporter struct {
	collectorURL  string
	client        *http.Client
	spanBuffer    []models.Span
	metricBuffer  []models.Metric
	batchSize     int
	flushInterval time.Duration
	mu            sync.Mutex
	stopCh        chan struct{}
	wg            sync.WaitGroup
	onError       func(error)
}

// ExporterConfig configures the exporter
type ExporterConfig struct {
	CollectorURL  string
	BatchSize     int
	FlushInterval time.Duration
	Timeout       time.Duration
	OnError       func(error)
}

// DefaultExporterConfig returns default exporter configuration
func DefaultExporterConfig() ExporterConfig {
	return ExporterConfig{
		CollectorURL:  "http://localhost:8080",
		BatchSize:     100,
		FlushInterval: 5 * time.Second,
		Timeout:       10 * time.Second,
	}
}

// NewExporter creates a new exporter
func NewExporter(config ExporterConfig) *Exporter {
	e := &Exporter{
		collectorURL:  config.CollectorURL,
		client:        &http.Client{Timeout: config.Timeout},
		spanBuffer:    make([]models.Span, 0, config.BatchSize),
		metricBuffer:  make([]models.Metric, 0, config.BatchSize),
		batchSize:     config.BatchSize,
		flushInterval: config.FlushInterval,
		stopCh:        make(chan struct{}),
		onError:       config.OnError,
	}

	e.wg.Add(1)
	go e.flushLoop()

	return e
}

// Export adds a span to the export buffer
func (e *Exporter) Export(span models.Span) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.spanBuffer = append(e.spanBuffer, span)

	if len(e.spanBuffer) >= e.batchSize {
		e.flushSpansLocked()
	}
}

// ExportMetric adds a metric to the export buffer
func (e *Exporter) ExportMetric(metric models.Metric) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.metricBuffer = append(e.metricBuffer, metric)

	if len(e.metricBuffer) >= e.batchSize {
		e.flushMetricsLocked()
	}
}

// Flush forces an immediate flush of all buffers
func (e *Exporter) Flush() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	var lastErr error
	if err := e.flushSpansLocked(); err != nil {
		lastErr = err
	}
	if err := e.flushMetricsLocked(); err != nil {
		lastErr = err
	}
	return lastErr
}

// Close stops the exporter and flushes remaining data
func (e *Exporter) Close() error {
	close(e.stopCh)
	e.wg.Wait()
	return e.Flush()
}

func (e *Exporter) flushLoop() {
	defer e.wg.Done()

	ticker := time.NewTicker(e.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.Flush()
		case <-e.stopCh:
			return
		}
	}
}

func (e *Exporter) flushSpansLocked() error {
	if len(e.spanBuffer) == 0 {
		return nil
	}

	spans := make([]models.Span, len(e.spanBuffer))
	copy(spans, e.spanBuffer)
	e.spanBuffer = e.spanBuffer[:0]

	// Send in background
	go func() {
		if err := e.sendSpans(spans); err != nil {
			if e.onError != nil {
				e.onError(err)
			}
		}
	}()

	return nil
}

func (e *Exporter) flushMetricsLocked() error {
	if len(e.metricBuffer) == 0 {
		return nil
	}

	metrics := make([]models.Metric, len(e.metricBuffer))
	copy(metrics, e.metricBuffer)
	e.metricBuffer = e.metricBuffer[:0]

	// Send in background
	go func() {
		if err := e.sendMetrics(metrics); err != nil {
			if e.onError != nil {
				e.onError(err)
			}
		}
	}()

	return nil
}

func (e *Exporter) sendSpans(spans []models.Span) error {
	batch := models.SpanBatch{Spans: spans}

	data, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("failed to marshal spans: %w", err)
	}

	resp, err := e.client.Post(
		e.collectorURL+"/api/v1/spans",
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		return fmt.Errorf("failed to send spans: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("collector returned status %d", resp.StatusCode)
	}

	return nil
}

func (e *Exporter) sendMetrics(metrics []models.Metric) error {
	batch := models.MetricBatch{Metrics: metrics}

	data, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	resp, err := e.client.Post(
		e.collectorURL+"/api/v1/metrics",
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		return fmt.Errorf("failed to send metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("collector returned status %d", resp.StatusCode)
	}

	return nil
}

// NoopExporter is an exporter that does nothing (for testing)
type NoopExporter struct{}

func (NoopExporter) Export(span models.Span)           {}
func (NoopExporter) ExportMetric(metric models.Metric) {}
func (NoopExporter) Flush() error                      { return nil }
func (NoopExporter) Close() error                      { return nil }
