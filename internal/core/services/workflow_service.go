// Package services implements core business logic services.
package services

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/forge-platform/forge/internal/core/domain"
	"github.com/forge-platform/forge/internal/core/ports"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// WorkflowService provides workflow management and execution.
type WorkflowService struct {
	workflowRepo  ports.WorkflowRepository
	executionRepo ports.WorkflowExecutionRepository
	actions       map[domain.StepType]StepAction
	logger        ports.Logger
	mu            sync.RWMutex
	running       map[uuid.UUID]context.CancelFunc // Active executions
}

// StepAction defines the interface for step execution.
type StepAction interface {
	Execute(ctx context.Context, step *domain.WorkflowStep, input map[string]interface{}) (map[string]interface{}, error)
}

// WorkflowConfig holds workflow service configuration.
type WorkflowConfig struct {
	DefaultTimeout    time.Duration
	DefaultRetries    int
	DefaultRetryDelay time.Duration
}

// DefaultWorkflowConfig returns default configuration.
func DefaultWorkflowConfig() WorkflowConfig {
	return WorkflowConfig{
		DefaultTimeout:    5 * time.Minute,
		DefaultRetries:    3,
		DefaultRetryDelay: 5 * time.Second,
	}
}

// NewWorkflowService creates a new workflow service.
func NewWorkflowService(
	workflowRepo ports.WorkflowRepository,
	executionRepo ports.WorkflowExecutionRepository,
	logger ports.Logger,
) *WorkflowService {
	return &WorkflowService{
		workflowRepo:  workflowRepo,
		executionRepo: executionRepo,
		actions:       make(map[domain.StepType]StepAction),
		logger:        logger,
		running:       make(map[uuid.UUID]context.CancelFunc),
	}
}

// RegisterAction registers a step action handler.
func (s *WorkflowService) RegisterAction(stepType domain.StepType, action StepAction) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.actions[stepType] = action
}

// LoadFromFile loads a workflow definition from a YAML file.
func (s *WorkflowService) LoadFromFile(ctx context.Context, path string) (*domain.Workflow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file: %w", err)
	}

	var workflow domain.Workflow
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		return nil, fmt.Errorf("failed to parse workflow YAML: %w", err)
	}

	// Generate ID and timestamps
	workflow.ID = uuid.Must(uuid.NewV7())
	now := time.Now()
	workflow.CreatedAt = now
	workflow.UpdatedAt = now
	workflow.Status = domain.WorkflowStatusPending

	// Validate workflow
	if err := s.validateWorkflow(&workflow); err != nil {
		return nil, err
	}

	return &workflow, nil
}

// validateWorkflow validates a workflow definition.
func (s *WorkflowService) validateWorkflow(w *domain.Workflow) error {
	if w.Name == "" {
		return fmt.Errorf("workflow name is required")
	}
	if len(w.Steps) == 0 {
		return fmt.Errorf("workflow must have at least one step")
	}

	// Check for duplicate step IDs
	stepIDs := make(map[string]bool)
	for _, step := range w.Steps {
		if step.ID == "" {
			return fmt.Errorf("step ID is required")
		}
		if stepIDs[step.ID] {
			return fmt.Errorf("duplicate step ID: %s", step.ID)
		}
		stepIDs[step.ID] = true
	}

	// Validate dependencies exist
	for _, step := range w.Steps {
		for _, depID := range step.DependsOn {
			if !stepIDs[depID] {
				return fmt.Errorf("step %s depends on unknown step: %s", step.ID, depID)
			}
		}
	}

	// Check for circular dependencies
	if err := s.checkCircularDeps(w); err != nil {
		return err
	}

	return nil
}

// checkCircularDeps detects circular dependencies in the workflow DAG.
func (s *WorkflowService) checkCircularDeps(w *domain.Workflow) error {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var dfs func(stepID string) bool
	dfs = func(stepID string) bool {
		visited[stepID] = true
		recStack[stepID] = true

		step := w.GetStep(stepID)
		if step == nil {
			return false
		}

		for _, depID := range step.DependsOn {
			if !visited[depID] {
				if dfs(depID) {
					return true
				}
			} else if recStack[depID] {
				return true
			}
		}

		recStack[stepID] = false
		return false
	}

	for _, step := range w.Steps {
		if !visited[step.ID] {
			if dfs(step.ID) {
				return fmt.Errorf("circular dependency detected involving step: %s", step.ID)
			}
		}
	}

	return nil
}

// Run executes a workflow with the given input.
func (s *WorkflowService) Run(ctx context.Context, workflow *domain.Workflow, input map[string]interface{}) (*domain.WorkflowExecution, error) {
	// Create execution instance
	execution := domain.NewWorkflowExecution(workflow, input)
	execution.Status = domain.WorkflowStatusRunning

	// Save initial execution state
	if s.executionRepo != nil {
		if err := s.executionRepo.Create(ctx, execution); err != nil {
			return nil, fmt.Errorf("failed to save execution: %w", err)
		}
	}

	// Create cancellable context
	execCtx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.running[execution.ID] = cancel
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.running, execution.ID)
		s.mu.Unlock()
	}()

	s.logger.Info("Starting workflow execution", "workflow", workflow.Name, "execution_id", execution.ID)

	// Execute workflow
	if err := s.executeWorkflow(execCtx, workflow, execution); err != nil {
		execution.Fail(err.Error())
		s.logger.Error("Workflow execution failed", "workflow", workflow.Name, "error", err)
	} else {
		execution.Complete(execution.Output)
		s.logger.Info("Workflow execution completed", "workflow", workflow.Name, "duration", execution.Duration)
	}

	// Save final state
	if s.executionRepo != nil {
		if err := s.executionRepo.Update(ctx, execution); err != nil {
			s.logger.Error("Failed to save execution state", "error", err)
		}
	}

	return execution, nil
}

