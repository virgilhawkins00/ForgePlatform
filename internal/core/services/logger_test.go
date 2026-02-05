// Package services implements core business logic services.
package services

import (
	"testing"

	"github.com/forge-platform/forge/internal/core/ports"
)

func TestNewSlogLogger(t *testing.T) {
	tests := []struct {
		name  string
		level string
		json  bool
	}{
		{"debug level text", "debug", false},
		{"info level text", "info", false},
		{"warn level text", "warn", false},
		{"error level text", "error", false},
		{"default level text", "unknown", false},
		{"debug level json", "debug", true},
		{"info level json", "info", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewSlogLogger(tt.level, tt.json)
			if logger == nil {
				t.Fatal("expected non-nil logger")
			}
			if logger.logger == nil {
				t.Error("internal slog logger not initialized")
			}
		})
	}
}

func TestSlogLogger_Methods(t *testing.T) {
	logger := NewSlogLogger("debug", false)

	// These should not panic
	logger.Debug("debug message", "key", "value")
	logger.Info("info message", "key", "value")
	logger.Warn("warn message", "key", "value")
	logger.Error("error message", "key", "value")
}

func TestSlogLogger_With(t *testing.T) {
	logger := NewSlogLogger("info", false)

	child := logger.With("context_key", "context_value")
	if child == nil {
		t.Fatal("expected non-nil child logger")
	}

	// Verify it implements ports.Logger
	var _ ports.Logger = child
}

func TestNopLogger(t *testing.T) {
	logger := &NopLogger{}

	// These should not panic and do nothing
	logger.Debug("debug message", "key", "value")
	logger.Info("info message", "key", "value")
	logger.Warn("warn message", "key", "value")
	logger.Error("error message", "key", "value")

	child := logger.With("key", "value")
	if child != logger {
		t.Error("NopLogger.With should return itself")
	}

	// Verify it implements ports.Logger
	var _ ports.Logger = logger
}

