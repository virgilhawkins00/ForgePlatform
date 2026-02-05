// Package services implements core business logic for the Forge platform.
package services

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/forge-platform/forge/internal/core/domain"
	"github.com/forge-platform/forge/internal/core/ports"
	"github.com/google/uuid"
)

// AlertService handles alert rule evaluation and alert management.
type AlertService struct {
	ruleRepo    ports.AlertRuleRepository
	alertRepo   ports.AlertRepository
	channelRepo ports.NotificationChannelRepository
	silenceRepo ports.SilenceRepository
	metricRepo  ports.MetricRepository
	logger      ports.Logger

	// Notification sender interface
	notifiers map[domain.NotificationChannelType]Notifier

	// Active alerts cache (fingerprint -> alert)
	activeAlerts map[string]*domain.Alert
	mu           sync.RWMutex

	// Evaluation state
	evaluating bool
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

// Notifier defines the interface for sending notifications.
type Notifier interface {
	Send(ctx context.Context, alert *domain.Alert, channel *domain.NotificationChannel) error
	Type() domain.NotificationChannelType
}

// NewAlertService creates a new alert service.
func NewAlertService(
	ruleRepo ports.AlertRuleRepository,
	alertRepo ports.AlertRepository,
	channelRepo ports.NotificationChannelRepository,
	silenceRepo ports.SilenceRepository,
	metricRepo ports.MetricRepository,
	logger ports.Logger,
) *AlertService {
	return &AlertService{
		ruleRepo:     ruleRepo,
		alertRepo:    alertRepo,
		channelRepo:  channelRepo,
		silenceRepo:  silenceRepo,
		metricRepo:   metricRepo,
		logger:       logger,
		notifiers:    make(map[domain.NotificationChannelType]Notifier),
		activeAlerts: make(map[string]*domain.Alert),
		stopCh:       make(chan struct{}),
	}
}

// RegisterNotifier registers a notification sender for a channel type.
func (s *AlertService) RegisterNotifier(notifier Notifier) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notifiers[notifier.Type()] = notifier
}

// Start begins the alert evaluation loop.
func (s *AlertService) Start(ctx context.Context, interval time.Duration) {
	s.mu.Lock()
	if s.evaluating {
		s.mu.Unlock()
		return
	}
	s.evaluating = true
	s.stopCh = make(chan struct{})
	s.mu.Unlock()

	s.wg.Add(1)
	go s.evaluationLoop(ctx, interval)
}

// Stop stops the alert evaluation loop.
func (s *AlertService) Stop() {
	s.mu.Lock()
	if !s.evaluating {
		s.mu.Unlock()
		return
	}
	s.evaluating = false
	close(s.stopCh)
	s.mu.Unlock()
	s.wg.Wait()
}

// evaluationLoop periodically evaluates alert rules.
func (s *AlertService) evaluationLoop(ctx context.Context, interval time.Duration) {
	defer s.wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial evaluation
	s.EvaluateAll(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.EvaluateAll(ctx)
		}
	}
}

// EvaluateAll evaluates all enabled alert rules.
func (s *AlertService) EvaluateAll(ctx context.Context) {
	if s.ruleRepo == nil {
		return
	}

	rules, err := s.ruleRepo.ListEnabled(ctx)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("Failed to list enabled rules", "error", err)
		}
		return
	}

	for _, rule := range rules {
		if err := s.EvaluateRule(ctx, rule); err != nil {
			if s.logger != nil {
				s.logger.Error("Failed to evaluate rule", "rule", rule.Name, "error", err)
			}
		}
	}
}

// EvaluateRule evaluates a single alert rule.
func (s *AlertService) EvaluateRule(ctx context.Context, rule *domain.AlertRule) error {
	// Query recent metrics
	query := ports.MetricQuery{
		Name:      rule.MetricName,
		Tags:      rule.Tags,
		StartTime: time.Now().Add(-rule.Duration * 2),
		EndTime:   time.Now(),
	}

	series, err := s.metricRepo.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query metrics: %w", err)
	}

	// Evaluate condition
	firing, value := s.evaluateCondition(rule, series)
	return s.processEvaluation(ctx, rule, firing, value)
}

