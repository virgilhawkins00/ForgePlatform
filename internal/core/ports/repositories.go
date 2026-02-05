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

// AlertRuleRepository defines the interface for alert rule persistence.
type AlertRuleRepository interface {
	// Create persists a new alert rule.
	Create(ctx context.Context, rule *domain.AlertRule) error

	// GetByID retrieves an alert rule by its ID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.AlertRule, error)

	// GetByName retrieves an alert rule by its name.
	GetByName(ctx context.Context, name string) (*domain.AlertRule, error)

	// Update updates an existing alert rule.
	Update(ctx context.Context, rule *domain.AlertRule) error

	// Delete removes an alert rule.
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves all alert rules.
	List(ctx context.Context) ([]*domain.AlertRule, error)

	// ListEnabled retrieves all enabled alert rules.
	ListEnabled(ctx context.Context) ([]*domain.AlertRule, error)

	// ListDue retrieves rules that are due for evaluation.
	ListDue(ctx context.Context, now time.Time) ([]*domain.AlertRule, error)
}

// AlertRepository defines the interface for alert instance persistence.
type AlertRepository interface {
	// Create persists a new alert.
	Create(ctx context.Context, alert *domain.Alert) error

	// GetByID retrieves an alert by its ID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Alert, error)

	// GetByFingerprint retrieves an alert by its fingerprint.
	GetByFingerprint(ctx context.Context, fingerprint string) (*domain.Alert, error)

	// Update updates an existing alert.
	Update(ctx context.Context, alert *domain.Alert) error

	// Delete removes an alert.
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves alerts with optional filtering.
	List(ctx context.Context, filter AlertFilter) ([]*domain.Alert, error)

	// ListActive retrieves all active (firing or pending) alerts.
	ListActive(ctx context.Context) ([]*domain.Alert, error)

	// CountByState returns alert counts grouped by state.
	CountByState(ctx context.Context) (map[domain.AlertState]int64, error)
}

// AlertFilter defines filtering options for alert queries.
type AlertFilter struct {
	RuleID    *uuid.UUID
	State     *domain.AlertState
	Severity  *domain.AlertSeverity
	Labels    map[string]string
	StartTime *time.Time
	EndTime   *time.Time
	Limit     int
	Offset    int
}

// NotificationChannelRepository defines the interface for notification channel persistence.
type NotificationChannelRepository interface {
	// Create persists a new notification channel.
	Create(ctx context.Context, channel *domain.NotificationChannel) error

	// GetByID retrieves a channel by its ID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.NotificationChannel, error)

	// GetByName retrieves a channel by its name.
	GetByName(ctx context.Context, name string) (*domain.NotificationChannel, error)

	// Update updates an existing channel.
	Update(ctx context.Context, channel *domain.NotificationChannel) error

	// Delete removes a channel.
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves all notification channels.
	List(ctx context.Context) ([]*domain.NotificationChannel, error)

	// ListEnabled retrieves all enabled channels.
	ListEnabled(ctx context.Context) ([]*domain.NotificationChannel, error)
}

// SilenceRepository defines the interface for silence persistence.
type SilenceRepository interface {
	// Create persists a new silence.
	Create(ctx context.Context, silence *domain.Silence) error

	// GetByID retrieves a silence by its ID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Silence, error)

	// Update updates an existing silence.
	Update(ctx context.Context, silence *domain.Silence) error

	// Delete removes a silence.
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves all silences.
	List(ctx context.Context) ([]*domain.Silence, error)

	// ListActive retrieves all active silences.
	ListActive(ctx context.Context, now time.Time) ([]*domain.Silence, error)
}

// ============================================================================
// Observability Repositories (Phase 8: v0.8.0)
// ============================================================================

// TraceFilter defines filtering options for trace queries.
type TraceFilter struct {
	ServiceName string
	Name        string
	Status      string
	MinDuration time.Duration
	MaxDuration time.Duration
	StartTime   time.Time
	EndTime     time.Time
	Limit       int
	Offset      int
}

// SpanFilter defines filtering options for span queries.
type SpanFilter struct {
	TraceID     domain.TraceID
	ServiceName string
	Name        string
	Kind        domain.SpanKind
	Status      domain.SpanStatus
	StartTime   time.Time
	EndTime     time.Time
	Limit       int
	Offset      int
}

// TraceRepository defines the interface for trace persistence.
type TraceRepository interface {
	// Create persists a new trace.
	Create(ctx context.Context, trace *domain.Trace) error

	// GetByID retrieves a trace by its UUID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Trace, error)

	// GetByTraceID retrieves a trace by its TraceID.
	GetByTraceID(ctx context.Context, traceID domain.TraceID) (*domain.Trace, error)

	// Update updates an existing trace.
	Update(ctx context.Context, trace *domain.Trace) error

	// Delete removes a trace.
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves traces with optional filtering.
	List(ctx context.Context, filter TraceFilter) ([]*domain.Trace, error)

	// GetServiceMap retrieves the service dependency map.
	GetServiceMap(ctx context.Context, startTime, endTime time.Time) (*domain.ServiceMap, error)

	// DeleteBefore removes traces older than the given timestamp.
	DeleteBefore(ctx context.Context, before time.Time) (int64, error)
}

