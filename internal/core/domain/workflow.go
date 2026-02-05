package domain

import (
	"time"

	"github.com/google/uuid"
)

// WorkflowStatus represents the current state of a workflow execution.
type WorkflowStatus string

const (
	WorkflowStatusPending   WorkflowStatus = "pending"
	WorkflowStatusRunning   WorkflowStatus = "running"
	WorkflowStatusCompleted WorkflowStatus = "completed"
	WorkflowStatusFailed    WorkflowStatus = "failed"
	WorkflowStatusCancelled WorkflowStatus = "cancelled"
	WorkflowStatusPaused    WorkflowStatus = "paused"
)

// StepType represents the type of workflow step.
type StepType string

const (
	StepTypeShell    StepType = "shell"    // Execute shell command
	StepTypeHTTP     StepType = "http"     // Make HTTP request
	StepTypeMetric   StepType = "metric"   // Query metrics
	StepTypeAI       StepType = "ai"       // AI analysis
	StepTypePlugin   StepType = "plugin"   // Call plugin
	StepTypeTask     StepType = "task"     // Create task
	StepTypeParallel StepType = "parallel" // Parallel execution group
	StepTypeDecision StepType = "decision" // Conditional branching
	StepTypeWorkflow StepType = "workflow" // Sub-workflow
)

// WorkflowStep represents a single step in a workflow.
type WorkflowStep struct {
	ID              string                 `json:"id" yaml:"id"`
	Name            string                 `json:"name" yaml:"name"`
	Description     string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Type            StepType               `json:"type" yaml:"type"`
	Config          map[string]interface{} `json:"config" yaml:"config"`
	DependsOn       []string               `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	Condition       string                 `json:"condition,omitempty" yaml:"condition,omitempty"`
	Timeout         time.Duration          `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Retries         int                    `json:"retries,omitempty" yaml:"retries,omitempty"`
	RetryDelay      time.Duration          `json:"retry_delay,omitempty" yaml:"retry_delay,omitempty"`
	ContinueOnError bool                   `json:"continue_on_error,omitempty" yaml:"continue_on_error,omitempty"`
	// Runtime state (not persisted in YAML)
	Status      WorkflowStatus `json:"status" yaml:"-"`
	StartedAt   *time.Time     `json:"started_at,omitempty" yaml:"-"`
	CompletedAt *time.Time     `json:"completed_at,omitempty" yaml:"-"`
	Output      interface{}    `json:"output,omitempty" yaml:"-"`
	Error       string         `json:"error,omitempty" yaml:"-"`
	RetryCount  int            `json:"retry_count,omitempty" yaml:"-"`
}

// Workflow represents a multi-step automation workflow definition.
type Workflow struct {
	ID          uuid.UUID              `json:"id" yaml:"-"`
	Name        string                 `json:"name" yaml:"name"`
	Description string                 `json:"description" yaml:"description"`
	Version     string                 `json:"version,omitempty" yaml:"version,omitempty"`
	Steps       []WorkflowStep         `json:"steps" yaml:"steps"`
	Variables   map[string]interface{} `json:"variables,omitempty" yaml:"variables,omitempty"`
	Env         map[string]string      `json:"env,omitempty" yaml:"env,omitempty"`
	Timeout     time.Duration          `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	MaxRetries  int                    `json:"max_retries,omitempty" yaml:"max_retries,omitempty"`
	// Runtime state
	Status      WorkflowStatus `json:"status" yaml:"-"`
	CurrentStep int            `json:"current_step" yaml:"-"`
	CreatedAt   time.Time      `json:"created_at" yaml:"-"`
	UpdatedAt   time.Time      `json:"updated_at" yaml:"-"`
	StartedAt   *time.Time     `json:"started_at,omitempty" yaml:"-"`
	CompletedAt *time.Time     `json:"completed_at,omitempty" yaml:"-"`
	Error       string         `json:"error,omitempty" yaml:"-"`
}

// WorkflowExecution represents a running or completed workflow instance.
type WorkflowExecution struct {
	ID           uuid.UUID              `json:"id"`
	WorkflowID   uuid.UUID              `json:"workflow_id"`
	WorkflowName string                 `json:"workflow_name"`
	Status       WorkflowStatus         `json:"status"`
	Steps        []StepExecution        `json:"steps"`
	Input        map[string]interface{} `json:"input,omitempty"`
	Output       map[string]interface{} `json:"output,omitempty"`
	Error        string                 `json:"error,omitempty"`
	StartedAt    time.Time              `json:"started_at"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	Duration     time.Duration          `json:"duration,omitempty"`
	Checkpoint   []byte                 `json:"checkpoint,omitempty"` // For durable execution
}

