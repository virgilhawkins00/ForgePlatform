package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/forge-platform/forge/internal/core/domain"
	"github.com/forge-platform/forge/internal/core/ports"
)

// MetricService handles metric operations.
type MetricService struct {
	repo   ports.MetricRepository
	logger ports.Logger

	// Buffering for batch writes
	buffer     []*domain.Metric
	bufferMu   sync.Mutex
	bufferSize int
	flushCh    chan struct{}
	stopCh     chan struct{}
}

// MetricServiceConfig holds configuration for the metric service.
type MetricServiceConfig struct {
	BufferSize    int
	FlushInterval time.Duration
}

// DefaultMetricServiceConfig returns the default configuration.
func DefaultMetricServiceConfig() MetricServiceConfig {
	return MetricServiceConfig{
		BufferSize:    1000,
		FlushInterval: time.Second,
	}
}

// NewMetricService creates a new metric service.
func NewMetricService(repo ports.MetricRepository, logger ports.Logger, config MetricServiceConfig) *MetricService {
	return &MetricService{
		repo:       repo,
		logger:     logger,
		buffer:     make([]*domain.Metric, 0, config.BufferSize),
		bufferSize: config.BufferSize,
		flushCh:    make(chan struct{}, 1),
		stopCh:     make(chan struct{}),
	}
}

// Record records a new metric.
func (s *MetricService) Record(ctx context.Context, name string, metricType domain.MetricType, value float64, tags map[string]string) error {
	metric := domain.NewMetric(name, metricType, value, tags)

	s.bufferMu.Lock()
	s.buffer = append(s.buffer, metric)
	shouldFlush := len(s.buffer) >= s.bufferSize
	s.bufferMu.Unlock()

	if shouldFlush {
		select {
		case s.flushCh <- struct{}{}:
		default:
		}
	}

	return nil
}

// Query retrieves metrics matching the given criteria.
func (s *MetricService) Query(ctx context.Context, query ports.MetricQuery) (*domain.MetricSeries, error) {
	// Flush buffer first to ensure we have latest data
	s.flush(ctx)

	return s.repo.Query(ctx, query)
}

// QueryRange retrieves metrics for a time range.
func (s *MetricService) QueryRange(ctx context.Context, name string, start, end time.Time, tags map[string]string) (*domain.MetricSeries, error) {
	query := ports.MetricQuery{
		Name:      name,
		Tags:      tags,
		StartTime: start,
		EndTime:   end,
	}
	return s.Query(ctx, query)
}

// Start starts the background flusher.
func (s *MetricService) Start(ctx context.Context, flushInterval time.Duration) {
	go s.flusher(ctx, flushInterval)
}

// Stop stops the metric service and flushes remaining data.
func (s *MetricService) Stop(ctx context.Context) {
	close(s.stopCh)
	s.flush(ctx)
}

// flusher periodically flushes the buffer to the database.
func (s *MetricService) flusher(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.flush(ctx)
		case <-s.flushCh:
			s.flush(ctx)
		}
	}
}

// flush writes buffered metrics to the database.
func (s *MetricService) flush(ctx context.Context) {
	s.bufferMu.Lock()
	if len(s.buffer) == 0 {
		s.bufferMu.Unlock()
		return
	}

	metrics := s.buffer
	s.buffer = make([]*domain.Metric, 0, s.bufferSize)
	s.bufferMu.Unlock()

	if err := s.repo.RecordBatch(ctx, metrics); err != nil {
		s.logger.Error("Failed to flush metrics", "count", len(metrics), "error", err)
		// Re-add to buffer on failure
		s.bufferMu.Lock()
		s.buffer = append(metrics, s.buffer...)
		s.bufferMu.Unlock()
	} else {
		s.logger.Debug("Flushed metrics", "count", len(metrics))
	}
}

