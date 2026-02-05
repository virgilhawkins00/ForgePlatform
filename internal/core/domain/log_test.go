package domain

import (
	"testing"
	"time"
)

func TestNewLogEntry(t *testing.T) {
	entry := NewLogEntry(LogLevelInfo, "Application started", "main.go", "myapp")

	if entry.ID.String() == "" {
		t.Error("ID is empty")
	}
	if entry.Level != LogLevelInfo {
		t.Errorf("Level = %v, want info", entry.Level)
	}
	if entry.Message != "Application started" {
		t.Errorf("Message = %v, want 'Application started'", entry.Message)
	}
	if entry.Source != "main.go" {
		t.Errorf("Source = %v, want main.go", entry.Source)
	}
	if entry.ServiceName != "myapp" {
		t.Errorf("ServiceName = %v, want myapp", entry.ServiceName)
	}
	if entry.Attributes == nil {
		t.Error("Attributes is nil")
	}
}

func TestLogEntry_SetTraceContext(t *testing.T) {
	entry := NewLogEntry(LogLevelInfo, "test", "test.go", "test")

	entry.SetTraceContext("trace-123", "span-456")

	if entry.TraceID != "trace-123" {
		t.Errorf("TraceID = %v, want trace-123", entry.TraceID)
	}
	if entry.SpanID != "span-456" {
		t.Errorf("SpanID = %v, want span-456", entry.SpanID)
	}
}

func TestLogEntry_SetAttribute(t *testing.T) {
	entry := NewLogEntry(LogLevelInfo, "test", "test.go", "test")

	entry.SetAttribute("user_id", "user123")
	entry.SetAttribute("request_id", "req456")

	if entry.Attributes["user_id"] != "user123" {
		t.Error("Attribute user_id not set correctly")
	}
	if entry.Attributes["request_id"] != "req456" {
		t.Error("Attribute request_id not set correctly")
	}
}

func TestLogEntry_IsError(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected bool
	}{
		{LogLevelTrace, false},
		{LogLevelDebug, false},
		{LogLevelInfo, false},
		{LogLevelWarning, false},
		{LogLevelError, true},
		{LogLevelFatal, true},
	}

	for _, tc := range tests {
		entry := NewLogEntry(tc.level, "test", "test.go", "test")
		if entry.IsError() != tc.expected {
			t.Errorf("IsError() for %v = %v, want %v", tc.level, entry.IsError(), tc.expected)
		}
	}
}

func TestLogLevelPriority(t *testing.T) {
	tests := []struct {
		level    LogLevel
		priority int
	}{
		{LogLevelTrace, 0},
		{LogLevelDebug, 1},
		{LogLevelInfo, 2},
		{LogLevelWarning, 3},
		{LogLevelError, 4},
		{LogLevelFatal, 5},
		{LogLevel("unknown"), 2}, // Default
	}

	for _, tc := range tests {
		priority := LogLevelPriority(tc.level)
		if priority != tc.priority {
			t.Errorf("LogLevelPriority(%v) = %d, want %d", tc.level, priority, tc.priority)
		}
	}
}

func TestNewLogParser(t *testing.T) {
	parser := NewLogParser("apache-parser", ParserTypeRegex, `(?P<ip>\d+\.\d+\.\d+\.\d+)`)

	if parser.ID.String() == "" {
		t.Error("ID is empty")
	}
	if parser.Name != "apache-parser" {
		t.Errorf("Name = %v, want apache-parser", parser.Name)
	}
	if parser.Type != ParserTypeRegex {
		t.Errorf("Type = %v, want regex", parser.Type)
	}
	if !parser.Enabled {
		t.Error("Enabled = false, want true")
	}
}

func TestLogParser_Compile(t *testing.T) {
	parser := NewLogParser("test", ParserTypeRegex, `(\d+)-(\w+)`)

	err := parser.Compile()
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	regex := parser.GetCompiledRegex()
	if regex == nil {
		t.Error("GetCompiledRegex() returned nil after Compile()")
	}
}

