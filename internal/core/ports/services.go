package ports

import (
	"context"

	"github.com/forge-platform/forge/internal/core/domain"
)

// WasmRuntime defines the interface for WebAssembly plugin execution.
type WasmRuntime interface {
	// LoadPlugin loads a plugin from the given path.
	LoadPlugin(ctx context.Context, plugin *domain.Plugin) error

	// UnloadPlugin unloads a plugin from the runtime.
	UnloadPlugin(ctx context.Context, pluginID string) error

	// CallFunction invokes a function exported by a plugin.
	CallFunction(ctx context.Context, pluginID, funcName string, args ...interface{}) (interface{}, error)

	// ListLoadedPlugins returns the IDs of all loaded plugins.
	ListLoadedPlugins() []string

	// Close shuts down the runtime and releases resources.
	Close() error
}

// AIProvider defines the interface for AI/LLM interactions.
type AIProvider interface {
	// Chat sends a conversation to the LLM and returns the response.
	Chat(ctx context.Context, conv *domain.Conversation) (*domain.Message, error)

	// ChatStream sends a conversation and streams the response.
	ChatStream(ctx context.Context, conv *domain.Conversation, callback func(chunk string)) (*domain.Message, error)

	// ListModels returns available models.
	ListModels(ctx context.Context) ([]string, error)

	// GetModel returns the current model name.
	GetModel() string

	// SetModel sets the model to use.
	SetModel(model string)
}

// AITool defines a tool that the AI can invoke.
type AITool struct {
	Name        string
	Description string
	Parameters  map[string]AIToolParameter
	Handler     func(ctx context.Context, args map[string]interface{}) (string, error)
}

// AIToolParameter defines a parameter for an AI tool.
type AIToolParameter struct {
	Type        string
	Description string
	Required    bool
	Enum        []string
}

// AIToolRegistry manages tools available to the AI.
type AIToolRegistry interface {
	// RegisterTool registers a new tool.
	RegisterTool(tool AITool) error

	// GetTool retrieves a tool by name.
	GetTool(name string) (*AITool, bool)

	// ListTools returns all registered tools.
	ListTools() []AITool

	// ExecuteTool executes a tool by name with the given arguments.
	ExecuteTool(ctx context.Context, name string, args map[string]interface{}) (string, error)
}

// DaemonService defines the interface for the background daemon.
type DaemonService interface {
	// Start starts the daemon.
	Start(ctx context.Context) error

	// Stop gracefully stops the daemon.
	Stop(ctx context.Context) error

	// IsRunning checks if the daemon is running.
	IsRunning() bool

	// GetStatus returns the daemon status.
	GetStatus() DaemonStatus
}

// DaemonStatus represents the current state of the daemon.
type DaemonStatus struct {
	Running       bool
	StartedAt     string
	Uptime        string
	TasksRunning  int
	PluginsLoaded int
	MetricsCount  int64
}

// Logger defines the interface for structured logging.
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	With(args ...interface{}) Logger
}

// EventBus defines the interface for internal event publishing.
type EventBus interface {
	// Publish publishes an event to all subscribers.
	Publish(ctx context.Context, event Event) error

	// Subscribe subscribes to events of a specific type.
	Subscribe(eventType string, handler EventHandler) error

	// Unsubscribe removes a subscription.
	Unsubscribe(eventType string, handler EventHandler) error
}

// Event represents an internal event.
type Event struct {
	Type      string
	Payload   interface{}
	Timestamp int64
}

// EventHandler is a function that handles events.
type EventHandler func(ctx context.Context, event Event) error

