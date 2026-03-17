// Package services implements core business logic services.
package services

import (
	"testing"
	"time"
)

func TestRAGConfig_Defaults(t *testing.T) {
	cfg := RAGConfig{}

	if cfg.MaxContextTokens != 0 {
		t.Errorf("expected default MaxContextTokens 0, got %d", cfg.MaxContextTokens)
	}
}

func TestRAGConfig_CustomValues(t *testing.T) {
	cfg := RAGConfig{MaxContextTokens: 8192}

	if cfg.MaxContextTokens != 8192 {
		t.Errorf("expected MaxContextTokens 8192, got %d", cfg.MaxContextTokens)
	}
}

func TestContextRequest_Fields(t *testing.T) {
	req := ContextRequest{
		TimeRange:      1 * time.Hour,
		MetricNames:   []string{"cpu_usage", "memory_free"},
		IncludeMetrics: true,
		IncludeTasks:   true,
		IncludeLogs:    false,
		Query:         "what is the current CPU usage?",
	}

	if req.TimeRange != 1*time.Hour {
		t.Error("TimeRange field mismatch")
	}
	if len(req.MetricNames) != 2 {
		t.Error("MetricNames field mismatch")
	}
	if !req.IncludeMetrics {
		t.Error("IncludeMetrics should be true")
	}
	if !req.IncludeTasks {
		t.Error("IncludeTasks should be true")
	}
	if req.IncludeLogs {
		t.Error("IncludeLogs should be false")
	}
	if req.Query != "what is the current CPU usage?" {
		t.Error("Query field mismatch")
	}
}

func TestContextResult_Fields(t *testing.T) {
	result := ContextResult{
		Metrics:      []MetricSummary{{Name: "cpu", Latest: 50.0}},
		Tasks:        []TaskSummary{{ID: "task-1", Status: "completed"}},
		Logs:         []LogEntry{{Level: "info", Message: "test"}},
		SystemPrompt: "You are a helpful assistant",
		TokenCount:   1024,
	}

	if len(result.Metrics) != 1 {
		t.Error("Metrics field mismatch")
	}
	if len(result.Tasks) != 1 {
		t.Error("Tasks field mismatch")
	}
	if len(result.Logs) != 1 {
		t.Error("Logs field mismatch")
	}
	if result.SystemPrompt == "" {
		t.Error("SystemPrompt should not be empty")
	}
	if result.TokenCount != 1024 {
		t.Error("TokenCount field mismatch")
	}
}

func TestMetricSummary_Fields(t *testing.T) {
	summary := MetricSummary{
		Name:      "cpu_usage",
		Tags:      map[string]string{"host": "server1"},
		Latest:    75.5,
		Min:       10.0,
		Max:       100.0,
		Avg:       55.5,
		Count:     100,
		Trend:     "increasing",
		Anomalies: []string{"spike at 10:00"},
	}

	if summary.Name != "cpu_usage" {
		t.Error("Name field mismatch")
	}
	if summary.Latest != 75.5 {
		t.Error("Latest field mismatch")
	}
	if summary.Trend != "increasing" {
		t.Error("Trend field mismatch")
	}
	if len(summary.Anomalies) != 1 {
		t.Error("Anomalies field mismatch")
	}
}

func TestTaskSummary_Fields(t *testing.T) {
	summary := TaskSummary{
		ID:     "task-123",
		Type:   "deploy",
		Status: "completed",
		Error:  "",
	}

	if summary.ID != "task-123" {
		t.Error("ID field mismatch")
	}
	if summary.Type != "deploy" {
		t.Error("Type field mismatch")
	}
	if summary.Status != "completed" {
		t.Error("Status field mismatch")
	}
}

func TestLogEntry_Fields(t *testing.T) {
	entry := LogEntry{
		Level:   "error",
		Message: "Something went wrong",
		Source:  "app.service",
	}

	if entry.Level != "error" {
		t.Error("Level field mismatch")
	}
	if entry.Message != "Something went wrong" {
		t.Error("Message field mismatch")
	}
	if entry.Source != "app.service" {
		t.Error("Source field mismatch")
	}
}

