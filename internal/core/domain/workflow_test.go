package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNewWorkflow(t *testing.T) {
	workflow := NewWorkflow("deploy-app", "Deploy application to production")

	if workflow.ID.String() == "" {
		t.Error("ID is empty")
	}
	if workflow.Name != "deploy-app" {
		t.Errorf("Name = %v, want deploy-app", workflow.Name)
	}
	if workflow.Description != "Deploy application to production" {
		t.Errorf("Description not set correctly")
	}
	if workflow.Status != WorkflowStatusPending {
		t.Errorf("Status = %v, want pending", workflow.Status)
	}
	if workflow.CurrentStep != 0 {
		t.Errorf("CurrentStep = %d, want 0", workflow.CurrentStep)
	}
	if len(workflow.Steps) != 0 {
		t.Errorf("Steps should be empty, got %d", len(workflow.Steps))
	}
}

func TestWorkflow_AddStep(t *testing.T) {
	workflow := NewWorkflow("test", "test workflow")
	config := map[string]interface{}{"command": "echo hello"}

	step := workflow.AddStep("echo", StepTypeShell, config, nil)

	if step == nil {
		t.Fatal("AddStep returned nil")
	}
	if len(workflow.Steps) != 1 {
		t.Errorf("Steps count = %d, want 1", len(workflow.Steps))
	}
	if step.Name != "echo" {
		t.Errorf("Step Name = %v, want echo", step.Name)
	}
	if step.Type != StepTypeShell {
		t.Errorf("Step Type = %v, want shell", step.Type)
	}
	if step.Status != WorkflowStatusPending {
		t.Errorf("Step Status = %v, want pending", step.Status)
	}
}

func TestWorkflow_AddStep_WithDependencies(t *testing.T) {
	workflow := NewWorkflow("test", "test workflow")

	step1 := workflow.AddStep("step1", StepTypeShell, nil, nil)
	step2 := workflow.AddStep("step2", StepTypeShell, nil, []string{step1.ID})

	if len(step2.DependsOn) != 1 {
		t.Errorf("DependsOn count = %d, want 1", len(step2.DependsOn))
	}
	if step2.DependsOn[0] != step1.ID {
		t.Errorf("DependsOn[0] = %v, want %v", step2.DependsOn[0], step1.ID)
	}
}

func TestWorkflow_Start(t *testing.T) {
	workflow := NewWorkflow("test", "test workflow")

	workflow.Start()

	if workflow.Status != WorkflowStatusRunning {
		t.Errorf("Status = %v, want running", workflow.Status)
	}
	if workflow.StartedAt == nil {
		t.Error("StartedAt is nil after Start()")
	}
}

func TestWorkflow_Complete(t *testing.T) {
	workflow := NewWorkflow("test", "test workflow")
	workflow.Start()

	workflow.Complete()

	if workflow.Status != WorkflowStatusCompleted {
		t.Errorf("Status = %v, want completed", workflow.Status)
	}
	if workflow.CompletedAt == nil {
		t.Error("CompletedAt is nil after Complete()")
	}
}

func TestWorkflow_Fail(t *testing.T) {
	workflow := NewWorkflow("test", "test workflow")
	workflow.Start()

	workflow.Fail(errors.New("connection timeout"))

	if workflow.Status != WorkflowStatusFailed {
		t.Errorf("Status = %v, want failed", workflow.Status)
	}
	if workflow.Error != "connection timeout" {
		t.Errorf("Error = %v, want 'connection timeout'", workflow.Error)
	}
}

func TestWorkflow_Cancel(t *testing.T) {
	workflow := NewWorkflow("test", "test workflow")
	workflow.Start()

	workflow.Cancel()

	if workflow.Status != WorkflowStatusCancelled {
		t.Errorf("Status = %v, want cancelled", workflow.Status)
	}
}

func TestWorkflow_Pause(t *testing.T) {
	workflow := NewWorkflow("test", "test workflow")
	workflow.Start()

	workflow.Pause()

	if workflow.Status != WorkflowStatusPaused {
		t.Errorf("Status = %v, want paused", workflow.Status)
	}
}

func TestWorkflow_Resume(t *testing.T) {
	workflow := NewWorkflow("test", "test workflow")
	workflow.Start()
	workflow.Pause()

	workflow.Resume()

	if workflow.Status != WorkflowStatusRunning {
		t.Errorf("Status = %v, want running", workflow.Status)
	}
}

func TestWorkflow_GetStep(t *testing.T) {
	workflow := NewWorkflow("test", "test workflow")
	step1 := workflow.AddStep("step1", StepTypeShell, nil, nil)
	workflow.AddStep("step2", StepTypeHTTP, nil, nil)

	found := workflow.GetStep(step1.ID)
	if found == nil {
		t.Fatal("GetStep returned nil for existing step")
	}
	if found.Name != "step1" {
		t.Errorf("GetStep returned wrong step: %v", found.Name)
	}

	notFound := workflow.GetStep("non-existent-id")
	if notFound != nil {
		t.Error("GetStep should return nil for non-existent step")
	}
}

