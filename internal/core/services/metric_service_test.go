package services

import (
	"context"
	"testing"
	"time"

	"github.com/forge-platform/forge/internal/core/domain"
	"github.com/forge-platform/forge/internal/core/ports"
)

// mockMetricRepository implements ports.MetricRepository for testing.
type mockMetricRepository struct {
	metrics          []*domain.Metric
	recordBatchCalls int
	queryCalls       int
}

func (m *mockMetricRepository) Record(ctx context.Context, metric *domain.Metric) error {
	m.metrics = append(m.metrics, metric)
	return nil
}

func (m *mockMetricRepository) RecordBatch(ctx context.Context, metrics []*domain.Metric) error {
	m.recordBatchCalls++
	m.metrics = append(m.metrics, metrics...)
	return nil
}

func (m *mockMetricRepository) Query(ctx context.Context, query ports.MetricQuery) (*domain.MetricSeries, error) {
	m.queryCalls++
	points := make([]domain.MetricPoint, len(m.metrics))
	for i, metric := range m.metrics {
		points[i] = domain.MetricPoint{Value: metric.Value, Timestamp: metric.Timestamp}
	}
	return &domain.MetricSeries{
		Name:   query.Name,
		Tags:   query.Tags,
		Points: points,
	}, nil
}

func (m *mockMetricRepository) QueryMultiple(ctx context.Context, query ports.MetricQuery) ([]*domain.MetricSeries, error) {
	series, _ := m.Query(ctx, query)
	return []*domain.MetricSeries{series}, nil
}

func (m *mockMetricRepository) QueryWithAggregation(ctx context.Context, query ports.MetricQuery) ([]ports.AggregatedResult, error) {
	return nil, nil
}

func (m *mockMetricRepository) Aggregate(ctx context.Context, query ports.MetricQuery, resolution string) (*domain.AggregatedMetric, error) {
	return nil, nil
}

func (m *mockMetricRepository) RecordAggregated(ctx context.Context, agg *domain.AggregatedMetric) error {
	return nil
}

func (m *mockMetricRepository) RecordAggregatedBatch(ctx context.Context, metrics []*domain.AggregatedMetric) error {
	return nil
}

func (m *mockMetricRepository) QueryAggregated(ctx context.Context, query ports.MetricQuery, resolution string) ([]*domain.AggregatedMetric, error) {
	return nil, nil
}

func (m *mockMetricRepository) DeleteBefore(ctx context.Context, before time.Time) (int64, error) {
	return 0, nil
}

func (m *mockMetricRepository) DeleteAggregatedBefore(ctx context.Context, before time.Time, resolution string) (int64, error) {
	return 0, nil
}

func (m *mockMetricRepository) GetDistinctSeries(ctx context.Context) ([]ports.SeriesInfo, error) {
	return nil, nil
}

func (m *mockMetricRepository) GetStats(ctx context.Context) (*ports.MetricStats, error) {
	return &ports.MetricStats{TotalPoints: int64(len(m.metrics))}, nil
}

func TestDefaultMetricServiceConfig(t *testing.T) {
	config := DefaultMetricServiceConfig()

	if config.BufferSize != 1000 {
		t.Errorf("BufferSize = %d, want 1000", config.BufferSize)
	}
	if config.FlushInterval != time.Second {
		t.Errorf("FlushInterval = %v, want 1s", config.FlushInterval)
	}
}

func TestNewMetricService(t *testing.T) {
	repo := &mockMetricRepository{}
	logger := &mockLogger{}
	config := DefaultMetricServiceConfig()

	svc := NewMetricService(repo, logger, config)

	if svc == nil {
		t.Fatal("NewMetricService() returned nil")
	}
	if svc.bufferSize != 1000 {
		t.Errorf("bufferSize = %d, want 1000", svc.bufferSize)
	}
}

func TestMetricService_Record(t *testing.T) {
	repo := &mockMetricRepository{}
	logger := &mockLogger{}
	config := MetricServiceConfig{BufferSize: 10, FlushInterval: time.Minute}

	svc := NewMetricService(repo, logger, config)
	ctx := context.Background()

	// Record a metric
	err := svc.Record(ctx, "test.metric", domain.MetricTypeGauge, 42.0, map[string]string{"host": "test"})
	if err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	// Check buffer
	if len(svc.buffer) != 1 {
		t.Errorf("len(buffer) = %d, want 1", len(svc.buffer))
	}
}

func TestMetricService_RecordTriggersFlush(t *testing.T) {
	repo := &mockMetricRepository{}
	logger := &mockLogger{}
	config := MetricServiceConfig{BufferSize: 2, FlushInterval: time.Minute}

	svc := NewMetricService(repo, logger, config)
	ctx := context.Background()

	// Record enough metrics to trigger flush
	_ = svc.Record(ctx, "test.metric", domain.MetricTypeGauge, 1.0, nil)
	_ = svc.Record(ctx, "test.metric", domain.MetricTypeGauge, 2.0, nil)

	// Buffer should be at capacity
	if len(svc.buffer) != 2 {
		t.Errorf("len(buffer) = %d, want 2", len(svc.buffer))
	}
}

func TestMetricService_Query(t *testing.T) {
	repo := &mockMetricRepository{}
	logger := &mockLogger{}
	config := DefaultMetricServiceConfig()

	svc := NewMetricService(repo, logger, config)
	ctx := context.Background()

	query := ports.MetricQuery{Name: "test.metric"}
	series, err := svc.Query(ctx, query)

	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if series == nil {
		t.Fatal("Query() returned nil series")
	}
	if series.Name != "test.metric" {
		t.Errorf("series.Name = %v, want test.metric", series.Name)
	}
}

func TestParseResolution(t *testing.T) {
	tests := []struct {
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"1m", time.Minute, false},
		{"5m", 5 * time.Minute, false},
		{"1h", time.Hour, false},
		{"1d", 24 * time.Hour, false},
		{"invalid", 0, true},
		{"2m", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseResolution(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseResolution(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseResolution(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

