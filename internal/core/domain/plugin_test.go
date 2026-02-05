package domain

import (
	"errors"
	"testing"
)

func TestNewPlugin(t *testing.T) {
	plugin := NewPlugin("metrics-collector", "1.0.0", "/plugins/metrics.wasm")

	if plugin.ID.String() == "" {
		t.Error("ID is empty")
	}
	if plugin.Name != "metrics-collector" {
		t.Errorf("Name = %v, want metrics-collector", plugin.Name)
	}
	if plugin.Version != "1.0.0" {
		t.Errorf("Version = %v, want 1.0.0", plugin.Version)
	}
	if plugin.Path != "/plugins/metrics.wasm" {
		t.Errorf("Path = %v, want /plugins/metrics.wasm", plugin.Path)
	}
	if plugin.Status != PluginStatusInactive {
		t.Errorf("Status = %v, want inactive", plugin.Status)
	}
	if len(plugin.Permissions) != 0 {
		t.Errorf("Permissions should be empty, got %d", len(plugin.Permissions))
	}
}

func TestPlugin_HasPermission(t *testing.T) {
	plugin := NewPlugin("test", "1.0.0", "/test.wasm")
	plugin.Permissions = []PluginPermission{
		PermissionMetricsRead,
		PermissionLogsWrite,
	}

	if !plugin.HasPermission(PermissionMetricsRead) {
		t.Error("HasPermission(MetricsRead) = false, want true")
	}
	if !plugin.HasPermission(PermissionLogsWrite) {
		t.Error("HasPermission(LogsWrite) = false, want true")
	}
	if plugin.HasPermission(PermissionNetwork) {
		t.Error("HasPermission(Network) = true, want false")
	}
	if plugin.HasPermission(PermissionFileSystem) {
		t.Error("HasPermission(FileSystem) = true, want false")
	}
}

func TestPlugin_MarkLoaded(t *testing.T) {
	plugin := NewPlugin("test", "1.0.0", "/test.wasm")
	plugin.Error = "previous error"

	plugin.MarkLoaded()

	if plugin.Status != PluginStatusActive {
		t.Errorf("Status = %v, want active", plugin.Status)
	}
	if plugin.LoadedAt == nil {
		t.Error("LoadedAt is nil after MarkLoaded()")
	}
	if plugin.Error != "" {
		t.Errorf("Error = %v, want empty string", plugin.Error)
	}
}

func TestPlugin_MarkError(t *testing.T) {
	plugin := NewPlugin("test", "1.0.0", "/test.wasm")
	plugin.MarkLoaded()

	plugin.MarkError(errors.New("failed to initialize"))

	if plugin.Status != PluginStatusError {
		t.Errorf("Status = %v, want error", plugin.Status)
	}
	if plugin.Error != "failed to initialize" {
		t.Errorf("Error = %v, want 'failed to initialize'", plugin.Error)
	}
}

func TestPluginStatusConstants(t *testing.T) {
	if PluginStatusInactive != "inactive" {
		t.Errorf("PluginStatusInactive = %v, want inactive", PluginStatusInactive)
	}
	if PluginStatusActive != "active" {
		t.Errorf("PluginStatusActive = %v, want active", PluginStatusActive)
	}
	if PluginStatusError != "error" {
		t.Errorf("PluginStatusError = %v, want error", PluginStatusError)
	}
	if PluginStatusLoading != "loading" {
		t.Errorf("PluginStatusLoading = %v, want loading", PluginStatusLoading)
	}
}

func TestPluginPermissionConstants(t *testing.T) {
	permissions := []PluginPermission{
		PermissionMetricsRead,
		PermissionMetricsWrite,
		PermissionLogsRead,
		PermissionLogsWrite,
		PermissionNetwork,
		PermissionFileSystem,
	}
	expected := []string{
		"metrics:read",
		"metrics:write",
		"logs:read",
		"logs:write",
		"network",
		"filesystem",
	}

	for i, perm := range permissions {
		if string(perm) != expected[i] {
			t.Errorf("Permission[%d] = %v, want %v", i, perm, expected[i])
		}
	}
}

func TestPluginManifest(t *testing.T) {
	manifest := PluginManifest{
		Name:        "test-plugin",
		Version:     "1.0.0",
		Description: "A test plugin",
		Author:      "Test Author",
		Entrypoint:  "main.wasm",
		Permissions: []PluginPermission{PermissionMetricsRead},
		Config: []PluginConfigDef{
			{Name: "interval", Type: "int", Default: "60", Required: true},
		},
		Hooks: []string{"on_tick", "handle_command"},
	}

	if manifest.Name != "test-plugin" {
		t.Errorf("Name = %v, want test-plugin", manifest.Name)
	}
	if len(manifest.Permissions) != 1 {
		t.Errorf("Permissions count = %d, want 1", len(manifest.Permissions))
	}
	if len(manifest.Config) != 1 {
		t.Errorf("Config count = %d, want 1", len(manifest.Config))
	}
	if len(manifest.Hooks) != 2 {
		t.Errorf("Hooks count = %d, want 2", len(manifest.Hooks))
	}
}