// SpanRepository defines the interface for span persistence.
type SpanRepository interface {
	// Create persists a new span.
	Create(ctx context.Context, span *domain.Span) error

	// CreateBatch persists multiple spans.
	CreateBatch(ctx context.Context, spans []*domain.Span) error

	// GetByID retrieves a span by its UUID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Span, error)

	// GetBySpanID retrieves a span by its SpanID.
	GetBySpanID(ctx context.Context, traceID domain.TraceID, spanID domain.SpanID) (*domain.Span, error)

	// ListByTraceID retrieves all spans for a trace.
	ListByTraceID(ctx context.Context, traceID domain.TraceID) ([]*domain.Span, error)

	// List retrieves spans with optional filtering.
	List(ctx context.Context, filter SpanFilter) ([]*domain.Span, error)

	// Delete removes a span.
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteByTraceID removes all spans for a trace.
	DeleteByTraceID(ctx context.Context, traceID domain.TraceID) (int64, error)
}

// LogFilter defines filtering options for log queries.
type LogFilter struct {
	Level       domain.LogLevel
	MinLevel    domain.LogLevel
	Source      string
	ServiceName string
	TraceID     string
	Search      string
	Attributes  map[string]string
	StartTime   time.Time
	EndTime     time.Time
	Limit       int
	Offset      int
}

// LogRepository defines the interface for log persistence.
type LogRepository interface {
	// Create persists a new log entry.
	Create(ctx context.Context, entry *domain.LogEntry) error

	// CreateBatch persists multiple log entries.
	CreateBatch(ctx context.Context, entries []*domain.LogEntry) error

	// GetByID retrieves a log entry by its ID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.LogEntry, error)

	// List retrieves log entries with optional filtering.
	List(ctx context.Context, filter LogFilter) ([]*domain.LogEntry, error)

	// Search performs full-text search on log messages.
	Search(ctx context.Context, query string, filter LogFilter) ([]*domain.LogEntry, error)

	// GetStats retrieves log statistics.
	GetStats(ctx context.Context, startTime, endTime time.Time) (*domain.LogStats, error)

	// Delete removes a log entry.
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteBefore removes log entries older than the given timestamp.
	DeleteBefore(ctx context.Context, before time.Time) (int64, error)
}

// LogParserRepository defines the interface for log parser persistence.
type LogParserRepository interface {
	// Create persists a new log parser.
	Create(ctx context.Context, parser *domain.LogParser) error

	// GetByID retrieves a parser by its ID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.LogParser, error)

	// Update updates an existing parser.
	Update(ctx context.Context, parser *domain.LogParser) error

	// Delete removes a parser.
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves all parsers.
	List(ctx context.Context) ([]*domain.LogParser, error)

	// ListEnabled retrieves all enabled parsers ordered by priority.
	ListEnabled(ctx context.Context) ([]*domain.LogParser, error)
}

// LogToMetricRuleRepository defines the interface for log-to-metric rule persistence.
type LogToMetricRuleRepository interface {
	// Create persists a new rule.
	Create(ctx context.Context, rule *domain.LogToMetricRule) error

	// GetByID retrieves a rule by its ID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.LogToMetricRule, error)

	// Update updates an existing rule.
	Update(ctx context.Context, rule *domain.LogToMetricRule) error

	// Delete removes a rule.
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves all rules.
	List(ctx context.Context) ([]*domain.LogToMetricRule, error)

	// ListEnabled retrieves all enabled rules.
	ListEnabled(ctx context.Context) ([]*domain.LogToMetricRule, error)
}

// ProfileFilter defines filtering options for profile queries.
type ProfileFilter struct {
	Type        domain.ProfileType
	Status      domain.ProfileStatus
	ServiceName string
	StartTime   time.Time
	EndTime     time.Time
	Limit       int
	Offset      int
}

// ProfileRepository defines the interface for profile persistence.
type ProfileRepository interface {
	// Create persists a new profile.
	Create(ctx context.Context, profile *domain.Profile) error

	// GetByID retrieves a profile by its ID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Profile, error)

	// Update updates an existing profile.
	Update(ctx context.Context, profile *domain.Profile) error

	// Delete removes a profile.
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves profiles with optional filtering.
	List(ctx context.Context, filter ProfileFilter) ([]*domain.Profile, error)

	// SaveProfileData saves the profile data.
	SaveProfileData(ctx context.Context, data *domain.ProfileData) error

	// GetProfileData retrieves the profile data.
	GetProfileData(ctx context.Context, profileID uuid.UUID) (*domain.ProfileData, error)

	// SaveFlameGraph saves a flame graph.
	SaveFlameGraph(ctx context.Context, fg *domain.FlameGraph) error

	// GetFlameGraph retrieves a flame graph.
	GetFlameGraph(ctx context.Context, profileID uuid.UUID) (*domain.FlameGraph, error)

	// DeleteBefore removes profiles older than the given timestamp.
	DeleteBefore(ctx context.Context, before time.Time) (int64, error)
}