// Downsample aggregates old metrics into lower resolution.
// Resolution can be "1m", "1h", or "1d".
// Retention policies (from ForgePlatform.md):
// - Raw data: 7 days
// - 1-minute aggregates: 30 days
// - 1-hour aggregates: 1 year
func (s *MetricService) Downsample(ctx context.Context, olderThan time.Duration, resolution string) error {
	s.logger.Info("Starting downsampling", "older_than", olderThan, "resolution", resolution)

	// Flush buffer first to ensure we have all data
	s.flush(ctx)

	// Parse resolution to get step duration
	step, err := parseResolution(resolution)
	if err != nil {
		return fmt.Errorf("invalid resolution: %w", err)
	}

	// Get all distinct series
	series, err := s.repo.GetDistinctSeries(ctx)
	if err != nil {
		return fmt.Errorf("failed to get distinct series: %w", err)
	}

	threshold := time.Now().Add(-olderThan)
	totalAggregated := 0
	totalDeleted := int64(0)

	for _, seriesInfo := range series {
		// Skip series with no data older than threshold
		if seriesInfo.LastTime.After(threshold) && seriesInfo.FirstTime.After(threshold) {
			continue
		}

		// Query with aggregation for this series
		endTime := threshold
		if seriesInfo.LastTime.Before(threshold) {
			endTime = seriesInfo.LastTime
		}

		query := ports.MetricQuery{
			Name:        seriesInfo.Name,
			SeriesHash:  &seriesInfo.SeriesHash,
			StartTime:   seriesInfo.FirstTime,
			EndTime:     endTime,
			Aggregation: ports.AggregationAvg,
			Step:        step,
		}

		results, err := s.repo.QueryWithAggregation(ctx, query)
		if err != nil {
			s.logger.Error("Failed to aggregate series", "series", seriesInfo.Name, "error", err)
			continue
		}

		if len(results) == 0 {
			continue
		}

		// Convert aggregated results to domain objects
		var aggregatedMetrics []*domain.AggregatedMetric
		for _, result := range results {
			agg := &domain.AggregatedMetric{
				ID:          domain.NewUUIDv7(),
				Name:        seriesInfo.Name,
				Tags:        seriesInfo.Tags,
				SeriesHash:  seriesInfo.SeriesHash,
				WindowStart: result.Timestamp,
				WindowEnd:   result.Timestamp.Add(step),
				Count:       result.Count,
				Sum:         result.Sum,
				Min:         result.Min,
				Max:         result.Max,
				Avg:         result.Avg,
				Resolution:  resolution,
			}
			aggregatedMetrics = append(aggregatedMetrics, agg)
		}

		// Batch insert aggregated metrics
		if err := s.repo.RecordAggregatedBatch(ctx, aggregatedMetrics); err != nil {
			s.logger.Error("Failed to record aggregated metrics", "series", seriesInfo.Name, "error", err)
			continue
		}

		totalAggregated += len(aggregatedMetrics)
	}

	// Delete raw metrics older than threshold
	deleted, err := s.repo.DeleteBefore(ctx, threshold)
	if err != nil {
		return fmt.Errorf("failed to delete old metrics: %w", err)
	}
	totalDeleted = deleted

	s.logger.Info("Downsampling complete",
		"resolution", resolution,
		"aggregated_buckets", totalAggregated,
		"deleted_raw", totalDeleted,
	)

	return nil
}

// parseResolution converts resolution string to duration.
func parseResolution(resolution string) (time.Duration, error) {
	switch resolution {
	case "1m":
		return time.Minute, nil
	case "5m":
		return 5 * time.Minute, nil
	case "1h":
		return time.Hour, nil
	case "1d":
		return 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported resolution: %s (use 1m, 5m, 1h, or 1d)", resolution)
	}
}

// QueryWithAggregation queries metrics with time-bucket aggregation.
func (s *MetricService) QueryWithAggregation(ctx context.Context, query ports.MetricQuery) ([]ports.AggregatedResult, error) {
	// Flush buffer first
	s.flush(ctx)
	return s.repo.QueryWithAggregation(ctx, query)
}

// QueryAggregated retrieves pre-aggregated metrics.
func (s *MetricService) QueryAggregated(ctx context.Context, query ports.MetricQuery, resolution string) ([]*domain.AggregatedMetric, error) {
	return s.repo.QueryAggregated(ctx, query, resolution)
}

// GetStats returns storage statistics.
func (s *MetricService) GetStats(ctx context.Context) (*ports.MetricStats, error) {
	return s.repo.GetStats(ctx)
}

// GetDistinctSeries returns all distinct metric series.
func (s *MetricService) GetDistinctSeries(ctx context.Context) ([]ports.SeriesInfo, error) {
	return s.repo.GetDistinctSeries(ctx)
}

// CleanupAggregated removes old aggregated metrics based on retention policy.
func (s *MetricService) CleanupAggregated(ctx context.Context) error {
	// Retention policies from ForgePlatform.md:
	// - 1-minute aggregates: 30 days
	// - 1-hour aggregates: 1 year
	// - 1-day aggregates: forever (no cleanup)

	retentionPolicies := map[string]time.Duration{
		"1m": 30 * 24 * time.Hour,      // 30 days
		"5m": 60 * 24 * time.Hour,      // 60 days
		"1h": 365 * 24 * time.Hour,     // 1 year
	}

	for resolution, retention := range retentionPolicies {
		before := time.Now().Add(-retention)
		deleted, err := s.repo.DeleteAggregatedBefore(ctx, before, resolution)
		if err != nil {
			s.logger.Error("Failed to cleanup aggregated metrics", "resolution", resolution, "error", err)
			continue
		}
		if deleted > 0 {
			s.logger.Info("Cleaned up aggregated metrics", "resolution", resolution, "deleted", deleted)
		}
	}

	return nil
}

// Cleanup removes metrics older than the retention period.
func (s *MetricService) Cleanup(ctx context.Context, retention time.Duration) (int64, error) {
	before := time.Now().Add(-retention)
	deleted, err := s.repo.DeleteBefore(ctx, before)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup metrics: %w", err)
	}

	s.logger.Info("Cleaned up old metrics", "deleted", deleted, "before", before)
	return deleted, nil
}

