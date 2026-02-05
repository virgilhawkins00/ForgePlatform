package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewProfile(t *testing.T) {
	profile := NewProfile("cpu-profile", ProfileTypeCPU, "my-service", 30*time.Second)

	if profile.ID.String() == "" {
		t.Error("ID is empty")
	}
	if profile.Name != "cpu-profile" {
		t.Errorf("Name = %v, want cpu-profile", profile.Name)
	}
	if profile.Type != ProfileTypeCPU {
		t.Errorf("Type = %v, want cpu", profile.Type)
	}
	if profile.Status != ProfileStatusPending {
		t.Errorf("Status = %v, want pending", profile.Status)
	}
	if profile.ServiceName != "my-service" {
		t.Errorf("ServiceName = %v, want my-service", profile.ServiceName)
	}
	if profile.Duration != 30*time.Second {
		t.Errorf("Duration = %v, want 30s", profile.Duration)
	}
	if profile.SampleRate != 100 {
		t.Errorf("SampleRate = %d, want 100", profile.SampleRate)
	}
	if profile.DataFormat != "pprof" {
		t.Errorf("DataFormat = %v, want pprof", profile.DataFormat)
	}
}

func TestProfile_Start(t *testing.T) {
	profile := NewProfile("test", ProfileTypeCPU, "svc", time.Minute)

	profile.Start()

	if profile.Status != ProfileStatusCapturing {
		t.Errorf("Status = %v, want capturing", profile.Status)
	}
	if profile.StartedAt.IsZero() {
		t.Error("StartedAt is zero after Start()")
	}
}

func TestProfile_Complete(t *testing.T) {
	profile := NewProfile("test", ProfileTypeCPU, "svc", time.Minute)
	profile.Start()

	profile.Complete(1024, "/profiles/cpu.pprof")

	if profile.Status != ProfileStatusCompleted {
		t.Errorf("Status = %v, want completed", profile.Status)
	}
	if profile.CompletedAt == nil {
		t.Error("CompletedAt is nil after Complete()")
	}
	if profile.DataSize != 1024 {
		t.Errorf("DataSize = %d, want 1024", profile.DataSize)
	}
	if profile.FilePath != "/profiles/cpu.pprof" {
		t.Errorf("FilePath = %v, want /profiles/cpu.pprof", profile.FilePath)
	}
}

func TestProfile_Fail(t *testing.T) {
	profile := NewProfile("test", ProfileTypeCPU, "svc", time.Minute)
	profile.Start()

	profile.Fail(errors.New("process not found"))

	if profile.Status != ProfileStatusFailed {
		t.Errorf("Status = %v, want failed", profile.Status)
	}
	if profile.CompletedAt == nil {
		t.Error("CompletedAt is nil after Fail()")
	}
	if profile.Error != "process not found" {
		t.Errorf("Error = %v, want 'process not found'", profile.Error)
	}
}

func TestProfile_Fail_NilError(t *testing.T) {
	profile := NewProfile("test", ProfileTypeCPU, "svc", time.Minute)
	profile.Start()

	profile.Fail(nil)

	if profile.Status != ProfileStatusFailed {
		t.Errorf("Status = %v, want failed", profile.Status)
	}
	if profile.Error != "" {
		t.Errorf("Error = %v, want empty", profile.Error)
	}
}

func TestProfileTypeConstants(t *testing.T) {
	types := []ProfileType{
		ProfileTypeCPU,
		ProfileTypeMemory,
		ProfileTypeHeap,
		ProfileTypeAllocs,
		ProfileTypeGoroutine,
		ProfileTypeBlock,
		ProfileTypeMutex,
		ProfileTypeTrace,
	}
	expected := []string{"cpu", "memory", "heap", "allocs", "goroutine", "block", "mutex", "trace"}

	for i, pt := range types {
		if string(pt) != expected[i] {
			t.Errorf("ProfileType[%d] = %v, want %v", i, pt, expected[i])
		}
	}
}

