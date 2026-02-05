package services

import (
	"log/slog"
	"os"

	"github.com/forge-platform/forge/internal/core/ports"
)

// SlogLogger implements ports.Logger using slog.
type SlogLogger struct {
	logger *slog.Logger
}

// NewSlogLogger creates a new slog-based logger.
func NewSlogLogger(level string, json bool) *SlogLogger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	var handler slog.Handler
	if json {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return &SlogLogger{
		logger: slog.New(handler),
	}
}

// Debug logs a debug message.
func (l *SlogLogger) Debug(msg string, args ...interface{}) {
	l.logger.Debug(msg, args...)
}

// Info logs an info message.
func (l *SlogLogger) Info(msg string, args ...interface{}) {
	l.logger.Info(msg, args...)
}

// Warn logs a warning message.
func (l *SlogLogger) Warn(msg string, args ...interface{}) {
	l.logger.Warn(msg, args...)
}

// Error logs an error message.
func (l *SlogLogger) Error(msg string, args ...interface{}) {
	l.logger.Error(msg, args...)
}

// With returns a logger with additional context.
func (l *SlogLogger) With(args ...interface{}) ports.Logger {
	return &SlogLogger{
		logger: l.logger.With(args...),
	}
}

var _ ports.Logger = (*SlogLogger)(nil)

// NopLogger is a no-op logger for testing.
type NopLogger struct{}

func (l *NopLogger) Debug(msg string, args ...interface{}) {}
func (l *NopLogger) Info(msg string, args ...interface{})  {}
func (l *NopLogger) Warn(msg string, args ...interface{})  {}
func (l *NopLogger) Error(msg string, args ...interface{}) {}
func (l *NopLogger) With(args ...interface{}) ports.Logger { return l }

var _ ports.Logger = (*NopLogger)(nil)

