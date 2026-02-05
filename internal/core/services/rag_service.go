// Package services provides the core business logic services.
package services

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/forge-platform/forge/internal/core/domain"
	"github.com/forge-platform/forge/internal/core/ports"
)

// RAGService provides Retrieval-Augmented Generation capabilities.
type RAGService struct {
	metricRepo ports.MetricRepository
	taskRepo   ports.TaskRepository
	logger     ports.Logger
	maxContext int // Maximum context window size in tokens (approximate)
}

// RAGConfig configures the RAG service.
type RAGConfig struct {
	MaxContextTokens int
}

// NewRAGService creates a new RAG service.
func NewRAGService(metricRepo ports.MetricRepository, taskRepo ports.TaskRepository, logger ports.Logger, cfg RAGConfig) *RAGService {
	if cfg.MaxContextTokens == 0 {
		cfg.MaxContextTokens = 4096
	}
	return &RAGService{
		metricRepo: metricRepo,
		taskRepo:   taskRepo,
		logger:     logger,
		maxContext: cfg.MaxContextTokens,
	}
}

// ContextRequest specifies what context to retrieve.
type ContextRequest struct {
	TimeRange     time.Duration
	MetricNames   []string
	IncludeMetrics bool
	IncludeTasks   bool
	IncludeLogs    bool
	Query         string // Natural language query for relevance filtering
}

// ContextResult contains the retrieved context.
type ContextResult struct {
	Metrics      []MetricSummary
	Tasks        []TaskSummary
	Logs         []LogEntry
	SystemPrompt string
	TokenCount   int
}

// MetricSummary summarizes metric data for context.
type MetricSummary struct {
	Name      string
	Tags      map[string]string
	Latest    float64
	Min       float64
	Max       float64
	Avg       float64
	Count     int
	Trend     string // "increasing", "decreasing", "stable"
	Anomalies []string
}

// TaskSummary summarizes task data for context.
type TaskSummary struct {
	ID        string
	Type      string
	Status    string
	CreatedAt time.Time
	Error     string
}

// LogEntry represents a log entry for context.
type LogEntry struct {
	Timestamp time.Time
	Level     string
	Message   string
	Source    string
}

// BuildContext retrieves and formats context for AI consumption.
func (s *RAGService) BuildContext(ctx context.Context, req ContextRequest) (*ContextResult, error) {
	result := &ContextResult{}
	var contextParts []string

	now := time.Now()
	startTime := now.Add(-req.TimeRange)

	// Add temporal awareness
	contextParts = append(contextParts, fmt.Sprintf(
		"Current time: %s\nAnalyzing data from %s to %s (%s window)",
		now.Format(time.RFC3339),
		startTime.Format(time.RFC3339),
		now.Format(time.RFC3339),
		req.TimeRange.String(),
	))

	// Retrieve metrics if requested
	if req.IncludeMetrics {
		metrics, err := s.retrieveMetrics(ctx, startTime, req.MetricNames)
		if err != nil {
			s.logger.Warn("Failed to retrieve metrics", "error", err)
		} else {
			result.Metrics = metrics
			contextParts = append(contextParts, s.formatMetricsContext(metrics))
		}
	}

	// Retrieve tasks if requested
	if req.IncludeTasks {
		tasks, err := s.retrieveTasks(ctx, startTime)
		if err != nil {
			s.logger.Warn("Failed to retrieve tasks", "error", err)
		} else {
			result.Tasks = tasks
			contextParts = append(contextParts, s.formatTasksContext(tasks))
		}
	}

	// Build system prompt
	result.SystemPrompt = s.buildSystemPrompt(contextParts)
	result.TokenCount = s.estimateTokens(result.SystemPrompt)

	return result, nil
}

// retrieveMetrics fetches and summarizes metrics.
func (s *RAGService) retrieveMetrics(ctx context.Context, since time.Time, names []string) ([]MetricSummary, error) {
	query := ports.MetricQuery{
		StartTime: since,
		EndTime:   time.Now(),
		Limit:     1000,
	}

	seriesList, err := s.metricRepo.QueryMultiple(ctx, query)
	if err != nil {
		return nil, err
	}

	var summaries []MetricSummary
	for _, series := range seriesList {
		if series == nil || len(series.Points) == 0 {
			continue
		}
		summary := s.summarizeMetricSeries(series)
		summaries = append(summaries, summary)
	}

	// Sort by name for consistent output
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Name < summaries[j].Name
	})

	return summaries, nil
}

