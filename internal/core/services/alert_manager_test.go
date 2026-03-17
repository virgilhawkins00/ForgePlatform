// Package services implements core business logic services.
package services

import (
	"context"
	"testing"
	"time"

	"github.com/forge-platform/forge/internal/core/domain"
	"github.com/forge-platform/forge/internal/core/ports"
	"github.com/google/uuid"
)

// mockAMLogger for testing
type mockAMLogger struct{}

func (m *mockAMLogger) Debug(msg string, args ...interface{}) {}
func (m *mockAMLogger) Info(msg string, args ...interface{})  {}
func (m *mockAMLogger) Warn(msg string, args ...interface{})  {}
func (m *mockAMLogger) Error(msg string, args ...interface{}) {}
func (m *mockAMLogger) With(args ...interface{}) ports.Logger { return m }

func TestNewAlertManager(t *testing.T) {
	logger := &mockAMLogger{}
	// Create a minimal alert service - manager can work with nil initially
	am := NewAlertManager(nil, logger)

	if am == nil {
		t.Fatal("expected non-nil alert manager")
	}
	if am.groupWait != 30*time.Second {
		t.Errorf("expected groupWait 30s, got %v", am.groupWait)
	}
	if am.groupInterval != 5*time.Minute {
		t.Errorf("expected groupInterval 5m, got %v", am.groupInterval)
	}
	if am.repeatInterval != 4*time.Hour {
		t.Errorf("expected repeatInterval 4h, got %v", am.repeatInterval)
	}
	if am.routes == nil {
		t.Error("routes should be initialized")
	}
	if am.escalationPolicies == nil {
		t.Error("escalationPolicies should be initialized")
	}
	if am.activeEscalations == nil {
		t.Error("activeEscalations should be initialized")
	}
	if am.alertGroups == nil {
		t.Error("alertGroups should be initialized")
	}
}

func TestAlertManager_AddRoute(t *testing.T) {
	logger := &mockAMLogger{}
	am := NewAlertManager(nil, logger)

	route := AlertRoute{
		Name:       "critical-alerts",
		Matchers:   map[string]string{"severity": "critical"},
		ChannelIDs: []string{"slack-ops", "pagerduty"},
		Continue:   false,
	}

	am.AddRoute(route)

	if len(am.routes) != 1 {
		t.Errorf("expected 1 route, got %d", len(am.routes))
	}
	if am.routes[0].Name != "critical-alerts" {
		t.Errorf("expected route name 'critical-alerts', got '%s'", am.routes[0].Name)
	}
}

func TestAlertManager_AddEscalationPolicy(t *testing.T) {
	logger := &mockAMLogger{}
	am := NewAlertManager(nil, logger)

	policy := &domain.EscalationPolicy{
		ID:   uuid.New(),
		Name: "default-escalation",
	}

	am.AddEscalationPolicy(policy)

	if len(am.escalationPolicies) != 1 {
		t.Errorf("expected 1 policy, got %d", len(am.escalationPolicies))
	}
	if stored := am.escalationPolicies[policy.ID]; stored.Name != "default-escalation" {
		t.Errorf("expected policy name 'default-escalation', got '%s'", stored.Name)
	}
}

func TestAlertManager_StartStop(t *testing.T) {
	logger := &mockAMLogger{}
	am := NewAlertManager(nil, logger)

	ctx := context.Background()

	// Start the manager
	am.Start(ctx)

	if !am.running {
		t.Error("expected manager to be running")
	}

	// Starting again should be a no-op
	am.Start(ctx)

	// Stop the manager
	am.Stop()

	if am.running {
		t.Error("expected manager to be stopped")
	}
}

func TestEscalationState_Fields(t *testing.T) {
	alertID := uuid.New()
	policyID := uuid.New()

	state := EscalationState{
		AlertID:      alertID,
		PolicyID:     policyID,
		CurrentLevel: 1,
		StartedAt:    time.Now(),
		Acknowledged: false,
	}

	if state.AlertID != alertID {
		t.Error("AlertID field mismatch")
	}
	if state.PolicyID != policyID {
		t.Error("PolicyID field mismatch")
	}
	if state.CurrentLevel != 1 {
		t.Error("CurrentLevel field mismatch")
	}
	if state.Acknowledged {
		t.Error("Acknowledged should be false")
	}
}

func TestAlertRoute_Fields(t *testing.T) {
	route := AlertRoute{
		Name:       "test-route",
		Matchers:   map[string]string{"env": "prod", "severity": "warning"},
		ChannelIDs: []string{"channel1", "channel2"},
		Continue:   true,
		MuteTimeIntervals: []string{"weekend", "maintenance"},
	}

	if route.Name != "test-route" {
		t.Error("Name field mismatch")
	}
	if len(route.Matchers) != 2 {
		t.Error("Matchers field mismatch")
	}
	if len(route.ChannelIDs) != 2 {
		t.Error("ChannelIDs field mismatch")
	}
	if !route.Continue {
		t.Error("Continue field should be true")
	}
	if len(route.MuteTimeIntervals) != 2 {
		t.Error("MuteTimeIntervals field mismatch")
	}
}

