// Package daemon implements the background daemon service.
package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	forgeDir := filepath.Join(os.TempDir(), "forge-test")
	cfg := DefaultConfig(forgeDir)

	expectedSocket := filepath.Join(forgeDir, "forge.sock")
	if cfg.SocketPath != expectedSocket {
		t.Errorf("expected SocketPath %s, got %s", expectedSocket, cfg.SocketPath)
	}

	expectedPID := filepath.Join(forgeDir, "forge.pid")
	if cfg.PIDFile != expectedPID {
		t.Errorf("expected PIDFile %s, got %s", expectedPID, cfg.PIDFile)
	}

	expectedData := filepath.Join(forgeDir, "data")
	if cfg.DataDir != expectedData {
		t.Errorf("expected DataDir %s, got %s", expectedData, cfg.DataDir)
	}

	if cfg.ShutdownTimeout.Seconds() != 10 {
		t.Errorf("expected ShutdownTimeout 10s, got %v", cfg.ShutdownTimeout)
	}

	if cfg.WorkerCount != 4 {
		t.Errorf("expected WorkerCount 4, got %d", cfg.WorkerCount)
	}

	if cfg.HTTPPort != "" {
		t.Errorf("expected HTTPPort empty, got %s", cfg.HTTPPort)
	}
}

func TestRequest_JSON(t *testing.T) {
	req := Request{
		Method: "task.create",
		Params: map[string]interface{}{
			"type":    "shell",
			"payload": "echo hello",
		},
		ID: "req-123",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	var decoded Request
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal request: %v", err)
	}

	if decoded.Method != req.Method {
		t.Errorf("expected method %s, got %s", req.Method, decoded.Method)
	}
	if decoded.ID != req.ID {
		t.Errorf("expected ID %s, got %s", req.ID, decoded.ID)
	}
}

func TestResponse_JSON(t *testing.T) {
	resp := Response{
		Result: map[string]interface{}{"status": "ok"},
		ID:     "req-123",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	var decoded Response
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if decoded.ID != resp.ID {
		t.Errorf("expected ID %s, got %s", resp.ID, decoded.ID)
	}
	if decoded.Error != "" {
		t.Error("expected no error in response")
	}
}

func TestResponse_WithError(t *testing.T) {
	resp := Response{
		Error: "something went wrong",
		ID:    "req-456",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	var decoded Response
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if decoded.Error != "something went wrong" {
		t.Errorf("expected error message, got %s", decoded.Error)
	}
	if decoded.Result != nil {
		t.Error("expected nil result when error is present")
	}
}

func TestConfig_Fields(t *testing.T) {
	cfg := Config{
		SocketPath:      filepath.Join(os.TempDir(), "forge.sock"),
		PIDFile:         filepath.Join(os.TempDir(), "forge.pid"),
		DataDir:         filepath.Join(os.TempDir(), "forge-data"),
		ShutdownTimeout: 30000000000, // 30 seconds
		WorkerCount:     8,
		HTTPPort:        "8080",
	}

	if cfg.SocketPath != filepath.Join(os.TempDir(), "forge.sock") {
		t.Error("SocketPath field mismatch")
	}
	if cfg.WorkerCount != 8 {
		t.Error("WorkerCount field mismatch")
	}
	if cfg.HTTPPort != "8080" {
		t.Error("HTTPPort field mismatch")
	}
}

func TestVersion(t *testing.T) {
	// Version should be defined
	if Version == "" {
		t.Error("Version constant should not be empty")
	}
}

func TestNewClient_MissingSocket(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix sockets not supported on Windows")
	}
	// Create a temp dir without socket
	tmpDir, err := os.MkdirTemp("", "forge-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	_, err = NewClient(tmpDir)
	if err == nil {
		t.Error("expected error for missing socket file")
	}
}