func TestLogParser_Compile_InvalidPattern(t *testing.T) {
	parser := NewLogParser("test", ParserTypeRegex, `[invalid`)

	err := parser.Compile()
	if err == nil {
		t.Error("Compile() should return error for invalid regex")
	}
}

func TestLogParser_Compile_JSONParser(t *testing.T) {
	parser := NewLogParser("json", ParserTypeJSON, "{}")

	err := parser.Compile()
	if err != nil {
		t.Errorf("Compile() should not error for JSON parser: %v", err)
	}
	if parser.GetCompiledRegex() != nil {
		t.Error("GetCompiledRegex() should return nil for JSON parser")
	}
}

func TestNewLogToMetricRule(t *testing.T) {
	rule := NewLogToMetricRule("error-count", "level", "error", "log.errors.total", MetricTypeCounter)

	if rule.ID.String() == "" {
		t.Error("ID is empty")
	}
	if rule.Name != "error-count" {
		t.Errorf("Name = %v, want error-count", rule.Name)
	}
	if rule.MatchField != "level" {
		t.Errorf("MatchField = %v, want level", rule.MatchField)
	}
	if rule.MetricType != MetricTypeCounter {
		t.Errorf("MetricType = %v, want counter", rule.MetricType)
	}
	if !rule.Enabled {
		t.Error("Enabled = false, want true")
	}
}

func TestNewLogStream(t *testing.T) {
	stream := NewLogStream("application-logs", "stdout", 7*24*time.Hour)

	if stream.ID.String() == "" {
		t.Error("ID is empty")
	}
	if stream.Name != "application-logs" {
		t.Errorf("Name = %v, want application-logs", stream.Name)
	}
	if stream.Source != "stdout" {
		t.Errorf("Source = %v, want stdout", stream.Source)
	}
	if stream.Retention != 7*24*time.Hour {
		t.Errorf("Retention = %v, want 168h", stream.Retention)
	}
	if !stream.Enabled {
		t.Error("Enabled = false, want true")
	}
}

func TestLogParserTypeConstants(t *testing.T) {
	if ParserTypeRegex != "regex" {
		t.Errorf("ParserTypeRegex = %v, want regex", ParserTypeRegex)
	}
	if ParserTypeJSON != "json" {
		t.Errorf("ParserTypeJSON = %v, want json", ParserTypeJSON)
	}
	if ParserTypeGrok != "grok" {
		t.Errorf("ParserTypeGrok = %v, want grok", ParserTypeGrok)
	}
	if ParserTypeKeyValue != "key_value" {
		t.Errorf("ParserTypeKeyValue = %v, want key_value", ParserTypeKeyValue)
	}
}

func TestLogLevelConstants(t *testing.T) {
	if LogLevelTrace != "trace" {
		t.Errorf("LogLevelTrace = %v, want trace", LogLevelTrace)
	}
	if LogLevelDebug != "debug" {
		t.Errorf("LogLevelDebug = %v, want debug", LogLevelDebug)
	}
	if LogLevelInfo != "info" {
		t.Errorf("LogLevelInfo = %v, want info", LogLevelInfo)
	}
	if LogLevelWarning != "warning" {
		t.Errorf("LogLevelWarning = %v, want warning", LogLevelWarning)
	}
	if LogLevelError != "error" {
		t.Errorf("LogLevelError = %v, want error", LogLevelError)
	}
	if LogLevelFatal != "fatal" {
		t.Errorf("LogLevelFatal = %v, want fatal", LogLevelFatal)
	}
}

func TestLogEntry_SetAttribute_NilMap(t *testing.T) {
	entry := &LogEntry{}
	entry.Attributes = nil

	entry.SetAttribute("key", "value")

	if entry.Attributes["key"] != "value" {
		t.Error("SetAttribute should initialize nil map")
	}
}

