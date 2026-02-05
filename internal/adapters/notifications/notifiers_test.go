// Package notifications provides notification channel implementations for alerts.
package notifications

import (
	"testing"

	"github.com/forge-platform/forge/internal/core/domain"
)

func TestNewWebhookNotifier(t *testing.T) {
	notifier := NewWebhookNotifier()
	if notifier == nil {
		t.Fatal("expected non-nil notifier")
	}
	if notifier.client == nil {
		t.Error("http client not initialized")
	}
}

func TestWebhookNotifier_Type(t *testing.T) {
	notifier := NewWebhookNotifier()
	if notifier.Type() != domain.ChannelWebhook {
		t.Errorf("expected type %v, got %v", domain.ChannelWebhook, notifier.Type())
	}
}

func TestNewSlackNotifier(t *testing.T) {
	notifier := NewSlackNotifier()
	if notifier == nil {
		t.Fatal("expected non-nil notifier")
	}
	if notifier.client == nil {
		t.Error("http client not initialized")
	}
}

func TestSlackNotifier_Type(t *testing.T) {
	notifier := NewSlackNotifier()
	if notifier.Type() != domain.ChannelSlack {
		t.Errorf("expected type %v, got %v", domain.ChannelSlack, notifier.Type())
	}
}

func TestSlackNotifier_getSeverityColor(t *testing.T) {
	notifier := NewSlackNotifier()

	tests := []struct {
		severity domain.AlertSeverity
		expected string
	}{
		{domain.AlertSeverityCritical, "#dc3545"},
		{domain.AlertSeverityWarning, "#ffc107"},
		{domain.AlertSeverityInfo, "#17a2b8"},
	}

	for _, tt := range tests {
		color := notifier.getSeverityColor(tt.severity)
		if color != tt.expected {
			t.Errorf("getSeverityColor(%v) = %s, expected %s", tt.severity, color, tt.expected)
		}
	}
}

func TestNewPagerDutyNotifier(t *testing.T) {
	notifier := NewPagerDutyNotifier()
	if notifier == nil {
		t.Fatal("expected non-nil notifier")
	}
	if notifier.client == nil {
		t.Error("http client not initialized")
	}
}

func TestPagerDutyNotifier_Type(t *testing.T) {
	notifier := NewPagerDutyNotifier()
	if notifier.Type() != domain.ChannelPagerDuty {
		t.Errorf("expected type %v, got %v", domain.ChannelPagerDuty, notifier.Type())
	}
}

func TestNewEmailNotifier(t *testing.T) {
	notifier := NewEmailNotifier()
	if notifier == nil {
		t.Fatal("expected non-nil notifier")
	}
}

func TestEmailNotifier_Type(t *testing.T) {
	notifier := NewEmailNotifier()
	if notifier.Type() != domain.ChannelEmail {
		t.Errorf("expected type %v, got %v", domain.ChannelEmail, notifier.Type())
	}
}

func TestSlackNotifier_buildFields(t *testing.T) {
	notifier := NewSlackNotifier()

	rule := domain.NewAlertRule("test-rule", "cpu_usage", domain.ConditionThresholdAbove, 90.0, domain.AlertSeverityWarning)
	alert := domain.NewAlert(rule, 95.5, "CPU usage exceeded threshold")

	fields := notifier.buildFields(alert)
	if len(fields) != 4 {
		t.Errorf("expected 4 fields, got %d", len(fields))
	}
}

