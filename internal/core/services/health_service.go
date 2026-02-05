// Package services implements core business logic services.
package services

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/forge-platform/forge/internal/core/ports"
)

// HealthStatus represents the overall health status.
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// ComponentHealth represents the health of a single component.
type ComponentHealth struct {
	Name      string            `json:"name"`
	Status    HealthStatus      `json:"status"`
	Message   string            `json:"message,omitempty"`
	Details   map[string]string `json:"details,omitempty"`
	CheckedAt time.Time         `json:"checked_at"`
	Latency   time.Duration     `json:"latency"`
}

// SystemHealth represents the overall system health.
type SystemHealth struct {
	Status     HealthStatus       `json:"status"`
	Version    string             `json:"version"`
	Uptime     time.Duration      `json:"uptime"`
	Components []ComponentHealth  `json:"components"`
	System     SystemMetrics      `json:"system"`
	CheckedAt  time.Time          `json:"checked_at"`
}

// SystemMetrics contains system-level metrics.
type SystemMetrics struct {
	GoVersion    string  `json:"go_version"`
	NumGoroutine int     `json:"num_goroutine"`
	NumCPU       int     `json:"num_cpu"`
	MemAlloc     uint64  `json:"mem_alloc"`
	MemSys       uint64  `json:"mem_sys"`
	HeapAlloc    uint64  `json:"heap_alloc"`
	HeapInuse    uint64  `json:"heap_inuse"`
	HeapObjects  uint64  `json:"heap_objects"`
	GCPauseNs    uint64  `json:"gc_pause_ns"`
	NumGC        uint32  `json:"num_gc"`
}

// HealthChecker is a function that checks the health of a component.
type HealthChecker func(ctx context.Context) ComponentHealth

// HealthService provides system health monitoring.
type HealthService struct {
	mu        sync.RWMutex
	startTime time.Time
	version   string
	checkers  map[string]HealthChecker
	logger    ports.Logger
}

// NewHealthService creates a new health service.
func NewHealthService(version string, logger ports.Logger) *HealthService {
	return &HealthService{
		startTime: time.Now(),
		version:   version,
		checkers:  make(map[string]HealthChecker),
		logger:    logger,
	}
}

// RegisterChecker registers a health checker for a component.
func (s *HealthService) RegisterChecker(name string, checker HealthChecker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkers[name] = checker
}

// Check performs a full health check.
func (s *HealthService) Check(ctx context.Context) *SystemHealth {
	s.mu.RLock()
	checkers := make(map[string]HealthChecker, len(s.checkers))
	for k, v := range s.checkers {
		checkers[k] = v
	}
	s.mu.RUnlock()

	components := make([]ComponentHealth, 0, len(checkers))
	overallStatus := HealthStatusHealthy

	for name, checker := range checkers {
		health := checker(ctx)
		health.Name = name
		components = append(components, health)

		// Update overall status
		if health.Status == HealthStatusUnhealthy {
			overallStatus = HealthStatusUnhealthy
		} else if health.Status == HealthStatusDegraded && overallStatus == HealthStatusHealthy {
			overallStatus = HealthStatusDegraded
		}
	}

	return &SystemHealth{
		Status:     overallStatus,
		Version:    s.version,
		Uptime:     time.Since(s.startTime),
		Components: components,
		System:     s.getSystemMetrics(),
		CheckedAt:  time.Now(),
	}
}

// CheckLiveness performs a simple liveness check.
func (s *HealthService) CheckLiveness(ctx context.Context) bool {
	return true
}

// CheckReadiness performs a readiness check.
func (s *HealthService) CheckReadiness(ctx context.Context) bool {
	health := s.Check(ctx)
	return health.Status != HealthStatusUnhealthy
}

// getSystemMetrics collects system-level metrics.
func (s *HealthService) getSystemMetrics() SystemMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return SystemMetrics{
		GoVersion:    runtime.Version(),
		NumGoroutine: runtime.NumGoroutine(),
		NumCPU:       runtime.NumCPU(),
		MemAlloc:     m.Alloc,
		MemSys:       m.Sys,
		HeapAlloc:    m.HeapAlloc,
		HeapInuse:    m.HeapInuse,
		HeapObjects:  m.HeapObjects,
		GCPauseNs:    m.PauseNs[(m.NumGC+255)%256],
		NumGC:        m.NumGC,
	}
}

// GetUptime returns the service uptime.
func (s *HealthService) GetUptime() time.Duration {
	return time.Since(s.startTime)
}

