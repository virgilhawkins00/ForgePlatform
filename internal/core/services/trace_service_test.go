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

// mockTraceLogger for testing
type mockTraceLogger struct{}

func (m *mockTraceLogger) Debug(msg string, args ...interface{}) {}
func (m *mockTraceLogger) Info(msg string, args ...interface{})  {}
func (m *mockTraceLogger) Warn(msg string, args ...interface{})  {}
func (m *mockTraceLogger) Error(msg string, args ...interface{}) {}
func (m *mockTraceLogger) With(args ...interface{}) ports.Logger { return m }

// mockTraceRepository for testing
type mockTraceRepository struct {
	mu     sync.RWMutex
	traces map[uuid.UUID]*domain.Trace
}

func newMockTraceRepository() *mockTraceRepository {
	return &mockTraceRepository{
		traces: make(map[uuid.UUID]*domain.Trace),
	}
}

func (m *mockTraceRepository) Create(ctx context.Context, trace *domain.Trace) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.traces[trace.ID] = trace
	return nil
}

func (m *mockTraceRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Trace, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.traces[id], nil
}

func (m *mockTraceRepository) GetByTraceID(ctx context.Context, traceID domain.TraceID) (*domain.Trace, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, t := range m.traces {
		if t.TraceID == traceID {
			return t, nil
		}
	}
	return nil, nil
}

func (m *mockTraceRepository) Update(ctx context.Context, trace *domain.Trace) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.traces[trace.ID] = trace
	return nil
}

func (m *mockTraceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.traces, id)
	return nil
}

func (m *mockTraceRepository) List(ctx context.Context, filter ports.TraceFilter) ([]*domain.Trace, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*domain.Trace, 0, len(m.traces))
	for _, t := range m.traces {
		result = append(result, t)
	}
	return result, nil
}

func (m *mockTraceRepository) DeleteBefore(ctx context.Context, before time.Time) (int64, error) {
	return 0, nil
}

func (m *mockTraceRepository) GetServiceMap(ctx context.Context, startTime, endTime time.Time) (*domain.ServiceMap, error) {
	return &domain.ServiceMap{
		Nodes:     []domain.ServiceMapNode{},
		UpdatedAt: time.Now(),
	}, nil
}

// mockSpanRepository for testing
type mockSpanRepository struct {
	mu    sync.RWMutex
	spans map[uuid.UUID]*domain.Span
}

func newMockSpanRepository() *mockSpanRepository {
	return &mockSpanRepository{
		spans: make(map[uuid.UUID]*domain.Span),
	}
}

func (m *mockSpanRepository) Create(ctx context.Context, span *domain.Span) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.spans[span.ID] = span
	return nil
}

func (m *mockSpanRepository) CreateBatch(ctx context.Context, spans []*domain.Span) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range spans {
		m.spans[s.ID] = s
	}
	return nil
}

func (m *mockSpanRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Span, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.spans[id], nil
}

func (m *mockSpanRepository) GetBySpanID(ctx context.Context, traceID domain.TraceID, spanID domain.SpanID) (*domain.Span, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, s := range m.spans {
		if s.TraceID == traceID && s.SpanID == spanID {
			return s, nil
		}
	}
	return nil, nil
}

func (m *mockSpanRepository) Update(ctx context.Context, span *domain.Span) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.spans[span.ID] = span
	return nil
}

func (m *mockSpanRepository) Delete(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.spans, id)
	return nil
}

func (m *mockSpanRepository) ListByTraceID(ctx context.Context, traceID domain.TraceID) ([]*domain.Span, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*domain.Span, 0)
	for _, s := range m.spans {
		if s.TraceID == traceID {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockSpanRepository) List(ctx context.Context, filter ports.SpanFilter) ([]*domain.Span, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*domain.Span, 0, len(m.spans))
	for _, s := range m.spans {
		result = append(result, s)
	}
	return result, nil
}

