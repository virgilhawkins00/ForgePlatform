// Package services contains the application services implementing business logic.
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

// TraceService provides distributed tracing capabilities.
type TraceService struct {
	traceRepo ports.TraceRepository
	spanRepo  ports.SpanRepository
	logger    ports.Logger

	// Active traces cache
	mu           sync.RWMutex
	activeTraces map[domain.TraceID]*domain.Trace
}

// NewTraceService creates a new trace service.
func NewTraceService(traceRepo ports.TraceRepository, spanRepo ports.SpanRepository, logger ports.Logger) *TraceService {
	return &TraceService{
		traceRepo:    traceRepo,
		spanRepo:     spanRepo,
		logger:       logger,
		activeTraces: make(map[domain.TraceID]*domain.Trace),
	}
}

// StartTrace creates a new trace.
func (s *TraceService) StartTrace(ctx context.Context, serviceName, operationName string) (*domain.Trace, error) {
	trace := domain.NewTrace(serviceName, operationName)

	s.mu.Lock()
	s.activeTraces[trace.TraceID] = trace
	s.mu.Unlock()

	if s.traceRepo != nil {
		if err := s.traceRepo.Create(ctx, trace); err != nil {
			s.logger.Error("failed to persist trace", "trace_id", trace.TraceID.String(), "error", err)
		}
	}

	s.logger.Debug("started trace", "trace_id", trace.TraceID.String(), "service", serviceName, "operation", operationName)
	return trace, nil
}

// StartSpan creates a new span within a trace.
func (s *TraceService) StartSpan(ctx context.Context, traceID domain.TraceID, name string, kind domain.SpanKind, serviceName string, parentSpanID *domain.SpanID) (*domain.Span, error) {
	span := domain.NewSpan(traceID, name, kind, serviceName)
	if parentSpanID != nil {
		span.SetParent(*parentSpanID)
	}

	// Get active trace
	s.mu.RLock()
	trace := s.activeTraces[traceID]
	s.mu.RUnlock()

	if trace != nil {
		s.mu.Lock()
		trace.AddSpan(span)
		s.mu.Unlock()
	}

	s.logger.Debug("started span", "trace_id", traceID.String(), "span_id", span.SpanID.String(), "name", name)
	return span, nil
}

// EndSpan marks a span as completed.
func (s *TraceService) EndSpan(ctx context.Context, span *domain.Span) error {
	span.End()

	if s.spanRepo != nil {
		if err := s.spanRepo.Create(ctx, span); err != nil {
			s.logger.Error("failed to persist span", "span_id", span.SpanID.String(), "error", err)
			return err
		}
	}

	// Update trace
	s.mu.RLock()
	trace := s.activeTraces[span.TraceID]
	s.mu.RUnlock()

	if trace != nil && s.traceRepo != nil {
		if err := s.traceRepo.Update(ctx, trace); err != nil {
			s.logger.Error("failed to update trace", "trace_id", span.TraceID.String(), "error", err)
		}
	}

	s.logger.Debug("ended span", "span_id", span.SpanID.String(), "duration", span.Duration)
	return nil
}

// EndTrace marks a trace as completed.
func (s *TraceService) EndTrace(ctx context.Context, traceID domain.TraceID) error {
	s.mu.Lock()
	trace, exists := s.activeTraces[traceID]
	if exists {
		delete(s.activeTraces, traceID)
	}
	s.mu.Unlock()

	if !exists {
		return fmt.Errorf("trace not found: %s", traceID.String())
	}

	trace.Complete()

	if s.traceRepo != nil {
		if err := s.traceRepo.Update(ctx, trace); err != nil {
			s.logger.Error("failed to persist trace", "trace_id", traceID.String(), "error", err)
			return err
		}
	}

	s.logger.Debug("ended trace", "trace_id", traceID.String(), "duration", trace.Duration, "spans", trace.SpanCount)
	return nil
}

// GetTrace retrieves a trace by ID.
func (s *TraceService) GetTrace(ctx context.Context, id uuid.UUID) (*domain.Trace, error) {
	if s.traceRepo == nil {
		return nil, fmt.Errorf("trace repository not configured")
	}
	return s.traceRepo.GetByID(ctx, id)
}

// GetTraceByTraceID retrieves a trace by TraceID.
func (s *TraceService) GetTraceByTraceID(ctx context.Context, traceID domain.TraceID) (*domain.Trace, error) {
	// Check active traces first
	s.mu.RLock()
	if trace, ok := s.activeTraces[traceID]; ok {
		s.mu.RUnlock()
		return trace, nil
	}
	s.mu.RUnlock()

	if s.traceRepo == nil {
		return nil, fmt.Errorf("trace repository not configured")
	}
	return s.traceRepo.GetByTraceID(ctx, traceID)
}

