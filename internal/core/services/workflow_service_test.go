// Package services implements core business logic services.
package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/forge-platform/forge/internal/core/domain"
	"github.com/forge-platform/forge/internal/core/ports"
	"github.com/google/uuid"
)

// mockWorkflowLogger for testing
type mockWorkflowLogger struct{}

func (m *mockWorkflowLogger) Debug(msg string, args ...interface{}) {}
func (m *mockWorkflowLogger) Info(msg string, args ...interface{})  {}
func (m *mockWorkflowLogger) Warn(msg string, args ...interface{})  {}
func (m *mockWorkflowLogger) Error(msg string, args ...interface{}) {}
func (m *mockWorkflowLogger) With(args ...interface{}) ports.Logger { return m }

// mockWorkflowRepository for testing
type mockWorkflowRepository struct {
	workflows map[uuid.UUID]*domain.Workflow
}

func newMockWorkflowRepository() *mockWorkflowRepository {
	return &mockWorkflowRepository{
		workflows: make(map[uuid.UUID]*domain.Workflow),
	}
}

func (m *mockWorkflowRepository) Create(ctx context.Context, w *domain.Workflow) error {
	m.workflows[w.ID] = w
	return nil
}

func (m *mockWorkflowRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Workflow, error) {
	return m.workflows[id], nil
}

func (m *mockWorkflowRepository) GetByName(ctx context.Context, name string) (*domain.Workflow, error) {
	for _, w := range m.workflows {
		if w.Name == name {
			return w, nil
		}
	}
	return nil, nil
}

func (m *mockWorkflowRepository) List(ctx context.Context) ([]*domain.Workflow, error) {
	result := make([]*domain.Workflow, 0, len(m.workflows))
	for _, w := range m.workflows {
		result = append(result, w)
	}
	return result, nil
}

func (m *mockWorkflowRepository) Update(ctx context.Context, w *domain.Workflow) error {
	m.workflows[w.ID] = w
	return nil
}

func (m *mockWorkflowRepository) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.workflows, id)
	return nil
}

// mockWorkflowExecutionRepository for testing
type mockWorkflowExecutionRepository struct {
	executions map[uuid.UUID]*domain.WorkflowExecution
}

func newMockWorkflowExecutionRepository() *mockWorkflowExecutionRepository {
	return &mockWorkflowExecutionRepository{
		executions: make(map[uuid.UUID]*domain.WorkflowExecution),
	}
}

func (m *mockWorkflowExecutionRepository) Create(ctx context.Context, e *domain.WorkflowExecution) error {
	m.executions[e.ID] = e
	return nil
}

func (m *mockWorkflowExecutionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.WorkflowExecution, error) {
	return m.executions[id], nil
}

func (m *mockWorkflowExecutionRepository) List(ctx context.Context, filter ports.ExecutionFilter) ([]*domain.WorkflowExecution, error) {
	result := make([]*domain.WorkflowExecution, 0)
	for _, e := range m.executions {
		if filter.WorkflowID != nil && e.WorkflowID == *filter.WorkflowID {
			result = append(result, e)
		}
	}
	return result, nil
}

func (m *mockWorkflowExecutionRepository) Update(ctx context.Context, e *domain.WorkflowExecution) error {
	m.executions[e.ID] = e
	return nil
}

func (m *mockWorkflowExecutionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.executions, id)
	return nil
}

func (m *mockWorkflowExecutionRepository) GetLatestByWorkflow(ctx context.Context, workflowID uuid.UUID) (*domain.WorkflowExecution, error) {
	for _, e := range m.executions {
		if e.WorkflowID == workflowID {
			return e, nil
		}
	}
	return nil, nil
}

func (m *mockWorkflowExecutionRepository) SaveCheckpoint(ctx context.Context, executionID uuid.UUID, checkpoint []byte) error {
	return nil
}

func (m *mockWorkflowExecutionRepository) LoadCheckpoint(ctx context.Context, executionID uuid.UUID) ([]byte, error) {
	return nil, nil
}