// evaluateCondition checks if the alert condition is met.
func (s *AlertService) evaluateCondition(rule *domain.AlertRule, series *domain.MetricSeries) (bool, float64) {
	if series == nil || len(series.Points) == 0 {
		// Check for absence of data condition
		if rule.Condition == domain.ConditionAbsenceOfData {
			return true, 0
		}
		return false, 0
	}

	// Get the latest value for threshold checks
	latestValue := series.Points[len(series.Points)-1].Value

	switch rule.Condition {
	case domain.ConditionThresholdAbove:
		return latestValue > rule.Threshold, latestValue

	case domain.ConditionThresholdBelow:
		return latestValue < rule.Threshold, latestValue

	case domain.ConditionThresholdEqual:
		return latestValue == rule.Threshold, latestValue

	case domain.ConditionRateOfChange:
		rate := s.calculateRateOfChange(series, rule.RateWindow)
		return math.Abs(rate) > rule.Threshold, rate

	case domain.ConditionAnomalyDetection:
		isAnomaly, zScore := s.detectAnomaly(series, rule.AnomalyStdDev)
		return isAnomaly, zScore

	case domain.ConditionAbsenceOfData:
		return false, latestValue // Data is present
	}

	return false, 0
}

// calculateRateOfChange calculates the rate of change over the given window.
func (s *AlertService) calculateRateOfChange(series *domain.MetricSeries, window time.Duration) float64 {
	if len(series.Points) < 2 {
		return 0
	}

	cutoff := time.Now().Add(-window)
	var firstPoint, lastPoint *domain.MetricPoint

	for i := range series.Points {
		if series.Points[i].Timestamp.After(cutoff) {
			if firstPoint == nil {
				firstPoint = &series.Points[i]
			}
			lastPoint = &series.Points[i]
		}
	}

	if firstPoint == nil || lastPoint == nil || firstPoint == lastPoint {
		return 0
	}

	timeDiff := lastPoint.Timestamp.Sub(firstPoint.Timestamp).Seconds()
	if timeDiff == 0 {
		return 0
	}

	return (lastPoint.Value - firstPoint.Value) / timeDiff
}

// detectAnomaly uses z-score to detect anomalies.
func (s *AlertService) detectAnomaly(series *domain.MetricSeries, stdDevThreshold float64) (bool, float64) {
	if len(series.Points) < 10 {
		return false, 0
	}

	// Calculate mean and standard deviation
	var sum, sumSq float64
	for _, p := range series.Points {
		sum += p.Value
		sumSq += p.Value * p.Value
	}

	n := float64(len(series.Points))
	mean := sum / n
	variance := (sumSq / n) - (mean * mean)
	stdDev := math.Sqrt(variance)

	if stdDev == 0 {
		return false, 0
	}

	// Check latest value
	latest := series.Points[len(series.Points)-1].Value
	zScore := (latest - mean) / stdDev

	return math.Abs(zScore) > stdDevThreshold, zScore
}