// retrieveTasks fetches and summarizes tasks.
func (s *RAGService) retrieveTasks(ctx context.Context, since time.Time) ([]TaskSummary, error) {
	// Get all tasks, we'll filter by time
	tasks, err := s.taskRepo.List(ctx, ports.TaskFilter{
		Limit:    100,
		OrderBy:  "created_at",
		OrderDir: "desc",
	})
	if err != nil {
		return nil, err
	}

	var summaries []TaskSummary
	for _, t := range tasks {
		if t.CreatedAt.Before(since) {
			continue
		}
		summaries = append(summaries, TaskSummary{
			ID:        t.ID.String(),
			Type:      string(t.Type),
			Status:    string(t.Status),
			CreatedAt: t.CreatedAt,
			Error:     t.Error,
		})
	}

	return summaries, nil
}

// summarizeMetricSeries computes statistics for a metric series.
func (s *RAGService) summarizeMetricSeries(series *domain.MetricSeries) MetricSummary {
	if series == nil || len(series.Points) == 0 {
		return MetricSummary{}
	}

	points := series.Points
	summary := MetricSummary{
		Name:   series.Name,
		Tags:   series.Tags,
		Count:  len(points),
		Min:    points[0].Value,
		Max:    points[0].Value,
		Latest: points[len(points)-1].Value,
	}

	// Calculate min, max, sum for avg
	var sum float64
	for _, p := range points {
		sum += p.Value
		if p.Value < summary.Min {
			summary.Min = p.Value
		}
		if p.Value > summary.Max {
			summary.Max = p.Value
		}
	}
	summary.Avg = sum / float64(len(points))

	// Detect trend using simple linear comparison
	summary.Trend = s.detectTrendFromPoints(points)

	// Detect anomalies (values > 2 standard deviations from mean)
	summary.Anomalies = s.detectAnomaliesFromPoints(points, summary.Avg)

	return summary
}

// detectTrendFromPoints determines if the metric is increasing, decreasing, or stable.
func (s *RAGService) detectTrendFromPoints(points []domain.MetricPoint) string {
	if len(points) < 2 {
		return "stable"
	}

	// Compare first third with last third
	n := len(points)
	third := n / 3
	if third < 1 {
		third = 1
	}

	var firstAvg, lastAvg float64
	for i := 0; i < third; i++ {
		firstAvg += points[i].Value
	}
	firstAvg /= float64(third)

	for i := n - third; i < n; i++ {
		lastAvg += points[i].Value
	}
	lastAvg /= float64(third)

	// 10% threshold for trend detection
	threshold := firstAvg * 0.1
	if threshold < 0 {
		threshold = -threshold
	}
	if threshold < 0.01 {
		threshold = 0.01
	}

	diff := lastAvg - firstAvg
	if diff > threshold {
		return "increasing"
	} else if diff < -threshold {
		return "decreasing"
	}
	return "stable"
}

// detectAnomaliesFromPoints finds values that deviate significantly from the mean.
func (s *RAGService) detectAnomaliesFromPoints(points []domain.MetricPoint, mean float64) []string {
	if len(points) < 3 {
		return nil
	}

	// Calculate variance
	var sumSquares float64
	for _, p := range points {
		diff := p.Value - mean
		sumSquares += diff * diff
	}
	variance := sumSquares / float64(len(points))
	// Threshold for anomaly detection: 2 std deviations (4 * variance for squared comparison)
	threshold := 4 * variance
	if threshold < 0.0001 {
		return nil // No significant variation
	}

	var anomalies []string
	for _, p := range points {
		diff := p.Value - mean
		if diff*diff > threshold {
			anomalies = append(anomalies, fmt.Sprintf(
				"%s: %.2f (mean: %.2f)",
				p.Timestamp.Format("15:04:05"),
				p.Value,
				mean,
			))
		}
	}

	// Limit anomalies to avoid overwhelming context
	if len(anomalies) > 5 {
		extra := len(anomalies) - 5
		anomalies = anomalies[:5]
		anomalies = append(anomalies, fmt.Sprintf("... and %d more", extra))
	}

	return anomalies
}