func TestDefaultWorkflowConfig(t *testing.T) {
	config := DefaultWorkflowConfig()

	if config.DefaultTimeout != 5*time.Minute {
		t.Errorf("expected default timeout 5m, got %v", config.DefaultTimeout)
	}
	if config.DefaultRetries != 3 {
		t.Errorf("expected default retries 3, got %d", config.DefaultRetries)
	}
	if config.DefaultRetryDelay != 5*time.Second {
		t.Errorf("expected default retry delay 5s, got %v", config.DefaultRetryDelay)
	}
}

func TestNewWorkflowService(t *testing.T) {
	logger := &mockWorkflowLogger{}
	workflowRepo := newMockWorkflowRepository()
	executionRepo := newMockWorkflowExecutionRepository()

	svc := NewWorkflowService(workflowRepo, executionRepo, logger)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.workflowRepo == nil {
		t.Error("workflow repo not set correctly")
	}
	if svc.executionRepo == nil {
		t.Error("execution repo not set correctly")
	}
	if svc.logger == nil {
		t.Error("logger not set correctly")
	}
}

func TestWorkflowService_RegisterAction(t *testing.T) {
	logger := &mockWorkflowLogger{}
	svc := NewWorkflowService(nil, nil, logger)

	// Create a mock action
	mockAction := &mockStepAction{output: map[string]interface{}{"result": "ok"}}

	svc.RegisterAction(domain.StepTypeShell, mockAction)

	// Verify action was registered
	if svc.actions[domain.StepTypeShell] != mockAction {
		t.Error("action not registered correctly")
	}
}

// mockStepAction for testing
type mockStepAction struct {
	output map[string]interface{}
	err    error
}

func (m *mockStepAction) Execute(ctx context.Context, step *domain.WorkflowStep, input map[string]interface{}) (map[string]interface{}, error) {
	return m.output, m.err
}

func TestWorkflowService_LoadFromFile(t *testing.T) {
	logger := &mockWorkflowLogger{}
	svc := NewWorkflowService(nil, nil, logger)

	// Create a temp workflow file
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "test-workflow.yaml")

	workflowContent := `
name: test-workflow
description: A test workflow
version: "1.0.0"
steps:
  - id: step1
    name: First Step
    type: shell
    config:
      command: echo hello
  - id: step2
    name: Second Step
    type: shell
    config:
      command: echo world
    depends_on:
      - step1
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("failed to write workflow file: %v", err)
	}

	workflow, err := svc.LoadFromFile(context.Background(), workflowPath)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if workflow.Name != "test-workflow" {
		t.Errorf("expected name 'test-workflow', got '%s'", workflow.Name)
	}
	if workflow.Description != "A test workflow" {
		t.Errorf("expected description 'A test workflow', got '%s'", workflow.Description)
	}
	if len(workflow.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(workflow.Steps))
	}
}

func TestWorkflowService_LoadFromFile_NotFound(t *testing.T) {
	logger := &mockWorkflowLogger{}
	svc := NewWorkflowService(nil, nil, logger)

	_, err := svc.LoadFromFile(context.Background(), "/nonexistent/path.yaml")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestWorkflowService_LoadFromFile_InvalidYAML(t *testing.T) {
	logger := &mockWorkflowLogger{}
	svc := NewWorkflowService(nil, nil, logger)

	// Create a temp file with invalid YAML
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "invalid.yaml")

	if err := os.WriteFile(workflowPath, []byte("::invalid::yaml::"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	_, err := svc.LoadFromFile(context.Background(), workflowPath)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestWorkflowService_LoadFromFile_MissingName(t *testing.T) {
	logger := &mockWorkflowLogger{}
	svc := NewWorkflowService(nil, nil, logger)

	// Create a temp file without name
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "noname.yaml")

	if err := os.WriteFile(workflowPath, []byte("description: no name"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	_, err := svc.LoadFromFile(context.Background(), workflowPath)
	if err == nil {
		t.Error("expected error for missing name")
	}
}

