// Package services implements core business logic services.
package services

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/forge-platform/forge/internal/core/domain"
	"github.com/forge-platform/forge/internal/core/ports"
	"github.com/google/uuid"
)

// mockAlertLogger for testing
type mockAlertLogger struct{}

func (m *mockAlertLogger) Debug(msg string, args ...interface{}) {}
func (m *mockAlertLogger) Info(msg string, args ...interface{})  {}
func (m *mockAlertLogger) Warn(msg string, args ...interface{})  {}
func (m *mockAlertLogger) Error(msg string, args ...interface{}) {}
func (m *mockAlertLogger) With(args ...interface{}) ports.Logger { return m }

// mockAlertRuleRepository for testing
type mockAlertRuleRepository struct {
	mu    sync.RWMutex
	rules map[uuid.UUID]*domain.AlertRule
}

func newMockAlertRuleRepository() *mockAlertRuleRepository {
	return &mockAlertRuleRepository{
		rules: make(map[uuid.UUID]*domain.AlertRule),
	}
}

func (m *mockAlertRuleRepository) Create(ctx context.Context, rule *domain.AlertRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rules[rule.ID] = rule
	return nil
}

func (m *mockAlertRuleRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.AlertRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.rules[id], nil
}

func (m *mockAlertRuleRepository) GetByName(ctx context.Context, name string) (*domain.AlertRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, r := range m.rules {
		if r.Name == name {
			return r, nil
		}
	}
	return nil, nil
}

func (m *mockAlertRuleRepository) Update(ctx context.Context, rule *domain.AlertRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rules[rule.ID] = rule
	return nil
}

func (m *mockAlertRuleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.rules, id)
	return nil
}

func (m *mockAlertRuleRepository) List(ctx context.Context) ([]*domain.AlertRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*domain.AlertRule, 0, len(m.rules))
	for _, r := range m.rules {
		result = append(result, r)
	}
	return result, nil
}

func (m *mockAlertRuleRepository) ListEnabled(ctx context.Context) ([]*domain.AlertRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*domain.AlertRule, 0)
	for _, r := range m.rules {
		if r.Enabled {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *mockAlertRuleRepository) ListDue(ctx context.Context, now time.Time) ([]*domain.AlertRule, error) {
	return m.ListEnabled(ctx)
}

// mockAlertRepository for testing
type mockAlertRepository struct {
	mu     sync.RWMutex
	alerts map[uuid.UUID]*domain.Alert
}

func newMockAlertRepository() *mockAlertRepository {
	return &mockAlertRepository{
		alerts: make(map[uuid.UUID]*domain.Alert),
	}
}

func (m *mockAlertRepository) Create(ctx context.Context, alert *domain.Alert) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alerts[alert.ID] = alert
	return nil
}

func (m *mockAlertRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Alert, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.alerts[id], nil
}

func (m *mockAlertRepository) GetByFingerprint(ctx context.Context, fp string) (*domain.Alert, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, a := range m.alerts {
		if a.Fingerprint == fp {
			return a, nil
		}
	}
	return nil, nil
}

func (m *mockAlertRepository) Update(ctx context.Context, alert *domain.Alert) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alerts[alert.ID] = alert
	return nil
}

func (m *mockAlertRepository) Delete(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.alerts, id)
	return nil
}

func (m *mockAlertRepository) List(ctx context.Context, filter ports.AlertFilter) ([]*domain.Alert, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*domain.Alert, 0, len(m.alerts))
	for _, a := range m.alerts {
		result = append(result, a)
	}
	return result, nil
}

func (m *mockAlertRepository) ListActive(ctx context.Context) ([]*domain.Alert, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*domain.Alert, 0)
	for _, a := range m.alerts {
		if a.State == domain.AlertStateFiring || a.State == domain.AlertStatePending {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockAlertRepository) CountByState(ctx context.Context) (map[domain.AlertState]int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	counts := make(map[domain.AlertState]int64)
	for _, a := range m.alerts {
		counts[a.State]++
	}
	return counts, nil
}

// mockNotificationChannelRepository for testing
type mockNotificationChannelRepository struct {
	mu       sync.RWMutex
	channels map[uuid.UUID]*domain.NotificationChannel
}

func newMockNotificationChannelRepository() *mockNotificationChannelRepository {
	return &mockNotificationChannelRepository{
		channels: make(map[uuid.UUID]*domain.NotificationChannel),
	}
}

func (m *mockNotificationChannelRepository) Create(ctx context.Context, ch *domain.NotificationChannel) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.channels[ch.ID] = ch
	return nil
}