// formatMetricsContext formats metrics for LLM consumption.
func (s *RAGService) formatMetricsContext(metrics []MetricSummary) string {
	if len(metrics) == 0 {
		return "No metrics data available."
	}

	var sb strings.Builder
	sb.WriteString("## System Metrics\n\n")

	for _, m := range metrics {
		sb.WriteString(fmt.Sprintf("### %s", m.Name))
		if len(m.Tags) > 0 {
			tags := make([]string, 0, len(m.Tags))
			for k, v := range m.Tags {
				tags = append(tags, fmt.Sprintf("%s=%s", k, v))
			}
			sb.WriteString(fmt.Sprintf(" {%s}", strings.Join(tags, ", ")))
		}
		sb.WriteString("\n")

		sb.WriteString(fmt.Sprintf("- Current: %.2f\n", m.Latest))
		sb.WriteString(fmt.Sprintf("- Range: %.2f - %.2f (avg: %.2f)\n", m.Min, m.Max, m.Avg))
		sb.WriteString(fmt.Sprintf("- Trend: %s\n", m.Trend))
		sb.WriteString(fmt.Sprintf("- Data points: %d\n", m.Count))

		if len(m.Anomalies) > 0 {
			sb.WriteString("- Anomalies detected:\n")
			for _, a := range m.Anomalies {
				sb.WriteString(fmt.Sprintf("  - %s\n", a))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// formatTasksContext formats tasks for LLM consumption.
func (s *RAGService) formatTasksContext(tasks []TaskSummary) string {
	if len(tasks) == 0 {
		return "No recent tasks."
	}

	var sb strings.Builder
	sb.WriteString("## Recent Tasks\n\n")

	// Group by status
	statusGroups := make(map[string][]TaskSummary)
	for _, t := range tasks {
		statusGroups[t.Status] = append(statusGroups[t.Status], t)
	}

	for status, group := range statusGroups {
		sb.WriteString(fmt.Sprintf("### %s (%d)\n", status, len(group)))
		for _, t := range group {
			sb.WriteString(fmt.Sprintf("- [%s] %s (created: %s)\n",
				t.ID[:8], t.Type, t.CreatedAt.Format("15:04:05")))
			if t.Error != "" {
				sb.WriteString(fmt.Sprintf("  Error: %s\n", t.Error))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// buildSystemPrompt combines context parts into a system prompt.
func (s *RAGService) buildSystemPrompt(contextParts []string) string {
	var sb strings.Builder

	sb.WriteString("You are Forge AI, an intelligent assistant for the Forge Platform. ")
	sb.WriteString("You help users understand system metrics, debug issues, and optimize performance.\n\n")
	sb.WriteString("You have access to real-time system data. Here is the current context:\n\n")

	for _, part := range contextParts {
		sb.WriteString(part)
		sb.WriteString("\n\n")
	}

	sb.WriteString("Based on this data, provide clear, actionable insights. ")
	sb.WriteString("If you notice any issues or anomalies, explain them and suggest solutions. ")
	sb.WriteString("Be concise but thorough.")

	return sb.String()
}

// estimateTokens provides a rough token count estimate.
// Uses the approximation of ~4 characters per token for English text.
func (s *RAGService) estimateTokens(text string) int {
	return len(text) / 4
}

// AnalyzeMetrics provides AI-powered analysis of system metrics.
func (s *RAGService) AnalyzeMetrics(ctx context.Context, timeRange time.Duration) (*AnalysisResult, error) {
	context, err := s.BuildContext(ctx, ContextRequest{
		TimeRange:      timeRange,
		IncludeMetrics: true,
		IncludeTasks:   true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build context: %w", err)
	}

	result := &AnalysisResult{
		Timestamp:    time.Now(),
		MetricCount:  len(context.Metrics),
		TaskCount:    len(context.Tasks),
		SystemPrompt: context.SystemPrompt,
	}

	// Identify issues from metrics
	for _, m := range context.Metrics {
		if m.Trend == "increasing" && strings.Contains(strings.ToLower(m.Name), "error") {
			result.Issues = append(result.Issues, Issue{
				Severity:    "warning",
				Component:   m.Name,
				Description: fmt.Sprintf("Increasing error rate detected: %.2f -> %.2f", m.Min, m.Latest),
				Suggestion:  "Review recent changes and check logs for root cause.",
			})
		}
		if len(m.Anomalies) > 0 {
			result.Issues = append(result.Issues, Issue{
				Severity:    "info",
				Component:   m.Name,
				Description: fmt.Sprintf("%d anomalies detected in %s", len(m.Anomalies), m.Name),
				Suggestion:  "Investigate unusual spikes in metric values.",
			})
		}
	}

	// Check for failed tasks
	failedCount := 0
	for _, t := range context.Tasks {
		if t.Status == "FAILED" || t.Status == "DEAD" {
			failedCount++
		}
	}
	if failedCount > 0 {
		result.Issues = append(result.Issues, Issue{
			Severity:    "warning",
			Component:   "task_queue",
			Description: fmt.Sprintf("%d failed tasks in the queue", failedCount),
			Suggestion:  "Review task errors and consider retry or manual intervention.",
		})
	}

	// Generate summary
	if len(result.Issues) == 0 {
		result.Summary = fmt.Sprintf("System health check completed. Analyzed %d metrics and %d tasks. No issues detected.",
			result.MetricCount, result.TaskCount)
	} else {
		result.Summary = fmt.Sprintf("System health check completed. Analyzed %d metrics and %d tasks. Found %d issue(s) requiring attention.",
			result.MetricCount, result.TaskCount, len(result.Issues))
	}

	return result, nil
}

// AnalysisResult contains the result of AI analysis.
type AnalysisResult struct {
	Timestamp    time.Time
	MetricCount  int
	TaskCount    int
	Issues       []Issue
	Summary      string
	SystemPrompt string
}

// Issue represents a detected issue.
type Issue struct {
	Severity    string // "info", "warning", "error", "critical"
	Component   string
	Description string
	Suggestion  string
}