// processEvaluation processes the result of rule evaluation.
func (s *AlertService) processEvaluation(ctx context.Context, rule *domain.AlertRule, firing bool, value float64) error {
	fingerprint := rule.ID.String() + ":" + rule.MetricName

	s.mu.Lock()
	existingAlert := s.activeAlerts[fingerprint]
	s.mu.Unlock()

	if firing {
		if existingAlert == nil {
			// Create new alert
			message := fmt.Sprintf("Alert %s: %s condition met (value: %.2f, threshold: %.2f)",
				rule.Name, rule.Condition, value, rule.Threshold)
			alert := domain.NewAlert(rule, value, message)

			// Check if should be silenced
			if s.shouldSilence(ctx, alert) {
				alert.Silence()
			} else {
				alert.Fire()
				// Send notifications
				s.sendNotifications(ctx, alert, rule.Channels)
			}

			if s.alertRepo != nil {
				if err := s.alertRepo.Create(ctx, alert); err != nil {
					return fmt.Errorf("failed to create alert: %w", err)
				}
			}

			s.mu.Lock()
			s.activeAlerts[fingerprint] = alert
			s.mu.Unlock()

			if s.logger != nil {
				s.logger.Info("Alert fired", "rule", rule.Name, "value", value)
			}
		} else {
			// Update existing alert
			existingAlert.Value = value
			existingAlert.LastEvaluated = time.Now()
			if s.alertRepo != nil {
				_ = s.alertRepo.Update(ctx, existingAlert)
			}
		}
	} else {
		if existingAlert != nil && existingAlert.State == domain.AlertStateFiring {
			// Resolve the alert
			existingAlert.Resolve()
			if s.alertRepo != nil {
				_ = s.alertRepo.Update(ctx, existingAlert)
			}

			s.mu.Lock()
			delete(s.activeAlerts, fingerprint)
			s.mu.Unlock()

			if s.logger != nil {
				s.logger.Info("Alert resolved", "rule", rule.Name)
			}
		}
	}

	return nil
}

// shouldSilence checks if an alert should be silenced.
func (s *AlertService) shouldSilence(ctx context.Context, alert *domain.Alert) bool {
	if s.silenceRepo == nil {
		return false
	}

	silences, err := s.silenceRepo.ListActive(ctx, time.Now())
	if err != nil {
		return false
	}

	for _, silence := range silences {
		if silence.Matches(alert.Labels) {
			return true
		}
	}
	return false
}

// sendNotifications sends notifications for an alert.
func (s *AlertService) sendNotifications(ctx context.Context, alert *domain.Alert, channelIDs []string) {
	if s.channelRepo == nil {
		return
	}

	for _, channelIDStr := range channelIDs {
		channelID, err := uuid.Parse(channelIDStr)
		if err != nil {
			continue
		}

		channel, err := s.channelRepo.GetByID(ctx, channelID)
		if err != nil || !channel.Enabled {
			continue
		}

		s.mu.RLock()
		notifier, ok := s.notifiers[channel.Type]
		s.mu.RUnlock()

		if !ok {
			if s.logger != nil {
				s.logger.Warn("No notifier for channel type", "type", channel.Type)
			}
			continue
		}

		go func(ch *domain.NotificationChannel) {
			if err := notifier.Send(ctx, alert, ch); err != nil {
				if s.logger != nil {
					s.logger.Error("Failed to send notification", "channel", ch.Name, "error", err)
				}
			}
		}(channel)
	}
}

// CreateRule creates a new alert rule.
func (s *AlertService) CreateRule(ctx context.Context, rule *domain.AlertRule) error {
	if s.ruleRepo == nil {
		return fmt.Errorf("rule repository not configured")
	}
	return s.ruleRepo.Create(ctx, rule)
}

// GetRule retrieves an alert rule by ID.
func (s *AlertService) GetRule(ctx context.Context, id uuid.UUID) (*domain.AlertRule, error) {
	if s.ruleRepo == nil {
		return nil, fmt.Errorf("rule repository not configured")
	}
	return s.ruleRepo.GetByID(ctx, id)
}

// UpdateRule updates an existing alert rule.
func (s *AlertService) UpdateRule(ctx context.Context, rule *domain.AlertRule) error {
	if s.ruleRepo == nil {
		return fmt.Errorf("rule repository not configured")
	}
	rule.UpdatedAt = time.Now()
	return s.ruleRepo.Update(ctx, rule)
}

// DeleteRule deletes an alert rule.
func (s *AlertService) DeleteRule(ctx context.Context, id uuid.UUID) error {
	if s.ruleRepo == nil {
		return fmt.Errorf("rule repository not configured")
	}
	return s.ruleRepo.Delete(ctx, id)
}

// ListRules lists all alert rules.
func (s *AlertService) ListRules(ctx context.Context) ([]*domain.AlertRule, error) {
	if s.ruleRepo == nil {
		return []*domain.AlertRule{}, nil
	}
	return s.ruleRepo.List(ctx)
}