func TestProfileStatusConstants(t *testing.T) {
	if ProfileStatusPending != "pending" {
		t.Errorf("ProfileStatusPending = %v, want pending", ProfileStatusPending)
	}
	if ProfileStatusCapturing != "capturing" {
		t.Errorf("ProfileStatusCapturing = %v, want capturing", ProfileStatusCapturing)
	}
	if ProfileStatusCompleted != "completed" {
		t.Errorf("ProfileStatusCompleted = %v, want completed", ProfileStatusCompleted)
	}
	if ProfileStatusFailed != "failed" {
		t.Errorf("ProfileStatusFailed = %v, want failed", ProfileStatusFailed)
	}
}

func TestNewFlameGraph(t *testing.T) {
	profileID := uuid.New()
	fg := NewFlameGraph(profileID, ProfileTypeCPU)

	if fg.ProfileID != profileID {
		t.Errorf("ProfileID mismatch")
	}
	if fg.Type != ProfileTypeCPU {
		t.Errorf("Type = %v, want cpu", fg.Type)
	}
	if fg.Root == nil {
		t.Error("Root is nil")
	}
	if fg.Root.Name != "root" {
		t.Errorf("Root.Name = %v, want root", fg.Root.Name)
	}
}

func TestNewGoroutineProfile(t *testing.T) {
	profileID := uuid.New()
	gp := NewGoroutineProfile(profileID)

	if gp.ID.String() == "" {
		t.Error("ID is empty")
	}
	if gp.ProfileID != profileID {
		t.Errorf("ProfileID mismatch")
	}
	if gp.TotalCount != 0 {
		t.Errorf("TotalCount = %d, want 0", gp.TotalCount)
	}
	if len(gp.Goroutines) != 0 {
		t.Errorf("Goroutines should be empty, got %d", len(gp.Goroutines))
	}
}

func TestGoroutineProfile_AddGoroutine(t *testing.T) {
	profileID := uuid.New()
	gp := NewGoroutineProfile(profileID)

	info1 := GoroutineInfo{
		ID:    1,
		State: GoroutineStateRunning,
	}
	info2 := GoroutineInfo{
		ID:    2,
		State: GoroutineStateWaiting,
	}

	gp.AddGoroutine(info1)
	gp.AddGoroutine(info2)

	if gp.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2", gp.TotalCount)
	}
	if len(gp.Goroutines) != 2 {
		t.Errorf("Goroutines count = %d, want 2", len(gp.Goroutines))
	}
	if gp.ByState["running"] != 1 {
		t.Errorf("ByState[running] = %d, want 1", gp.ByState["running"])
	}
	if gp.ByState["waiting"] != 1 {
		t.Errorf("ByState[waiting] = %d, want 1", gp.ByState["waiting"])
	}
}

func TestGoroutineStateConstants(t *testing.T) {
	states := []GoroutineState{
		GoroutineStateRunning,
		GoroutineStateWaiting,
		GoroutineStateSyscall,
		GoroutineStateIdle,
		GoroutineStateSleep,
		GoroutineStateChanRecv,
		GoroutineStateChanSend,
		GoroutineStateSelect,
		GoroutineStateSemacquire,
		GoroutineStateIOWait,
	}
	expected := []string{
		"running", "waiting", "syscall", "idle", "sleep",
		"chan receive", "chan send", "select", "semacquire", "IO wait",
	}

	for i, state := range states {
		if string(state) != expected[i] {
			t.Errorf("GoroutineState[%d] = %v, want %v", i, state, expected[i])
		}
	}
}

func TestStackFrame(t *testing.T) {
	frame := StackFrame{
		Function: "main.handleRequest",
		File:     "/app/main.go",
		Line:     42,
		Package:  "main",
		Module:   "github.com/myapp",
	}

	if frame.Function != "main.handleRequest" {
		t.Errorf("Function = %v", frame.Function)
	}
	if frame.Line != 42 {
		t.Errorf("Line = %d", frame.Line)
	}
}

func TestProfileSample(t *testing.T) {
	sample := ProfileSample{
		Stack: StackTrace{
			Frames: []StackFrame{
				{Function: "main", File: "main.go", Line: 1},
			},
		},
		Value:  1000000,
		Labels: map[string]string{"goroutine": "1"},
	}

	if sample.Value != 1000000 {
		t.Errorf("Value = %d", sample.Value)
	}
	if len(sample.Stack.Frames) != 1 {
		t.Errorf("Frames count = %d", len(sample.Stack.Frames))
	}
}

