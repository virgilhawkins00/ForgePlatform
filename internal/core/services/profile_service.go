// Package services contains the application services implementing business logic.
package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/forge-platform/forge/internal/core/domain"
	"github.com/forge-platform/forge/internal/core/ports"
	"github.com/google/uuid"
)

// ProfileService provides profiling capabilities.
type ProfileService struct {
	profileRepo ports.ProfileRepository
	logger      ports.Logger
	profileDir  string

	// Active profiles
	mu             sync.RWMutex
	activeProfiles map[uuid.UUID]*activeProfile
}

// activeProfile tracks an in-progress profile capture.
type activeProfile struct {
	profile    *domain.Profile
	file       *os.File
	stopCh     chan struct{}
	cpuProfile bool
}

// NewProfileService creates a new profile service.
func NewProfileService(profileRepo ports.ProfileRepository, profileDir string, logger ports.Logger) *ProfileService {
	if profileDir == "" {
		profileDir = filepath.Join(os.TempDir(), "forge-profiles")
	}
	_ = os.MkdirAll(profileDir, 0755)

	return &ProfileService{
		profileRepo:    profileRepo,
		logger:         logger,
		profileDir:     profileDir,
		activeProfiles: make(map[uuid.UUID]*activeProfile),
	}
}

// StartCPUProfile starts a CPU profile capture.
func (s *ProfileService) StartCPUProfile(ctx context.Context, name, serviceName string, duration time.Duration) (*domain.Profile, error) {
	profile := domain.NewProfile(name, domain.ProfileTypeCPU, serviceName, duration)
	filePath := filepath.Join(s.profileDir, fmt.Sprintf("cpu-%s.pprof", profile.ID.String()))

	f, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create profile file: %w", err)
	}

	if err := pprof.StartCPUProfile(f); err != nil {
		f.Close()
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to start CPU profile: %w", err)
	}

	profile.Start()
	profile.FilePath = filePath

	ap := &activeProfile{
		profile:    profile,
		file:       f,
		stopCh:     make(chan struct{}),
		cpuProfile: true,
	}

	s.mu.Lock()
	s.activeProfiles[profile.ID] = ap
	s.mu.Unlock()

	// Auto-stop after duration
	go func() {
		select {
		case <-time.After(duration):
		case <-ap.stopCh:
		}
		_, _ = s.StopProfile(context.Background(), profile.ID)
	}()

	if s.profileRepo != nil {
		if err := s.profileRepo.Create(ctx, profile); err != nil {
			s.logger.Error("failed to persist profile", "profile_id", profile.ID, "error", err)
		}
	}

	s.logger.Info("started CPU profile", "profile_id", profile.ID, "duration", duration)
	return profile, nil
}

// StopProfile stops an active profile.
func (s *ProfileService) StopProfile(ctx context.Context, id uuid.UUID) (*domain.Profile, error) {
	s.mu.Lock()
	ap, exists := s.activeProfiles[id]
	if exists {
		delete(s.activeProfiles, id)
	}
	s.mu.Unlock()

	if !exists {
		return nil, fmt.Errorf("profile not found or already stopped: %s", id)
	}

	// Stop CPU profiling
	if ap.cpuProfile {
		pprof.StopCPUProfile()
	}

	// Close file
	if ap.file != nil {
		ap.file.Close()
	}

	// Get file size
	if info, err := os.Stat(ap.profile.FilePath); err == nil {
		ap.profile.Complete(info.Size(), ap.profile.FilePath)
	} else {
		ap.profile.Fail(err)
	}

	// Close stop channel
	select {
	case <-ap.stopCh:
	default:
		close(ap.stopCh)
	}

	if s.profileRepo != nil {
		if err := s.profileRepo.Update(ctx, ap.profile); err != nil {
			s.logger.Error("failed to update profile", "profile_id", id, "error", err)
		}
	}

	s.logger.Info("stopped profile", "profile_id", id, "size", ap.profile.DataSize)
	return ap.profile, nil
}

// CaptureHeapProfile captures a heap profile snapshot.
func (s *ProfileService) CaptureHeapProfile(ctx context.Context, name, serviceName string) (*domain.Profile, error) {
	profile := domain.NewProfile(name, domain.ProfileTypeHeap, serviceName, 0)
	filePath := filepath.Join(s.profileDir, fmt.Sprintf("heap-%s.pprof", profile.ID.String()))

	f, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create profile file: %w", err)
	}
	defer f.Close()

	profile.Start()
	if err := pprof.WriteHeapProfile(f); err != nil {
		profile.Fail(err)
		return nil, fmt.Errorf("failed to write heap profile: %w", err)
	}

	if info, errStat := os.Stat(filePath); errStat == nil {
		profile.Complete(info.Size(), filePath)
	}

	if s.profileRepo != nil {
		if errPersist := s.profileRepo.Create(ctx, profile); errPersist != nil {
			s.logger.Error("failed to persist profile", "profile_id", profile.ID, "error", errPersist)
		}
	}

	s.logger.Info("captured heap profile", "profile_id", profile.ID)
	return profile, nil
}

