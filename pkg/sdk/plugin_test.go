// Package sdk provides the SDK for developing Forge WebAssembly plugins.
package sdk

import (
	"testing"
)

func TestLogLevel_Constants(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected int32
	}{
		{LogDebug, 0},
		{LogInfo, 1},
		{LogWarn, 2},
		{LogError, 3},
	}

	for _, tt := range tests {
		if int32(tt.level) != tt.expected {
			t.Errorf("expected LogLevel %d, got %d", tt.expected, int32(tt.level))
		}
	}
}

func TestLog(t *testing.T) {
	// Log function calls stub forgeLog - shouldn't panic
	Log(LogInfo, "test message")
	Log(LogDebug, "")
	Log(LogWarn, "warning")
	Log(LogError, "error message")
}

func TestDebug(t *testing.T) {
	// Should not panic
	Debug("debug message")
	Debug("")
}

func TestInfo(t *testing.T) {
	// Should not panic
	Info("info message")
	Info("")
}

func TestWarn(t *testing.T) {
	// Should not panic
	Warn("warning message")
	Warn("")
}

func TestError(t *testing.T) {
	// Should not panic
	Error("error message")
	Error("")
}

func TestRecordMetric(t *testing.T) {
	// Should not panic with stub implementation
	RecordMetric("cpu_usage", 45.5)
	RecordMetric("memory_free", 1024.0)
	RecordMetric("", 0)
}

func TestRecordMetricWithTags(t *testing.T) {
	// Should not panic with stub implementation
	RecordMetricWithTags("http_requests", 100.0, map[string]string{
		"method": "GET",
		"path":   "/api/v1",
	})

	RecordMetricWithTags("cpu_usage", 50.0, map[string]string{})
	RecordMetricWithTags("disk_io", 200.0, nil)
}

func TestGetConfig(t *testing.T) {
	// Should return empty string with stub implementation
	value, ok := GetConfig("test_key")
	if ok {
		t.Error("expected ok=false from stub")
	}
	if value != "" {
		t.Errorf("expected empty string from stub, got %q", value)
	}

	value, ok = GetConfig("")
	if ok {
		t.Error("expected ok=false for empty key")
	}
	if value != "" {
		t.Errorf("expected empty string for empty key, got %q", value)
	}
}

func TestHTTPGet(t *testing.T) {
	// Stub returns error
	resp, err := HTTPGet("http://example.com")
	if err == nil {
		t.Error("expected error from stub implementation")
	}
	if resp != nil {
		t.Error("expected nil response from stub")
	}
}

func TestHTTPPost(t *testing.T) {
	// Stub returns error
	resp, err := HTTPPost("http://example.com", []byte(`{"test": true}`))
	if err == nil {
		t.Error("expected error from stub implementation")
	}
	if resp != nil {
		t.Error("expected nil response from stub")
	}
}

func TestHTTPPut(t *testing.T) {
	// Stub returns error
	resp, err := HTTPPut("http://example.com", []byte(`{"test": true}`))
	if err == nil {
		t.Error("expected error from stub implementation")
	}
	if resp != nil {
		t.Error("expected nil response from stub")
	}
}

func TestHTTPDelete(t *testing.T) {
	// Stub returns error
	resp, err := HTTPDelete("http://example.com")
	if err == nil {
		t.Error("expected error from stub implementation")
	}
	if resp != nil {
		t.Error("expected nil response from stub")
	}
}

func TestEmitEvent(t *testing.T) {
	// Stub returns error
	err := EmitEvent("test_event", []byte(`{"test": true}`))
	if err == nil {
		t.Error("expected error from stub implementation")
	}

	err = EmitEvent("", nil)
	if err == nil {
		t.Error("expected error from stub implementation")
	}
}

func TestReadFile(t *testing.T) {
	// Stub returns error
	data, err := ReadFile("/test/path")
	if err == nil {
		t.Error("expected error from stub implementation")
	}
	if data != nil {
		t.Error("expected nil data from stub")
	}
}

func TestWriteFile(t *testing.T) {
	// Stub returns error
	err := WriteFile("/test/path", []byte("test content"))
	if err == nil {
		t.Error("expected error from stub implementation")
	}
}

