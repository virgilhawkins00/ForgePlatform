package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/forge-platform/forge/internal/core/domain"
	"github.com/forge-platform/forge/internal/core/ports"
	"github.com/google/uuid"
)

// MetricRepository implements ports.MetricRepository using SQLite.
type MetricRepository struct {
	db *DB
}

// NewMetricRepository creates a new metric repository.
func NewMetricRepository(db *DB) *MetricRepository {
	return &MetricRepository{db: db}
}

// Record persists a new metric.
func (r *MetricRepository) Record(ctx context.Context, metric *domain.Metric) error {
	tagsJSON, err := json.Marshal(metric.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	query := `
		INSERT INTO metrics (id, name, type, value, timestamp, series_hash, tags)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	idBytes, _ := metric.ID.MarshalBinary()
	_, err = r.db.conn.ExecContext(ctx, query,
		idBytes,
		metric.Name,
		string(metric.Type),
		metric.Value,
		metric.Timestamp.UnixMilli(),
		hashToInt64(metric.SeriesHash),
		tagsJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to insert metric: %w", err)
	}

	return nil
}

// RecordBatch persists multiple metrics in a single transaction.
func (r *MetricRepository) RecordBatch(ctx context.Context, metrics []*domain.Metric) error {
	tx, err := r.db.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO metrics (id, name, type, value, timestamp, series_hash, tags)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, metric := range metrics {
		tagsJSON, _ := json.Marshal(metric.Tags)
		idBytes, _ := metric.ID.MarshalBinary()

		_, err = stmt.ExecContext(ctx,
			idBytes,
			metric.Name,
			string(metric.Type),
			metric.Value,
			metric.Timestamp.UnixMilli(),
			hashToInt64(metric.SeriesHash),
			tagsJSON,
		)
		if err != nil {
			return fmt.Errorf("failed to insert metric: %w", err)
		}
	}

	return tx.Commit()
}

// Query retrieves metrics matching the given criteria.
func (r *MetricRepository) Query(ctx context.Context, query ports.MetricQuery) (*domain.MetricSeries, error) {
	sqlQuery := `
		SELECT id, name, type, value, timestamp, series_hash, tags
		FROM metrics
		WHERE name = ? AND timestamp >= ? AND timestamp <= ?
	`
	args := []interface{}{query.Name, query.StartTime.UnixMilli(), query.EndTime.UnixMilli()}

	if query.SeriesHash != nil {
		sqlQuery += " AND series_hash = ?"
		args = append(args, hashToInt64(*query.SeriesHash))
	}

	sqlQuery += " ORDER BY timestamp ASC"

	if query.Limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT %d", query.Limit)
	}

	rows, err := r.db.conn.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query metrics: %w", err)
	}
	defer rows.Close()

	series := &domain.MetricSeries{
		Name:   query.Name,
		Tags:   query.Tags,
		Points: []domain.MetricPoint{},
	}

	for rows.Next() {
		var (
			idBytes    []byte
			name       string
			metricType string
			value      float64
			timestamp  int64
			seriesHash int64
			tagsJSON   []byte
		)

		if err := rows.Scan(&idBytes, &name, &metricType, &value, &timestamp, &seriesHash, &tagsJSON); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		series.SeriesHash = int64ToHash(seriesHash)
		series.Points = append(series.Points, domain.MetricPoint{
			Value:     value,
			Timestamp: time.UnixMilli(timestamp),
		})

		if series.Tags == nil && len(tagsJSON) > 0 {
			json.Unmarshal(tagsJSON, &series.Tags)
		}
	}

	return series, nil
}

// QueryMultiple retrieves multiple series matching the criteria.
func (r *MetricRepository) QueryMultiple(ctx context.Context, query ports.MetricQuery) ([]*domain.MetricSeries, error) {
	// For now, delegate to Query - can be optimized later
	series, err := r.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	return []*domain.MetricSeries{series}, nil
}

// QueryWithAggregation retrieves metrics with time-bucket aggregation.
func (r *MetricRepository) QueryWithAggregation(ctx context.Context, query ports.MetricQuery) ([]ports.AggregatedResult, error) {
	if query.Step == 0 {
		return nil, fmt.Errorf("step duration is required for aggregation")
	}

	stepMs := query.Step.Milliseconds()

	// Build aggregation SQL based on type
	var aggExpr string
	switch query.Aggregation {
	case ports.AggregationAvg:
		aggExpr = "AVG(value)"
	case ports.AggregationSum:
		aggExpr = "SUM(value)"
	case ports.AggregationMin:
		aggExpr = "MIN(value)"
	case ports.AggregationMax:
		aggExpr = "MAX(value)"
	case ports.AggregationCount:
		aggExpr = "COUNT(*)"
	case ports.AggregationLast:
		aggExpr = "value" // Will use ORDER BY timestamp DESC LIMIT 1 per bucket
	case ports.AggregationFirst:
		aggExpr = "value" // Will use ORDER BY timestamp ASC LIMIT 1 per bucket
	default:
		aggExpr = "AVG(value)"
	}

	sqlQuery := fmt.Sprintf(`
		SELECT
			(timestamp / %d) * %d as bucket,
			%s as agg_value,
			COUNT(*) as cnt,
			MIN(value) as min_val,
			MAX(value) as max_val,
			SUM(value) as sum_val,
			AVG(value) as avg_val
		FROM metrics
		WHERE name = ? AND timestamp >= ? AND timestamp <= ?
	`, stepMs, stepMs, aggExpr)

	args := []interface{}{query.Name, query.StartTime.UnixMilli(), query.EndTime.UnixMilli()}

	if query.SeriesHash != nil {
		sqlQuery += " AND series_hash = ?"
		args = append(args, hashToInt64(*query.SeriesHash))
	}

	sqlQuery += fmt.Sprintf(" GROUP BY bucket ORDER BY bucket ASC")

	if query.Limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT %d", query.Limit)
	}

	rows, err := r.db.conn.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query aggregated metrics: %w", err)
	}
	defer rows.Close()

	var results []ports.AggregatedResult
	for rows.Next() {
		var (
			bucket   int64
			aggValue float64
			count    int64
			minVal   float64
			maxVal   float64
			sumVal   float64
			avgVal   float64
		)

		if err := rows.Scan(&bucket, &aggValue, &count, &minVal, &maxVal, &sumVal, &avgVal); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		results = append(results, ports.AggregatedResult{
			Timestamp: time.UnixMilli(bucket),
			Value:     aggValue,
			Count:     count,
			Min:       minVal,
			Max:       maxVal,
			Sum:       sumVal,
			Avg:       avgVal,
		})
	}

	return results, nil
}

// Aggregate performs aggregation on metrics.
func (r *MetricRepository) Aggregate(ctx context.Context, query ports.MetricQuery, resolution string) (*domain.AggregatedMetric, error) {
	sqlQuery := `
		SELECT
			COUNT(*) as cnt,
			SUM(value) as sum_val,
			MIN(value) as min_val,
			MAX(value) as max_val,
			AVG(value) as avg_val,
			MIN(timestamp) as first_ts,
			MAX(timestamp) as last_ts
		FROM metrics
		WHERE name = ? AND timestamp >= ? AND timestamp <= ?
	`
	args := []interface{}{query.Name, query.StartTime.UnixMilli(), query.EndTime.UnixMilli()}

	if query.SeriesHash != nil {
		sqlQuery += " AND series_hash = ?"
		args = append(args, hashToInt64(*query.SeriesHash))
	}

	var (
		count   int64
		sum     float64
		min     float64
		max     float64
		avg     float64
		firstTs int64
		lastTs  int64
	)

	err := r.db.conn.QueryRowContext(ctx, sqlQuery, args...).Scan(&count, &sum, &min, &max, &avg, &firstTs, &lastTs)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate metrics: %w", err)
	}

	if count == 0 {
		return nil, nil
	}

	return &domain.AggregatedMetric{
		ID:          uuid.Must(uuid.NewV7()),
		Name:        query.Name,
		Tags:        query.Tags,
		SeriesHash:  0, // Will be computed if needed
		WindowStart: time.UnixMilli(firstTs),
		WindowEnd:   time.UnixMilli(lastTs),
		Count:       count,
		Sum:         sum,
		Min:         min,
		Max:         max,
		Avg:         avg,
		Resolution:  resolution,
	}, nil
}

// DeleteBefore removes metrics older than the given timestamp.
func (r *MetricRepository) DeleteBefore(ctx context.Context, before time.Time) (int64, error) {
	result, err := r.db.conn.ExecContext(ctx,
		"DELETE FROM metrics WHERE timestamp < ?",
		before.UnixMilli(),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to delete metrics: %w", err)
	}
	return result.RowsAffected()
}

// RecordAggregated persists an aggregated metric.
func (r *MetricRepository) RecordAggregated(ctx context.Context, agg *domain.AggregatedMetric) error {
	tagsJSON, err := json.Marshal(agg.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	query := `
		INSERT INTO metrics_aggregated (id, name, series_hash, window_start, window_end, resolution, count, sum, min, max, avg, tags)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	idBytes, _ := agg.ID.MarshalBinary()
	_, err = r.db.conn.ExecContext(ctx, query,
		idBytes,
		agg.Name,
		hashToInt64(agg.SeriesHash),
		agg.WindowStart.UnixMilli(),
		agg.WindowEnd.UnixMilli(),
		agg.Resolution,
		agg.Count,
		agg.Sum,
		agg.Min,
		agg.Max,
		agg.Avg,
		tagsJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to insert aggregated metric: %w", err)
	}

	return nil
}

// RecordAggregatedBatch persists multiple aggregated metrics.
func (r *MetricRepository) RecordAggregatedBatch(ctx context.Context, aggs []*domain.AggregatedMetric) error {
	tx, err := r.db.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO metrics_aggregated (id, name, series_hash, window_start, window_end, resolution, count, sum, min, max, avg, tags)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, agg := range aggs {
		tagsJSON, _ := json.Marshal(agg.Tags)
		idBytes, _ := agg.ID.MarshalBinary()

		_, err = stmt.ExecContext(ctx,
			idBytes,
			agg.Name,
			hashToInt64(agg.SeriesHash),
			agg.WindowStart.UnixMilli(),
			agg.WindowEnd.UnixMilli(),
			agg.Resolution,
			agg.Count,
			agg.Sum,
			agg.Min,
			agg.Max,
			agg.Avg,
			tagsJSON,
		)
		if err != nil {
			return fmt.Errorf("failed to insert aggregated metric: %w", err)
		}
	}

	return tx.Commit()
}

// QueryAggregated retrieves pre-aggregated metrics.
func (r *MetricRepository) QueryAggregated(ctx context.Context, query ports.MetricQuery, resolution string) ([]*domain.AggregatedMetric, error) {
	sqlQuery := `
		SELECT id, name, series_hash, window_start, window_end, resolution, count, sum, min, max, avg, tags
		FROM metrics_aggregated
		WHERE name = ? AND resolution = ? AND window_start >= ? AND window_end <= ?
	`
	args := []interface{}{query.Name, resolution, query.StartTime.UnixMilli(), query.EndTime.UnixMilli()}

	if query.SeriesHash != nil {
		sqlQuery += " AND series_hash = ?"
		args = append(args, hashToInt64(*query.SeriesHash))
	}

	sqlQuery += " ORDER BY window_start ASC"

	if query.Limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT %d", query.Limit)
	}

	rows, err := r.db.conn.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query aggregated metrics: %w", err)
	}
	defer rows.Close()

	var results []*domain.AggregatedMetric
	for rows.Next() {
		var (
			idBytes     []byte
			name        string
			seriesHash  int64
			windowStart int64
			windowEnd   int64
			res         string
			count       int64
			sum         float64
			min         float64
			max         float64
			avg         float64
			tagsJSON    []byte
		)

		if err := rows.Scan(&idBytes, &name, &seriesHash, &windowStart, &windowEnd, &res, &count, &sum, &min, &max, &avg, &tagsJSON); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		agg := &domain.AggregatedMetric{
			ID:          uuidFromBytes(idBytes),
			Name:        name,
			SeriesHash:  int64ToHash(seriesHash),
			WindowStart: time.UnixMilli(windowStart),
			WindowEnd:   time.UnixMilli(windowEnd),
			Resolution:  res,
			Count:       count,
			Sum:         sum,
			Min:         min,
			Max:         max,
			Avg:         avg,
		}

		if len(tagsJSON) > 0 {
			json.Unmarshal(tagsJSON, &agg.Tags)
		}

		results = append(results, agg)
	}

	return results, nil
}

