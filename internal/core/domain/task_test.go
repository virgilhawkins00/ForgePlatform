package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNewTask(t *testing.T) {
	payload := map[string]interface{}{"command": "echo hello", "description": "Test task"}
	task := NewTask(TaskTypePluginExec, payload)

	if task.ID.String() == "" {
		t.Error("Task ID should not be empty")
	}

	if task.Type != TaskTypePluginExec {
		t.Errorf("Expected type %s, got %s", TaskTypePluginExec, task.Type)
	}

	if task.Status != TaskStatusPending {
		t.Errorf("Expected status %s, got %s", TaskStatusPending, task.Status)
	}

	if task.Payload["command"] != "echo hello" {
		t.Errorf("Expected payload command 'echo hello', got '%v'", task.Payload["command"])
	}

	if task.MaxRetries != 3 {
		t.Errorf("Expected max retries 3, got %d", task.MaxRetries)
	}
}

func TestTaskMarkRunning(t *testing.T) {
	payload := map[string]interface{}{"command": "echo hello"}
	task := NewTask(TaskTypePluginExec, payload)

	task.MarkRunning(5 * time.Minute)

	if task.Status != TaskStatusRunning {
		t.Errorf("Expected status %s, got %s", TaskStatusRunning, task.Status)
	}

	if task.LockedUntil == nil {
		t.Error("LockedUntil should not be nil")
	}
}

func TestTaskMarkCompleted(t *testing.T) {
	payload := map[string]interface{}{"command": "echo hello"}
	task := NewTask(TaskTypePluginExec, payload)
	task.MarkRunning(5 * time.Minute)

	task.MarkCompleted()

	if task.Status != TaskStatusCompleted {
		t.Errorf("Expected status %s, got %s", TaskStatusCompleted, task.Status)
	}

	if task.CompletedAt == nil {
		t.Error("CompletedAt should not be nil")
	}

	if task.LockedUntil != nil {
		t.Error("LockedUntil should be nil after completion")
	}
}

func TestTaskMarkFailed(t *testing.T) {
	payload := map[string]interface{}{"command": "echo hello"}
	task := NewTask(TaskTypePluginExec, payload)
	task.MarkRunning(5 * time.Minute)

	task.MarkFailed(errors.New("execution failed"))

	if task.Status != TaskStatusPending {
		t.Errorf("Expected status %s after retry, got %s", TaskStatusPending, task.Status)
	}

	if task.RetryCount != 1 {
		t.Errorf("Expected retry count 1, got %d", task.RetryCount)
	}

	if task.Error != "execution failed" {
		t.Errorf("Expected error message 'execution failed', got '%s'", task.Error)
	}
}

func TestTaskMarkFailedExhausted(t *testing.T) {
	payload := map[string]interface{}{"command": "echo hello"}
	task := NewTask(TaskTypePluginExec, payload)
	task.MaxRetries = 1
	task.RetryCount = 1
	task.MarkRunning(5 * time.Minute)

	task.MarkFailed(errors.New("execution failed"))

	if task.Status != TaskStatusDead {
		t.Errorf("Expected status %s after exhausted retries, got %s", TaskStatusDead, task.Status)
	}
}

func TestTaskCanRetry(t *testing.T) {
	payload := map[string]interface{}{"command": "echo hello"}
	task := NewTask(TaskTypePluginExec, payload)
	task.MaxRetries = 3
	task.RetryCount = 2

	if !task.CanRetry() {
		t.Error("Task should be able to retry")
	}

	task.RetryCount = 3
	if task.CanRetry() {
		t.Error("Task should not be able to retry")
	}
}

func TestTaskIsLocked(t *testing.T) {
	payload := map[string]interface{}{"command": "echo hello"}
	task := NewTask(TaskTypePluginExec, payload)

	if task.IsLocked() {
		t.Error("New task should not be locked")
	}

	task.MarkRunning(5 * time.Minute)
	if !task.IsLocked() {
		t.Error("Running task should be locked")
	}

	past := time.Now().Add(-1 * time.Hour)
	task.LockedUntil = &past
	if task.IsLocked() {
		t.Error("Task with expired lock should not be locked")
	}
}

