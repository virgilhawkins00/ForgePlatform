package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/forge-platform/forge/internal/core/ports"
)

// DaemonCaller is an interface for calling daemon RPC methods.
type DaemonCaller interface {
	Call(ctx context.Context, method string, params map[string]interface{}) (interface{}, error)
}

// ToolRegistry manages AI tools.
type ToolRegistry struct {
	tools        map[string]ports.AITool
	mu           sync.RWMutex
	daemonClient DaemonCaller
}

// NewToolRegistry creates a new tool registry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]ports.AITool),
	}
}

// SetDaemonClient sets the daemon client for tool execution.
func (r *ToolRegistry) SetDaemonClient(client DaemonCaller) {
	r.daemonClient = client
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
			if r.daemonClient == nil {
				return "Daemon not connected. Cannot list metrics.", nil
			}
			resp, err := r.daemonClient.Call(ctx, "metric.list", nil)
			if err != nil {
				return fmt.Sprintf("Error listing metrics: %v", err), nil
			}
			data, _ := json.Marshal(resp)
			return string(data), nil
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
			if r.daemonClient == nil {
				return "Daemon not connected. Cannot retrieve logs.", nil
			}
			params := map[string]interface{}{"limit": 20}
			if level, ok := args["level"].(string); ok {
				params["level"] = level
			}
			if limit, ok := args["limit"].(float64); ok {
				params["limit"] = int(limit)
			}
			resp, err := r.daemonClient.Call(ctx, "log.list", params)
			if err != nil {
				return fmt.Sprintf("Error retrieving logs: %v", err), nil
			}
			data, _ := json.Marshal(resp)
			return string(data), nil
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
			if r.daemonClient == nil {
				return "Daemon not connected. Cannot list tasks.", nil
			}
			resp, err := r.daemonClient.Call(ctx, "task.list", args)
			if err != nil {
				return fmt.Sprintf("Error listing tasks: %v", err), nil
			}
			data, _ := json.Marshal(resp)
			return string(data), nil
		},
	})

	// List plugins tool
	_ = r.RegisterTool(ports.AITool{
		Name:        "list_plugins",
		Description: "List installed WebAssembly plugins",
		Parameters:  map[string]ports.AIToolParameter{},
		Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
			if r.daemonClient == nil {
				return "Daemon not connected. Cannot list plugins.", nil
			}
			resp, err := r.daemonClient.Call(ctx, "plugin.list", nil)
			if err != nil {
				return fmt.Sprintf("Error listing plugins: %v", err), nil
			}
			data, _ := json.Marshal(resp)
			return string(data), nil
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
			if r.daemonClient == nil {
				return fmt.Sprintf("Daemon not connected. Cannot restart plugin %s.", pluginID), nil
			}
			_, err := r.daemonClient.Call(ctx, "plugin.restart", map[string]interface{}{"name": pluginID})
			if err != nil {
				return fmt.Sprintf("Error restarting plugin: %v", err), nil
			}
			return fmt.Sprintf("Plugin %s restarted successfully", pluginID), nil
		},
	})
}

var _ ports.AIToolRegistry = (*ToolRegistry)(nil)

