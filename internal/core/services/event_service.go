// Package services provides the core business logic services.
package services

import (
	"context"
	"sync"
	"time"

	"github.com/forge-platform/forge/internal/core/ports"
)

// Event represents an event in the system.
type Event struct {
	ID        string
	Type      string
	Source    string // plugin ID or "system"
	Payload   []byte
	Timestamp time.Time
}

// EventHandler is a function that handles events.
type EventHandler func(event Event) error

// EventService manages event subscriptions and publishing.
type EventService struct {
	mu           sync.RWMutex
	subscribers  map[string][]EventHandler // eventType -> handlers
	allHandlers  []EventHandler            // handlers for all events
	eventHistory []Event                   // recent events for replay
	maxHistory   int
	logger       ports.Logger
}

// NewEventService creates a new event service.
func NewEventService(logger ports.Logger) *EventService {
	return &EventService{
		subscribers:  make(map[string][]EventHandler),
		allHandlers:  make([]EventHandler, 0),
		eventHistory: make([]Event, 0),
		maxHistory:   1000,
		logger:       logger,
	}
}

// Subscribe registers a handler for a specific event type.
func (s *EventService) Subscribe(eventType string, handler EventHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.subscribers[eventType] = append(s.subscribers[eventType], handler)
	s.logger.Debug("Event subscription added", "type", eventType)
}

// SubscribeAll registers a handler for all events.
func (s *EventService) SubscribeAll(handler EventHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.allHandlers = append(s.allHandlers, handler)
	s.logger.Debug("Global event subscription added")
}

// Publish publishes an event to all subscribers.
func (s *EventService) Publish(ctx context.Context, event Event) error {
	s.mu.RLock()
	handlers := make([]EventHandler, 0)

	// Get specific handlers
	if h, ok := s.subscribers[event.Type]; ok {
		handlers = append(handlers, h...)
	}

	// Get all-event handlers
	handlers = append(handlers, s.allHandlers...)
	s.mu.RUnlock()

	// Set timestamp if not set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Add to history
	s.addToHistory(event)

	// Dispatch to handlers
	var firstErr error
	for _, h := range handlers {
		if err := h(event); err != nil {
			s.logger.Error("Event handler error", "type", event.Type, "error", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	s.logger.Debug("Event published", "type", event.Type, "handlers", len(handlers))
	return firstErr
}

// addToHistory adds an event to the history buffer.
func (s *EventService) addToHistory(event Event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.eventHistory = append(s.eventHistory, event)

	// Trim if over max
	if len(s.eventHistory) > s.maxHistory {
		s.eventHistory = s.eventHistory[len(s.eventHistory)-s.maxHistory:]
	}
}

// GetHistory returns recent events, optionally filtered by type.
func (s *EventService) GetHistory(eventType string, limit int) []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}

	var result []Event
	for i := len(s.eventHistory) - 1; i >= 0 && len(result) < limit; i-- {
		e := s.eventHistory[i]
		if eventType == "" || e.Type == eventType {
			result = append(result, e)
		}
	}

	return result
}

// Unsubscribe removes all handlers for a specific event type.
func (s *EventService) Unsubscribe(eventType string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.subscribers, eventType)
	s.logger.Debug("Event subscriptions removed", "type", eventType)
}

// Clear removes all subscriptions and history.
func (s *EventService) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.subscribers = make(map[string][]EventHandler)
	s.allHandlers = make([]EventHandler, 0)
	s.eventHistory = make([]Event, 0)
	s.logger.Debug("Event service cleared")
}

