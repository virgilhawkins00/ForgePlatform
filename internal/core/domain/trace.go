// Package domain contains the core business entities for the Forge platform.
package domain

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TraceID represents a 128-bit trace identifier (OpenTelemetry compatible).
type TraceID [16]byte

// SpanID represents a 64-bit span identifier (OpenTelemetry compatible).
type SpanID [8]byte

// String returns the hex string representation of the TraceID.
func (t TraceID) String() string {
	return hex.EncodeToString(t[:])
}

// ParseTraceID parses a hex string into a TraceID.
func ParseTraceID(s string) (TraceID, error) {
	var t TraceID
	b, err := hex.DecodeString(s)
	if err != nil {
		return t, err
	}
	if len(b) != 16 {
		return t, fmt.Errorf("invalid trace ID length: expected 16 bytes, got %d", len(b))
	}
	copy(t[:], b)
	return t, nil
}

// String returns the hex string representation of the SpanID.
func (s SpanID) String() string {
	return hex.EncodeToString(s[:])
}

// ParseSpanID parses a hex string into a SpanID.
func ParseSpanID(str string) (SpanID, error) {
	var s SpanID
	b, err := hex.DecodeString(str)
	if err != nil {
		return s, err
	}
	if len(b) != 8 {
		return s, fmt.Errorf("invalid span ID length: expected 8 bytes, got %d", len(b))
	}
	copy(s[:], b)
	return s, nil
}

// IsValid returns true if the TraceID is not zero.
func (t TraceID) IsValid() bool {
	for _, b := range t {
		if b != 0 {
			return true
		}
	}
	return false
}

// IsValid returns true if the SpanID is not zero.
func (s SpanID) IsValid() bool {
	for _, b := range s {
		if b != 0 {
			return true
		}
	}
	return false
}

// SpanKind represents the role of the span in the trace.
type SpanKind string

const (
	SpanKindUnspecified SpanKind = "unspecified"
	SpanKindInternal    SpanKind = "internal"
	SpanKindServer      SpanKind = "server"
	SpanKindClient      SpanKind = "client"
	SpanKindProducer    SpanKind = "producer"
	SpanKindConsumer    SpanKind = "consumer"
)

// SpanStatus represents the status of a span.
type SpanStatus string

const (
	SpanStatusUnset SpanStatus = "unset"
	SpanStatusOK    SpanStatus = "ok"
	SpanStatusError SpanStatus = "error"
)

// SpanContext carries trace context across process boundaries.
type SpanContext struct {
	TraceID    TraceID `json:"trace_id"`
	SpanID     SpanID  `json:"span_id"`
	TraceFlags byte    `json:"trace_flags"`
	TraceState string  `json:"trace_state,omitempty"`
	Remote     bool    `json:"remote"`
}

