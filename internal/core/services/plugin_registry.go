// Package services provides the core business logic services.
package services

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/forge-platform/forge/internal/core/domain"
	"github.com/forge-platform/forge/internal/core/ports"
)

// PluginManifest describes a plugin in the registry.
type PluginManifest struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	Author       string            `json:"author"`
	License      string            `json:"license"`
	Homepage     string            `json:"homepage,omitempty"`
	Repository   string            `json:"repository,omitempty"`
	Tags         []string          `json:"tags,omitempty"`
	Permissions  []string          `json:"permissions"`
	Dependencies []PluginDep       `json:"dependencies,omitempty"`
	SHA256       string            `json:"sha256"`
	Signature    string            `json:"signature,omitempty"`
	DownloadURL  string            `json:"download_url"`
	Size         int64             `json:"size"`
	PublishedAt  time.Time         `json:"published_at"`
	Config       map[string]string `json:"config,omitempty"`
}

// PluginDep describes a plugin dependency.
type PluginDep struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	Optional   bool   `json:"optional,omitempty"`
}

// RegistryIndex contains the list of available plugins.
type RegistryIndex struct {
	Version   string           `json:"version"`
	UpdatedAt time.Time        `json:"updated_at"`
	Plugins   []PluginManifest `json:"plugins"`
}

// PluginRegistry manages plugin discovery, installation, and updates.
type PluginRegistry struct {
	mu           sync.RWMutex
	registryURL  string
	cacheDir     string
	pluginsDir   string
	index        *RegistryIndex
	installed    map[string]*domain.Plugin
	publicKeys   []ed25519.PublicKey
	httpClient   *http.Client
	logger       ports.Logger
}

// RegistryConfig configures the plugin registry.
type RegistryConfig struct {
	RegistryURL string   // Remote registry URL
	CacheDir    string   // Local cache directory
	PluginsDir  string   // Plugins installation directory
	PublicKeys  []string // Trusted public keys (hex-encoded)
}

// NewPluginRegistry creates a new plugin registry.
func NewPluginRegistry(cfg RegistryConfig, logger ports.Logger) (*PluginRegistry, error) {
	// Set defaults
	if cfg.RegistryURL == "" {
		cfg.RegistryURL = "https://registry.forgeplatform.dev"
	}
	if cfg.CacheDir == "" {
		home, _ := os.UserHomeDir()
		cfg.CacheDir = filepath.Join(home, ".forge", "cache")
	}
	if cfg.PluginsDir == "" {
		home, _ := os.UserHomeDir()
		cfg.PluginsDir = filepath.Join(home, ".forge", "plugins")
	}

	// Create directories
	for _, dir := range []string{cfg.CacheDir, cfg.PluginsDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Parse public keys
	var publicKeys []ed25519.PublicKey
	for _, keyHex := range cfg.PublicKeys {
		keyBytes, err := hex.DecodeString(keyHex)
		if err != nil {
			return nil, fmt.Errorf("invalid public key: %w", err)
		}
		if len(keyBytes) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("invalid public key size")
		}
		publicKeys = append(publicKeys, ed25519.PublicKey(keyBytes))
	}

	return &PluginRegistry{
		registryURL: cfg.RegistryURL,
		cacheDir:    cfg.CacheDir,
		pluginsDir:  cfg.PluginsDir,
		installed:   make(map[string]*domain.Plugin),
		publicKeys:  publicKeys,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		logger: logger,
	}, nil
}

// Refresh updates the plugin index from the remote registry.
func (r *PluginRegistry) Refresh(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	url := r.registryURL + "/index.json"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	var index RegistryIndex
	if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
		return fmt.Errorf("failed to decode index: %w", err)
	}

	r.index = &index
	r.logger.Info("Registry refreshed", "plugins", len(index.Plugins))
	return nil
}

// Search searches for plugins by name or tags.
func (r *PluginRegistry) Search(query string) []PluginManifest {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.index == nil {
		return nil
	}

	query = strings.ToLower(query)
	var results []PluginManifest

	for _, p := range r.index.Plugins {
		if strings.Contains(strings.ToLower(p.Name), query) ||
			strings.Contains(strings.ToLower(p.Description), query) {
			results = append(results, p)
			continue
		}
		for _, tag := range p.Tags {
			if strings.Contains(strings.ToLower(tag), query) {
				results = append(results, p)
				break
			}
		}
	}

	return results
}

