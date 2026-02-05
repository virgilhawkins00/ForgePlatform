package domain

import (
	"testing"
	"time"
)

func TestNewAlertRule(t *testing.T) {
	rule := NewAlertRule("high-cpu", "cpu.usage", ConditionThresholdAbove, 90.0, AlertSeverityCritical)

	if rule.ID.String() == "" {
		t.Error("ID is empty")
	}
	if rule.Name != "high-cpu" {
		t.Errorf("Name = %v, want high-cpu", rule.Name)
	}
	if rule.MetricName != "cpu.usage" {
		t.Errorf("MetricName = %v, want cpu.usage", rule.MetricName)
	}
	if rule.Condition != ConditionThresholdAbove {
		t.Errorf("Condition = %v, want threshold_above", rule.Condition)
	}
	if rule.Threshold != 90.0 {
		t.Errorf("Threshold = %v, want 90.0", rule.Threshold)
	}
	if rule.Severity != AlertSeverityCritical {
		t.Errorf("Severity = %v, want critical", rule.Severity)
	}
	if !rule.Enabled {
		t.Error("Enabled = false, want true")
	}
	if rule.Duration != time.Minute {
		t.Errorf("Duration = %v, want 1m", rule.Duration)
	}
}

func TestNewAlert(t *testing.T) {
	rule := NewAlertRule("test-rule", "test.metric", ConditionThresholdAbove, 50.0, AlertSeverityWarning)

	alert := NewAlert(rule, 75.0, "Value exceeded threshold")

	if alert.ID.String() == "" {
		t.Error("ID is empty")
	}
	if alert.RuleID != rule.ID {
		t.Errorf("RuleID = %v, want %v", alert.RuleID, rule.ID)
	}
	if alert.RuleName != "test-rule" {
		t.Errorf("RuleName = %v, want test-rule", alert.RuleName)
	}
	if alert.Severity != AlertSeverityWarning {
		t.Errorf("Severity = %v, want warning", alert.Severity)
	}
	if alert.State != AlertStatePending {
		t.Errorf("State = %v, want pending", alert.State)
	}
	if alert.Value != 75.0 {
		t.Errorf("Value = %v, want 75.0", alert.Value)
	}
	if alert.Message != "Value exceeded threshold" {
		t.Errorf("Message = %v, want 'Value exceeded threshold'", alert.Message)
	}
}

func TestAlert_Fire(t *testing.T) {
	rule := NewAlertRule("test", "test.metric", ConditionThresholdAbove, 50.0, AlertSeverityInfo)
	alert := NewAlert(rule, 60.0, "test alert")

	alert.Fire()

	if alert.State != AlertStateFiring {
		t.Errorf("State = %v, want firing", alert.State)
	}
}

func TestAlert_Resolve(t *testing.T) {
	rule := NewAlertRule("test", "test.metric", ConditionThresholdAbove, 50.0, AlertSeverityInfo)
	alert := NewAlert(rule, 60.0, "test alert")
	alert.Fire()

	alert.Resolve()

	if alert.State != AlertStateResolved {
		t.Errorf("State = %v, want resolved", alert.State)
	}
	if alert.EndsAt == nil {
		t.Error("EndsAt is nil after Resolve()")
	}
}

func TestAlert_Acknowledge(t *testing.T) {
	rule := NewAlertRule("test", "test.metric", ConditionThresholdAbove, 50.0, AlertSeverityInfo)
	alert := NewAlert(rule, 60.0, "test alert")

	alert.Acknowledge("user123", "investigating")

	if alert.State != AlertStateAcknowledged {
		t.Errorf("State = %v, want acknowledged", alert.State)
	}
	if alert.AcknowledgedBy != "user123" {
		t.Errorf("AcknowledgedBy = %v, want user123", alert.AcknowledgedBy)
	}
	if alert.AckComment != "investigating" {
		t.Errorf("AckComment = %v, want investigating", alert.AckComment)
	}
	if alert.AcknowledgedAt == nil {
		t.Error("AcknowledgedAt is nil after Acknowledge()")
	}
}

func TestAlert_Silence(t *testing.T) {
	rule := NewAlertRule("test", "test.metric", ConditionThresholdAbove, 50.0, AlertSeverityInfo)
	alert := NewAlert(rule, 60.0, "test alert")

	alert.Silence()

	if alert.State != AlertStateSilenced {
		t.Errorf("State = %v, want silenced", alert.State)
	}
}

func TestNewNotificationChannel(t *testing.T) {
	config := map[string]string{"webhook_url": "https://hooks.slack.com/xxx"}
	channel := NewNotificationChannel("slack-alerts", ChannelSlack, config)

	if channel.ID.String() == "" {
		t.Error("ID is empty")
	}
	if channel.Name != "slack-alerts" {
		t.Errorf("Name = %v, want slack-alerts", channel.Name)
	}
	if channel.Type != ChannelSlack {
		t.Errorf("Type = %v, want slack", channel.Type)
	}
	if !channel.Enabled {
		t.Error("Enabled = false, want true")
	}
	if channel.Config["webhook_url"] != "https://hooks.slack.com/xxx" {
		t.Error("Config not set correctly")
	}
}

