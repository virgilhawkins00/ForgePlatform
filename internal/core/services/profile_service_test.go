// Package services implements core business logic services.
package services

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/forge-platform/forge/internal/core/domain"
	"github.com/forge-platform/forge/internal/core/ports"
	"github.com/google/uuid"
)

// mockProfileLogger for testing
type mockProfileLogger struct{}

func (m *mockProfileLogger) Debug(msg string, args ...interface{}) {}
func (m *mockProfileLogger) Info(msg string, args ...interface{})  {}
func (m *mockProfileLogger) Warn(msg string, args ...interface{})  {}
func (m *mockProfileLogger) Error(msg string, args ...interface{}) {}
func (m *mockProfileLogger) With(args ...interface{}) ports.Logger { return m }

// mockProfileRepository for testing
type mockProfileRepository struct {
	mu       sync.RWMutex
	profiles map[uuid.UUID]*domain.Profile
}

func newMockProfileRepository() *mockProfileRepository {
	return &mockProfileRepository{
		profiles: make(map[uuid.UUID]*domain.Profile),
	}
}

func (m *mockProfileRepository) Create(ctx context.Context, p *domain.Profile) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.profiles[p.ID] = p
	return nil
}

func (m *mockProfileRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Profile, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.profiles[id], nil
}

func (m *mockProfileRepository) Update(ctx context.Context, p *domain.Profile) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.profiles[p.ID] = p
	return nil
}

func (m *mockProfileRepository) Delete(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.profiles, id)
	return nil
}

func (m *mockProfileRepository) List(ctx context.Context, filter ports.ProfileFilter) ([]*domain.Profile, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*domain.Profile, 0, len(m.profiles))
	for _, p := range m.profiles {
		result = append(result, p)
	}
	return result, nil
}

func (m *mockProfileRepository) SaveProfileData(ctx context.Context, data *domain.ProfileData) error {
	return nil
}

func (m *mockProfileRepository) GetProfileData(ctx context.Context, profileID uuid.UUID) (*domain.ProfileData, error) {
	return &domain.ProfileData{ProfileID: profileID}, nil
}

func (m *mockProfileRepository) SaveFlameGraph(ctx context.Context, fg *domain.FlameGraph) error {
	return nil
}

func (m *mockProfileRepository) GetFlameGraph(ctx context.Context, profileID uuid.UUID) (*domain.FlameGraph, error) {
	return &domain.FlameGraph{ProfileID: profileID}, nil
}

func (m *mockProfileRepository) DeleteBefore(ctx context.Context, before time.Time) (int64, error) {
	return 0, nil
}

func TestNewProfileService(t *testing.T) {
	logger := &mockProfileLogger{}
	repo := newMockProfileRepository()
	tmpDir := filepath.Join(os.TempDir(), "forge-profile-test")
	defer os.RemoveAll(tmpDir)

	svc := NewProfileService(repo, tmpDir, logger)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.profileRepo == nil {
		t.Error("profile repo not set correctly")
	}
	if svc.logger == nil {
		t.Error("logger not set correctly")
	}
	if svc.profileDir != tmpDir {
		t.Errorf("expected profile dir '%s', got '%s'", tmpDir, svc.profileDir)
	}
	if svc.activeProfiles == nil {
		t.Error("active profiles map not initialized")
	}
}

func TestNewProfileService_DefaultDir(t *testing.T) {
	logger := &mockProfileLogger{}
	repo := newMockProfileRepository()

	svc := NewProfileService(repo, "", logger)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	// Should use default temp directory
	expectedDir := filepath.Join(os.TempDir(), "forge-profiles")
	if svc.profileDir != expectedDir {
		t.Errorf("expected default profile dir '%s', got '%s'", expectedDir, svc.profileDir)
	}
}

func TestProfileService_CaptureHeapProfile(t *testing.T) {
	logger := &mockProfileLogger{}
	repo := newMockProfileRepository()
	tmpDir := filepath.Join(os.TempDir(), "forge-profile-test-heap")
	defer os.RemoveAll(tmpDir)

	svc := NewProfileService(repo, tmpDir, logger)

	profile, err := svc.CaptureHeapProfile(context.Background(), "test-heap", "test-service")
	if err != nil {
		t.Fatalf("CaptureHeapProfile failed: %v", err)
	}

	if profile == nil {
		t.Fatal("expected non-nil profile")
	}
	if profile.Name != "test-heap" {
		t.Errorf("expected name 'test-heap', got '%s'", profile.Name)
	}
}

