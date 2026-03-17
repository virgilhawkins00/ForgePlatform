// Package wasm implements the WebAssembly plugin runtime using wazero.
package wasm

import (
	"testing"
	"time"
)

func TestRuntimeOptions_Defaults(t *testing.T) {
	opts := RuntimeOptions{}

	if opts.DataDir != "" {
		t.Error("expected empty DataDir as default")
	}
	if opts.HTTPTimeout != 0 {
		t.Error("expected zero HTTPTimeout as default")
	}
	if opts.EventBufSize != 0 {
		t.Error("expected zero EventBufSize as default")
	}
	if opts.Config != nil {
		t.Error("expected nil Config as default")
	}
}

func TestRuntimeOptions_CustomValues(t *testing.T) {
	opts := RuntimeOptions{
		DataDir:      "/tmp/plugins",
		HTTPTimeout:  60 * time.Second,
		EventBufSize: 500,
		Config: map[string]string{
			"key1": "value1",
		},
		AllowedHosts: []string{"api.example.com"},
	}

	if opts.DataDir != "/tmp/plugins" {
		t.Error("DataDir mismatch")
	}
	if opts.HTTPTimeout != 60*time.Second {
		t.Error("HTTPTimeout mismatch")
	}
	if opts.EventBufSize != 500 {
		t.Error("EventBufSize mismatch")
	}
	if opts.Config["key1"] != "value1" {
		t.Error("Config mismatch")
	}
	if len(opts.AllowedHosts) != 1 {
		t.Error("AllowedHosts mismatch")
	}
}

func TestPluginEvent_Fields(t *testing.T) {
	event := PluginEvent{
		PluginID:  "plugin-123",
		EventType: "metric_collected",
		Payload:   []byte(`{"cpu": 50.5}`),
	}

	if event.PluginID != "plugin-123" {
		t.Error("PluginID mismatch")
	}
	if event.EventType != "metric_collected" {
		t.Error("EventType mismatch")
	}
	if len(event.Payload) == 0 {
		t.Error("Payload should not be empty")
	}
}

func TestPluginMemoryAllocator_Fields(t *testing.T) {
	allocator := PluginMemoryAllocator{
		memory: make(map[uint32][]byte),
		nextID: 0,
	}

	// Store some data
	allocator.memory[1] = []byte("test data")
	allocator.nextID = 2

	if len(allocator.memory) != 1 {
		t.Error("memory map should have 1 entry")
	}
	if allocator.nextID != 2 {
		t.Error("nextID should be 2")
	}
}

func TestLoadedPlugin_Fields(t *testing.T) {
	loaded := LoadedPlugin{
		Plugin:  nil, // Domain plugin would be set here
		Module:  nil, // WASM module would be set here
		Exports: nil, // Map of api.Function would be set here
	}

	if loaded.Plugin != nil {
		t.Error("Plugin should be nil in this test")
	}
	if loaded.Module != nil {
		t.Error("Module should be nil in this test")
	}
}