func TestNewSilence(t *testing.T) {
	startsAt := time.Now()
	endsAt := startsAt.Add(2 * time.Hour)
	matchers := map[string]string{"severity": "warning"}

	silence := NewSilence(matchers, startsAt, endsAt, "operator", "planned maintenance")

	if silence.ID.String() == "" {
		t.Error("ID is empty")
	}
	if silence.CreatedBy != "operator" {
		t.Errorf("CreatedBy = %v, want operator", silence.CreatedBy)
	}
	if silence.Comment != "planned maintenance" {
		t.Errorf("Comment = %v, want 'planned maintenance'", silence.Comment)
	}
	if !silence.Active {
		t.Error("Active = false, want true")
	}
}

func TestSilence_IsActive(t *testing.T) {
	now := time.Now()
	matchers := map[string]string{"severity": "critical"}

	// Active silence (started in past, ends in future)
	silence := NewSilence(matchers, now.Add(-time.Hour), now.Add(time.Hour), "admin", "test")
	if !silence.IsActive() {
		t.Error("IsActive() = false for active silence")
	}

	// Inactive silence (not yet started)
	silence2 := NewSilence(matchers, now.Add(time.Hour), now.Add(2*time.Hour), "admin", "test")
	if silence2.IsActive() {
		t.Error("IsActive() = true for future silence")
	}

	// Expired silence
	silence3 := NewSilence(matchers, now.Add(-2*time.Hour), now.Add(-time.Hour), "admin", "test")
	if silence3.IsActive() {
		t.Error("IsActive() = true for expired silence")
	}

	// Deactivated silence
	silence4 := NewSilence(matchers, now.Add(-time.Hour), now.Add(time.Hour), "admin", "test")
	silence4.Active = false
	if silence4.IsActive() {
		t.Error("IsActive() = true for deactivated silence")
	}
}

func TestSilence_Matches(t *testing.T) {
	matchers := map[string]string{
		"severity": "critical",
		"team":     "platform",
	}
	silence := NewSilence(matchers, time.Now(), time.Now().Add(time.Hour), "admin", "test")

	// All matchers present and matching
	labels := map[string]string{"severity": "critical", "team": "platform", "host": "server1"}
	if !silence.Matches(labels) {
		t.Error("Matches() = false when all matchers match")
	}

	// Missing matcher
	labels2 := map[string]string{"severity": "critical"}
	if silence.Matches(labels2) {
		t.Error("Matches() = true when matcher is missing")
	}

	// Wrong value
	labels3 := map[string]string{"severity": "warning", "team": "platform"}
	if silence.Matches(labels3) {
		t.Error("Matches() = true when matcher value is wrong")
	}
}

func TestAlertSeverityConstants(t *testing.T) {
	if AlertSeverityInfo != "info" {
		t.Errorf("AlertSeverityInfo = %v, want info", AlertSeverityInfo)
	}
	if AlertSeverityWarning != "warning" {
		t.Errorf("AlertSeverityWarning = %v, want warning", AlertSeverityWarning)
	}
	if AlertSeverityCritical != "critical" {
		t.Errorf("AlertSeverityCritical = %v, want critical", AlertSeverityCritical)
	}
}

func TestAlertStateConstants(t *testing.T) {
	if AlertStatePending != "pending" {
		t.Errorf("AlertStatePending = %v, want pending", AlertStatePending)
	}
	if AlertStateFiring != "firing" {
		t.Errorf("AlertStateFiring = %v, want firing", AlertStateFiring)
	}
	if AlertStateResolved != "resolved" {
		t.Errorf("AlertStateResolved = %v, want resolved", AlertStateResolved)
	}
	if AlertStateSilenced != "silenced" {
		t.Errorf("AlertStateSilenced = %v, want silenced", AlertStateSilenced)
	}
	if AlertStateAcknowledged != "acknowledged" {
		t.Errorf("AlertStateAcknowledged = %v, want acknowledged", AlertStateAcknowledged)
	}
}

func TestRuleConditionTypeConstants(t *testing.T) {
	types := []RuleConditionType{
		ConditionThresholdAbove,
		ConditionThresholdBelow,
		ConditionThresholdEqual,
		ConditionRateOfChange,
		ConditionAnomalyDetection,
		ConditionAbsenceOfData,
		ConditionComposite,
	}
	expected := []string{
		"threshold_above",
		"threshold_below",
		"threshold_equal",
		"rate_of_change",
		"anomaly_detection",
		"absence_of_data",
		"composite",
	}

	for i, ct := range types {
		if string(ct) != expected[i] {
			t.Errorf("ConditionType[%d] = %v, want %v", i, ct, expected[i])
		}
	}
}

func TestNotificationChannelTypeConstants(t *testing.T) {
	if ChannelEmail != "email" {
		t.Errorf("ChannelEmail = %v, want email", ChannelEmail)
	}
	if ChannelSlack != "slack" {
		t.Errorf("ChannelSlack = %v, want slack", ChannelSlack)
	}
	if ChannelWebhook != "webhook" {
		t.Errorf("ChannelWebhook = %v, want webhook", ChannelWebhook)
	}
	if ChannelPagerDuty != "pagerduty" {
		t.Errorf("ChannelPagerDuty = %v, want pagerduty", ChannelPagerDuty)
	}
}

