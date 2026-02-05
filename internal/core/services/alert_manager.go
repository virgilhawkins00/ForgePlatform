// Package services implements core business logic for the Forge platform.
package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/forge-platform/forge/internal/core/domain"
	"github.com/forge-platform/forge/internal/core/ports"
	"github.com/google/uuid"
)

// AlertManager coordinates alert routing, grouping, silencing, and escalation.
type AlertManager struct {
	alertSvc     *AlertService
	logger       ports.Logger

	// Routing rules (label matchers -> channel IDs)
	routes []AlertRoute

	// Escalation policies
	escalationPolicies map[uuid.UUID]*domain.EscalationPolicy

	// Active escalations (alert ID -> escalation state)
	activeEscalations map[uuid.UUID]*EscalationState

	// Alert groups (group key -> alerts)
	alertGroups map[string][]*domain.Alert

	// Configuration
	groupWait     time.Duration // How long to wait before sending first notification for a group
	groupInterval time.Duration // How long to wait before sending notifications for new alerts in group
	repeatInterval time.Duration // How long to wait before re-sending notification

	mu       sync.RWMutex
	stopCh   chan struct{}
	wg       sync.WaitGroup
	running  bool
}

// AlertRoute defines a routing rule for alerts.
type AlertRoute struct {
	Name       string            // Route name
	Matchers   map[string]string // Label matchers (AND semantics)
	ChannelIDs []string          // Channels to notify
	Continue   bool              // If true, continue checking other routes
	MuteTimeIntervals []string   // Names of mute time intervals
}

// EscalationState tracks the escalation state of an alert.
type EscalationState struct {
	AlertID        uuid.UUID
	PolicyID       uuid.UUID
	CurrentLevel   int
	StartedAt      time.Time
	LastNotifiedAt time.Time
	NextEscalateAt time.Time
	Acknowledged   bool
}

// NewAlertManager creates a new alert manager.
func NewAlertManager(alertSvc *AlertService, logger ports.Logger) *AlertManager {
	return &AlertManager{
		alertSvc:           alertSvc,
		logger:             logger,
		routes:             make([]AlertRoute, 0),
		escalationPolicies: make(map[uuid.UUID]*domain.EscalationPolicy),
		activeEscalations:  make(map[uuid.UUID]*EscalationState),
		alertGroups:        make(map[string][]*domain.Alert),
		groupWait:          30 * time.Second,
		groupInterval:      5 * time.Minute,
		repeatInterval:     4 * time.Hour,
		stopCh:             make(chan struct{}),
	}
}

// AddRoute adds a routing rule.
func (m *AlertManager) AddRoute(route AlertRoute) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.routes = append(m.routes, route)
}

// AddEscalationPolicy adds an escalation policy.
func (m *AlertManager) AddEscalationPolicy(policy *domain.EscalationPolicy) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.escalationPolicies[policy.ID] = policy
}

// Start starts the alert manager background processes.
func (m *AlertManager) Start(ctx context.Context) {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	m.running = true
	m.stopCh = make(chan struct{})
	m.mu.Unlock()

	m.wg.Add(1)
	go m.escalationLoop(ctx)
}

// Stop stops the alert manager.
func (m *AlertManager) Stop() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}
	m.running = false
	close(m.stopCh)
	m.mu.Unlock()
	m.wg.Wait()
}

// escalationLoop handles alert escalations.
func (m *AlertManager) escalationLoop(ctx context.Context) {
	defer m.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.processEscalations(ctx)
		}
	}
}

// processEscalations checks and processes escalations.
func (m *AlertManager) processEscalations(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	for alertID, state := range m.activeEscalations {
		if state.Acknowledged {
			continue
		}

		if now.After(state.NextEscalateAt) {
			policy, ok := m.escalationPolicies[state.PolicyID]
			if !ok {
				continue
			}

			// Move to next level
			if state.CurrentLevel < len(policy.Levels)-1 {
				state.CurrentLevel++
				level := policy.Levels[state.CurrentLevel]
				state.NextEscalateAt = now.Add(level.Delay)
				state.LastNotifiedAt = now

				// Get alert and send notifications for this level
				if m.alertSvc != nil {
					alert, err := m.alertSvc.GetAlert(ctx, alertID)
					if err == nil && alert != nil {
						m.alertSvc.sendNotifications(ctx, alert, level.ChannelIDs)
					}
				}

				if m.logger != nil {
					m.logger.Info("Alert escalated",
						"alert_id", alertID,
						"level", state.CurrentLevel,
						"policy", policy.Name)
				}
			}
		}
	}
}

