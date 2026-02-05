// Package ai provides AI provider integrations.
package ai

import (
	"context"
	"testing"

	"github.com/forge-platform/forge/internal/core/ports"
)

func TestNewToolRegistry(t *testing.T) {
	registry := NewToolRegistry()
	if registry == nil {
		t.Fatal("expected non-nil registry")
	}
	if registry.tools == nil {
		t.Error("tools map not initialized")
	}
}

func TestToolRegistry_RegisterTool(t *testing.T) {
	registry := NewToolRegistry()

	tool := ports.AITool{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters:  map[string]ports.AIToolParameter{},
		Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
			return "test result", nil
		},
	}

	err := registry.RegisterTool(tool)
	if err != nil {
		t.Fatalf("RegisterTool failed: %v", err)
	}

	// Try to register same tool again
	err = registry.RegisterTool(tool)
	if err == nil {
		t.Error("expected error when registering duplicate tool")
	}
}

func TestToolRegistry_GetTool(t *testing.T) {
	registry := NewToolRegistry()

	tool := ports.AITool{
		Name:        "my_tool",
		Description: "My tool description",
	}
	_ = registry.RegisterTool(tool)

	// Get existing tool
	got, ok := registry.GetTool("my_tool")
	if !ok {
		t.Fatal("expected to find tool")
	}
	if got.Name != "my_tool" {
		t.Errorf("expected name 'my_tool', got '%s'", got.Name)
	}

	// Get non-existent tool
	_, ok = registry.GetTool("nonexistent")
	if ok {
		t.Error("expected not to find nonexistent tool")
	}
}

func TestToolRegistry_ListTools(t *testing.T) {
	registry := NewToolRegistry()

	// Empty list
	tools := registry.ListTools()
	if len(tools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(tools))
	}

	// Add tools
	_ = registry.RegisterTool(ports.AITool{Name: "tool1"})
	_ = registry.RegisterTool(ports.AITool{Name: "tool2"})
	_ = registry.RegisterTool(ports.AITool{Name: "tool3"})

	tools = registry.ListTools()
	if len(tools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(tools))
	}
}

func TestToolRegistry_ExecuteTool(t *testing.T) {
	registry := NewToolRegistry()
	ctx := context.Background()

	// Register tool with handler
	tool := ports.AITool{
		Name: "echo",
		Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
			msg, _ := args["message"].(string)
			return "echo: " + msg, nil
		},
	}
	_ = registry.RegisterTool(tool)

	// Execute existing tool
	result, err := registry.ExecuteTool(ctx, "echo", map[string]interface{}{"message": "hello"})
	if err != nil {
		t.Fatalf("ExecuteTool failed: %v", err)
	}
	if result != "echo: hello" {
		t.Errorf("expected 'echo: hello', got '%s'", result)
	}

	// Execute non-existent tool
	_, err = registry.ExecuteTool(ctx, "nonexistent", nil)
	if err == nil {
		t.Error("expected error for nonexistent tool")
	}

	// Register tool without handler
	_ = registry.RegisterTool(ports.AITool{Name: "no_handler"})
	_, err = registry.ExecuteTool(ctx, "no_handler", nil)
	if err == nil {
		t.Error("expected error for tool without handler")
	}
}

func TestToolRegistry_RegisterDefaultTools(t *testing.T) {
	registry := NewToolRegistry()
	registry.RegisterDefaultTools()

	// Should have default tools registered
	tools := registry.ListTools()
	if len(tools) < 3 {
		t.Errorf("expected at least 3 default tools, got %d", len(tools))
	}

	// Check specific default tools exist
	expectedTools := []string{"list_metrics", "get_logs", "list_tasks", "list_plugins", "restart_plugin"}
	for _, name := range expectedTools {
		if _, ok := registry.GetTool(name); !ok {
			t.Errorf("expected default tool '%s' to exist", name)
		}
	}
}