// GetManifest returns the manifest for a specific plugin version.
func (r *PluginRegistry) GetManifest(name, version string) (*PluginManifest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.index == nil {
		return nil, fmt.Errorf("registry not loaded")
	}

	for _, p := range r.index.Plugins {
		if p.Name == name && (version == "" || version == "latest" || p.Version == version) {
			return &p, nil
		}
	}

	return nil, fmt.Errorf("plugin not found: %s@%s", name, version)
}

// GetVersions returns all available versions of a plugin, sorted newest first.
func (r *PluginRegistry) GetVersions(name string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.index == nil {
		return nil
	}

	var versions []string
	for _, p := range r.index.Plugins {
		if p.Name == name {
			versions = append(versions, p.Version)
		}
	}

	// Sort versions (simple string sort - ideally use semver)
	sort.Sort(sort.Reverse(sort.StringSlice(versions)))
	return versions
}

// Install downloads and installs a plugin.
func (r *PluginRegistry) Install(ctx context.Context, name, version string) (*domain.Plugin, error) {
	manifest, err := r.GetManifest(name, version)
	if err != nil {
		return nil, err
	}

	// Download plugin
	r.logger.Info("Downloading plugin", "name", name, "version", manifest.Version)

	req, err := http.NewRequestWithContext(ctx, "GET", manifest.DownloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download plugin: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Read plugin data
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin data: %w", err)
	}

	// Verify hash
	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])
	if hashStr != manifest.SHA256 {
		return nil, fmt.Errorf("hash mismatch: expected %s, got %s", manifest.SHA256, hashStr)
	}

	// Verify signature if required
	if len(r.publicKeys) > 0 && manifest.Signature != "" {
		if err := r.verifySignature(data, manifest.Signature); err != nil {
			return nil, fmt.Errorf("signature verification failed: %w", err)
		}
	}

	// Save plugin
	pluginPath := filepath.Join(r.pluginsDir, fmt.Sprintf("%s-%s.wasm", name, manifest.Version))
	if err := os.WriteFile(pluginPath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to save plugin: %w", err)
	}

	// Create domain plugin
	plugin := domain.NewPlugin(name, manifest.Version, pluginPath)
	plugin.Hash = hashStr

	r.mu.Lock()
	r.installed[name] = plugin
	r.mu.Unlock()

	r.logger.Info("Plugin installed", "name", name, "version", manifest.Version)
	return plugin, nil
}

// verifySignature verifies the plugin signature using trusted public keys.
func (r *PluginRegistry) verifySignature(data []byte, signatureHex string) error {
	signature, err := hex.DecodeString(signatureHex)
	if err != nil {
		return fmt.Errorf("invalid signature format: %w", err)
	}

	// Try each trusted public key
	for _, pubKey := range r.publicKeys {
		if ed25519.Verify(pubKey, data, signature) {
			return nil
		}
	}

	return fmt.Errorf("signature not verified by any trusted key")
}

// Uninstall removes an installed plugin.
func (r *PluginRegistry) Uninstall(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	plugin, ok := r.installed[name]
	if !ok {
		return fmt.Errorf("plugin not installed: %s", name)
	}

	// Remove file
	if err := os.Remove(plugin.Path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove plugin file: %w", err)
	}

	delete(r.installed, name)
	r.logger.Info("Plugin uninstalled", "name", name)
	return nil
}

// ListInstalled returns all installed plugins.
func (r *PluginRegistry) ListInstalled() []*domain.Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugins := make([]*domain.Plugin, 0, len(r.installed))
	for _, p := range r.installed {
		plugins = append(plugins, p)
	}
	return plugins
}

// CheckUpdates checks for available updates for installed plugins.
func (r *PluginRegistry) CheckUpdates() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	updates := make(map[string]string)
	for name, plugin := range r.installed {
		versions := r.GetVersions(name)
		if len(versions) > 0 && versions[0] != plugin.Version {
			updates[name] = versions[0]
		}
	}
	return updates
}
