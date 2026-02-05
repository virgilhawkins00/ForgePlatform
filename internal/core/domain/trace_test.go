package domain

import (
	"errors"
	"testing"
)

func TestTraceID_String(t *testing.T) {
	traceID := NewTraceID()

	str := traceID.String()
	if len(str) != 32 {
		t.Errorf("TraceID String length = %d, want 32", len(str))
	}
}

func TestParseTraceID(t *testing.T) {
	original := NewTraceID()
	str := original.String()

	parsed, err := ParseTraceID(str)
	if err != nil {
		t.Fatalf("ParseTraceID error: %v", err)
	}
	if parsed != original {
		t.Error("Parsed TraceID does not match original")
	}
}

func TestParseTraceID_Invalid(t *testing.T) {
	_, err := ParseTraceID("invalid-hex")
	if err == nil {
		t.Error("ParseTraceID should error for invalid hex")
	}

	_, err = ParseTraceID("1234") // Wrong length
	if err == nil {
		t.Error("ParseTraceID should error for wrong length")
	}
}

func TestSpanID_String(t *testing.T) {
	spanID := NewSpanID()

	str := spanID.String()
	if len(str) != 16 {
		t.Errorf("SpanID String length = %d, want 16", len(str))
	}
}

func TestParseSpanID(t *testing.T) {
	original := NewSpanID()
	str := original.String()

	parsed, err := ParseSpanID(str)
	if err != nil {
		t.Fatalf("ParseSpanID error: %v", err)
	}
	if parsed != original {
		t.Error("Parsed SpanID does not match original")
	}
}

func TestParseSpanID_Invalid(t *testing.T) {
	_, err := ParseSpanID("invalid")
	if err == nil {
		t.Error("ParseSpanID should error for invalid hex")
	}

	_, err = ParseSpanID("12345678") // Wrong length (8 chars = 4 bytes)
	if err == nil {
		t.Error("ParseSpanID should error for wrong length")
	}
}

func TestTraceID_IsValid(t *testing.T) {
	valid := NewTraceID()
	if !valid.IsValid() {
		t.Error("NewTraceID should be valid")
	}

	var zero TraceID
	if zero.IsValid() {
		t.Error("Zero TraceID should not be valid")
	}
}

func TestSpanID_IsValid(t *testing.T) {
	valid := NewSpanID()
	if !valid.IsValid() {
		t.Error("NewSpanID should be valid")
	}

	var zero SpanID
	if zero.IsValid() {
		t.Error("Zero SpanID should not be valid")
	}
}

func TestNewSpan(t *testing.T) {
	traceID := NewTraceID()
	span := NewSpan(traceID, "http.request", SpanKindServer, "api-gateway")

	if span.ID.String() == "" {
		t.Error("ID is empty")
	}
	if span.TraceID != traceID {
		t.Error("TraceID mismatch")
	}
	if !span.SpanID.IsValid() {
		t.Error("SpanID is not valid")
	}
	if span.Name != "http.request" {
		t.Errorf("Name = %v, want http.request", span.Name)
	}
	if span.Kind != SpanKindServer {
		t.Errorf("Kind = %v, want server", span.Kind)
	}
	if span.ServiceName != "api-gateway" {
		t.Errorf("ServiceName = %v, want api-gateway", span.ServiceName)
	}
	if span.Status != SpanStatusUnset {
		t.Errorf("Status = %v, want unset", span.Status)
	}
}

func TestSpan_End(t *testing.T) {
	traceID := NewTraceID()
	span := NewSpan(traceID, "test", SpanKindInternal, "svc")

	span.End()

	if span.EndTime.IsZero() {
		t.Error("EndTime is zero after End()")
	}
	if span.Duration == 0 {
		t.Error("Duration is zero after End()")
	}
}

func TestSpan_SetStatus(t *testing.T) {
	traceID := NewTraceID()
	span := NewSpan(traceID, "test", SpanKindInternal, "svc")

	span.SetStatus(SpanStatusOK, "success")

	if span.Status != SpanStatusOK {
		t.Errorf("Status = %v, want ok", span.Status)
	}
	if span.StatusMessage != "success" {
		t.Errorf("StatusMessage = %v, want success", span.StatusMessage)
	}
}

func TestSpan_SetError(t *testing.T) {
	traceID := NewTraceID()
	span := NewSpan(traceID, "test", SpanKindInternal, "svc")

	span.SetError(errors.New("connection refused"))

	if span.Status != SpanStatusError {
		t.Errorf("Status = %v, want error", span.Status)
	}
	if span.StatusMessage != "connection refused" {
		t.Errorf("StatusMessage = %v, want 'connection refused'", span.StatusMessage)
	}
}

func TestSpan_SetError_Nil(t *testing.T) {
	traceID := NewTraceID()
	span := NewSpan(traceID, "test", SpanKindInternal, "svc")

	span.SetError(nil)

	if span.Status != SpanStatusError {
		t.Errorf("Status = %v, want error", span.Status)
	}
	if span.StatusMessage != "" {
		t.Errorf("StatusMessage = %v, want empty", span.StatusMessage)
	}
}

