package models

import (
	"time"
)

// MetricType represents the type of metric
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
)

// Metric represents a single metric data point
type Metric struct {
	Name      string            `json:"name"`
	Type      MetricType        `json:"type"`
	Value     float64           `json:"value"`
	Timestamp time.Time         `json:"timestamp"`
	Labels    map[string]string `json:"labels,omitempty"`
	Service   string            `json:"service"`
}

// HistogramBucket represents a histogram bucket
type HistogramBucket struct {
	UpperBound float64 `json:"upper_bound"`
	Count      uint64  `json:"count"`
}

// HistogramMetric represents a histogram with buckets
type HistogramMetric struct {
	Metric
	Buckets []HistogramBucket `json:"buckets"`
	Sum     float64           `json:"sum"`
	Count   uint64            `json:"count"`
}

// MetricBatch represents a batch of metrics for ingestion
type MetricBatch struct {
	Metrics []Metric `json:"metrics"`
}

// AggregatedMetric represents pre-aggregated metric data
type AggregatedMetric struct {
	Name      string            `json:"name"`
	Labels    map[string]string `json:"labels,omitempty"`
	Service   string            `json:"service"`
	StartTime time.Time         `json:"start_time"`
	EndTime   time.Time         `json:"end_time"`
	Count     int64             `json:"count"`
	Sum       float64           `json:"sum"`
	Min       float64           `json:"min"`
	Max       float64           `json:"max"`
	Avg       float64           `json:"avg"`
}

// MetricQuery represents a query for metrics
type MetricQuery struct {
	Name      string            `json:"name"`
	Labels    map[string]string `json:"labels,omitempty"`
	Service   string            `json:"service,omitempty"`
	StartTime time.Time         `json:"start_time"`
	EndTime   time.Time         `json:"end_time"`
	Step      time.Duration     `json:"step"`
}

// WithLabel adds a label to the metric
func (m *Metric) WithLabel(key, value string) *Metric {
	if m.Labels == nil {
		m.Labels = make(map[string]string)
	}
	m.Labels[key] = value
	return m
}

// NewCounter creates a new counter metric
func NewCounter(name string, value float64, service string) *Metric {
	return &Metric{
		Name:      name,
		Type:      MetricTypeCounter,
		Value:     value,
		Timestamp: time.Now(),
		Service:   service,
	}
}

// NewGauge creates a new gauge metric
func NewGauge(name string, value float64, service string) *Metric {
	return &Metric{
		Name:      name,
		Type:      MetricTypeGauge,
		Value:     value,
		Timestamp: time.Now(),
		Service:   service,
	}
}