// CaptureGoroutineProfile captures a goroutine profile snapshot.
func (s *ProfileService) CaptureGoroutineProfile(ctx context.Context, name, serviceName string) (*domain.Profile, error) {
	profile := domain.NewProfile(name, domain.ProfileTypeGoroutine, serviceName, 0)
	filePath := filepath.Join(s.profileDir, fmt.Sprintf("goroutine-%s.pprof", profile.ID.String()))

	f, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create profile file: %w", err)
	}
	defer f.Close()

	profile.Start()
	p := pprof.Lookup("goroutine")
	if p == nil {
		profile.Fail(fmt.Errorf("goroutine profile not found"))
		return nil, fmt.Errorf("goroutine profile not found")
	}

	if err := p.WriteTo(f, 1); err != nil {
		profile.Fail(err)
		return nil, fmt.Errorf("failed to write goroutine profile: %w", err)
	}

	if info, errStat := os.Stat(filePath); errStat == nil {
		profile.Complete(info.Size(), filePath)
	}

	if s.profileRepo != nil {
		if errPersist := s.profileRepo.Create(ctx, profile); errPersist != nil {
			s.logger.Error("failed to persist profile", "profile_id", profile.ID, "error", errPersist)
		}
	}

	s.logger.Info("captured goroutine profile", "profile_id", profile.ID)
	return profile, nil
}

// GetMemoryStats returns current memory statistics.
func (s *ProfileService) GetMemoryStats() *domain.MemoryStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &domain.MemoryStats{
		Alloc:        m.Alloc,
		TotalAlloc:   m.TotalAlloc,
		Sys:          m.Sys,
		HeapAlloc:    m.HeapAlloc,
		HeapSys:      m.HeapSys,
		HeapIdle:     m.HeapIdle,
		HeapInuse:    m.HeapInuse,
		HeapReleased: m.HeapReleased,
		HeapObjects:  m.HeapObjects,
		StackInuse:   m.StackInuse,
		StackSys:     m.StackSys,
		NumGC:        m.NumGC,
		LastGC:       time.Unix(0, int64(m.LastGC)),
		PauseTotalNs: m.PauseTotalNs,
		NumGoroutine: runtime.NumGoroutine(),
		CapturedAt:   time.Now(),
	}
}

// GetProfile retrieves a profile by ID.
func (s *ProfileService) GetProfile(ctx context.Context, id uuid.UUID) (*domain.Profile, error) {
	// Check active profiles first
	s.mu.RLock()
	if ap, ok := s.activeProfiles[id]; ok {
		s.mu.RUnlock()
		return ap.profile, nil
	}
	s.mu.RUnlock()

	if s.profileRepo == nil {
		return nil, fmt.Errorf("profile repository not configured")
	}
	return s.profileRepo.GetByID(ctx, id)
}

// ListProfiles lists profiles with optional filtering.
func (s *ProfileService) ListProfiles(ctx context.Context, filter ports.ProfileFilter) ([]*domain.Profile, error) {
	if s.profileRepo == nil {
		return []*domain.Profile{}, nil
	}
	return s.profileRepo.List(ctx, filter)
}

// DeleteProfile deletes a profile and its data.
func (s *ProfileService) DeleteProfile(ctx context.Context, id uuid.UUID) error {
	profile, err := s.GetProfile(ctx, id)
	if err != nil {
		return err
	}

	// Remove file
	if profile.FilePath != "" {
		os.Remove(profile.FilePath)
	}

	if s.profileRepo != nil {
		return s.profileRepo.Delete(ctx, id)
	}

	return nil
}

// GetActiveProfiles returns the list of active profiles.
func (s *ProfileService) GetActiveProfiles() []*domain.Profile {
	s.mu.RLock()
	defer s.mu.RUnlock()

	profiles := make([]*domain.Profile, 0, len(s.activeProfiles))
	for _, ap := range s.activeProfiles {
		profiles = append(profiles, ap.profile)
	}
	return profiles
}

// GetProfileStats returns profiling statistics.
func (s *ProfileService) GetProfileStats(ctx context.Context) (map[string]interface{}, error) {
	s.mu.RLock()
	activeCount := len(s.activeProfiles)
	s.mu.RUnlock()

	memStats := s.GetMemoryStats()

	stats := map[string]interface{}{
		"active_profiles": activeCount,
		"num_goroutine":   memStats.NumGoroutine,
		"heap_alloc_mb":   float64(memStats.HeapAlloc) / 1024 / 1024,
		"sys_mb":          float64(memStats.Sys) / 1024 / 1024,
		"num_gc":          memStats.NumGC,
	}

	return stats, nil
}

