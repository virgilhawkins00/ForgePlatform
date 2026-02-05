package ai

import (
	"context"
	"fmt"
	"sync"

	"github.com/forge-platform/forge/internal/core/ports"
)

// ToolRegistry manages AI tools.
type ToolRegistry struct {
	tools map[string]ports.AITool
	mu    sync.RWMutex
}

// NewToolRegistry creates a new tool registry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]ports.AITool),
	}
}

// RegisterTool registers a new tool.
func (r *ToolRegistry) RegisterTool(tool ports.AITool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[tool.Name]; exists {
		return fmt.Errorf("tool already registered: %s", tool.Name)
	}

	r.tools[tool.Name] = tool
	return nil
}

// GetTool retrieves a tool by name.
func (r *ToolRegistry) GetTool(name string) (*ports.AITool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, ok := r.tools[name]
	if !ok {
		return nil, false
	}
	return &tool, true
}

// ListTools returns all registered tools.
func (r *ToolRegistry) ListTools() []ports.AITool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]ports.AITool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// ExecuteTool executes a tool by name.
func (r *ToolRegistry) ExecuteTool(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	r.mu.RLock()
	tool, ok := r.tools[name]
	r.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("tool not found: %s", name)
	}

	if tool.Handler == nil {
		return "", fmt.Errorf("tool has no handler: %s", name)
	}

	return tool.Handler(ctx, args)
}

// RegisterDefaultTools registers the default Forge tools.
func (r *ToolRegistry) RegisterDefaultTools() {
	// List metrics tool
	_ = r.RegisterTool(ports.AITool{
		Name:        "list_metrics",
		Description: "List available metrics from the time-series database",
		Parameters: map[string]ports.AIToolParameter{
			"name_filter": {
				Type:        "string",
				Description: "Optional filter for metric names",
				Required:    false,
			},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
			// TODO: Implement actual metric listing
			return "Available metrics: cpu_usage, memory_usage, disk_io", nil
		},
	})

	// Get logs tool
	_ = r.RegisterTool(ports.AITool{
		Name:        "get_logs",
		Description: "Retrieve logs from the system",
		Parameters: map[string]ports.AIToolParameter{
			"level": {
				Type:        "string",
				Description: "Log level filter (debug, info, warn, error)",
				Required:    false,
				Enum:        []string{"debug", "info", "warn", "error"},
			},
			"limit": {
				Type:        "integer",
				Description: "Maximum number of logs to return",
				Required:    false,
			},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
			// TODO: Implement actual log retrieval
			return "No recent logs found", nil
		},
	})

	// List tasks tool
	_ = r.RegisterTool(ports.AITool{
		Name:        "list_tasks",
		Description: "List tasks in the execution queue",
		Parameters: map[string]ports.AIToolParameter{
			"status": {
				Type:        "string",
				Description: "Filter by task status",
				Required:    false,
				Enum:        []string{"PENDING", "RUNNING", "COMPLETED", "FAILED"},
			},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
			// TODO: Implement actual task listing
			return "No tasks in queue", nil
		},
	})

	// List plugins tool
	_ = r.RegisterTool(ports.AITool{
		Name:        "list_plugins",
		Description: "List installed WebAssembly plugins",
		Parameters:  map[string]ports.AIToolParameter{},
		Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
			// TODO: Implement actual plugin listing
			return "No plugins installed", nil
		},
	})

	// Restart plugin tool
	_ = r.RegisterTool(ports.AITool{
		Name:        "restart_plugin",
		Description: "Restart a WebAssembly plugin",
		Parameters: map[string]ports.AIToolParameter{
			"plugin_id": {
				Type:        "string",
				Description: "The ID of the plugin to restart",
				Required:    true,
			},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
			pluginID, ok := args["plugin_id"].(string)
			if !ok {
				return "", fmt.Errorf("plugin_id is required")
			}
			// TODO: Implement actual plugin restart
			return fmt.Sprintf("Plugin %s restarted", pluginID), nil
		},
	})
}

var _ ports.AIToolRegistry = (*ToolRegistry)(nil)