// DeleteAggregatedBefore removes aggregated metrics older than the given timestamp.
func (r *MetricRepository) DeleteAggregatedBefore(ctx context.Context, before time.Time, resolution string) (int64, error) {
	result, err := r.db.conn.ExecContext(ctx,
		"DELETE FROM metrics_aggregated WHERE window_end < ? AND resolution = ?",
		before.UnixMilli(),
		resolution,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to delete aggregated metrics: %w", err)
	}
	return result.RowsAffected()
}

// GetDistinctSeries returns all distinct series.
func (r *MetricRepository) GetDistinctSeries(ctx context.Context) ([]ports.SeriesInfo, error) {
	sqlQuery := `
		SELECT
			name,
			series_hash,
			tags,
			COUNT(*) as point_count,
			MIN(timestamp) as first_time,
			MAX(timestamp) as last_time
		FROM metrics
		GROUP BY series_hash
		ORDER BY name, series_hash
	`

	rows, err := r.db.conn.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query distinct series: %w", err)
	}
	defer rows.Close()

	var results []ports.SeriesInfo
	for rows.Next() {
		var (
			name       string
			seriesHash int64
			tagsJSON   []byte
			pointCount int64
			firstTime  int64
			lastTime   int64
		)

		if err := rows.Scan(&name, &seriesHash, &tagsJSON, &pointCount, &firstTime, &lastTime); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		info := ports.SeriesInfo{
			Name:       name,
			SeriesHash: int64ToHash(seriesHash),
			PointCount: pointCount,
			FirstTime:  time.UnixMilli(firstTime),
			LastTime:   time.UnixMilli(lastTime),
		}

		if len(tagsJSON) > 0 {
			json.Unmarshal(tagsJSON, &info.Tags)
		}

		results = append(results, info)
	}

	return results, nil
}