// GetAlert retrieves an alert by ID.
func (s *AlertService) GetAlert(ctx context.Context, id uuid.UUID) (*domain.Alert, error) {
	if s.alertRepo == nil {
		return nil, fmt.Errorf("alert repository not configured")
	}
	return s.alertRepo.GetByID(ctx, id)
}

// ListAlerts lists alerts with optional filtering.
func (s *AlertService) ListAlerts(ctx context.Context, filter ports.AlertFilter) ([]*domain.Alert, error) {
	if s.alertRepo == nil {
		return []*domain.Alert{}, nil
	}
	return s.alertRepo.List(ctx, filter)
}

// ListActiveAlerts lists all currently active alerts.
func (s *AlertService) ListActiveAlerts(ctx context.Context) ([]*domain.Alert, error) {
	if s.alertRepo == nil {
		// Return from in-memory cache
		s.mu.RLock()
		defer s.mu.RUnlock()
		alerts := make([]*domain.Alert, 0, len(s.activeAlerts))
		for _, a := range s.activeAlerts {
			alerts = append(alerts, a)
		}
		return alerts, nil
	}
	return s.alertRepo.ListActive(ctx)
}

// AcknowledgeAlert acknowledges an alert.
func (s *AlertService) AcknowledgeAlert(ctx context.Context, id uuid.UUID, by, comment string) error {
	alert, err := s.GetAlert(ctx, id)
	if err != nil {
		return err
	}

	alert.Acknowledge(by, comment)

	if s.alertRepo != nil {
		return s.alertRepo.Update(ctx, alert)
	}
	return nil
}

// CreateSilence creates a new silence.
func (s *AlertService) CreateSilence(ctx context.Context, silence *domain.Silence) error {
	if s.silenceRepo == nil {
		return fmt.Errorf("silence repository not configured")
	}
	return s.silenceRepo.Create(ctx, silence)
}

// ListSilences lists all silences.
func (s *AlertService) ListSilences(ctx context.Context) ([]*domain.Silence, error) {
	if s.silenceRepo == nil {
		return []*domain.Silence{}, nil
	}
	return s.silenceRepo.List(ctx)
}

// DeleteSilence deletes a silence.
func (s *AlertService) DeleteSilence(ctx context.Context, id uuid.UUID) error {
	if s.silenceRepo == nil {
		return fmt.Errorf("silence repository not configured")
	}
	return s.silenceRepo.Delete(ctx, id)
}

// CreateChannel creates a new notification channel.
func (s *AlertService) CreateChannel(ctx context.Context, channel *domain.NotificationChannel) error {
	if s.channelRepo == nil {
		return fmt.Errorf("channel repository not configured")
	}
	return s.channelRepo.Create(ctx, channel)
}

// ListChannels lists all notification channels.
func (s *AlertService) ListChannels(ctx context.Context) ([]*domain.NotificationChannel, error) {
	if s.channelRepo == nil {
		return []*domain.NotificationChannel{}, nil
	}
	return s.channelRepo.List(ctx)
}

// DeleteChannel deletes a notification channel.
func (s *AlertService) DeleteChannel(ctx context.Context, id uuid.UUID) error {
	if s.channelRepo == nil {
		return fmt.Errorf("channel repository not configured")
	}
	return s.channelRepo.Delete(ctx, id)
}

// GetAlertStats returns alert statistics.
func (s *AlertService) GetAlertStats(ctx context.Context) (map[string]interface{}, error) {
	stats := map[string]interface{}{
		"active_alerts": len(s.activeAlerts),
	}

	if s.alertRepo != nil {
		counts, err := s.alertRepo.CountByState(ctx)
		if err == nil {
			stats["by_state"] = counts
		}
	}

	if s.ruleRepo != nil {
		rules, err := s.ruleRepo.List(ctx)
		if err == nil {
			stats["total_rules"] = len(rules)
			enabled := 0
			for _, r := range rules {
				if r.Enabled {
					enabled++
				}
			}
			stats["enabled_rules"] = enabled
		}
	}

	return stats, nil
}

