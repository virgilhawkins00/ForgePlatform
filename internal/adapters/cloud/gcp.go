// Package cloud provides cloud provider integrations.
package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/forge-platform/forge/internal/core/domain"
	"github.com/forge-platform/forge/internal/core/ports"
)

// GCPConfig holds GCP Cloud Monitoring configuration.
type GCPConfig struct {
	ProjectID       string        `json:"project_id"`
	CredentialsPath string        `json:"credentials_path,omitempty"`
	Region          string        `json:"region,omitempty"`
	MetricPrefix    string        `json:"metric_prefix"`
	FlushInterval   time.Duration `json:"flush_interval"`
	BatchSize       int           `json:"batch_size"`
}

// DefaultGCPConfig returns default GCP configuration.
func DefaultGCPConfig() GCPConfig {
	return GCPConfig{
		MetricPrefix:  "custom.googleapis.com/forge",
		FlushInterval: 60 * time.Second,
		BatchSize:     200,
	}
}

// GCPExporter exports metrics to GCP Cloud Monitoring.
type GCPExporter struct {
	config     GCPConfig
	httpClient *http.Client
	logger     ports.Logger
	metricCh   chan *domain.Metric
	stopCh     chan struct{}
}

// NewGCPExporter creates a new GCP exporter.
func NewGCPExporter(config GCPConfig, logger ports.Logger) (*GCPExporter, error) {
	if config.ProjectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}

	return &GCPExporter{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:   logger,
		metricCh: make(chan *domain.Metric, 1000),
		stopCh:   make(chan struct{}),
	}, nil
}

// Start starts the exporter.
func (e *GCPExporter) Start(ctx context.Context) error {
	go e.flushLoop(ctx)
	e.logger.Info("GCP exporter started", "project", e.config.ProjectID)
	return nil
}

// Stop stops the exporter.
func (e *GCPExporter) Stop() error {
	close(e.stopCh)
	return nil
}

// Export exports a metric to GCP.
func (e *GCPExporter) Export(metric *domain.Metric) error {
	select {
	case e.metricCh <- metric:
		return nil
	default:
		return fmt.Errorf("metric buffer full")
	}
}

// flushLoop periodically flushes metrics to GCP.
func (e *GCPExporter) flushLoop(ctx context.Context) {
	ticker := time.NewTicker(e.config.FlushInterval)
	defer ticker.Stop()

	var batch []*domain.Metric

	for {
		select {
		case <-ctx.Done():
			e.flush(batch)
			return
		case <-e.stopCh:
			e.flush(batch)
			return
		case m := <-e.metricCh:
			batch = append(batch, m)
			if len(batch) >= e.config.BatchSize {
				e.flush(batch)
				batch = nil
			}
		case <-ticker.C:
			if len(batch) > 0 {
				e.flush(batch)
				batch = nil
			}
		}
	}
}

// flush sends metrics to GCP Cloud Monitoring.
func (e *GCPExporter) flush(metrics []*domain.Metric) {
	if len(metrics) == 0 {
		return
	}

	timeSeries := e.metricsToTimeSeries(metrics)
	e.logger.Debug("Flushing metrics to GCP", "count", len(timeSeries))

	// In production, this would use the Cloud Monitoring API
	// For now, we'll log the metrics
	for _, ts := range timeSeries {
		e.logger.Debug("GCP metric",
			"type", ts.MetricType,
			"value", ts.Value,
			"time", ts.EndTime,
		)
	}
}

// TimeSeries represents a GCP time series point.
type TimeSeries struct {
	MetricType string            `json:"metric_type"`
	Labels     map[string]string `json:"labels"`
	Value      float64           `json:"value"`
	EndTime    time.Time         `json:"end_time"`
}

// metricsToTimeSeries converts Forge metrics to GCP time series.
func (e *GCPExporter) metricsToTimeSeries(metrics []*domain.Metric) []TimeSeries {
	result := make([]TimeSeries, len(metrics))
	for i, m := range metrics {
		result[i] = TimeSeries{
			MetricType: fmt.Sprintf("%s/%s", e.config.MetricPrefix, m.Name),
			Labels:     m.Tags,
			Value:      m.Value,
			EndTime:    m.Timestamp,
		}
	}
	return result
}

// GetConfig returns the current configuration.
func (e *GCPExporter) GetConfig() GCPConfig {
	return e.config
}

// ============================================================================
// GCP Cloud Monitoring API Types (for reference)
// ============================================================================

// CreateTimeSeriesRequest represents a request to create time series.
type CreateTimeSeriesRequest struct {
	TimeSeries []GCPTimeSeries `json:"timeSeries"`
}

// GCPTimeSeries represents a Cloud Monitoring time series.
type GCPTimeSeries struct {
	Metric   GCPMetric   `json:"metric"`
	Resource GCPResource `json:"resource"`
	Points   []GCPPoint  `json:"points"`
}

// GCPMetric represents a metric descriptor.
type GCPMetric struct {
	Type   string            `json:"type"`
	Labels map[string]string `json:"labels"`
}

// GCPResource represents a monitored resource.
type GCPResource struct {
	Type   string            `json:"type"`
	Labels map[string]string `json:"labels"`
}

// GCPPoint represents a single data point.
type GCPPoint struct {
	Interval GCPInterval `json:"interval"`
	Value    GCPValue    `json:"value"`
}

// GCPInterval represents a time interval.
type GCPInterval struct {
	EndTime   string `json:"endTime"`
	StartTime string `json:"startTime,omitempty"`
}

// GCPValue represents a typed value.
type GCPValue struct {
	DoubleValue *float64 `json:"doubleValue,omitempty"`
	Int64Value  *int64   `json:"int64Value,omitempty"`
	BoolValue   *bool    `json:"boolValue,omitempty"`
	StringValue *string  `json:"stringValue,omitempty"`
}

// ToGCPTimeSeries converts metrics to GCP API format.
func (e *GCPExporter) ToGCPTimeSeries(metrics []*domain.Metric) []GCPTimeSeries {
	result := make([]GCPTimeSeries, len(metrics))
	for i, m := range metrics {
		val := m.Value
		result[i] = GCPTimeSeries{
			Metric: GCPMetric{
				Type:   fmt.Sprintf("%s/%s", e.config.MetricPrefix, m.Name),
				Labels: m.Tags,
			},
			Resource: GCPResource{
				Type: "global",
				Labels: map[string]string{
					"project_id": e.config.ProjectID,
				},
			},
			Points: []GCPPoint{{
				Interval: GCPInterval{
					EndTime: m.Timestamp.Format(time.RFC3339Nano),
				},
				Value: GCPValue{
					DoubleValue: &val,
				},
			}},
		}
	}
	return result
}

// MarshalJSON returns the JSON representation for API calls.
func (r *CreateTimeSeriesRequest) MarshalJSON() ([]byte, error) {
	type Alias CreateTimeSeriesRequest
	return json.Marshal((*Alias)(r))
}