// GetStats returns statistics about the metric storage.
func (r *MetricRepository) GetStats(ctx context.Context) (*ports.MetricStats, error) {
	stats := &ports.MetricStats{
		AggregatedPoints: make(map[string]int64),
	}

	// Get raw metrics stats
	err := r.db.conn.QueryRowContext(ctx, `
		SELECT COUNT(*), COUNT(DISTINCT series_hash), MIN(timestamp), MAX(timestamp)
		FROM metrics
	`).Scan(&stats.TotalPoints, &stats.TotalSeries, &stats.OldestPoint, &stats.NewestPoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics stats: %w", err)
	}

	// Get aggregated metrics stats by resolution
	rows, err := r.db.conn.QueryContext(ctx, `
		SELECT resolution, COUNT(*) FROM metrics_aggregated GROUP BY resolution
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get aggregated stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var resolution string
		var count int64
		if err := rows.Scan(&resolution, &count); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		stats.AggregatedPoints[resolution] = count
	}

	// Get storage size
	var pageCount, pageSize int64
	r.db.conn.QueryRowContext(ctx, "PRAGMA page_count").Scan(&pageCount)
	r.db.conn.QueryRowContext(ctx, "PRAGMA page_size").Scan(&pageSize)
	stats.StorageBytes = pageCount * pageSize

	return stats, nil
}

// Ensure MetricRepository implements the interface
var _ ports.MetricRepository = (*MetricRepository)(nil)

// Helper to convert UUID bytes
func uuidFromBytes(b []byte) uuid.UUID {
	var id uuid.UUID
	copy(id[:], b)
	return id
}

// hashToInt64 converts uint64 to int64 for SQLite compatibility.
// SQLite doesn't support uint64 with high bit set.
func hashToInt64(h uint64) int64 {
	return int64(h)
}

// int64ToHash converts int64 back to uint64.
func int64ToHash(i int64) uint64 {
	return uint64(i)
}

