// Package domain contains the core business entities for the Forge platform.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// ProfileType represents the type of profile.
type ProfileType string

const (
	ProfileTypeCPU       ProfileType = "cpu"
	ProfileTypeMemory    ProfileType = "memory"
	ProfileTypeHeap      ProfileType = "heap"
	ProfileTypeAllocs    ProfileType = "allocs"
	ProfileTypeGoroutine ProfileType = "goroutine"
	ProfileTypeBlock     ProfileType = "block"
	ProfileTypeMutex     ProfileType = "mutex"
	ProfileTypeTrace     ProfileType = "trace"
)

// ProfileStatus represents the status of a profile capture.
type ProfileStatus string

const (
	ProfileStatusPending   ProfileStatus = "pending"
	ProfileStatusCapturing ProfileStatus = "capturing"
	ProfileStatusCompleted ProfileStatus = "completed"
	ProfileStatusFailed    ProfileStatus = "failed"
)

// Profile represents a profiling session.
type Profile struct {
	ID          uuid.UUID         `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Type        ProfileType       `json:"type"`
	Status      ProfileStatus     `json:"status"`
	ServiceName string            `json:"service_name"`
	ProcessID   int               `json:"process_id,omitempty"`
	Duration    time.Duration     `json:"duration"`
	SampleRate  int               `json:"sample_rate,omitempty"` // Hz
	Labels      map[string]string `json:"labels,omitempty"`
	// Data
	DataSize   int64  `json:"data_size"`
	DataFormat string `json:"data_format"` // pprof, flamegraph, etc.
	FilePath   string `json:"file_path,omitempty"`
	// Timestamps
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	// Error info
	Error string `json:"error,omitempty"`
}

// NewProfile creates a new profile.
func NewProfile(name string, profileType ProfileType, serviceName string, duration time.Duration) *Profile {
	now := time.Now()
	return &Profile{
		ID:          uuid.Must(uuid.NewV7()),
		Name:        name,
		Type:        profileType,
		Status:      ProfileStatusPending,
		ServiceName: serviceName,
		Duration:    duration,
		SampleRate:  100, // Default 100 Hz
		Labels:      make(map[string]string),
		DataFormat:  "pprof",
		CreatedAt:   now,
	}
}

// Start marks the profile as capturing.
func (p *Profile) Start() {
	p.Status = ProfileStatusCapturing
	p.StartedAt = time.Now()
}

// Complete marks the profile as completed.
func (p *Profile) Complete(dataSize int64, filePath string) {
	now := time.Now()
	p.Status = ProfileStatusCompleted
	p.CompletedAt = &now
	p.DataSize = dataSize
	p.FilePath = filePath
}

// Fail marks the profile as failed.
func (p *Profile) Fail(err error) {
	now := time.Now()
	p.Status = ProfileStatusFailed
	p.CompletedAt = &now
	if err != nil {
		p.Error = err.Error()
	}
}

// StackFrame represents a single frame in a stack trace.
type StackFrame struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Package  string `json:"package,omitempty"`
	Module   string `json:"module,omitempty"`
}

// StackTrace represents a full stack trace.
type StackTrace struct {
	Frames []StackFrame `json:"frames"`
}

// ProfileSample represents a single sample in a profile.
type ProfileSample struct {
	Stack    StackTrace        `json:"stack"`
	Value    int64             `json:"value"`     // CPU: nanoseconds, Memory: bytes
	Labels   map[string]string `json:"labels,omitempty"`
}

// ProfileData represents parsed profile data.
type ProfileData struct {
	ProfileID   uuid.UUID       `json:"profile_id"`
	Type        ProfileType     `json:"type"`
	Samples     []ProfileSample `json:"samples"`
	TotalValue  int64           `json:"total_value"`
	SampleCount int             `json:"sample_count"`
}

// FlameGraphNode represents a node in a flame graph.
type FlameGraphNode struct {
	Name     string            `json:"name"`
	Value    int64             `json:"value"`
	Self     int64             `json:"self"` // Value for this node only (not children)
	Children []*FlameGraphNode `json:"children,omitempty"`
}

// FlameGraph represents a flame graph visualization.
type FlameGraph struct {
	ProfileID  uuid.UUID       `json:"profile_id"`
	Type       ProfileType     `json:"type"`
	Root       *FlameGraphNode `json:"root"`
	TotalValue int64           `json:"total_value"`
	MaxDepth   int             `json:"max_depth"`
	CreatedAt  time.Time       `json:"created_at"`
}

// NewFlameGraph creates a new flame graph.
func NewFlameGraph(profileID uuid.UUID, profileType ProfileType) *FlameGraph {
	return &FlameGraph{
		ProfileID: profileID,
		Type:      profileType,
		Root: &FlameGraphNode{
			Name:     "root",
			Children: []*FlameGraphNode{},
		},
		CreatedAt: time.Now(),
	}
}

// GoroutineState represents the state of a goroutine.
type GoroutineState string

const (
	GoroutineStateRunning   GoroutineState = "running"
	GoroutineStateWaiting   GoroutineState = "waiting"
	GoroutineStateSyscall   GoroutineState = "syscall"
	GoroutineStateIdle      GoroutineState = "idle"
	GoroutineStateSleep     GoroutineState = "sleep"
	GoroutineStateChanRecv  GoroutineState = "chan receive"
	GoroutineStateChanSend  GoroutineState = "chan send"
	GoroutineStateSelect    GoroutineState = "select"
	GoroutineStateSemacquire GoroutineState = "semacquire"
	GoroutineStateIOWait    GoroutineState = "IO wait"
)

// GoroutineInfo represents information about a single goroutine.
type GoroutineInfo struct {
	ID           int64          `json:"id"`
	State        GoroutineState `json:"state"`
	WaitReason   string         `json:"wait_reason,omitempty"`
	WaitDuration time.Duration  `json:"wait_duration,omitempty"`
	Stack        StackTrace     `json:"stack"`
	CreatedBy    string         `json:"created_by,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
}