// ListTraces retrieves traces with optional filtering.
func (s *TraceService) ListTraces(ctx context.Context, filter ports.TraceFilter) ([]*domain.Trace, error) {
	if s.traceRepo == nil {
		return []*domain.Trace{}, nil
	}
	return s.traceRepo.List(ctx, filter)
}

// GetSpansByTraceID retrieves all spans for a trace.
func (s *TraceService) GetSpansByTraceID(ctx context.Context, traceID domain.TraceID) ([]*domain.Span, error) {
	if s.spanRepo == nil {
		return []*domain.Span{}, nil
	}
	return s.spanRepo.ListByTraceID(ctx, traceID)
}

// GetServiceMap retrieves the service dependency map.
func (s *TraceService) GetServiceMap(ctx context.Context, startTime, endTime time.Time) (*domain.ServiceMap, error) {
	if s.traceRepo == nil {
		return &domain.ServiceMap{
			Nodes:     []domain.ServiceMapNode{},
			UpdatedAt: time.Now(),
		}, nil
	}
	return s.traceRepo.GetServiceMap(ctx, startTime, endTime)
}

// IngestSpan ingests a span from external source.
func (s *TraceService) IngestSpan(ctx context.Context, span *domain.Span) error {
	// Ensure we have a trace
	s.mu.Lock()
	trace, exists := s.activeTraces[span.TraceID]
	if !exists {
		// Create trace for this span
		trace = &domain.Trace{
			ID:          uuid.Must(uuid.NewV7()),
			TraceID:     span.TraceID,
			Spans:       []*domain.Span{},
			ServiceName: span.ServiceName,
			Name:        span.Name,
			StartTime:   span.StartTime,
			Status:      domain.SpanStatusUnset,
			Attributes:  make(map[string]string),
			CreatedAt:   time.Now(),
		}
		s.activeTraces[span.TraceID] = trace
	}
	trace.AddSpan(span)
	s.mu.Unlock()

	// Persist span
	if s.spanRepo != nil {
		if err := s.spanRepo.Create(ctx, span); err != nil {
			return fmt.Errorf("failed to persist span: %w", err)
		}
	}

	return nil
}

// IngestSpanBatch ingests multiple spans.
func (s *TraceService) IngestSpanBatch(ctx context.Context, spans []*domain.Span) error {
	for _, span := range spans {
		s.mu.Lock()
		trace, exists := s.activeTraces[span.TraceID]
		if !exists {
			trace = &domain.Trace{
				ID:          uuid.Must(uuid.NewV7()),
				TraceID:     span.TraceID,
				Spans:       []*domain.Span{},
				ServiceName: span.ServiceName,
				Name:        span.Name,
				StartTime:   span.StartTime,
				Status:      domain.SpanStatusUnset,
				Attributes:  make(map[string]string),
				CreatedAt:   time.Now(),
			}
			s.activeTraces[span.TraceID] = trace
		}
		trace.AddSpan(span)
		s.mu.Unlock()
	}

	if s.spanRepo != nil {
		if err := s.spanRepo.CreateBatch(ctx, spans); err != nil {
			return fmt.Errorf("failed to persist spans: %w", err)
		}
	}

	return nil
}

// GetTraceStats returns tracing statistics.
func (s *TraceService) GetTraceStats(ctx context.Context) (map[string]interface{}, error) {
	s.mu.RLock()
	activeCount := len(s.activeTraces)
	s.mu.RUnlock()

	stats := map[string]interface{}{
		"active_traces": activeCount,
	}

	return stats, nil
}

// CleanupInactiveTraces removes traces that haven't been updated recently.
func (s *TraceService) CleanupInactiveTraces(ctx context.Context, inactiveThreshold time.Duration) int {
	now := time.Now()
	var cleaned int

	s.mu.Lock()
	for traceID, trace := range s.activeTraces {
		if now.Sub(trace.EndTime) > inactiveThreshold || (trace.EndTime.IsZero() && now.Sub(trace.StartTime) > inactiveThreshold) {
			// Finalize and persist
			trace.Complete()
			if s.traceRepo != nil {
				if err := s.traceRepo.Update(ctx, trace); err != nil {
					s.logger.Error("failed to persist inactive trace", "trace_id", traceID.String(), "error", err)
				}
			}
			delete(s.activeTraces, traceID)
			cleaned++
		}
	}
	s.mu.Unlock()

	if cleaned > 0 {
		s.logger.Info("cleaned up inactive traces", "count", cleaned)
	}

	return cleaned
}

