// Package notifications provides notification channel implementations for alerts.
package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/forge-platform/forge/internal/core/domain"
)

// WebhookNotifier sends alerts via HTTP webhooks.
type WebhookNotifier struct {
	client *http.Client
}

// NewWebhookNotifier creates a new webhook notifier.
func NewWebhookNotifier() *WebhookNotifier {
	return &WebhookNotifier{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Type returns the notification channel type.
func (n *WebhookNotifier) Type() domain.NotificationChannelType {
	return domain.ChannelWebhook
}

// Send sends an alert notification via webhook.
func (n *WebhookNotifier) Send(ctx context.Context, alert *domain.Alert, channel *domain.NotificationChannel) error {
	url := channel.Config["url"]
	if url == "" {
		return fmt.Errorf("webhook URL not configured")
	}

	payload := map[string]interface{}{
		"id":         alert.ID.String(),
		"rule_id":    alert.RuleID.String(),
		"rule_name":  alert.RuleName,
		"state":      alert.State,
		"severity":   alert.Severity,
		"message":    alert.Message,
		"value":      alert.Value,
		"threshold":  alert.Threshold,
		"labels":     alert.Labels,
		"starts_at":  alert.StartsAt.Format(time.RFC3339),
		"fingerprint": alert.Fingerprint,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add custom headers if configured
	if headers := channel.Config["headers"]; headers != "" {
		for _, h := range strings.Split(headers, ",") {
			parts := strings.SplitN(h, ":", 2)
			if len(parts) == 2 {
				req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
			}
		}
	}

	// Add auth if configured
	if token := channel.Config["auth_token"]; token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook returned error: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

// SlackNotifier sends alerts to Slack.
type SlackNotifier struct {
	client *http.Client
}

// NewSlackNotifier creates a new Slack notifier.
func NewSlackNotifier() *SlackNotifier {
	return &SlackNotifier{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Type returns the notification channel type.
func (n *SlackNotifier) Type() domain.NotificationChannelType {
	return domain.ChannelSlack
}

// Send sends an alert notification to Slack.
func (n *SlackNotifier) Send(ctx context.Context, alert *domain.Alert, channel *domain.NotificationChannel) error {
	webhookURL := channel.Config["webhook_url"]
	if webhookURL == "" {
		return fmt.Errorf("Slack webhook URL not configured")
	}

	// Build Slack message
	color := n.getSeverityColor(alert.Severity)
	payload := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"color":  color,
				"title":  fmt.Sprintf("[%s] %s", strings.ToUpper(string(alert.Severity)), alert.RuleName),
				"text":   alert.Message,
				"fields": n.buildFields(alert),
				"ts":     alert.StartsAt.Unix(),
			},
		},
	}

	if alertChannel := channel.Config["channel"]; alertChannel != "" {
		payload["channel"] = alertChannel
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create Slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("Slack request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Slack returned error: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

func (n *SlackNotifier) getSeverityColor(severity domain.AlertSeverity) string {
	switch severity {
	case domain.AlertSeverityCritical:
		return "#dc3545" // red
	case domain.AlertSeverityWarning:
		return "#ffc107" // yellow
	default:
		return "#17a2b8" // blue
	}
}

func (n *SlackNotifier) buildFields(alert *domain.Alert) []map[string]interface{} {
	fields := []map[string]interface{}{
		{"title": "State", "value": string(alert.State), "short": true},
		{"title": "Severity", "value": string(alert.Severity), "short": true},
		{"title": "Value", "value": fmt.Sprintf("%.2f", alert.Value), "short": true},
		{"title": "Threshold", "value": fmt.Sprintf("%.2f", alert.Threshold), "short": true},
	}
	return fields
}

// EmailNotifier sends alerts via email.
type EmailNotifier struct{}

// NewEmailNotifier creates a new email notifier.
func NewEmailNotifier() *EmailNotifier {
	return &EmailNotifier{}
}

// Type returns the notification channel type.
func (n *EmailNotifier) Type() domain.NotificationChannelType {
	return domain.ChannelEmail
}

// Send sends an alert notification via email.
func (n *EmailNotifier) Send(ctx context.Context, alert *domain.Alert, channel *domain.NotificationChannel) error {
	smtpHost := channel.Config["smtp_host"]
	smtpPort := channel.Config["smtp_port"]
	from := channel.Config["from"]
	to := channel.Config["to"]
	username := channel.Config["username"]
	password := channel.Config["password"]

	if smtpHost == "" || from == "" || to == "" {
		return fmt.Errorf("email configuration incomplete: need smtp_host, from, to")
	}

	if smtpPort == "" {
		smtpPort = "587"
	}

	subject := fmt.Sprintf("[%s] Alert: %s", strings.ToUpper(string(alert.Severity)), alert.RuleName)
	body := fmt.Sprintf(`Alert Notification

Rule: %s
State: %s
Severity: %s

Message: %s

Value: %.2f
Threshold: %.2f

Started At: %s
Fingerprint: %s
`,
		alert.RuleName,
		alert.State,
		alert.Severity,
		alert.Message,
		alert.Value,
		alert.Threshold,
		alert.StartsAt.Format(time.RFC3339),
		alert.Fingerprint,
	)

	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", to, subject, body))

	addr := smtpHost + ":" + smtpPort

	var auth smtp.Auth
	if username != "" && password != "" {
		auth = smtp.PlainAuth("", username, password, smtpHost)
	}

	return smtp.SendMail(addr, auth, from, strings.Split(to, ","), msg)
}

// PagerDutyNotifier sends alerts to PagerDuty.
type PagerDutyNotifier struct {
	client *http.Client
}

// NewPagerDutyNotifier creates a new PagerDuty notifier.
func NewPagerDutyNotifier() *PagerDutyNotifier {
	return &PagerDutyNotifier{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Type returns the notification channel type.
func (n *PagerDutyNotifier) Type() domain.NotificationChannelType {
	return domain.ChannelPagerDuty
}

// Send sends an alert notification to PagerDuty.
func (n *PagerDutyNotifier) Send(ctx context.Context, alert *domain.Alert, channel *domain.NotificationChannel) error {
	routingKey := channel.Config["routing_key"]
	if routingKey == "" {
		return fmt.Errorf("PagerDuty routing key not configured")
	}

	eventAction := "trigger"
	if alert.State == domain.AlertStateResolved {
		eventAction = "resolve"
	}

	payload := map[string]interface{}{
		"routing_key":  routingKey,
		"event_action": eventAction,
		"dedup_key":    alert.Fingerprint,
		"payload": map[string]interface{}{
			"summary":   alert.Message,
			"source":    "forge-platform",
			"severity":  n.mapSeverity(alert.Severity),
			"timestamp": alert.StartsAt.Format(time.RFC3339),
			"custom_details": map[string]interface{}{
				"rule_name": alert.RuleName,
				"value":     alert.Value,
				"threshold": alert.Threshold,
				"labels":    alert.Labels,
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal PagerDuty payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://events.pagerduty.com/v2/enqueue", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create PagerDuty request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("PagerDuty request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("PagerDuty returned error: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

func (n *PagerDutyNotifier) mapSeverity(severity domain.AlertSeverity) string {
	switch severity {
	case domain.AlertSeverityCritical:
		return "critical"
	case domain.AlertSeverityWarning:
		return "warning"
	default:
		return "info"
	}
}

