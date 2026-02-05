package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/forge-platform/forge/internal/core/domain"
	"github.com/forge-platform/forge/internal/core/ports"
	"github.com/google/uuid"
)

// Mock implementations
type mockTaskRepository struct {
	tasks       map[uuid.UUID]*domain.Task
	createError error
	getError    error
	claimError  error
}

func newMockTaskRepository() *mockTaskRepository {
	return &mockTaskRepository{
		tasks: make(map[uuid.UUID]*domain.Task),
	}
}

func (m *mockTaskRepository) Create(_ context.Context, task *domain.Task) error {
	if m.createError != nil {
		return m.createError
	}
	m.tasks[task.ID] = task
	return nil
}

func (m *mockTaskRepository) GetByID(_ context.Context, id uuid.UUID) (*domain.Task, error) {
	if m.getError != nil {
		return nil, m.getError
	}
	task, ok := m.tasks[id]
	if !ok {
		return nil, errors.New("task not found")
	}
	return task, nil
}

func (m *mockTaskRepository) Update(_ context.Context, task *domain.Task) error {
	m.tasks[task.ID] = task
	return nil
}

func (m *mockTaskRepository) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.tasks, id)
	return nil
}

func (m *mockTaskRepository) List(_ context.Context, _ ports.TaskFilter) ([]*domain.Task, error) {
	result := make([]*domain.Task, 0, len(m.tasks))
	for _, task := range m.tasks {
		result = append(result, task)
	}
	return result, nil
}

func (m *mockTaskRepository) ClaimNext(_ context.Context, lockDuration time.Duration) (*domain.Task, error) {
	if m.claimError != nil {
		return nil, m.claimError
	}
	for _, task := range m.tasks {
		if task.Status == domain.TaskStatusPending {
			task.MarkRunning(lockDuration)
			return task, nil
		}
	}
	return nil, nil
}

func (m *mockTaskRepository) ReleaseExpired(_ context.Context) (int64, error) {
	return 0, nil
}

// Tests
func TestNewTaskService(t *testing.T) {
	repo := newMockTaskRepository()
	logger := &mockLogger{}

	svc := NewTaskService(repo, logger)

	if svc == nil {
		t.Fatal("NewTaskService returned nil")
	}
	if svc.handlers == nil {
		t.Error("handlers map is nil")
	}
}

func TestTaskService_CreateTask(t *testing.T) {
	repo := newMockTaskRepository()
	logger := &mockLogger{}
	svc := NewTaskService(repo, logger)

	payload := map[string]interface{}{"key": "value"}
	task, err := svc.CreateTask(context.Background(), domain.TaskTypeMetricIngest, payload)

	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if task == nil {
		t.Fatal("Task is nil")
	}
	if task.Type != domain.TaskTypeMetricIngest {
		t.Errorf("Type = %v, want metric_export", task.Type)
	}
	if len(repo.tasks) != 1 {
		t.Errorf("Task not saved to repo")
	}
}

func TestTaskService_CreateTask_RepoError(t *testing.T) {
	repo := newMockTaskRepository()
	repo.createError = errors.New("database error")
	logger := &mockLogger{}
	svc := NewTaskService(repo, logger)

	_, err := svc.CreateTask(context.Background(), domain.TaskTypeMetricIngest, nil)

	if err == nil {
		t.Error("Expected error when repo fails")
	}
}

func TestTaskService_GetTask(t *testing.T) {
	repo := newMockTaskRepository()
	logger := &mockLogger{}
	svc := NewTaskService(repo, logger)

	created, _ := svc.CreateTask(context.Background(), domain.TaskTypeMetricIngest, nil)

	task, err := svc.GetTask(context.Background(), created.ID)

	if err != nil {
		t.Fatalf("GetTask error: %v", err)
	}
	if task.ID != created.ID {
		t.Error("Task ID mismatch")
	}
}

func TestTaskService_ListTasks(t *testing.T) {
	repo := newMockTaskRepository()
	logger := &mockLogger{}
	svc := NewTaskService(repo, logger)

	svc.CreateTask(context.Background(), domain.TaskTypeMetricIngest, nil)
	svc.CreateTask(context.Background(), domain.TaskTypeAIAnalysis, nil)

	tasks, err := svc.ListTasks(context.Background(), ports.TaskFilter{})

	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("Tasks count = %d, want 2", len(tasks))
	}
}