// SpanEvent represents a time-stamped event within a span.
type SpanEvent struct {
	Name       string            `json:"name"`
	Timestamp  time.Time         `json:"timestamp"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// SpanLink represents a link to another span.
type SpanLink struct {
	TraceID    TraceID           `json:"trace_id"`
	SpanID     SpanID            `json:"span_id"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// Span represents a single unit of work within a trace.
type Span struct {
	ID           uuid.UUID         `json:"id"`
	TraceID      TraceID           `json:"trace_id"`
	SpanID       SpanID            `json:"span_id"`
	ParentSpanID *SpanID           `json:"parent_span_id,omitempty"`
	Name         string            `json:"name"`
	Kind         SpanKind          `json:"kind"`
	StartTime    time.Time         `json:"start_time"`
	EndTime      time.Time         `json:"end_time"`
	Duration     time.Duration     `json:"duration"`
	Status       SpanStatus        `json:"status"`
	StatusMessage string           `json:"status_message,omitempty"`
	Attributes   map[string]string `json:"attributes,omitempty"`
	Events       []SpanEvent       `json:"events,omitempty"`
	Links        []SpanLink        `json:"links,omitempty"`

	// Resource attributes (service info)
	ServiceName    string `json:"service_name"`
	ServiceVersion string `json:"service_version,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
}

// Trace represents a distributed trace containing multiple spans.
type Trace struct {
	ID          uuid.UUID         `json:"id"`
	TraceID     TraceID           `json:"trace_id"`
	RootSpan    *Span             `json:"root_span,omitempty"`
	Spans       []*Span           `json:"spans"`
	ServiceName string            `json:"service_name"`
	Name        string            `json:"name"`
	StartTime   time.Time         `json:"start_time"`
	EndTime     time.Time         `json:"end_time"`
	Duration    time.Duration     `json:"duration"`
	SpanCount   int               `json:"span_count"`
	ErrorCount  int               `json:"error_count"`
	Status      SpanStatus        `json:"status"`
	Attributes  map[string]string `json:"attributes,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

// ServiceMapNode represents a node in the service dependency map.
type ServiceMapNode struct {
	ServiceName  string   `json:"service_name"`
	SpanCount    int64    `json:"span_count"`
	ErrorCount   int64    `json:"error_count"`
	AvgDuration  float64  `json:"avg_duration_ms"`
	Dependencies []string `json:"dependencies"`
}

// ServiceMap represents the service dependency graph.
type ServiceMap struct {
	Nodes     []ServiceMapNode `json:"nodes"`
	UpdatedAt time.Time        `json:"updated_at"`
}

// NewTraceID generates a new random TraceID.
func NewTraceID() TraceID {
	var t TraceID
	id := uuid.New()
	copy(t[:], id[:])
	return t
}

// NewSpanID generates a new random SpanID.
func NewSpanID() SpanID {
	var s SpanID
	id := uuid.New()
	copy(s[:], id[:8])
	return s
}

// NewSpan creates a new span with the given parameters.
func NewSpan(traceID TraceID, name string, kind SpanKind, serviceName string) *Span {
	now := time.Now()
	return &Span{
		ID:          uuid.Must(uuid.NewV7()),
		TraceID:     traceID,
		SpanID:      NewSpanID(),
		Name:        name,
		Kind:        kind,
		StartTime:   now,
		Status:      SpanStatusUnset,
		Attributes:  make(map[string]string),
		Events:      []SpanEvent{},
		Links:       []SpanLink{},
		ServiceName: serviceName,
		CreatedAt:   now,
	}
}

// End marks the span as completed.
func (s *Span) End() {
	s.EndTime = time.Now()
	s.Duration = s.EndTime.Sub(s.StartTime)
}

// SetStatus sets the span status.
func (s *Span) SetStatus(status SpanStatus, message string) {
	s.Status = status
	s.StatusMessage = message
}

// SetError marks the span as error.
func (s *Span) SetError(err error) {
	s.Status = SpanStatusError
	if err != nil {
		s.StatusMessage = err.Error()
	}
}

// AddEvent adds a timestamped event to the span.
func (s *Span) AddEvent(name string, attributes map[string]string) {
	s.Events = append(s.Events, SpanEvent{
		Name:       name,
		Timestamp:  time.Now(),
		Attributes: attributes,
	})
}

// SetAttribute sets an attribute on the span.
func (s *Span) SetAttribute(key, value string) {
	if s.Attributes == nil {
		s.Attributes = make(map[string]string)
	}
	s.Attributes[key] = value
}

// SetParent sets the parent span ID.
func (s *Span) SetParent(parentID SpanID) {
	s.ParentSpanID = &parentID
}

// NewTrace creates a new trace.
func NewTrace(serviceName, name string) *Trace {
	now := time.Now()
	traceID := NewTraceID()
	return &Trace{
		ID:          uuid.Must(uuid.NewV7()),
		TraceID:     traceID,
		Spans:       []*Span{},
		ServiceName: serviceName,
		Name:        name,
		StartTime:   now,
		Status:      SpanStatusUnset,
		Attributes:  make(map[string]string),
		CreatedAt:   now,
	}
}

// AddSpan adds a span to the trace.
func (t *Trace) AddSpan(span *Span) {
	t.Spans = append(t.Spans, span)
	t.SpanCount = len(t.Spans)
	if span.Status == SpanStatusError {
		t.ErrorCount++
	}
	// Update root span if this is the first or has no parent
	if t.RootSpan == nil && span.ParentSpanID == nil {
		t.RootSpan = span
		t.Name = span.Name
	}
	// Update trace end time
	if span.EndTime.After(t.EndTime) {
		t.EndTime = span.EndTime
		t.Duration = t.EndTime.Sub(t.StartTime)
	}
}

// Complete finalizes the trace.
func (t *Trace) Complete() {
	t.EndTime = time.Now()
	t.Duration = t.EndTime.Sub(t.StartTime)
	// Determine overall status
	if t.ErrorCount > 0 {
		t.Status = SpanStatusError
	} else {
		t.Status = SpanStatusOK
	}
}

