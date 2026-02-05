// Package services implements core business logic services.
package services

import (
	"context"
	"testing"

	"github.com/forge-platform/forge/internal/core/domain"
	"github.com/forge-platform/forge/internal/core/ports"
)

// mockAgentLogger for testing
type mockAgentLogger struct{}

func (m *mockAgentLogger) Debug(msg string, args ...interface{}) {}
func (m *mockAgentLogger) Info(msg string, args ...interface{})  {}
func (m *mockAgentLogger) Warn(msg string, args ...interface{})  {}
func (m *mockAgentLogger) Error(msg string, args ...interface{}) {}
func (m *mockAgentLogger) With(args ...interface{}) ports.Logger { return m }

// mockAIProvider for testing
type mockAIProvider struct{}

func (m *mockAIProvider) Chat(ctx context.Context, conv *domain.Conversation) (*domain.Message, error) {
	return &domain.Message{
		Role:    domain.RoleAssistant,
		Content: "I'll help you with that.",
	}, nil
}

func (m *mockAIProvider) ChatStream(ctx context.Context, conv *domain.Conversation, callback func(chunk string)) (*domain.Message, error) {
	callback("Hello")
	return &domain.Message{
		Role:    domain.RoleAssistant,
		Content: "Hello",
	}, nil
}

func (m *mockAIProvider) ListModels(ctx context.Context) ([]string, error) {
	return []string{"test-model"}, nil
}

func (m *mockAIProvider) GetModel() string {
	return "test-model"
}

func (m *mockAIProvider) SetModel(model string) {}

// mockToolRegistry for testing
type mockToolRegistry struct {
	tools map[string]ports.AITool
}

func newMockToolRegistry() *mockToolRegistry {
	return &mockToolRegistry{
		tools: make(map[string]ports.AITool),
	}
}

func (m *mockToolRegistry) RegisterTool(tool ports.AITool) error {
	m.tools[tool.Name] = tool
	return nil
}

func (m *mockToolRegistry) GetTool(name string) (*ports.AITool, bool) {
	tool, ok := m.tools[name]
	if !ok {
		return nil, false
	}
	return &tool, true
}

func (m *mockToolRegistry) ListTools() []ports.AITool {
	tools := make([]ports.AITool, 0, len(m.tools))
	for _, t := range m.tools {
		tools = append(tools, t)
	}
	return tools
}

func (m *mockToolRegistry) ExecuteTool(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	return "executed", nil
}

func TestDefaultAgentConfig(t *testing.T) {
	cfg := DefaultAgentConfig()

	if cfg.MaxSteps != 10 {
		t.Errorf("expected MaxSteps 10, got %d", cfg.MaxSteps)
	}
	if !cfg.RequireConfirm {
		t.Error("expected RequireConfirm to be true")
	}
	if cfg.ConfirmFn != nil {
		t.Error("expected ConfirmFn to be nil")
	}
}

func TestNewAgentService(t *testing.T) {
	logger := &mockAgentLogger{}
	provider := &mockAIProvider{}
	registry := newMockToolRegistry()
	config := DefaultAgentConfig()

	svc := NewAgentService(provider, registry, logger, config)

	if svc == nil {
		t.Fatal("expected non-nil agent service")
	}
	if svc.maxSteps != 10 {
		t.Errorf("expected maxSteps 10, got %d", svc.maxSteps)
	}
	if svc.aiProvider == nil {
		t.Error("aiProvider not set")
	}
	if svc.toolRegistry == nil {
		t.Error("toolRegistry not set")
	}
}

func TestNewAgentService_DefaultMaxSteps(t *testing.T) {
	logger := &mockAgentLogger{}
	provider := &mockAIProvider{}
	registry := newMockToolRegistry()
	config := AgentConfig{MaxSteps: 0}

	svc := NewAgentService(provider, registry, logger, config)

	// Should default to 10 when 0 is provided
	if svc.maxSteps != 10 {
		t.Errorf("expected default maxSteps 10, got %d", svc.maxSteps)
	}
}

