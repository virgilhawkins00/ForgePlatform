// Package services implements core business logic services.
package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/forge-platform/forge/internal/core/ports"
)

// mockPluginRegistryLogger for testing
type mockPluginRegistryLogger struct{}

func (m *mockPluginRegistryLogger) Debug(msg string, args ...interface{}) {}
func (m *mockPluginRegistryLogger) Info(msg string, args ...interface{})  {}
func (m *mockPluginRegistryLogger) Warn(msg string, args ...interface{})  {}
func (m *mockPluginRegistryLogger) Error(msg string, args ...interface{}) {}
func (m *mockPluginRegistryLogger) With(args ...interface{}) ports.Logger { return m }

func TestNewPluginRegistry(t *testing.T) {
	logger := &mockPluginRegistryLogger{}
	tmpDir := filepath.Join(os.TempDir(), "forge-plugin-test")
	defer os.RemoveAll(tmpDir)

	cfg := RegistryConfig{
		RegistryURL: "https://plugins.example.com",
		CacheDir:    filepath.Join(tmpDir, "cache"),
		PluginsDir:  filepath.Join(tmpDir, "plugins"),
	}

	registry, err := NewPluginRegistry(cfg, logger)
	if err != nil {
		t.Fatalf("NewPluginRegistry failed: %v", err)
	}

	if registry == nil {
		t.Fatal("expected non-nil registry")
	}
	if registry.registryURL != cfg.RegistryURL {
		t.Errorf("expected registry URL '%s', got '%s'", cfg.RegistryURL, registry.registryURL)
	}
	if registry.cacheDir != cfg.CacheDir {
		t.Errorf("expected cache dir '%s', got '%s'", cfg.CacheDir, registry.cacheDir)
	}
	if registry.pluginsDir != cfg.PluginsDir {
		t.Errorf("expected plugins dir '%s', got '%s'", cfg.PluginsDir, registry.pluginsDir)
	}
	if registry.installed == nil {
		t.Error("installed map not initialized")
	}
	if registry.httpClient == nil {
		t.Error("http client not initialized")
	}
	if registry.logger == nil {
		t.Error("logger not set correctly")
	}
}

func TestNewPluginRegistry_DefaultDirs(t *testing.T) {
	logger := &mockPluginRegistryLogger{}

	cfg := RegistryConfig{
		RegistryURL: "https://plugins.example.com",
	}

	registry, err := NewPluginRegistry(cfg, logger)
	if err != nil {
		t.Fatalf("NewPluginRegistry failed: %v", err)
	}

	// Should use default directories
	if registry.cacheDir == "" {
		t.Error("cache dir should have a default value")
	}
	if registry.pluginsDir == "" {
		t.Error("plugins dir should have a default value")
	}

	// Cleanup
	os.RemoveAll(registry.cacheDir)
	os.RemoveAll(registry.pluginsDir)
}

func TestNewPluginRegistry_WithPublicKeys(t *testing.T) {
	logger := &mockPluginRegistryLogger{}
	tmpDir := filepath.Join(os.TempDir(), "forge-plugin-test-keys")
	defer os.RemoveAll(tmpDir)

	// Valid ed25519 public key (hex encoded, 64 characters for 32 bytes)
	validKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	cfg := RegistryConfig{
		RegistryURL: "https://plugins.example.com",
		CacheDir:    filepath.Join(tmpDir, "cache"),
		PluginsDir:  filepath.Join(tmpDir, "plugins"),
		PublicKeys:  []string{validKey},
	}

	registry, err := NewPluginRegistry(cfg, logger)
	if err != nil {
		t.Fatalf("NewPluginRegistry failed: %v", err)
	}

	if len(registry.publicKeys) != 1 {
		t.Errorf("expected 1 public key, got %d", len(registry.publicKeys))
	}
}

func TestPluginRegistry_List_Empty(t *testing.T) {
	logger := &mockPluginRegistryLogger{}
	tmpDir := filepath.Join(os.TempDir(), "forge-plugin-test-list")
	defer os.RemoveAll(tmpDir)

	cfg := RegistryConfig{
		CacheDir:   filepath.Join(tmpDir, "cache"),
		PluginsDir: filepath.Join(tmpDir, "plugins"),
	}

	registry, err := NewPluginRegistry(cfg, logger)
	if err != nil {
		t.Fatalf("NewPluginRegistry failed: %v", err)
	}

	plugins := registry.ListInstalled()
	if len(plugins) != 0 {
		t.Errorf("expected 0 installed plugins, got %d", len(plugins))
	}
}

func TestPluginRegistry_Search_NoIndex(t *testing.T) {
	logger := &mockPluginRegistryLogger{}
	tmpDir := filepath.Join(os.TempDir(), "forge-plugin-test-search")
	defer os.RemoveAll(tmpDir)

	cfg := RegistryConfig{
		CacheDir:   filepath.Join(tmpDir, "cache"),
		PluginsDir: filepath.Join(tmpDir, "plugins"),
	}

	registry, err := NewPluginRegistry(cfg, logger)
	if err != nil {
		t.Fatalf("NewPluginRegistry failed: %v", err)
	}

	// Search with no index loaded
	results := registry.Search("test")
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestPluginRegistry_GetVersions_NoIndex(t *testing.T) {
	logger := &mockPluginRegistryLogger{}
	tmpDir := filepath.Join(os.TempDir(), "forge-plugin-test-versions")
	defer os.RemoveAll(tmpDir)

	cfg := RegistryConfig{
		CacheDir:   filepath.Join(tmpDir, "cache"),
		PluginsDir: filepath.Join(tmpDir, "plugins"),
	}

	registry, err := NewPluginRegistry(cfg, logger)
	if err != nil {
		t.Fatalf("NewPluginRegistry failed: %v", err)
	}

	// Get versions with no index loaded
	versions := registry.GetVersions("test-plugin")
	if versions != nil && len(versions) != 0 {
		t.Errorf("expected nil or empty versions, got %v", versions)
	}
}

func TestPluginRegistry_CheckUpdates_Empty(t *testing.T) {
	logger := &mockPluginRegistryLogger{}
	tmpDir := filepath.Join(os.TempDir(), "forge-plugin-test-updates")
	defer os.RemoveAll(tmpDir)

	cfg := RegistryConfig{
		CacheDir:   filepath.Join(tmpDir, "cache"),
		PluginsDir: filepath.Join(tmpDir, "plugins"),
	}

	registry, err := NewPluginRegistry(cfg, logger)
	if err != nil {
		t.Fatalf("NewPluginRegistry failed: %v", err)
	}

	// Check updates with no plugins installed
	updates := registry.CheckUpdates()
	if len(updates) != 0 {
		t.Errorf("expected 0 updates, got %d", len(updates))
	}
}