// executeWorkflow runs the workflow DAG.
func (s *WorkflowService) executeWorkflow(ctx context.Context, workflow *domain.Workflow, execution *domain.WorkflowExecution) error {
	// Build step map for quick lookup
	stepMap := make(map[string]*domain.WorkflowStep)
	for i := range workflow.Steps {
		stepMap[workflow.Steps[i].ID] = &workflow.Steps[i]
	}

	// Track completed steps
	completed := make(map[string]bool)
	outputs := make(map[string]map[string]interface{})

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Find ready steps (all dependencies completed)
		var ready []*domain.WorkflowStep
		for _, step := range workflow.Steps {
			if completed[step.ID] {
				continue
			}
			allDepsComplete := true
			for _, depID := range step.DependsOn {
				if !completed[depID] {
					allDepsComplete = false
					break
				}
			}
			if allDepsComplete {
				ready = append(ready, stepMap[step.ID])
			}
		}

		// No more steps to execute
		if len(ready) == 0 {
			break
		}

		// Execute ready steps (could be parallelized)
		for _, step := range ready {
			// Build step input from dependencies
			stepInput := make(map[string]interface{})
			for k, v := range execution.Input {
				stepInput[k] = v
			}
			for _, depID := range step.DependsOn {
				if depOutput, ok := outputs[depID]; ok {
					for k, v := range depOutput {
						stepInput[depID+"_"+k] = v
					}
				}
			}

			// Execute step
			output, err := s.executeStep(ctx, step, stepInput, execution)
			if err != nil {
				if !step.ContinueOnError {
					return fmt.Errorf("step %s failed: %w", step.ID, err)
				}
				s.logger.Warn("Step failed but continuing", "step", step.ID, "error", err)
			}

			completed[step.ID] = true
			outputs[step.ID] = output
		}
	}

	// Collect final outputs
	execution.Output = make(map[string]interface{})
	for stepID, output := range outputs {
		execution.Output[stepID] = output
	}

	return nil
}

// executeStep runs a single step with retry logic.
func (s *WorkflowService) executeStep(ctx context.Context, step *domain.WorkflowStep, input map[string]interface{}, execution *domain.WorkflowExecution) (map[string]interface{}, error) {
	stepExec := execution.GetStepExecution(step.ID)
	if stepExec == nil {
		return nil, fmt.Errorf("step execution not found: %s", step.ID)
	}

	now := time.Now()
	stepExec.Status = domain.WorkflowStatusRunning
	stepExec.StartedAt = &now
	stepExec.Input = input

	s.logger.Debug("Executing step", "step", step.ID, "type", step.Type)

	// Get action handler
	s.mu.RLock()
	action, ok := s.actions[step.Type]
	s.mu.RUnlock()

	if !ok {
		err := fmt.Errorf("no action handler for step type: %s", step.Type)
		stepExec.Status = domain.WorkflowStatusFailed
		stepExec.Error = err.Error()
		return nil, err
	}

	// Execute with retries
	maxRetries := step.Retries
	if maxRetries == 0 {
		maxRetries = 1
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		stepExec.RetryCount = attempt

		// Apply timeout if specified
		execCtx := ctx
		if step.Timeout > 0 {
			var cancel context.CancelFunc
			execCtx, cancel = context.WithTimeout(ctx, step.Timeout)
			defer cancel()
		}

		output, err := action.Execute(execCtx, step, input)
		if err == nil {
			completedAt := time.Now()
			stepExec.Status = domain.WorkflowStatusCompleted
			stepExec.CompletedAt = &completedAt
			stepExec.Duration = completedAt.Sub(*stepExec.StartedAt)
			stepExec.Output = output
			return output, nil
		}

		lastErr = err
		s.logger.Warn("Step attempt failed", "step", step.ID, "attempt", attempt+1, "error", err)

		// Wait before retry
		if attempt < maxRetries-1 && step.RetryDelay > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(step.RetryDelay):
			}
		}
	}

	completedAt := time.Now()
	stepExec.Status = domain.WorkflowStatusFailed
	stepExec.CompletedAt = &completedAt
	stepExec.Duration = completedAt.Sub(*stepExec.StartedAt)
	stepExec.Error = lastErr.Error()

	return nil, lastErr
}

// Cancel cancels a running workflow execution.
func (s *WorkflowService) Cancel(executionID uuid.UUID) error {
	s.mu.RLock()
	cancel, ok := s.running[executionID]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("execution not found or not running: %s", executionID)
	}

	cancel()
	return nil
}

// GetExecution retrieves a workflow execution by ID.
func (s *WorkflowService) GetExecution(ctx context.Context, id uuid.UUID) (*domain.WorkflowExecution, error) {
	if s.executionRepo == nil {
		return nil, fmt.Errorf("execution repository not configured")
	}
	return s.executionRepo.GetByID(ctx, id)
}

// ListExecutions lists workflow executions with optional filtering.
func (s *WorkflowService) ListExecutions(ctx context.Context, filter ports.ExecutionFilter) ([]*domain.WorkflowExecution, error) {
	if s.executionRepo == nil {
		return nil, fmt.Errorf("execution repository not configured")
	}
	return s.executionRepo.List(ctx, filter)
}
