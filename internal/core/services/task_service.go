// Package services implements the application layer business logic.
package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/forge-platform/forge/internal/core/domain"
	"github.com/forge-platform/forge/internal/core/ports"
	"github.com/google/uuid"
)

// TaskService handles task queue operations.
type TaskService struct {
	repo       ports.TaskRepository
	logger     ports.Logger
	handlers   map[domain.TaskType]TaskHandler
	handlersMu sync.RWMutex
	workerWg   sync.WaitGroup
	stopCh     chan struct{}
}

// TaskHandler is a function that processes a task.
type TaskHandler func(ctx context.Context, task *domain.Task) error

// NewTaskService creates a new task service.
func NewTaskService(repo ports.TaskRepository, logger ports.Logger) *TaskService {
	return &TaskService{
		repo:     repo,
		logger:   logger,
		handlers: make(map[domain.TaskType]TaskHandler),
		stopCh:   make(chan struct{}),
	}
}

// RegisterHandler registers a handler for a task type.
func (s *TaskService) RegisterHandler(taskType domain.TaskType, handler TaskHandler) {
	s.handlersMu.Lock()
	defer s.handlersMu.Unlock()
	s.handlers[taskType] = handler
}

// CreateTask creates a new task in the queue.
func (s *TaskService) CreateTask(ctx context.Context, taskType domain.TaskType, payload map[string]interface{}) (*domain.Task, error) {
	task := domain.NewTask(taskType, payload)

	if err := s.repo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	s.logger.Info("Task created", "id", task.ID, "type", taskType)
	return task, nil
}

// GetTask retrieves a task by ID.
func (s *TaskService) GetTask(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	return s.repo.GetByID(ctx, id)
}

// ListTasks lists tasks with optional filtering.
func (s *TaskService) ListTasks(ctx context.Context, filter ports.TaskFilter) ([]*domain.Task, error) {
	return s.repo.List(ctx, filter)
}

// CancelTask cancels a pending task.
func (s *TaskService) CancelTask(ctx context.Context, id uuid.UUID) error {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if task.Status != domain.TaskStatusPending {
		return fmt.Errorf("can only cancel pending tasks")
	}

	task.Status = domain.TaskStatusDead
	task.Error = "cancelled by user"
	task.UpdatedAt = time.Now()

	return s.repo.Update(ctx, task)
}

// StartWorkers starts the task processing workers.
func (s *TaskService) StartWorkers(ctx context.Context, numWorkers int) {
	for i := 0; i < numWorkers; i++ {
		s.workerWg.Add(1)
		go s.worker(ctx, i)
	}

	// Start expired lock releaser
	go s.releaseExpiredLocks(ctx)
}

// StopWorkers stops all workers gracefully.
func (s *TaskService) StopWorkers() {
	close(s.stopCh)
	s.workerWg.Wait()
}

// worker is the main worker loop.
func (s *TaskService) worker(ctx context.Context, id int) {
	defer s.workerWg.Done()

	s.logger.Debug("Worker started", "worker_id", id)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.processNextTask(ctx)
		}
	}
}

// processNextTask claims and processes the next available task.
func (s *TaskService) processNextTask(ctx context.Context) {
	// Claim a task with 5-minute lock
	task, err := s.repo.ClaimNext(ctx, 5*time.Minute)
	if err != nil {
		s.logger.Error("Failed to claim task", "error", err)
		return
	}

	if task == nil {
		return // No tasks available
	}

	s.logger.Debug("Processing task", "id", task.ID, "type", task.Type)

	// Get handler
	s.handlersMu.RLock()
	handler, ok := s.handlers[task.Type]
	s.handlersMu.RUnlock()

	if !ok {
		s.logger.Error("No handler for task type", "type", task.Type)
		task.MarkFailed(fmt.Errorf("no handler for task type: %s", task.Type))
		_ = s.repo.Update(ctx, task)
		return
	}

	// Execute handler
	if err := handler(ctx, task); err != nil {
		s.logger.Error("Task failed", "id", task.ID, "error", err)
		task.MarkFailed(err)
	} else {
		s.logger.Info("Task completed", "id", task.ID)
		task.MarkCompleted()
	}

	_ = s.repo.Update(ctx, task)
}

// releaseExpiredLocks periodically releases expired task locks.
func (s *TaskService) releaseExpiredLocks(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			released, err := s.repo.ReleaseExpired(ctx)
			if err != nil {
				s.logger.Error("Failed to release expired locks", "error", err)
			} else if released > 0 {
				s.logger.Info("Released expired task locks", "count", released)
			}
		}
	}
}

