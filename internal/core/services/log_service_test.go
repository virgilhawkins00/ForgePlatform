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

// mockLogLogger for testing
type mockLogLogger struct{}

func (m *mockLogLogger) Debug(msg string, args ...interface{}) {}
func (m *mockLogLogger) Info(msg string, args ...interface{})  {}
func (m *mockLogLogger) Warn(msg string, args ...interface{})  {}
func (m *mockLogLogger) Error(msg string, args ...interface{}) {}
func (m *mockLogLogger) With(args ...interface{}) ports.Logger { return m }

// mockLogRepository for testing
type mockLogRepository struct {
	mu      sync.RWMutex
	entries []*domain.LogEntry
}

func newMockLogRepository() *mockLogRepository {
	return &mockLogRepository{
		entries: make([]*domain.LogEntry, 0),
	}
}

func (m *mockLogRepository) Create(ctx context.Context, entry *domain.LogEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = append(m.entries, entry)
	return nil
}

func (m *mockLogRepository) CreateBatch(ctx context.Context, entries []*domain.LogEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = append(m.entries, entries...)
	return nil
}

func (m *mockLogRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.LogEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, e := range m.entries {
		if e.ID == id {
			return e, nil
		}
	}
	return nil, nil
}

func (m *mockLogRepository) List(ctx context.Context, filter ports.LogFilter) ([]*domain.LogEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.entries, nil
}

func (m *mockLogRepository) Search(ctx context.Context, query string, filter ports.LogFilter) ([]*domain.LogEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.entries, nil
}

func (m *mockLogRepository) GetStats(ctx context.Context, startTime, endTime time.Time) (*domain.LogStats, error) {
	return &domain.LogStats{}, nil
}

func (m *mockLogRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockLogRepository) DeleteBefore(ctx context.Context, before time.Time) (int64, error) {
	return 0, nil
}

// mockLogParserRepository for testing
type mockLogParserRepository struct {
	mu      sync.RWMutex
	parsers []*domain.LogParser
}

func newMockLogParserRepository() *mockLogParserRepository {
	return &mockLogParserRepository{
		parsers: make([]*domain.LogParser, 0),
	}
}

func (m *mockLogParserRepository) Create(ctx context.Context, p *domain.LogParser) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.parsers = append(m.parsers, p)
	return nil
}

func (m *mockLogParserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.LogParser, error) {
	return nil, nil
}

func (m *mockLogParserRepository) GetByName(ctx context.Context, name string) (*domain.LogParser, error) {
	return nil, nil
}

func (m *mockLogParserRepository) Update(ctx context.Context, p *domain.LogParser) error {
	return nil
}

func (m *mockLogParserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockLogParserRepository) List(ctx context.Context) ([]*domain.LogParser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.parsers, nil
}

func (m *mockLogParserRepository) ListEnabled(ctx context.Context) ([]*domain.LogParser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*domain.LogParser, 0)
	for _, p := range m.parsers {
		if p.Enabled {
			result = append(result, p)
		}
	}
	return result, nil
}

// mockLogToMetricRuleRepository for testing
type mockLogToMetricRuleRepository struct {
	mu    sync.RWMutex
	rules []*domain.LogToMetricRule
}

func newMockLogToMetricRuleRepository() *mockLogToMetricRuleRepository {
	return &mockLogToMetricRuleRepository{
		rules: make([]*domain.LogToMetricRule, 0),
	}
}

func (m *mockLogToMetricRuleRepository) Create(ctx context.Context, r *domain.LogToMetricRule) error {
	return nil
}

func (m *mockLogToMetricRuleRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.LogToMetricRule, error) {
	return nil, nil
}

func (m *mockLogToMetricRuleRepository) Update(ctx context.Context, r *domain.LogToMetricRule) error {
	return nil
}

func (m *mockLogToMetricRuleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockLogToMetricRuleRepository) List(ctx context.Context) ([]*domain.LogToMetricRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.rules, nil
}

func (m *mockLogToMetricRuleRepository) ListEnabled(ctx context.Context) ([]*domain.LogToMetricRule, error) {
	return []*domain.LogToMetricRule{}, nil
}

func TestNewLogService(t *testing.T) {
	logger := &mockLogLogger{}
	logRepo := newMockLogRepository()
	parserRepo := newMockLogParserRepository()
	logToMetricRepo := newMockLogToMetricRuleRepository()
	metricRepo := newMockMetricRepositoryForAlert()

	svc := NewLogService(logRepo, parserRepo, logToMetricRepo, metricRepo, logger)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.logRepo == nil {
		t.Error("log repo not set correctly")
	}
	if svc.parserRepo == nil {
		t.Error("parser repo not set correctly")
	}
	if svc.logToMetricRepo == nil {
		t.Error("log to metric repo not set correctly")
	}
	if svc.metricRepo == nil {
		t.Error("metric repo not set correctly")
	}
	if svc.logger == nil {
		t.Error("logger not set correctly")
	}
	if svc.bufferSize != 1000 {
		t.Errorf("expected buffer size 1000, got %d", svc.bufferSize)
	}
	if svc.flushInterval != 5*time.Second {
		t.Errorf("expected flush interval 5s, got %v", svc.flushInterval)
	}
}

func TestLogService_RefreshParsers(t *testing.T) {
	logger := &mockLogLogger{}
	parserRepo := newMockLogParserRepository()

	svc := NewLogService(nil, parserRepo, nil, nil, logger)

	// Add a parser
	parser := &domain.LogParser{
		ID:      uuid.New(),
		Name:    "test-parser",
		Pattern: `(?P<level>\w+): (?P<message>.*)`,
		Enabled: true,
	}
	_ = parserRepo.Create(context.Background(), parser)

	// Refresh parsers
	err := svc.RefreshParsers(context.Background())
	if err != nil {
		t.Fatalf("RefreshParsers failed: %v", err)
	}

	if len(svc.parsers) != 1 {
		t.Errorf("expected 1 parser, got %d", len(svc.parsers))
	}
}

func TestLogService_RefreshParsers_NoRepo(t *testing.T) {
	logger := &mockLogLogger{}
	svc := NewLogService(nil, nil, nil, nil, logger)

	// Should not error with nil parser repo
	err := svc.RefreshParsers(context.Background())
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestLogService_Ingest(t *testing.T) {
	logger := &mockLogLogger{}
	logRepo := newMockLogRepository()

	svc := NewLogService(logRepo, nil, nil, nil, logger)

	entry := &domain.LogEntry{
		ID:        uuid.New(),
		Timestamp: time.Now(),
		Level:     domain.LogLevelInfo,
		Message:   "Test log message",
		Source:    "test",
	}

	err := svc.Ingest(context.Background(), entry)
	if err != nil {
		t.Fatalf("Ingest failed: %v", err)
	}

	if len(logRepo.entries) != 1 {
		t.Errorf("expected 1 entry in repo, got %d", len(logRepo.entries))
	}
}

func TestLogService_Ingest_NoRepo(t *testing.T) {
	logger := &mockLogLogger{}
	svc := NewLogService(nil, nil, nil, nil, logger)

	entry := &domain.LogEntry{
		ID:        uuid.New(),
		Timestamp: time.Now(),
		Level:     domain.LogLevelInfo,
		Message:   "Test log message",
	}

	// Should not error with nil log repo
	err := svc.Ingest(context.Background(), entry)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