func TestSpan_AddEvent(t *testing.T) {
	traceID := NewTraceID()
	span := NewSpan(traceID, "test", SpanKindInternal, "svc")

	attrs := map[string]string{"key": "value"}
	span.AddEvent("query.executed", attrs)

	if len(span.Events) != 1 {
		t.Errorf("Events count = %d, want 1", len(span.Events))
	}
	if span.Events[0].Name != "query.executed" {
		t.Errorf("Event name = %v, want query.executed", span.Events[0].Name)
	}
	if span.Events[0].Attributes["key"] != "value" {
		t.Error("Event attributes not set correctly")
	}
}

func TestSpan_SetAttribute(t *testing.T) {
	traceID := NewTraceID()
	span := NewSpan(traceID, "test", SpanKindInternal, "svc")

	span.SetAttribute("http.method", "GET")
	span.SetAttribute("http.url", "/api/users")

	if span.Attributes["http.method"] != "GET" {
		t.Error("Attribute http.method not set")
	}
	if span.Attributes["http.url"] != "/api/users" {
		t.Error("Attribute http.url not set")
	}
}

func TestSpan_SetParent(t *testing.T) {
	traceID := NewTraceID()
	parentSpan := NewSpan(traceID, "parent", SpanKindServer, "svc")
	childSpan := NewSpan(traceID, "child", SpanKindInternal, "svc")

	childSpan.SetParent(parentSpan.SpanID)

	if childSpan.ParentSpanID == nil {
		t.Error("ParentSpanID is nil after SetParent()")
	}
	if *childSpan.ParentSpanID != parentSpan.SpanID {
		t.Error("ParentSpanID does not match")
	}
}

func TestNewTrace(t *testing.T) {
	trace := NewTrace("user-service", "GetUser")

	if trace.ID.String() == "" {
		t.Error("ID is empty")
	}
	if !trace.TraceID.IsValid() {
		t.Error("TraceID is not valid")
	}
	if trace.ServiceName != "user-service" {
		t.Errorf("ServiceName = %v, want user-service", trace.ServiceName)
	}
	if trace.Name != "GetUser" {
		t.Errorf("Name = %v, want GetUser", trace.Name)
	}
	if trace.Status != SpanStatusUnset {
		t.Errorf("Status = %v, want unset", trace.Status)
	}
	if len(trace.Spans) != 0 {
		t.Errorf("Spans should be empty, got %d", len(trace.Spans))
	}
}

func TestTrace_AddSpan(t *testing.T) {
	trace := NewTrace("svc", "op")
	span := NewSpan(trace.TraceID, "span1", SpanKindInternal, "svc")
	span.End()

	trace.AddSpan(span)

	if trace.SpanCount != 1 {
		t.Errorf("SpanCount = %d, want 1", trace.SpanCount)
	}
	if len(trace.Spans) != 1 {
		t.Errorf("Spans len = %d, want 1", len(trace.Spans))
	}
	if trace.RootSpan == nil {
		t.Error("RootSpan is nil after adding first span")
	}
}

func TestTrace_AddSpan_ErrorSpan(t *testing.T) {
	trace := NewTrace("svc", "op")
	span := NewSpan(trace.TraceID, "failing", SpanKindInternal, "svc")
	span.SetStatus(SpanStatusError, "failed")
	span.End()

	trace.AddSpan(span)

	if trace.ErrorCount != 1 {
		t.Errorf("ErrorCount = %d, want 1", trace.ErrorCount)
	}
}

func TestTrace_Complete(t *testing.T) {
	trace := NewTrace("svc", "op")
	span := NewSpan(trace.TraceID, "span1", SpanKindInternal, "svc")
	span.End()
	trace.AddSpan(span)

	trace.Complete()

	if trace.EndTime.IsZero() {
		t.Error("EndTime is zero after Complete()")
	}
	if trace.Duration == 0 {
		t.Error("Duration is zero after Complete()")
	}
	if trace.Status != SpanStatusOK {
		t.Errorf("Status = %v, want ok", trace.Status)
	}
}

func TestTrace_Complete_WithErrors(t *testing.T) {
	trace := NewTrace("svc", "op")
	span := NewSpan(trace.TraceID, "failing", SpanKindInternal, "svc")
	span.SetStatus(SpanStatusError, "failed")
	span.End()
	trace.AddSpan(span)

	trace.Complete()

	if trace.Status != SpanStatusError {
		t.Errorf("Status = %v, want error", trace.Status)
	}
}

func TestSpanKindConstants(t *testing.T) {
	kinds := []SpanKind{
		SpanKindUnspecified,
		SpanKindInternal,
		SpanKindServer,
		SpanKindClient,
		SpanKindProducer,
		SpanKindConsumer,
	}
	expected := []string{"unspecified", "internal", "server", "client", "producer", "consumer"}

	for i, kind := range kinds {
		if string(kind) != expected[i] {
			t.Errorf("SpanKind[%d] = %v, want %v", i, kind, expected[i])
		}
	}
}

func TestSpanStatusConstants(t *testing.T) {
	if SpanStatusUnset != "unset" {
		t.Errorf("SpanStatusUnset = %v, want unset", SpanStatusUnset)
	}
	if SpanStatusOK != "ok" {
		t.Errorf("SpanStatusOK = %v, want ok", SpanStatusOK)
	}
	if SpanStatusError != "error" {
		t.Errorf("SpanStatusError = %v, want error", SpanStatusError)
	}
}