func (m *mockSpanRepository) DeleteByTraceID(ctx context.Context, traceID domain.TraceID) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := int64(0)
	for id, s := range m.spans {
		if s.TraceID == traceID {
			delete(m.spans, id)
			count++
		}
	}
	return count, nil
}

func TestNewTraceService(t *testing.T) {
	logger := &mockTraceLogger{}
	traceRepo := newMockTraceRepository()
	spanRepo := newMockSpanRepository()

	svc := NewTraceService(traceRepo, spanRepo, logger)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.traceRepo == nil {
		t.Error("trace repo not set correctly")
	}
	if svc.spanRepo == nil {
		t.Error("span repo not set correctly")
	}
	if svc.logger == nil {
		t.Error("logger not set correctly")
	}
	if svc.activeTraces == nil {
		t.Error("active traces map not initialized")
	}
}

func TestTraceService_StartTrace(t *testing.T) {
	logger := &mockTraceLogger{}
	traceRepo := newMockTraceRepository()
	spanRepo := newMockSpanRepository()

	svc := NewTraceService(traceRepo, spanRepo, logger)

	trace, err := svc.StartTrace(context.Background(), "test-service", "test-operation")
	if err != nil {
		t.Fatalf("StartTrace failed: %v", err)
	}

	if trace == nil {
		t.Fatal("expected non-nil trace")
	}
	if trace.ServiceName != "test-service" {
		t.Errorf("expected service name 'test-service', got '%s'", trace.ServiceName)
	}
	if trace.Name != "test-operation" {
		t.Errorf("expected name 'test-operation', got '%s'", trace.Name)
	}
	if !trace.TraceID.IsValid() {
		t.Error("expected valid trace ID")
	}

	// Check it's in the cache
	svc.mu.RLock()
	cached := svc.activeTraces[trace.TraceID]
	svc.mu.RUnlock()

	if cached != trace {
		t.Error("trace not cached correctly")
	}
}

func TestTraceService_StartTrace_NoRepo(t *testing.T) {
	logger := &mockTraceLogger{}
	svc := NewTraceService(nil, nil, logger)

	// Should work even without repos
	trace, err := svc.StartTrace(context.Background(), "test-service", "test-operation")
	if err != nil {
		t.Fatalf("StartTrace failed: %v", err)
	}
	if trace == nil {
		t.Fatal("expected non-nil trace")
	}
}

func TestTraceService_StartSpan(t *testing.T) {
	logger := &mockTraceLogger{}
	traceRepo := newMockTraceRepository()
	spanRepo := newMockSpanRepository()

	svc := NewTraceService(traceRepo, spanRepo, logger)

	// First start a trace
	trace, _ := svc.StartTrace(context.Background(), "test-service", "test-operation")

	// Now start a span
	span, err := svc.StartSpan(context.Background(), trace.TraceID, "test-span", domain.SpanKindInternal, "test-service", nil)
	if err != nil {
		t.Fatalf("StartSpan failed: %v", err)
	}

	if span == nil {
		t.Fatal("expected non-nil span")
	}
	if span.Name != "test-span" {
		t.Errorf("expected span name 'test-span', got '%s'", span.Name)
	}
	if span.TraceID != trace.TraceID {
		t.Error("span trace ID doesn't match")
	}
	if span.Kind != domain.SpanKindInternal {
		t.Errorf("expected span kind Internal, got %v", span.Kind)
	}
}

func TestTraceService_EndSpan(t *testing.T) {
	logger := &mockTraceLogger{}
	spanRepo := newMockSpanRepository()

	svc := NewTraceService(nil, spanRepo, logger)

	// Start a trace and span
	trace, _ := svc.StartTrace(context.Background(), "test-service", "test-operation")
	span, _ := svc.StartSpan(context.Background(), trace.TraceID, "test-span", domain.SpanKindInternal, "test-service", nil)

	// End the span
	err := svc.EndSpan(context.Background(), span)
	if err != nil {
		t.Fatalf("EndSpan failed: %v", err)
	}
}

