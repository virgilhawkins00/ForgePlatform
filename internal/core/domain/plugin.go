package domain

import (
	"time"

	"github.com/google/uuid"
)

// PluginStatus represents the current state of a plugin.
type PluginStatus string

const (
	PluginStatusInactive PluginStatus = "inactive"
	PluginStatusActive   PluginStatus = "active"
	PluginStatusError    PluginStatus = "error"
	PluginStatusLoading  PluginStatus = "loading"
)

// PluginPermission represents a capability that a plugin can request.
type PluginPermission string

const (
	PermissionMetricsRead  PluginPermission = "metrics:read"
	PermissionMetricsWrite PluginPermission = "metrics:write"
	PermissionLogsRead     PluginPermission = "logs:read"
	PermissionLogsWrite    PluginPermission = "logs:write"
	PermissionNetwork      PluginPermission = "network"
	PermissionFileSystem   PluginPermission = "filesystem"
)

// Plugin represents a WebAssembly plugin loaded into the Forge runtime.
type Plugin struct {
	ID          uuid.UUID          `json:"id"`
	Name        string             `json:"name"`
	Version     string             `json:"version"`
	Description string             `json:"description"`
	Author      string             `json:"author"`
	Path        string             `json:"path"`
	Hash        string             `json:"hash"` // SHA256 of the .wasm binary
	Status      PluginStatus       `json:"status"`
	Permissions []PluginPermission `json:"permissions"`
	Config      map[string]string  `json:"config"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
	LoadedAt    *time.Time         `json:"loaded_at,omitempty"`
	Error       string             `json:"error,omitempty"`
}

// NewPlugin creates a new plugin with default values.
func NewPlugin(name, version, path string) *Plugin {
	now := time.Now()
	return &Plugin{
		ID:          uuid.Must(uuid.NewV7()),
		Name:        name,
		Version:     version,
		Path:        path,
		Status:      PluginStatusInactive,
		Permissions: []PluginPermission{},
		Config:      make(map[string]string),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// HasPermission checks if the plugin has a specific permission.
func (p *Plugin) HasPermission(perm PluginPermission) bool {
	for _, permission := range p.Permissions {
		if permission == perm {
			return true
		}
	}
	return false
}

// MarkLoaded marks the plugin as successfully loaded.
func (p *Plugin) MarkLoaded() {
	now := time.Now()
	p.Status = PluginStatusActive
	p.LoadedAt = &now
	p.UpdatedAt = now
	p.Error = ""
}

// MarkError marks the plugin as having an error.
func (p *Plugin) MarkError(err error) {
	p.Status = PluginStatusError
	p.Error = err.Error()
	p.UpdatedAt = time.Now()
}

// PluginManifest represents the plugin.yaml configuration file.
type PluginManifest struct {
	Name        string             `yaml:"name"`
	Version     string             `yaml:"version"`
	Description string             `yaml:"description"`
	Author      string             `yaml:"author"`
	Entrypoint  string             `yaml:"entrypoint"`
	Permissions []PluginPermission `yaml:"permissions"`
	Config      []PluginConfigDef  `yaml:"config"`
	Hooks       []string           `yaml:"hooks"` // e.g., "on_tick", "handle_command"
}

// PluginConfigDef represents a configuration option definition.
type PluginConfigDef struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"` // string, int, bool
	Default     string `yaml:"default"`
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
}