// RouteAlert routes an alert to the appropriate channels based on routing rules.
func (m *AlertManager) RouteAlert(ctx context.Context, alert *domain.Alert) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var channels []string

	for _, route := range m.routes {
		if m.matchesRoute(alert, route) {
			channels = append(channels, route.ChannelIDs...)
			if !route.Continue {
				break
			}
		}
	}

	return channels
}

// matchesRoute checks if an alert matches a routing rule.
func (m *AlertManager) matchesRoute(alert *domain.Alert, route AlertRoute) bool {
	for key, value := range route.Matchers {
		if alert.Labels[key] != value {
			return false
		}
	}
	return true
}

// GroupAlert groups an alert with related alerts.
func (m *AlertManager) GroupAlert(alert *domain.Alert, groupByLabels []string) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Build group key from specified labels
	groupKey := m.buildGroupKey(alert, groupByLabels)

	m.alertGroups[groupKey] = append(m.alertGroups[groupKey], alert)

	return groupKey
}

// buildGroupKey creates a unique key for grouping alerts.
func (m *AlertManager) buildGroupKey(alert *domain.Alert, groupByLabels []string) string {
	if len(groupByLabels) == 0 {
		return alert.RuleName
	}

	key := ""
	for _, label := range groupByLabels {
		if v, ok := alert.Labels[label]; ok {
			key += label + "=" + v + ","
		}
	}
	return key
}

// GetAlertGroup returns all alerts in a group.
func (m *AlertManager) GetAlertGroup(groupKey string) []*domain.Alert {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if alerts, ok := m.alertGroups[groupKey]; ok {
		result := make([]*domain.Alert, len(alerts))
		copy(result, alerts)
		return result
	}
	return nil
}

// StartEscalation starts escalation for an alert.
func (m *AlertManager) StartEscalation(alertID, policyID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	policy, ok := m.escalationPolicies[policyID]
	if !ok {
		return fmt.Errorf("escalation policy not found: %s", policyID)
	}

	if len(policy.Levels) == 0 {
		return fmt.Errorf("escalation policy has no levels")
	}

	now := time.Now()
	m.activeEscalations[alertID] = &EscalationState{
		AlertID:        alertID,
		PolicyID:       policyID,
		CurrentLevel:   0,
		StartedAt:      now,
		LastNotifiedAt: now,
		NextEscalateAt: now.Add(policy.Levels[0].Delay),
	}

	return nil
}

// AcknowledgeEscalation stops escalation for an alert.
func (m *AlertManager) AcknowledgeEscalation(alertID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if state, ok := m.activeEscalations[alertID]; ok {
		state.Acknowledged = true
	}
}

// StopEscalation removes an alert from active escalations.
func (m *AlertManager) StopEscalation(alertID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.activeEscalations, alertID)
}

// GetEscalationState returns the escalation state for an alert.
func (m *AlertManager) GetEscalationState(alertID uuid.UUID) *EscalationState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if state, ok := m.activeEscalations[alertID]; ok {
		return &EscalationState{
			AlertID:        state.AlertID,
			PolicyID:       state.PolicyID,
			CurrentLevel:   state.CurrentLevel,
			StartedAt:      state.StartedAt,
			LastNotifiedAt: state.LastNotifiedAt,
			NextEscalateAt: state.NextEscalateAt,
			Acknowledged:   state.Acknowledged,
		}
	}
	return nil
}

// GetStats returns alert manager statistics.
func (m *AlertManager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"routes":              len(m.routes),
		"escalation_policies": len(m.escalationPolicies),
		"active_escalations":  len(m.activeEscalations),
		"alert_groups":        len(m.alertGroups),
	}
}