func (m *mockNotificationChannelRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.NotificationChannel, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.channels[id], nil
}

func (m *mockNotificationChannelRepository) GetByName(ctx context.Context, name string) (*domain.NotificationChannel, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, ch := range m.channels {
		if ch.Name == name {
			return ch, nil
		}
	}
	return nil, nil
}

func (m *mockNotificationChannelRepository) Update(ctx context.Context, ch *domain.NotificationChannel) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.channels[ch.ID] = ch
	return nil
}

func (m *mockNotificationChannelRepository) Delete(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.channels, id)
	return nil
}

func (m *mockNotificationChannelRepository) List(ctx context.Context) ([]*domain.NotificationChannel, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*domain.NotificationChannel, 0, len(m.channels))
	for _, ch := range m.channels {
		result = append(result, ch)
	}
	return result, nil
}

func (m *mockNotificationChannelRepository) ListEnabled(ctx context.Context) ([]*domain.NotificationChannel, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*domain.NotificationChannel, 0)
	for _, ch := range m.channels {
		if ch.Enabled {
			result = append(result, ch)
		}
	}
	return result, nil
}

// mockSilenceRepository for testing
type mockSilenceRepository struct {
	mu       sync.RWMutex
	silences map[uuid.UUID]*domain.Silence
}

func newMockSilenceRepository() *mockSilenceRepository {
	return &mockSilenceRepository{
		silences: make(map[uuid.UUID]*domain.Silence),
	}
}

func (m *mockSilenceRepository) Create(ctx context.Context, s *domain.Silence) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.silences[s.ID] = s
	return nil
}

func (m *mockSilenceRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Silence, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.silences[id], nil
}

func (m *mockSilenceRepository) Update(ctx context.Context, s *domain.Silence) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.silences[s.ID] = s
	return nil
}

func (m *mockSilenceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.silences, id)
	return nil
}

func (m *mockSilenceRepository) List(ctx context.Context) ([]*domain.Silence, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*domain.Silence, 0, len(m.silences))
	for _, s := range m.silences {
		result = append(result, s)
	}
	return result, nil
}

func (m *mockSilenceRepository) ListActive(ctx context.Context, now time.Time) ([]*domain.Silence, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*domain.Silence, 0)
	for _, s := range m.silences {
		if now.After(s.StartsAt) && now.Before(s.EndsAt) {
			result = append(result, s)
		}
	}
	return result, nil
}

// mockMetricRepositoryForAlert for testing
type mockMetricRepositoryForAlert struct {
	mu      sync.RWMutex
	metrics []*domain.Metric
}

func newMockMetricRepositoryForAlert() *mockMetricRepositoryForAlert {
	return &mockMetricRepositoryForAlert{
		metrics: make([]*domain.Metric, 0),
	}
}

func (m *mockMetricRepositoryForAlert) Record(ctx context.Context, metric *domain.Metric) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metrics = append(m.metrics, metric)
	return nil
}

func (m *mockMetricRepositoryForAlert) RecordBatch(ctx context.Context, metrics []*domain.Metric) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metrics = append(m.metrics, metrics...)
	return nil
}

func (m *mockMetricRepositoryForAlert) Query(ctx context.Context, query ports.MetricQuery) (*domain.MetricSeries, error) {
	return &domain.MetricSeries{Name: query.Name}, nil
}

func (m *mockMetricRepositoryForAlert) QueryMultiple(ctx context.Context, query ports.MetricQuery) ([]*domain.MetricSeries, error) {
	return []*domain.MetricSeries{{Name: query.Name}}, nil
}

func (m *mockMetricRepositoryForAlert) QueryWithAggregation(ctx context.Context, query ports.MetricQuery) ([]ports.AggregatedResult, error) {
	return []ports.AggregatedResult{}, nil
}

func (m *mockMetricRepositoryForAlert) Aggregate(ctx context.Context, query ports.MetricQuery, resolution string) (*domain.AggregatedMetric, error) {
	return &domain.AggregatedMetric{}, nil
}

func (m *mockMetricRepositoryForAlert) RecordAggregated(ctx context.Context, agg *domain.AggregatedMetric) error {
	return nil
}

func (m *mockMetricRepositoryForAlert) RecordAggregatedBatch(ctx context.Context, aggs []*domain.AggregatedMetric) error {
	return nil
}