func TestWorkflow_GetNextSteps(t *testing.T) {
	workflow := NewWorkflow("test", "test workflow")
	step1 := workflow.AddStep("step1", StepTypeShell, nil, nil)
	step2 := workflow.AddStep("step2", StepTypeShell, nil, []string{step1.ID})
	workflow.AddStep("step3", StepTypeShell, nil, []string{step2.ID})

	// Initially, only step1 should be next (no dependencies)
	workflow.Steps[0].Status = WorkflowStatusPending
	workflow.Steps[1].Status = WorkflowStatusPending
	workflow.Steps[2].Status = WorkflowStatusPending

	next := workflow.GetNextSteps()
	if len(next) != 1 {
		t.Errorf("GetNextSteps count = %d, want 1", len(next))
	}
	if len(next) > 0 && next[0].Name != "step1" {
		t.Errorf("GetNextSteps[0].Name = %v, want step1", next[0].Name)
	}

	// After step1 completes, step2 should be next
	workflow.Steps[0].Status = WorkflowStatusCompleted
	next = workflow.GetNextSteps()
	if len(next) != 1 {
		t.Errorf("After step1 complete: GetNextSteps count = %d, want 1", len(next))
	}
	if len(next) > 0 && next[0].Name != "step2" {
		t.Errorf("After step1 complete: GetNextSteps[0].Name = %v, want step2", next[0].Name)
	}
}

func TestNewWorkflowExecution(t *testing.T) {
	workflow := NewWorkflow("deploy", "deploy app")
	workflow.AddStep("build", StepTypeShell, nil, nil)
	workflow.AddStep("deploy", StepTypeShell, nil, nil)
	input := map[string]interface{}{"version": "1.0.0"}

	exec := NewWorkflowExecution(workflow, input)

	if exec.ID.String() == "" {
		t.Error("ID is empty")
	}
	if exec.WorkflowID != workflow.ID {
		t.Errorf("WorkflowID = %v, want %v", exec.WorkflowID, workflow.ID)
	}
	if exec.WorkflowName != "deploy" {
		t.Errorf("WorkflowName = %v, want deploy", exec.WorkflowName)
	}
	if exec.Status != WorkflowStatusPending {
		t.Errorf("Status = %v, want pending", exec.Status)
	}
	if len(exec.Steps) != 2 {
		t.Errorf("Steps count = %d, want 2", len(exec.Steps))
	}
	if exec.Input["version"] != "1.0.0" {
		t.Error("Input not set correctly")
	}
}

func TestWorkflowExecution_GetStepExecution(t *testing.T) {
	workflow := NewWorkflow("test", "test")
	step := workflow.AddStep("step1", StepTypeShell, nil, nil)

	exec := NewWorkflowExecution(workflow, nil)

	stepExec := exec.GetStepExecution(step.ID)
	if stepExec == nil {
		t.Fatal("GetStepExecution returned nil for existing step")
	}
	if stepExec.StepID != step.ID {
		t.Errorf("StepID = %v, want %v", stepExec.StepID, step.ID)
	}

	notFound := exec.GetStepExecution("non-existent")
	if notFound != nil {
		t.Error("GetStepExecution should return nil for non-existent step")
	}
}

func TestWorkflowExecution_IsComplete(t *testing.T) {
	workflow := NewWorkflow("test", "test")
	workflow.AddStep("step1", StepTypeShell, nil, nil)

	exec := NewWorkflowExecution(workflow, nil)

	// Pending status - not complete
	if exec.IsComplete() {
		t.Error("IsComplete = true when status is pending")
	}

	// Running status - not complete
	exec.Status = WorkflowStatusRunning
	if exec.IsComplete() {
		t.Error("IsComplete = true when status is running")
	}

	// Completed status - is complete
	exec.Status = WorkflowStatusCompleted
	if !exec.IsComplete() {
		t.Error("IsComplete = false when status is completed")
	}

	// Failed status - is complete
	exec.Status = WorkflowStatusFailed
	if !exec.IsComplete() {
		t.Error("IsComplete = false when status is failed")
	}

	// Cancelled status - is complete
	exec.Status = WorkflowStatusCancelled
	if !exec.IsComplete() {
		t.Error("IsComplete = false when status is cancelled")
	}
}

func TestWorkflowExecution_Complete(t *testing.T) {
	workflow := NewWorkflow("test", "test")
	exec := NewWorkflowExecution(workflow, nil)
	output := map[string]interface{}{"result": "success"}

	exec.Complete(output)

	if exec.Status != WorkflowStatusCompleted {
		t.Errorf("Status = %v, want completed", exec.Status)
	}
	if exec.CompletedAt == nil {
		t.Error("CompletedAt is nil after Complete()")
	}
	if exec.Output["result"] != "success" {
		t.Error("Output not set correctly")
	}
}

func TestWorkflowExecution_Fail(t *testing.T) {
	workflow := NewWorkflow("test", "test")
	exec := NewWorkflowExecution(workflow, nil)

	exec.Fail("step failed: connection refused")

	if exec.Status != WorkflowStatusFailed {
		t.Errorf("Status = %v, want failed", exec.Status)
	}
	if exec.Error != "step failed: connection refused" {
		t.Errorf("Error = %v, want error message", exec.Error)
	}
	if exec.CompletedAt == nil {
		t.Error("CompletedAt is nil after Fail()")
	}
}

func TestStepTypes(t *testing.T) {
	types := []StepType{
		StepTypeShell,
		StepTypeHTTP,
		StepTypeMetric,
		StepTypeAI,
		StepTypePlugin,
		StepTypeTask,
		StepTypeParallel,
		StepTypeDecision,
		StepTypeWorkflow,
	}

	expected := []string{"shell", "http", "metric", "ai", "plugin", "task", "parallel", "decision", "workflow"}

	for i, st := range types {
		if string(st) != expected[i] {
			t.Errorf("StepType[%d] = %v, want %v", i, st, expected[i])
		}
	}
}

func TestWorkflowTimeout(t *testing.T) {
	workflow := NewWorkflow("test", "test")
	workflow.Timeout = 30 * time.Minute

	if workflow.Timeout != 30*time.Minute {
		t.Errorf("Timeout = %v, want 30m", workflow.Timeout)
	}
}

