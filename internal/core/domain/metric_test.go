package domain

import (
	"testing"
	"time"
)

func TestNewMetric(t *testing.T) {
	tags := map[string]string{"host": "localhost", "service": "api"}
	metric := NewMetric("cpu_usage", MetricTypeGauge, 75.5, tags)

	if metric.ID.String() == "" {
		t.Error("Metric ID should not be empty")
	}

	if metric.Name != "cpu_usage" {
		t.Errorf("Expected name 'cpu_usage', got '%s'", metric.Name)
	}

	if metric.Type != MetricTypeGauge {
		t.Errorf("Expected type %s, got %s", MetricTypeGauge, metric.Type)
	}

	if metric.Value != 75.5 {
		t.Errorf("Expected value 75.5, got %f", metric.Value)
	}

	if metric.SeriesHash == 0 {
		t.Error("SeriesHash should not be zero")
	}

	if len(metric.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(metric.Tags))
	}
}

func TestMetricSeriesHashConsistency(t *testing.T) {
	// Note: Map iteration order in Go is not guaranteed, so we use a single-tag map
	tags1 := map[string]string{"host": "localhost"}
	tags2 := map[string]string{"host": "localhost"}

	metric1 := NewMetric("cpu_usage", MetricTypeGauge, 75.5, tags1)
	metric2 := NewMetric("cpu_usage", MetricTypeGauge, 80.0, tags2)

	if metric1.SeriesHash != metric2.SeriesHash {
		t.Error("Metrics with same name and tags should have same SeriesHash")
	}
}

func TestMetricSeriesHashDifferent(t *testing.T) {
	tags1 := map[string]string{"host": "host1"}
	tags2 := map[string]string{"host": "host2"}

	metric1 := NewMetric("cpu_usage", MetricTypeGauge, 75.5, tags1)
	metric2 := NewMetric("cpu_usage", MetricTypeGauge, 75.5, tags2)

	if metric1.SeriesHash == metric2.SeriesHash {
		t.Error("Metrics with different tags should have different SeriesHash")
	}
}

func TestNewAggregatedMetric(t *testing.T) {
	tags := map[string]string{"host": "localhost"}
	now := time.Now()
	points := []MetricPoint{
		{Value: 70.0, Timestamp: now.Add(-2 * time.Minute)},
		{Value: 75.0, Timestamp: now.Add(-1 * time.Minute)},
		{Value: 80.0, Timestamp: now},
	}

	agg := NewAggregatedMetric("cpu_usage", tags, points, "1m")

	if agg == nil {
		t.Fatal("AggregatedMetric should not be nil")
	}

	if agg.Name != "cpu_usage" {
		t.Errorf("Expected name 'cpu_usage', got '%s'", agg.Name)
	}

	if agg.Count != 3 {
		t.Errorf("Expected count 3, got %d", agg.Count)
	}

	if agg.Min != 70.0 {
		t.Errorf("Expected min 70.0, got %f", agg.Min)
	}

	if agg.Max != 80.0 {
		t.Errorf("Expected max 80.0, got %f", agg.Max)
	}

	expectedSum := 70.0 + 75.0 + 80.0
	if agg.Sum != expectedSum {
		t.Errorf("Expected sum %f, got %f", expectedSum, agg.Sum)
	}

	expectedAvg := expectedSum / 3.0
	if agg.Avg != expectedAvg {
		t.Errorf("Expected avg %f, got %f", expectedAvg, agg.Avg)
	}

	if agg.Resolution != "1m" {
		t.Errorf("Expected resolution '1m', got '%s'", agg.Resolution)
	}
}

func TestNewAggregatedMetricEmptyPoints(t *testing.T) {
	tags := map[string]string{"host": "localhost"}
	agg := NewAggregatedMetric("cpu_usage", tags, []MetricPoint{}, "1m")

	if agg != nil {
		t.Error("AggregatedMetric with empty points should return nil")
	}
}

func TestMetricNilTags(t *testing.T) {
	metric := NewMetric("cpu_usage", MetricTypeGauge, 75.5, nil)

	// Metric with nil tags is acceptable - hash still works
	if metric.SeriesHash == 0 {
		t.Error("SeriesHash should not be zero even with nil tags")
	}
}

