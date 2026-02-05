package domain

import (
	"hash/fnv"
	"time"

	"github.com/google/uuid"
)

// NewUUIDv7 generates a new UUIDv7 (time-ordered).
func NewUUIDv7() uuid.UUID {
	return uuid.Must(uuid.NewV7())
}

// MetricType represents the type of metric being recorded.
type MetricType string

const (
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeCounter   MetricType = "counter"
	MetricTypeHistogram MetricType = "histogram"
)

// Metric represents a single time-series data point.
// Uses UUIDv7 for monotonic, time-ordered primary keys.
type Metric struct {
	ID         uuid.UUID         `json:"id"`
	Name       string            `json:"name"`
	Type       MetricType        `json:"type"`
	Value      float64           `json:"value"`
	Timestamp  time.Time         `json:"timestamp"`
	Tags       map[string]string `json:"tags"`
	SeriesHash uint64            `json:"series_hash"`
}

// NewMetric creates a new metric with UUIDv7 and computed series hash.
func NewMetric(name string, metricType MetricType, value float64, tags map[string]string) *Metric {
	now := time.Now()
	m := &Metric{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      name,
		Type:      metricType,
		Value:     value,
		Timestamp: now,
		Tags:      tags,
	}
	m.SeriesHash = m.computeSeriesHash()
	return m
}

// computeSeriesHash generates a FNV-1a hash of the metric name and tags.
// This enables fast lookups for time-series queries.
func (m *Metric) computeSeriesHash() uint64 {
	h := fnv.New64a()
	h.Write([]byte(m.Name))
	for k, v := range m.Tags {
		h.Write([]byte(k))
		h.Write([]byte(v))
	}
	return h.Sum64()
}

// MetricSeries represents a collection of metrics with the same identity.
type MetricSeries struct {
	Name       string            `json:"name"`
	Tags       map[string]string `json:"tags"`
	SeriesHash uint64            `json:"series_hash"`
	Points     []MetricPoint     `json:"points"`
}

// MetricPoint represents a single value-timestamp pair in a series.
type MetricPoint struct {
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

// AggregatedMetric represents a downsampled metric for long-term storage.
type AggregatedMetric struct {
	ID           uuid.UUID         `json:"id"`
	Name         string            `json:"name"`
	Tags         map[string]string `json:"tags"`
	SeriesHash   uint64            `json:"series_hash"`
	WindowStart  time.Time         `json:"window_start"`
	WindowEnd    time.Time         `json:"window_end"`
	Count        int64             `json:"count"`
	Sum          float64           `json:"sum"`
	Min          float64           `json:"min"`
	Max          float64           `json:"max"`
	Avg          float64           `json:"avg"`
	Resolution   string            `json:"resolution"` // "1m", "1h", "1d"
}

// NewAggregatedMetric creates a new aggregated metric from a series of points.
func NewAggregatedMetric(name string, tags map[string]string, points []MetricPoint, resolution string) *AggregatedMetric {
	if len(points) == 0 {
		return nil
	}

	agg := &AggregatedMetric{
		ID:          uuid.Must(uuid.NewV7()),
		Name:        name,
		Tags:        tags,
		WindowStart: points[0].Timestamp,
		WindowEnd:   points[len(points)-1].Timestamp,
		Count:       int64(len(points)),
		Resolution:  resolution,
	}

	// Compute series hash
	h := fnv.New64a()
	h.Write([]byte(name))
	for k, v := range tags {
		h.Write([]byte(k))
		h.Write([]byte(v))
	}
	agg.SeriesHash = h.Sum64()

	// Compute aggregations
	agg.Min = points[0].Value
	agg.Max = points[0].Value
	for _, p := range points {
		agg.Sum += p.Value
		if p.Value < agg.Min {
			agg.Min = p.Value
		}
		if p.Value > agg.Max {
			agg.Max = p.Value
		}
	}
	agg.Avg = agg.Sum / float64(agg.Count)

	return agg
}

