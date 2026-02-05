// Package ports defines the interfaces (ports) for the hexagonal architecture.
// These interfaces decouple the domain from infrastructure implementations.
package ports

import (
	"context"
	"time"

	"github.com/forge-platform/forge/internal/core/domain"
	"github.com/google/uuid"
)

// TaskRepository defines the interface for task persistence.
type TaskRepository interface {
	// Create persists a new task.
	Create(ctx context.Context, task *domain.Task) error

	// GetByID retrieves a task by its ID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Task, error)

	// Update updates an existing task.
	Update(ctx context.Context, task *domain.Task) error

	// Delete removes a task.
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves tasks with optional filtering.
	List(ctx context.Context, filter TaskFilter) ([]*domain.Task, error)

	// ClaimNext atomically claims the next pending task for processing.
	ClaimNext(ctx context.Context, lockDuration time.Duration) (*domain.Task, error)

	// ReleaseExpired releases tasks with expired locks.
	ReleaseExpired(ctx context.Context) (int64, error)
}

// TaskFilter defines filtering options for task queries.
type TaskFilter struct {
	Status   *domain.TaskStatus
	Type     *domain.TaskType
	Limit    int
	Offset   int
	OrderBy  string
	OrderDir string
}

// MetricRepository defines the interface for metric persistence.
type MetricRepository interface {
	// Record persists a new metric.
	Record(ctx context.Context, metric *domain.Metric) error

	// RecordBatch persists multiple metrics in a single transaction.
	RecordBatch(ctx context.Context, metrics []*domain.Metric) error

	// Query retrieves metrics matching the given criteria.
	Query(ctx context.Context, query MetricQuery) (*domain.MetricSeries, error)

	// QueryMultiple retrieves multiple series matching the criteria.
	QueryMultiple(ctx context.Context, query MetricQuery) ([]*domain.MetricSeries, error)

	// QueryWithAggregation retrieves metrics with time-bucket aggregation.
	QueryWithAggregation(ctx context.Context, query MetricQuery) ([]AggregatedResult, error)

	// Aggregate performs aggregation on metrics.
	Aggregate(ctx context.Context, query MetricQuery, resolution string) (*domain.AggregatedMetric, error)

	// RecordAggregated persists an aggregated metric (for downsampling).
	RecordAggregated(ctx context.Context, agg *domain.AggregatedMetric) error

	// RecordAggregatedBatch persists multiple aggregated metrics.
	RecordAggregatedBatch(ctx context.Context, aggs []*domain.AggregatedMetric) error

	// QueryAggregated retrieves pre-aggregated metrics.
	QueryAggregated(ctx context.Context, query MetricQuery, resolution string) ([]*domain.AggregatedMetric, error)

	// DeleteBefore removes metrics older than the given timestamp.
	DeleteBefore(ctx context.Context, before time.Time) (int64, error)

	// DeleteAggregatedBefore removes aggregated metrics older than the given timestamp.
	DeleteAggregatedBefore(ctx context.Context, before time.Time, resolution string) (int64, error)

	// GetDistinctSeries returns all distinct series (name + tags combinations).
	GetDistinctSeries(ctx context.Context) ([]SeriesInfo, error)

	// GetStats returns statistics about the metric storage.
	GetStats(ctx context.Context) (*MetricStats, error)
}

// SeriesInfo contains information about a metric series.
type SeriesInfo struct {
	Name       string
	Tags       map[string]string
	SeriesHash uint64
	PointCount int64
	FirstTime  time.Time
	LastTime   time.Time
}

// MetricStats contains statistics about metric storage.
type MetricStats struct {
	TotalPoints      int64
	TotalSeries      int64
	OldestPoint      time.Time
	NewestPoint      time.Time
	StorageBytes     int64
	AggregatedPoints map[string]int64 // resolution -> count
}

// MetricQuery defines query parameters for metric retrieval.
type MetricQuery struct {
	Name       string
	Tags       map[string]string
	SeriesHash *uint64
	StartTime  time.Time
	EndTime    time.Time
	Limit      int

	// Aggregation options
	Aggregation AggregationType
	GroupBy     []string // Tag keys to group by
	Step        time.Duration // Time bucket size for aggregation
}

