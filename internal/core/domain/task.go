// Package domain contains the core business entities of the Forge Platform.
// These entities are pure and have no knowledge of persistence or presentation.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// TaskStatus represents the current state of a task in the queue.
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "PENDING"
	TaskStatusRunning   TaskStatus = "RUNNING"
	TaskStatusCompleted TaskStatus = "COMPLETED"
	TaskStatusFailed    TaskStatus = "FAILED"
	TaskStatusDead      TaskStatus = "DEAD"
)

// TaskType represents the type of task to be executed.
type TaskType string

const (
	TaskTypeAIAnalysis   TaskType = "ai_analysis"
	TaskTypeMetricIngest TaskType = "metric_ingest"
	TaskTypePluginExec   TaskType = "plugin_exec"
	TaskTypeMaintenance  TaskType = "maintenance"
	TaskTypeDownsample   TaskType = "downsample"
)

// Task represents a durable task in the execution queue.
// Tasks are persisted to SQLite for durability across restarts.
type Task struct {
	ID          uuid.UUID              `json:"id"`
	Type        TaskType               `json:"type"`
	Payload     map[string]interface{} `json:"payload"`
	Status      TaskStatus             `json:"status"`
	Priority    int                    `json:"priority"`
	MaxRetries  int                    `json:"max_retries"`
	RetryCount  int                    `json:"retry_count"`
	RunAt       time.Time              `json:"run_at"`
	LockedUntil *time.Time             `json:"locked_until,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// NewTask creates a new task with default values.
func NewTask(taskType TaskType, payload map[string]interface{}) *Task {
	now := time.Now()
	return &Task{
		ID:         uuid.Must(uuid.NewV7()),
		Type:       taskType,
		Payload:    payload,
		Status:     TaskStatusPending,
		Priority:   0,
		MaxRetries: 3,
		RetryCount: 0,
		RunAt:      now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// CanRetry checks if the task can be retried.
func (t *Task) CanRetry() bool {
	return t.RetryCount < t.MaxRetries
}

// MarkRunning marks the task as running with a lock timeout.
func (t *Task) MarkRunning(lockDuration time.Duration) {
	now := time.Now()
	lockedUntil := now.Add(lockDuration)
	t.Status = TaskStatusRunning
	t.LockedUntil = &lockedUntil
	t.UpdatedAt = now
}

// MarkCompleted marks the task as completed.
func (t *Task) MarkCompleted() {
	now := time.Now()
	t.Status = TaskStatusCompleted
	t.CompletedAt = &now
	t.LockedUntil = nil
	t.UpdatedAt = now
}

// MarkFailed marks the task as failed with an error message.
func (t *Task) MarkFailed(err error) {
	now := time.Now()
	t.RetryCount++
	t.Error = err.Error()
	t.LockedUntil = nil
	t.UpdatedAt = now

	if t.CanRetry() {
		t.Status = TaskStatusPending
		t.RunAt = now.Add(time.Duration(t.RetryCount) * time.Minute) // Exponential backoff
	} else {
		t.Status = TaskStatusDead
	}
}

// IsLocked checks if the task is currently locked by a worker.
func (t *Task) IsLocked() bool {
	if t.LockedUntil == nil {
		return false
	}
	return time.Now().Before(*t.LockedUntil)
}

