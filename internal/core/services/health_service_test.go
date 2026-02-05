package services

import (
	"context"
	"testing"
	"time"

	"github.com/forge-platform/forge/internal/core/ports"
)

// mockLogger implements ports.Logger for testing.
type mockLogger struct{}

func (l *mockLogger) Debug(msg string, keysAndValues ...interface{}) {}
func (l *mockLogger) Info(msg string, keysAndValues ...interface{})  {}
func (l *mockLogger) Warn(msg string, keysAndValues ...interface{})  {}
func (l *mockLogger) Error(msg string, keysAndValues ...interface{}) {}
func (l *mockLogger) With(keysAndValues ...interface{}) ports.Logger { return l }

func TestNewHealthService(t *testing.T) {
	svc := NewHealthService("1.0.0", &mockLogger{})
	if svc == nil {
		t.Fatal("NewHealthService() returned nil")
	}
	if svc.version != "1.0.0" {
		t.Errorf("version = %v, want 1.0.0", svc.version)
	}
}

func TestHealthService_RegisterChecker(t *testing.T) {
	svc := NewHealthService("1.0.0", &mockLogger{})

	checker := func(ctx context.Context) ComponentHealth {
		return ComponentHealth{
			Status:  HealthStatusHealthy,
			Message: "OK",
		}
	}

	svc.RegisterChecker("test", checker)

	if len(svc.checkers) != 1 {
		t.Errorf("len(checkers) = %d, want 1", len(svc.checkers))
	}
}

func TestHealthService_Check(t *testing.T) {
	svc := NewHealthService("1.0.0", &mockLogger{})

	// Register healthy component
	svc.RegisterChecker("healthy", func(ctx context.Context) ComponentHealth {
		return ComponentHealth{
			Status:    HealthStatusHealthy,
			Message:   "All good",
			CheckedAt: time.Now(),
		}
	})

	ctx := context.Background()
	health := svc.Check(ctx)

	if health.Status != HealthStatusHealthy {
		t.Errorf("Status = %v, want healthy", health.Status)
	}
	if health.Version != "1.0.0" {
		t.Errorf("Version = %v, want 1.0.0", health.Version)
	}
	if len(health.Components) != 1 {
		t.Errorf("len(Components) = %d, want 1", len(health.Components))
	}
}

func TestHealthService_Check_Degraded(t *testing.T) {
	svc := NewHealthService("1.0.0", &mockLogger{})

	svc.RegisterChecker("healthy", func(ctx context.Context) ComponentHealth {
		return ComponentHealth{Status: HealthStatusHealthy}
	})
	svc.RegisterChecker("degraded", func(ctx context.Context) ComponentHealth {
		return ComponentHealth{Status: HealthStatusDegraded}
	})

	ctx := context.Background()
	health := svc.Check(ctx)

	if health.Status != HealthStatusDegraded {
		t.Errorf("Status = %v, want degraded", health.Status)
	}
}

func TestHealthService_Check_Unhealthy(t *testing.T) {
	svc := NewHealthService("1.0.0", &mockLogger{})

	svc.RegisterChecker("healthy", func(ctx context.Context) ComponentHealth {
		return ComponentHealth{Status: HealthStatusHealthy}
	})
	svc.RegisterChecker("unhealthy", func(ctx context.Context) ComponentHealth {
		return ComponentHealth{Status: HealthStatusUnhealthy}
	})

	ctx := context.Background()
	health := svc.Check(ctx)

	if health.Status != HealthStatusUnhealthy {
		t.Errorf("Status = %v, want unhealthy", health.Status)
	}
}

func TestHealthService_CheckLiveness(t *testing.T) {
	svc := NewHealthService("1.0.0", &mockLogger{})
	ctx := context.Background()

	if !svc.CheckLiveness(ctx) {
		t.Error("CheckLiveness() = false, want true")
	}
}

func TestHealthService_CheckReadiness(t *testing.T) {
	svc := NewHealthService("1.0.0", &mockLogger{})
	ctx := context.Background()

	// No checkers = healthy
	if !svc.CheckReadiness(ctx) {
		t.Error("CheckReadiness() = false, want true (no checkers)")
	}

	// Add unhealthy checker
	svc.RegisterChecker("unhealthy", func(ctx context.Context) ComponentHealth {
		return ComponentHealth{Status: HealthStatusUnhealthy}
	})

	if svc.CheckReadiness(ctx) {
		t.Error("CheckReadiness() = true, want false (unhealthy component)")
	}
}

func TestHealthService_GetUptime(t *testing.T) {
	svc := NewHealthService("1.0.0", &mockLogger{})

	time.Sleep(10 * time.Millisecond)
	uptime := svc.GetUptime()

	if uptime < 10*time.Millisecond {
		t.Errorf("Uptime = %v, want >= 10ms", uptime)
	}
}

func TestHealthService_getSystemMetrics(t *testing.T) {
	svc := NewHealthService("1.0.0", &mockLogger{})
	metrics := svc.getSystemMetrics()

	if metrics.GoVersion == "" {
		t.Error("GoVersion is empty")
	}
	if metrics.NumCPU <= 0 {
		t.Errorf("NumCPU = %d, want > 0", metrics.NumCPU)
	}
	if metrics.NumGoroutine <= 0 {
		t.Errorf("NumGoroutine = %d, want > 0", metrics.NumGoroutine)
	}
}