// GoroutineProfile represents a snapshot of all goroutines.
type GoroutineProfile struct {
	ID          uuid.UUID        `json:"id"`
	ProfileID   uuid.UUID        `json:"profile_id"`
	Goroutines  []GoroutineInfo  `json:"goroutines"`
	TotalCount  int              `json:"total_count"`
	ByState     map[string]int   `json:"by_state"`
	CapturedAt  time.Time        `json:"captured_at"`
}

// NewGoroutineProfile creates a new goroutine profile.
func NewGoroutineProfile(profileID uuid.UUID) *GoroutineProfile {
	return &GoroutineProfile{
		ID:         uuid.Must(uuid.NewV7()),
		ProfileID:  profileID,
		Goroutines: []GoroutineInfo{},
		ByState:    make(map[string]int),
		CapturedAt: time.Now(),
	}
}

// AddGoroutine adds a goroutine to the profile.
func (g *GoroutineProfile) AddGoroutine(info GoroutineInfo) {
	g.Goroutines = append(g.Goroutines, info)
	g.TotalCount = len(g.Goroutines)
	g.ByState[string(info.State)]++
}

// MemoryStats represents memory statistics.
type MemoryStats struct {
	Alloc         uint64 `json:"alloc"`          // Bytes allocated and in use
	TotalAlloc    uint64 `json:"total_alloc"`    // Bytes allocated (even if freed)
	Sys           uint64 `json:"sys"`            // Bytes obtained from system
	HeapAlloc     uint64 `json:"heap_alloc"`     // Bytes in heap
	HeapSys       uint64 `json:"heap_sys"`       // Bytes obtained from OS for heap
	HeapIdle      uint64 `json:"heap_idle"`      // Bytes in idle spans
	HeapInuse     uint64 `json:"heap_inuse"`     // Bytes in non-idle spans
	HeapReleased  uint64 `json:"heap_released"`  // Bytes released to OS
	HeapObjects   uint64 `json:"heap_objects"`   // Number of allocated objects
	StackInuse    uint64 `json:"stack_inuse"`    // Bytes in stack spans
	StackSys      uint64 `json:"stack_sys"`      // Bytes obtained from OS for stack
	NumGC         uint32 `json:"num_gc"`         // Number of GC cycles
	LastGC        time.Time `json:"last_gc"`     // Time of last GC
	PauseTotalNs  uint64 `json:"pause_total_ns"` // Total GC pause time
	NumGoroutine  int    `json:"num_goroutine"`  // Number of goroutines
	CapturedAt    time.Time `json:"captured_at"`
}

// ProfileQuery represents a query for profiles.
type ProfileQuery struct {
	Type        ProfileType   `json:"type,omitempty"`
	Status      ProfileStatus `json:"status,omitempty"`
	ServiceName string        `json:"service_name,omitempty"`
	StartTime   time.Time     `json:"start_time,omitempty"`
	EndTime     time.Time     `json:"end_time,omitempty"`
	Limit       int           `json:"limit,omitempty"`
	Offset      int           `json:"offset,omitempty"`
}