// AggregationType defines the type of aggregation to perform.
type AggregationType string

const (
	AggregationNone  AggregationType = ""
	AggregationAvg   AggregationType = "avg"
	AggregationSum   AggregationType = "sum"
	AggregationMin   AggregationType = "min"
	AggregationMax   AggregationType = "max"
	AggregationCount AggregationType = "count"
	AggregationLast  AggregationType = "last"
	AggregationFirst AggregationType = "first"
)

// AggregatedResult represents a single aggregated data point.
type AggregatedResult struct {
	Timestamp time.Time
	Value     float64
	Count     int64
	Min       float64
	Max       float64
	Sum       float64
	Avg       float64
}

// PluginRepository defines the interface for plugin persistence.
type PluginRepository interface {
	// Create persists a new plugin.
	Create(ctx context.Context, plugin *domain.Plugin) error

	// GetByID retrieves a plugin by its ID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Plugin, error)

	// GetByName retrieves a plugin by its name.
	GetByName(ctx context.Context, name string) (*domain.Plugin, error)

	// Update updates an existing plugin.
	Update(ctx context.Context, plugin *domain.Plugin) error

	// Delete removes a plugin.
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves all plugins.
	List(ctx context.Context) ([]*domain.Plugin, error)

	// ListActive retrieves all active plugins.
	ListActive(ctx context.Context) ([]*domain.Plugin, error)
}

// ConversationRepository defines the interface for conversation persistence.
type ConversationRepository interface {
	// Create persists a new conversation.
	Create(ctx context.Context, conv *domain.Conversation) error

	// GetByID retrieves a conversation by its ID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Conversation, error)

	// Update updates an existing conversation.
	Update(ctx context.Context, conv *domain.Conversation) error

	// Delete removes a conversation.
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves conversations with pagination.
	List(ctx context.Context, limit, offset int) ([]*domain.Conversation, error)
}

// WorkflowRepository defines the interface for workflow definition persistence.
type WorkflowRepository interface {
	// Create persists a new workflow definition.
	Create(ctx context.Context, workflow *domain.Workflow) error

	// GetByID retrieves a workflow by its ID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Workflow, error)

	// GetByName retrieves a workflow by its name.
	GetByName(ctx context.Context, name string) (*domain.Workflow, error)

	// Update updates an existing workflow.
	Update(ctx context.Context, workflow *domain.Workflow) error

	// Delete removes a workflow.
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves all workflow definitions.
	List(ctx context.Context) ([]*domain.Workflow, error)
}

// WorkflowExecutionRepository defines the interface for workflow execution persistence.
type WorkflowExecutionRepository interface {
	// Create persists a new workflow execution.
	Create(ctx context.Context, execution *domain.WorkflowExecution) error

	// GetByID retrieves an execution by its ID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.WorkflowExecution, error)

	// Update updates an existing execution.
	Update(ctx context.Context, execution *domain.WorkflowExecution) error

	// Delete removes an execution.
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves executions with optional filtering.
	List(ctx context.Context, filter ExecutionFilter) ([]*domain.WorkflowExecution, error)

	// GetLatestByWorkflow retrieves the latest execution for a workflow.
	GetLatestByWorkflow(ctx context.Context, workflowID uuid.UUID) (*domain.WorkflowExecution, error)

	// SaveCheckpoint saves a checkpoint for durable execution.
	SaveCheckpoint(ctx context.Context, executionID uuid.UUID, checkpoint []byte) error

	// LoadCheckpoint loads a checkpoint for resuming execution.
	LoadCheckpoint(ctx context.Context, executionID uuid.UUID) ([]byte, error)
}

// ExecutionFilter defines filtering options for execution queries.
type ExecutionFilter struct {
	WorkflowID   *uuid.UUID
	WorkflowName string
	Status       *domain.WorkflowStatus
	StartedAfter *time.Time
	Limit        int
	Offset       int
}
