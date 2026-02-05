// Package services implements core business logic services.
package services

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/forge-platform/forge/internal/core/ports"
)

// mockEventLogger for testing
type mockEventLogger struct{}

func (m *mockEventLogger) Debug(msg string, args ...interface{}) {}
func (m *mockEventLogger) Info(msg string, args ...interface{})  {}
func (m *mockEventLogger) Warn(msg string, args ...interface{})  {}
func (m *mockEventLogger) Error(msg string, args ...interface{}) {}
func (m *mockEventLogger) With(args ...interface{}) ports.Logger { return m }

func TestNewEventService(t *testing.T) {
	logger := &mockEventLogger{}

	svc := NewEventService(logger)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.subscribers == nil {
		t.Error("subscribers map not initialized")
	}
	if svc.allHandlers == nil {
		t.Error("allHandlers slice not initialized")
	}
	if svc.eventHistory == nil {
		t.Error("eventHistory slice not initialized")
	}
	if svc.maxHistory != 1000 {
		t.Errorf("expected maxHistory 1000, got %d", svc.maxHistory)
	}
	if svc.logger == nil {
		t.Error("logger not set correctly")
	}
}

func TestEventService_Subscribe(t *testing.T) {
	logger := &mockEventLogger{}
	svc := NewEventService(logger)

	handlerCalled := false
	handler := func(e Event) error {
		handlerCalled = true
		return nil
	}

	svc.Subscribe("test.event", handler)

	if len(svc.subscribers["test.event"]) != 1 {
		t.Errorf("expected 1 subscriber, got %d", len(svc.subscribers["test.event"]))
	}

	// Trigger event
	err := svc.Publish(context.Background(), Event{
		Type:   "test.event",
		Source: "test",
	})
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	if !handlerCalled {
		t.Error("handler was not called")
	}
}

func TestEventService_SubscribeAll(t *testing.T) {
	logger := &mockEventLogger{}
	svc := NewEventService(logger)

	var callCount int32
	handler := func(e Event) error {
		atomic.AddInt32(&callCount, 1)
		return nil
	}

	svc.SubscribeAll(handler)

	// Publish multiple events of different types
	_ = svc.Publish(context.Background(), Event{Type: "type1", Source: "test"})
	_ = svc.Publish(context.Background(), Event{Type: "type2", Source: "test"})

	if atomic.LoadInt32(&callCount) != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestEventService_Publish_SetsTimestamp(t *testing.T) {
	logger := &mockEventLogger{}
	svc := NewEventService(logger)

	var receivedEvent Event
	svc.SubscribeAll(func(e Event) error {
		receivedEvent = e
		return nil
	})

	err := svc.Publish(context.Background(), Event{
		Type:   "test",
		Source: "test",
	})
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	if receivedEvent.Timestamp.IsZero() {
		t.Error("expected timestamp to be set")
	}
}

func TestEventService_Publish_AddsToHistory(t *testing.T) {
	logger := &mockEventLogger{}
	svc := NewEventService(logger)

	event := Event{
		ID:     "test-id",
		Type:   "test.event",
		Source: "test",
	}

	_ = svc.Publish(context.Background(), event)

	history := svc.GetHistory("test.event", 10)
	if len(history) != 1 {
		t.Errorf("expected 1 event in history, got %d", len(history))
	}
	if history[0].ID != "test-id" {
		t.Errorf("expected event ID 'test-id', got '%s'", history[0].ID)
	}
}

func TestEventService_GetHistory_Limit(t *testing.T) {
	logger := &mockEventLogger{}
	svc := NewEventService(logger)

	// Publish multiple events
	for i := 0; i < 10; i++ {
		_ = svc.Publish(context.Background(), Event{
			Type:      "test.event",
			Source:    "test",
			Timestamp: time.Now(),
		})
	}

	// Get limited history
	history := svc.GetHistory("test.event", 5)
	if len(history) != 5 {
		t.Errorf("expected 5 events in history, got %d", len(history))
	}
}

