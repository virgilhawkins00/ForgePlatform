// Package services implements core business logic services.
package services

import (
	"context"
	"testing"
	"time"

	"github.com/forge-platform/forge/internal/core/domain"
)

func TestNewShellAction(t *testing.T) {
	action := NewShellAction("/tmp")

	if action == nil {
		t.Fatal("expected non-nil shell action")
	}
	if action.workDir != "/tmp" {
		t.Errorf("expected workDir '/tmp', got '%s'", action.workDir)
	}
}

func TestShellAction_Execute(t *testing.T) {
	action := NewShellAction("")

	step := &domain.WorkflowStep{
		ID:     "step-1",
		Name:   "test-shell",
		Type:   domain.StepTypeShell,
		Config: map[string]interface{}{"command": "echo hello"},
	}

	result, err := action.Execute(context.Background(), step, nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	stdout, ok := result["stdout"].(string)
	if !ok {
		t.Fatal("expected stdout in result")
	}
	if stdout != "hello\n" {
		t.Errorf("expected 'hello\\n', got %q", stdout)
	}

	exitCode, ok := result["exit_code"].(int)
	if !ok {
		t.Fatal("expected exit_code in result")
	}
	if exitCode != 0 {
		t.Errorf("expected exit_code 0, got %d", exitCode)
	}
}

func TestShellAction_Execute_WithVariables(t *testing.T) {
	action := NewShellAction("")

	step := &domain.WorkflowStep{
		ID:     "step-2",
		Name:   "test-shell-vars",
		Type:   domain.StepTypeShell,
		Config: map[string]interface{}{"command": "echo ${message}"},
	}

	input := map[string]interface{}{"message": "world"}
	result, err := action.Execute(context.Background(), step, input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	stdout := result["stdout"].(string)
	if stdout != "world\n" {
		t.Errorf("expected 'world\\n', got %q", stdout)
	}
}

func TestShellAction_Execute_NoCommand(t *testing.T) {
	action := NewShellAction("")

	step := &domain.WorkflowStep{
		ID:     "step-3",
		Name:   "test-no-command",
		Type:   domain.StepTypeShell,
		Config: map[string]interface{}{},
	}

	_, err := action.Execute(context.Background(), step, nil)
	if err == nil {
		t.Error("expected error for missing command")
	}
}

func TestNewHTTPAction(t *testing.T) {
	action := NewHTTPAction(30 * time.Second)

	if action == nil {
		t.Fatal("expected non-nil HTTP action")
	}
	if action.client == nil {
		t.Error("expected non-nil HTTP client")
	}
	if action.client.Timeout != 30*time.Second {
		t.Errorf("expected timeout 30s, got %v", action.client.Timeout)
	}
}

func TestNewPluginAction(t *testing.T) {
	runner := func(ctx context.Context, pluginName string, input map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"result": "ok"}, nil
	}
	action := NewPluginAction(runner)

	if action == nil {
		t.Fatal("expected non-nil plugin action")
	}
}

func TestPluginAction_Execute(t *testing.T) {
	runner := func(ctx context.Context, pluginName string, input map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"result": "executed", "plugin": pluginName}, nil
	}
	action := NewPluginAction(runner)

	step := &domain.WorkflowStep{
		ID:     "plugin-step-1",
		Name:   "test-plugin",
		Type:   domain.StepTypePlugin,
		Config: map[string]interface{}{
			"plugin": "my-plugin",
		},
	}

	result, err := action.Execute(context.Background(), step, nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

