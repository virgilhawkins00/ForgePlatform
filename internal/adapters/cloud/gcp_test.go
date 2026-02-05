// Package cloud provides cloud provider integrations.
package cloud

import (
	"testing"
	"time"

	"github.com/forge-platform/forge/internal/core/ports"
)

// mockGCPLogger for testing
type mockGCPLogger struct{}

func (m *mockGCPLogger) Debug(msg string, args ...interface{}) {}
func (m *mockGCPLogger) Info(msg string, args ...interface{})  {}
func (m *mockGCPLogger) Warn(msg string, args ...interface{})  {}
func (m *mockGCPLogger) Error(msg string, args ...interface{}) {}
func (m *mockGCPLogger) With(args ...interface{}) ports.Logger { return m }

func TestDefaultGCPConfig(t *testing.T) {
	cfg := DefaultGCPConfig()

	if cfg.MetricPrefix != "custom.googleapis.com/forge" {
		t.Errorf("expected MetricPrefix 'custom.googleapis.com/forge', got '%s'", cfg.MetricPrefix)
	}
	if cfg.FlushInterval != 60*time.Second {
		t.Errorf("expected FlushInterval 60s, got %v", cfg.FlushInterval)
	}
	if cfg.BatchSize != 200 {
		t.Errorf("expected BatchSize 200, got %d", cfg.BatchSize)
	}
	if cfg.ProjectID != "" {
		t.Errorf("expected empty ProjectID, got '%s'", cfg.ProjectID)
	}
}

func TestNewGCPExporter(t *testing.T) {
	logger := &mockGCPLogger{}

	cfg := DefaultGCPConfig()
	cfg.ProjectID = "test-project"

	exporter, err := NewGCPExporter(cfg, logger)
	if err != nil {
		t.Fatalf("NewGCPExporter failed: %v", err)
	}

	if exporter == nil {
		t.Fatal("expected non-nil exporter")
	}
	if exporter.config.ProjectID != "test-project" {
		t.Errorf("expected project ID 'test-project', got '%s'", exporter.config.ProjectID)
	}
	if exporter.httpClient == nil {
		t.Error("http client not initialized")
	}
	if exporter.logger == nil {
		t.Error("logger not set correctly")
	}
	if exporter.metricCh == nil {
		t.Error("metric channel not initialized")
	}
	if exporter.stopCh == nil {
		t.Error("stop channel not initialized")
	}
}

func TestNewGCPExporter_MissingProjectID(t *testing.T) {
	logger := &mockGCPLogger{}

	cfg := DefaultGCPConfig()
	// ProjectID is empty

	_, err := NewGCPExporter(cfg, logger)
	if err == nil {
		t.Error("expected error for missing project ID")
	}
}

