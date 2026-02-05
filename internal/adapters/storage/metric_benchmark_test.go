package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/forge-platform/forge/internal/core/domain"
	"github.com/forge-platform/forge/internal/core/ports"
)

// BenchmarkMetricRecordBatch benchmarks batch metric insertion.
// Target: 100K+ writes/sec
func BenchmarkMetricRecordBatch(b *testing.B) {
	db := setupBenchmarkDB(b)
	defer cleanupBenchmarkDB(b, db)

	repo := NewMetricRepository(db)

	// Prepare batch of metrics
	batchSize := 1000
	metrics := make([]*domain.Metric, batchSize)
	for i := 0; i < batchSize; i++ {
		metrics[i] = domain.NewMetric(
			"benchmark.metric",
			domain.MetricTypeGauge,
			float64(i),
			map[string]string{"host": "localhost", "cpu": fmt.Sprintf("%d", i%8)},
		)
	}

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Create new batch with fresh timestamps
		batch := make([]*domain.Metric, batchSize)
		for j := 0; j < batchSize; j++ {
			batch[j] = domain.NewMetric(
				"benchmark.metric",
				domain.MetricTypeGauge,
				float64(j),
				map[string]string{"host": "localhost", "cpu": fmt.Sprintf("%d", j%8)},
			)
		}

		if err := repo.RecordBatch(ctx, batch); err != nil {
			b.Fatalf("RecordBatch failed: %v", err)
		}
	}

	b.StopTimer()

	// Calculate writes per second
	totalWrites := int64(b.N) * int64(batchSize)
	writesPerSec := float64(totalWrites) / b.Elapsed().Seconds()
	b.ReportMetric(writesPerSec, "writes/sec")
}

// BenchmarkMetricQuery benchmarks metric query performance.
// Target: < 10ms latency for 1M points
func BenchmarkMetricQuery(b *testing.B) {
	db := setupBenchmarkDB(b)
	defer cleanupBenchmarkDB(b, db)

	repo := NewMetricRepository(db)
	ctx := context.Background()

	// Insert test data - 10K metrics for benchmark (1M takes too long in CI)
	numMetrics := 10000
	batchSize := 1000
	for i := 0; i < numMetrics/batchSize; i++ {
		batch := make([]*domain.Metric, batchSize)
		for j := 0; j < batchSize; j++ {
			batch[j] = domain.NewMetric(
				"query.benchmark",
				domain.MetricTypeGauge,
				float64(i*batchSize+j),
				map[string]string{"host": "localhost"},
			)
		}
		if err := repo.RecordBatch(ctx, batch); err != nil {
			b.Fatalf("RecordBatch failed: %v", err)
		}
	}

	query := ports.MetricQuery{
		Name:      "query.benchmark",
		StartTime: time.Now().Add(-24 * time.Hour),
		EndTime:   time.Now().Add(time.Hour),
		Limit:     1000,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := repo.Query(ctx, query)
		if err != nil {
			b.Fatalf("Query failed: %v", err)
		}
	}

	b.StopTimer()

	// Report average latency
	avgLatency := b.Elapsed().Seconds() / float64(b.N) * 1000
	b.ReportMetric(avgLatency, "ms/query")
}

// BenchmarkMetricQueryWithAggregation benchmarks aggregated queries.
func BenchmarkMetricQueryWithAggregation(b *testing.B) {
	db := setupBenchmarkDB(b)
	defer cleanupBenchmarkDB(b, db)

	repo := NewMetricRepository(db)
	ctx := context.Background()

	// Insert 10K metrics
	for i := 0; i < 10; i++ {
		batch := make([]*domain.Metric, 1000)
		for j := 0; j < 1000; j++ {
			batch[j] = domain.NewMetric(
				"agg.benchmark",
				domain.MetricTypeGauge,
				float64(i*1000+j),
				map[string]string{"host": "localhost"},
			)
		}
		_ = repo.RecordBatch(ctx, batch)
	}

	query := ports.MetricQuery{
		Name:        "agg.benchmark",
		StartTime:   time.Now().Add(-24 * time.Hour),
		EndTime:     time.Now().Add(time.Hour),
		Aggregation: ports.AggregationAvg,
		Step:        time.Minute,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := repo.QueryWithAggregation(ctx, query)
		if err != nil {
			b.Fatalf("QueryWithAggregation failed: %v", err)
		}
	}
}

func setupBenchmarkDB(b *testing.B) *DB {
	tmpDir, err := os.MkdirTemp("", "forge-benchmark-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}

	config := Config{
		Path:        filepath.Join(tmpDir, "benchmark.db"),
		JournalMode: "WAL",
		Synchronous: "OFF", // Faster for benchmarks
		CacheSize:   -64000,
		MmapSize:    268435456,
		BusyTimeout: 5000,
	}

	db, err := New(config)
	if err != nil {
		os.RemoveAll(tmpDir)
		b.Fatalf("Failed to create database: %v", err)
	}

	return db
}

func cleanupBenchmarkDB(b *testing.B, db *DB) {
	dbPath := db.config.Path
	db.Close()
	os.RemoveAll(filepath.Dir(dbPath))
}