// StepExecution represents the execution state of a single step.
type StepExecution struct {
	StepID      string                 `json:"step_id"`
	StepName    string                 `json:"step_name"`
	Status      WorkflowStatus         `json:"status"`
	Input       map[string]interface{} `json:"input,omitempty"`
	Output      map[string]interface{} `json:"output,omitempty"`
	Error       string                 `json:"error,omitempty"`
	RetryCount  int                    `json:"retry_count"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Duration    time.Duration          `json:"duration,omitempty"`
	Logs        []string               `json:"logs,omitempty"`
}

// NewWorkflow creates a new workflow.
func NewWorkflow(name, description string) *Workflow {
	now := time.Now()
	return &Workflow{
		ID:          uuid.Must(uuid.NewV7()),
		Name:        name,
		Description: description,
		Steps:       []WorkflowStep{},
		Variables:   make(map[string]interface{}),
		Status:      WorkflowStatusPending,
		CurrentStep: 0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// AddStep adds a new step to the workflow.
func (w *Workflow) AddStep(name string, stepType StepType, config map[string]interface{}, dependsOn []string) *WorkflowStep {
	step := WorkflowStep{
		ID:        uuid.Must(uuid.NewV7()).String(),
		Name:      name,
		Type:      stepType,
		Config:    config,
		DependsOn: dependsOn,
		Status:    WorkflowStatusPending,
	}
	w.Steps = append(w.Steps, step)
	w.UpdatedAt = time.Now()
	return &step
}

// Start marks the workflow as running.
func (w *Workflow) Start() {
	now := time.Now()
	w.Status = WorkflowStatusRunning
	w.StartedAt = &now
	w.UpdatedAt = now
}

// Complete marks the workflow as completed.
func (w *Workflow) Complete() {
	now := time.Now()
	w.Status = WorkflowStatusCompleted
	w.CompletedAt = &now
	w.UpdatedAt = now
}

// Fail marks the workflow as failed.
func (w *Workflow) Fail(err error) {
	now := time.Now()
	w.Status = WorkflowStatusFailed
	w.Error = err.Error()
	w.CompletedAt = &now
	w.UpdatedAt = now
}

// GetNextSteps returns steps that are ready to execute.
func (w *Workflow) GetNextSteps() []WorkflowStep {
	var ready []WorkflowStep
	for _, step := range w.Steps {
		if step.Status != WorkflowStatusPending {
			continue
		}
		allDepsComplete := true
		for _, depID := range step.DependsOn {
			for _, s := range w.Steps {
				if s.ID == depID && s.Status != WorkflowStatusCompleted {
					allDepsComplete = false
					break
				}
			}
		}
		if allDepsComplete {
			ready = append(ready, step)
		}
	}
	return ready
}

// GetStep returns a step by ID.
func (w *Workflow) GetStep(stepID string) *WorkflowStep {
	for i := range w.Steps {
		if w.Steps[i].ID == stepID {
			return &w.Steps[i]
		}
	}
	return nil
}

// Cancel marks the workflow as cancelled.
func (w *Workflow) Cancel() {
	now := time.Now()
	w.Status = WorkflowStatusCancelled
	w.CompletedAt = &now
	w.UpdatedAt = now
}

// Pause marks the workflow as paused.
func (w *Workflow) Pause() {
	w.Status = WorkflowStatusPaused
	w.UpdatedAt = time.Now()
}

// Resume marks the workflow as running again.
func (w *Workflow) Resume() {
	w.Status = WorkflowStatusRunning
	w.UpdatedAt = time.Now()
}

// NewWorkflowExecution creates a new workflow execution instance.
func NewWorkflowExecution(workflow *Workflow, input map[string]interface{}) *WorkflowExecution {
	steps := make([]StepExecution, len(workflow.Steps))
	for i, step := range workflow.Steps {
		steps[i] = StepExecution{
			StepID:   step.ID,
			StepName: step.Name,
			Status:   WorkflowStatusPending,
		}
	}

	return &WorkflowExecution{
		ID:           uuid.Must(uuid.NewV7()),
		WorkflowID:   workflow.ID,
		WorkflowName: workflow.Name,
		Status:       WorkflowStatusPending,
		Steps:        steps,
		Input:        input,
		StartedAt:    time.Now(),
	}
}

// GetStepExecution returns a step execution by step ID.
func (e *WorkflowExecution) GetStepExecution(stepID string) *StepExecution {
	for i := range e.Steps {
		if e.Steps[i].StepID == stepID {
			return &e.Steps[i]
		}
	}
	return nil
}

// IsComplete returns true if the execution has finished (completed, failed, or cancelled).
func (e *WorkflowExecution) IsComplete() bool {
	return e.Status == WorkflowStatusCompleted ||
		e.Status == WorkflowStatusFailed ||
		e.Status == WorkflowStatusCancelled
}

// Complete marks the execution as completed.
func (e *WorkflowExecution) Complete(output map[string]interface{}) {
	now := time.Now()
	e.Status = WorkflowStatusCompleted
	e.Output = output
	e.CompletedAt = &now
	e.Duration = now.Sub(e.StartedAt)
}

// Fail marks the execution as failed.
func (e *WorkflowExecution) Fail(err string) {
	now := time.Now()
	e.Status = WorkflowStatusFailed
	e.Error = err
	e.CompletedAt = &now
	e.Duration = now.Sub(e.StartedAt)
}