func (m *mockMetricRepositoryForAlert) QueryAggregated(ctx context.Context, query ports.MetricQuery, resolution string) ([]*domain.AggregatedMetric, error) {
	return []*domain.AggregatedMetric{}, nil
}

func (m *mockMetricRepositoryForAlert) DeleteBefore(ctx context.Context, before time.Time) (int64, error) {
	return 0, nil
}

func (m *mockMetricRepositoryForAlert) DeleteAggregatedBefore(ctx context.Context, before time.Time, resolution string) (int64, error) {
	return 0, nil
}

func (m *mockMetricRepositoryForAlert) GetDistinctSeries(ctx context.Context) ([]ports.SeriesInfo, error) {
	return []ports.SeriesInfo{}, nil
}

func (m *mockMetricRepositoryForAlert) GetStats(ctx context.Context) (*ports.MetricStats, error) {
	return &ports.MetricStats{}, nil
}

// mockNotifier for testing
type mockNotifier struct {
	channelType domain.NotificationChannelType
	sendCalled  bool
	sendErr     error
}

func (m *mockNotifier) Send(ctx context.Context, alert *domain.Alert, channel *domain.NotificationChannel) error {
	m.sendCalled = true
	return m.sendErr
}

func (m *mockNotifier) Type() domain.NotificationChannelType {
	return m.channelType
}

func TestNewAlertService(t *testing.T) {
	logger := &mockAlertLogger{}
	ruleRepo := newMockAlertRuleRepository()
	alertRepo := newMockAlertRepository()
	channelRepo := newMockNotificationChannelRepository()
	silenceRepo := newMockSilenceRepository()
	metricRepo := newMockMetricRepositoryForAlert()

	svc := NewAlertService(ruleRepo, alertRepo, channelRepo, silenceRepo, metricRepo, logger)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.ruleRepo == nil {
		t.Error("rule repo not set correctly")
	}
	if svc.alertRepo == nil {
		t.Error("alert repo not set correctly")
	}
	if svc.channelRepo == nil {
		t.Error("channel repo not set correctly")
	}
	if svc.silenceRepo == nil {
		t.Error("silence repo not set correctly")
	}
	if svc.metricRepo == nil {
		t.Error("metric repo not set correctly")
	}
	if svc.logger == nil {
		t.Error("logger not set correctly")
	}
}

func TestAlertService_RegisterNotifier(t *testing.T) {
	logger := &mockAlertLogger{}
	svc := NewAlertService(nil, nil, nil, nil, nil, logger)

	notifier := &mockNotifier{channelType: domain.ChannelSlack}
	svc.RegisterNotifier(notifier)

	if svc.notifiers[domain.ChannelSlack] != notifier {
		t.Error("notifier not registered correctly")
	}
}

func TestAlertService_StartStop(t *testing.T) {
	logger := &mockAlertLogger{}
	ruleRepo := newMockAlertRuleRepository()
	svc := NewAlertService(ruleRepo, nil, nil, nil, nil, logger)

	ctx := context.Background()

	// Start the service
	svc.Start(ctx, 100*time.Millisecond)

	// Give it a moment to start
	time.Sleep(50 * time.Millisecond)

	if !svc.evaluating {
		t.Error("expected evaluating to be true after start")
	}

	// Stop the service
	svc.Stop()

	if svc.evaluating {
		t.Error("expected evaluating to be false after stop")
	}
}

func TestAlertService_StartTwice(t *testing.T) {
	logger := &mockAlertLogger{}
	ruleRepo := newMockAlertRuleRepository()
	svc := NewAlertService(ruleRepo, nil, nil, nil, nil, logger)

	ctx := context.Background()

	// Start twice should be idempotent
	svc.Start(ctx, 100*time.Millisecond)
	svc.Start(ctx, 100*time.Millisecond)

	if !svc.evaluating {
		t.Error("expected evaluating to be true")
	}

	svc.Stop()
}

func TestAlertService_StopNotStarted(t *testing.T) {
	logger := &mockAlertLogger{}
	svc := NewAlertService(nil, nil, nil, nil, nil, logger)

	// Stop without start should not panic
	svc.Stop()

	if svc.evaluating {
		t.Error("expected evaluating to be false")
	}
}

func TestAlertService_EvaluateAll_NoRuleRepo(t *testing.T) {
	logger := &mockAlertLogger{}
	svc := NewAlertService(nil, nil, nil, nil, nil, logger)

	// Should not panic with nil rule repo
	svc.EvaluateAll(context.Background())
}

