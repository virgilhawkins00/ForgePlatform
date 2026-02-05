// Package domain contains the core business entities for the Forge platform.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// AlertSeverity represents the severity level of an alert.
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityCritical AlertSeverity = "critical"
)

// AlertState represents the current state of an alert.
type AlertState string

const (
	AlertStatePending      AlertState = "pending"      // Alert condition detected, waiting for duration
	AlertStateFiring       AlertState = "firing"       // Alert is active
	AlertStateResolved     AlertState = "resolved"     // Alert condition no longer met
	AlertStateSilenced     AlertState = "silenced"     // Alert is silenced
	AlertStateAcknowledged AlertState = "acknowledged" // Alert has been acknowledged
)

// RuleConditionType represents the type of condition for alert rules.
type RuleConditionType string

const (
	ConditionThresholdAbove    RuleConditionType = "threshold_above"    // Value > threshold
	ConditionThresholdBelow    RuleConditionType = "threshold_below"    // Value < threshold
	ConditionThresholdEqual    RuleConditionType = "threshold_equal"    // Value == threshold
	ConditionRateOfChange      RuleConditionType = "rate_of_change"     // Rate of change exceeds threshold
	ConditionAnomalyDetection  RuleConditionType = "anomaly_detection"  // Statistical anomaly detected
	ConditionAbsenceOfData     RuleConditionType = "absence_of_data"    // No data received for duration
	ConditionComposite         RuleConditionType = "composite"          // Multiple conditions combined
)

// NotificationChannelType represents the type of notification channel.
type NotificationChannelType string

const (
	ChannelEmail     NotificationChannelType = "email"
	ChannelSlack     NotificationChannelType = "slack"
	ChannelWebhook   NotificationChannelType = "webhook"
	ChannelPagerDuty NotificationChannelType = "pagerduty"
)

// AlertRule defines the conditions under which an alert should fire.
type AlertRule struct {
	ID          uuid.UUID              `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Enabled     bool                   `json:"enabled"`

	// Metric targeting
	MetricName string            `json:"metric_name"`
	Tags       map[string]string `json:"tags,omitempty"`

	// Condition configuration
	Condition RuleConditionType `json:"condition"`
	Threshold float64           `json:"threshold"`

	// For rate of change: the time window to calculate rate
	RateWindow time.Duration `json:"rate_window,omitempty"`

	// For anomaly detection: number of standard deviations
	AnomalyStdDev float64 `json:"anomaly_std_dev,omitempty"`

	// For composite conditions: list of sub-rule IDs and operator (AND/OR)
	CompositeRules    []uuid.UUID `json:"composite_rules,omitempty"`
	CompositeOperator string      `json:"composite_operator,omitempty"` // "and" or "or"

	// Timing
	Duration   time.Duration `json:"duration"`    // How long condition must be true before firing
	Interval   time.Duration `json:"interval"`    // How often to evaluate the rule
	LastCheck  time.Time     `json:"last_check"`
	NextCheck  time.Time     `json:"next_check"`

	// Notification configuration
	Severity AlertSeverity `json:"severity"`
	Channels []string      `json:"channels"` // Channel IDs to notify

	// Labels for routing and grouping
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations for alert messages
	Annotations map[string]string `json:"annotations,omitempty"`

	// Metadata
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewAlertRule creates a new alert rule with defaults.
func NewAlertRule(name, metricName string, condition RuleConditionType, threshold float64, severity AlertSeverity) *AlertRule {
	now := time.Now()
	return &AlertRule{
		ID:          uuid.New(),
		Name:        name,
		MetricName:  metricName,
		Condition:   condition,
		Threshold:   threshold,
		Severity:    severity,
		Enabled:     true,
		Duration:    time.Minute,
		Interval:    time.Minute,
		Labels:      make(map[string]string),
		Annotations: make(map[string]string),
		Tags:        make(map[string]string),
		Channels:    []string{},
		CreatedAt:   now,
		UpdatedAt:   now,
		NextCheck:   now,
	}
}

// Alert represents an instance of a fired alert.
type Alert struct {
	ID        uuid.UUID     `json:"id"`
	RuleID    uuid.UUID     `json:"rule_id"`
	RuleName  string        `json:"rule_name"`
	State     AlertState    `json:"state"`
	Severity  AlertSeverity `json:"severity"`

	// Alert details
	Message     string            `json:"message"`
	Value       float64           `json:"value"`       // The value that triggered the alert
	Threshold   float64           `json:"threshold"`   // The threshold that was exceeded
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`

	// Timing
	StartsAt      time.Time  `json:"starts_at"`
	EndsAt        *time.Time `json:"ends_at,omitempty"`
	LastEvaluated time.Time  `json:"last_evaluated"`

	// Acknowledgement
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
	AcknowledgedBy string     `json:"acknowledged_by,omitempty"`
	AckComment     string     `json:"ack_comment,omitempty"`

	// Fingerprint for deduplication
	Fingerprint string `json:"fingerprint"`
}

