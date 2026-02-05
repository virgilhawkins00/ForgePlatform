// Package cloud provides cloud provider integrations.
package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/forge-platform/forge/internal/core/domain"
	"github.com/forge-platform/forge/internal/core/ports"
	"google.golang.org/api/option"
	metricpb "google.golang.org/genproto/googleapis/api/metric"
	monitoredrespb "google.golang.org/genproto/googleapis/api/monitoredres"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	config       GCPConfig
	client       *monitoring.MetricClient
	httpClient   *http.Client
	logger       ports.Logger
	metricCh     chan *domain.Metric
	stopCh       chan struct{}
	mu           sync.RWMutex
	metricsCount int64
	errorsCount  int64
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

// NewGCPExporterWithClient creates a new GCP exporter with an initialized client.
func NewGCPExporterWithClient(ctx context.Context, config GCPConfig, logger ports.Logger) (*GCPExporter, error) {
	if config.ProjectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}

	var opts []option.ClientOption

	// Use credentials file if provided, otherwise use ADC
	if config.CredentialsPath != "" {
		if _, err := os.Stat(config.CredentialsPath); err != nil {
			return nil, fmt.Errorf("credentials file not found: %s", config.CredentialsPath)
		}
		opts = append(opts, option.WithCredentialsFile(config.CredentialsPath))
	}
	// If no credentials path, the client will use Application Default Credentials (ADC)

	client, err := monitoring.NewMetricClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create monitoring client: %w", err)
	}

	return &GCPExporter{
		config: config,
		client: client,
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

	e.logger.Debug("Flushing metrics to GCP", "count", len(metrics))

	// If we have a real client, use the Cloud Monitoring API
	if e.client != nil {
		e.flushToAPI(context.Background(), metrics)
		return
	}

	// Fallback: log metrics (for testing without credentials)
	timeSeries := e.metricsToTimeSeries(metrics)
	for _, ts := range timeSeries {
		e.logger.Debug("GCP metric (dry-run)",
			"type", ts.MetricType,
			"value", ts.Value,
			"time", ts.EndTime,
		)
	}
}

// flushToAPI sends metrics to GCP Cloud Monitoring API.
func (e *GCPExporter) flushToAPI(ctx context.Context, metrics []*domain.Metric) {
	// Convert to protobuf time series
	timeSeries := make([]*monitoringpb.TimeSeries, 0, len(metrics))

	for _, m := range metrics {
		ts := &monitoringpb.TimeSeries{
			Metric: &metricpb.Metric{
				Type:   fmt.Sprintf("%s/%s", e.config.MetricPrefix, m.Name),
				Labels: m.Tags,
			},
			Resource: &monitoredrespb.MonitoredResource{
				Type: "global",
				Labels: map[string]string{
					"project_id": e.config.ProjectID,
				},
			},
			Points: []*monitoringpb.Point{{
				Interval: &monitoringpb.TimeInterval{
					EndTime: timestamppb.New(m.Timestamp),
				},
				Value: &monitoringpb.TypedValue{
					Value: &monitoringpb.TypedValue_DoubleValue{
						DoubleValue: m.Value,
					},
				},
			}},
		}
		timeSeries = append(timeSeries, ts)
	}

	// Create the request
	req := &monitoringpb.CreateTimeSeriesRequest{
		Name:       fmt.Sprintf("projects/%s", e.config.ProjectID),
		TimeSeries: timeSeries,
	}

	// Send to Cloud Monitoring
	if err := e.client.CreateTimeSeries(ctx, req); err != nil {
		e.mu.Lock()
		e.errorsCount++
		e.mu.Unlock()
		e.logger.Error("Failed to send metrics to GCP", "error", err, "count", len(metrics))
		return
	}

	e.mu.Lock()
	e.metricsCount += int64(len(metrics))
	e.mu.Unlock()
	e.logger.Debug("Successfully sent metrics to GCP", "count", len(metrics))
}

// Stats returns exporter statistics.
func (e *GCPExporter) Stats() (metricsCount, errorsCount int64) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.metricsCount, e.errorsCount
}

// Close closes the GCP client.
func (e *GCPExporter) Close() error {
	if e.client != nil {
		return e.client.Close()
	}
	return nil
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

