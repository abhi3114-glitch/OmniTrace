package storage

import (
	"sync"
	"time"

	"github.com/omnitrace/omnitrace/internal/models"
)

// MetricStore implements in-memory storage for metrics
type MetricStore struct {
	metrics   map[string][]models.Metric // Key (Name+Tags) -> Metrics
	mu        sync.RWMutex
	maxPoints int
	ttl       time.Duration
}

// NewMetricStore creates a new metric store
func NewMetricStore(maxPoints int, ttl time.Duration) *MetricStore {
	store := &MetricStore{
		metrics:   make(map[string][]models.Metric),
		maxPoints: maxPoints,
		ttl:       ttl,
	}

	go store.cleanupLoop()

	return store
}

// Store adds a metric to storage
func (s *MetricStore) Store(metric models.Metric) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := generateMetricKey(metric)
	s.metrics[key] = append(s.metrics[key], metric)

	return nil
}

// QueryMetrics retrieves aggregated metrics
func (s *MetricStore) QueryMetrics(query models.MetricQuery) ([]models.AggregatedMetric, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []models.AggregatedMetric

	// Filter by name and labels
	for _, metrics := range s.metrics {
		if len(metrics) == 0 {
			continue
		}

		// Check name match
		if metrics[0].Name != query.Name {
			continue
		}

		// Check label match
		match := true
		for k, v := range query.Labels {
			if metrics[0].Labels[k] != v {
				match = false
				break
			}
		}
		if !match {
			continue
		}

		// Aggregate buckets
		// This is a simplification. We align to steps.
		buckets := make(map[int64]*models.AggregatedMetric)

		for _, m := range metrics {
			if m.Timestamp.Before(query.StartTime) || m.Timestamp.After(query.EndTime) {
				continue
			}

			// Determine bucket
			bucketTime := m.Timestamp.Truncate(query.Step).Unix()

			agg, exists := buckets[bucketTime]
			if !exists {
				agg = &models.AggregatedMetric{
					Name:      m.Name,
					Labels:    m.Labels,
					Service:   m.Service,
					StartTime: time.Unix(bucketTime, 0),
					EndTime:   time.Unix(bucketTime, 0).Add(query.Step),
					Min:       m.Value,
					Max:       m.Value,
				}
				buckets[bucketTime] = agg
			}

			agg.Count++
			agg.Sum += m.Value
			if m.Value < agg.Min {
				agg.Min = m.Value
			}
			if m.Value > agg.Max {
				agg.Max = m.Value
			}
		}

		for _, agg := range buckets {
			agg.Avg = agg.Sum / float64(agg.Count)
			results = append(results, *agg)
		}
	}

	return results, nil
}

func generateMetricKey(m models.Metric) string {
	// composite key: name|service|sorted_labels
	// implementation simplified for prototype
	return m.Name + "|" + m.Service
}

func (s *MetricStore) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	for range ticker.C {
		s.cleanup()
	}
}

func (s *MetricStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-s.ttl)

	for key, metrics := range s.metrics {
		// Filter in place
		n := 0
		for _, m := range metrics {
			if m.Timestamp.After(cutoff) {
				metrics[n] = m
				n++
			}
		}
		s.metrics[key] = metrics[:n]

		if n == 0 {
			delete(s.metrics, key)
		}
	}
}