// NewAlert creates a new alert instance.
func NewAlert(rule *AlertRule, value float64, message string) *Alert {
	now := time.Now()
	return &Alert{
		ID:            uuid.New(),
		RuleID:        rule.ID,
		RuleName:      rule.Name,
		State:         AlertStatePending,
		Severity:      rule.Severity,
		Message:       message,
		Value:         value,
		Threshold:     rule.Threshold,
		Labels:        copyMap(rule.Labels),
		Annotations:   copyMap(rule.Annotations),
		StartsAt:      now,
		LastEvaluated: now,
		Fingerprint:   generateFingerprint(rule),
	}
}

// Fire transitions the alert to firing state.
func (a *Alert) Fire() {
	a.State = AlertStateFiring
	a.LastEvaluated = time.Now()
}

// Resolve transitions the alert to resolved state.
func (a *Alert) Resolve() {
	a.State = AlertStateResolved
	now := time.Now()
	a.EndsAt = &now
	a.LastEvaluated = now
}

// Acknowledge marks the alert as acknowledged.
func (a *Alert) Acknowledge(by, comment string) {
	a.State = AlertStateAcknowledged
	now := time.Now()
	a.AcknowledgedAt = &now
	a.AcknowledgedBy = by
	a.AckComment = comment
}

// Silence marks the alert as silenced.
func (a *Alert) Silence() {
	a.State = AlertStateSilenced
}

// NotificationChannel defines a channel for sending alert notifications.
type NotificationChannel struct {
	ID          uuid.UUID               `json:"id"`
	Name        string                  `json:"name"`
	Type        NotificationChannelType `json:"type"`
	Enabled     bool                    `json:"enabled"`
	Config      map[string]string       `json:"config"` // Channel-specific configuration
	CreatedAt   time.Time               `json:"created_at"`
	UpdatedAt   time.Time               `json:"updated_at"`
}

// NewNotificationChannel creates a new notification channel.
func NewNotificationChannel(name string, channelType NotificationChannelType, config map[string]string) *NotificationChannel {
	now := time.Now()
	return &NotificationChannel{
		ID:        uuid.New(),
		Name:      name,
		Type:      channelType,
		Enabled:   true,
		Config:    config,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Silence defines a time period during which alerts matching certain criteria are silenced.
type Silence struct {
	ID        uuid.UUID         `json:"id"`
	Matchers  map[string]string `json:"matchers"`   // Labels that must match for silence to apply
	StartsAt  time.Time         `json:"starts_at"`
	EndsAt    time.Time         `json:"ends_at"`
	CreatedBy string            `json:"created_by"`
	Comment   string            `json:"comment"`
	Active    bool              `json:"active"`
	CreatedAt time.Time         `json:"created_at"`
}

// NewSilence creates a new silence.
func NewSilence(matchers map[string]string, startsAt, endsAt time.Time, createdBy, comment string) *Silence {
	return &Silence{
		ID:        uuid.New(),
		Matchers:  matchers,
		StartsAt:  startsAt,
		EndsAt:    endsAt,
		CreatedBy: createdBy,
		Comment:   comment,
		Active:    true,
		CreatedAt: time.Now(),
	}
}

// IsActive returns whether the silence is currently active.
func (s *Silence) IsActive() bool {
	if !s.Active {
		return false
	}
	now := time.Now()
	return now.After(s.StartsAt) && now.Before(s.EndsAt)
}

// Matches checks if an alert's labels match the silence matchers.
func (s *Silence) Matches(labels map[string]string) bool {
	for key, value := range s.Matchers {
		if labels[key] != value {
			return false
		}
	}
	return true
}

// AlertNotification tracks notification attempts for an alert.
type AlertNotification struct {
	ID         uuid.UUID `json:"id"`
	AlertID    uuid.UUID `json:"alert_id"`
	ChannelID  uuid.UUID `json:"channel_id"`
	SentAt     time.Time `json:"sent_at"`
	Success    bool      `json:"success"`
	Error      string    `json:"error,omitempty"`
	RetryCount int       `json:"retry_count"`
}

// EscalationPolicy defines how alerts should be escalated.
type EscalationPolicy struct {
	ID          uuid.UUID           `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Levels      []EscalationLevel   `json:"levels"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

// EscalationLevel defines a single level in an escalation policy.
type EscalationLevel struct {
	Level        int           `json:"level"`
	Delay        time.Duration `json:"delay"`       // Time to wait before escalating to this level
	ChannelIDs   []string      `json:"channel_ids"` // Channels to notify at this level
	RepeatEvery  time.Duration `json:"repeat_every,omitempty"` // Repeat notification interval
}

// Helper functions

func copyMap(m map[string]string) map[string]string {
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

func generateFingerprint(rule *AlertRule) string {
	// Simple fingerprint based on rule ID and metric name
	return rule.ID.String() + ":" + rule.MetricName
}

