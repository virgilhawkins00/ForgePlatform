// Package domain contains the core business entities for the Forge platform.
package domain

import (
	"regexp"
	"time"

	"github.com/google/uuid"
)

// LogLevel represents the severity level of a log entry.
type LogLevel string

const (
	LogLevelTrace   LogLevel = "trace"
	LogLevelDebug   LogLevel = "debug"
	LogLevelInfo    LogLevel = "info"
	LogLevelWarning LogLevel = "warning"
	LogLevelError   LogLevel = "error"
	LogLevelFatal   LogLevel = "fatal"
)

// LogLevelPriority returns the priority of a log level (higher = more severe).
func LogLevelPriority(level LogLevel) int {
	switch level {
	case LogLevelTrace:
		return 0
	case LogLevelDebug:
		return 1
	case LogLevelInfo:
		return 2
	case LogLevelWarning:
		return 3
	case LogLevelError:
		return 4
	case LogLevelFatal:
		return 5
	default:
		return 2
	}
}

// LogEntry represents a single log entry.
type LogEntry struct {
	ID          uuid.UUID         `json:"id"`
	Timestamp   time.Time         `json:"timestamp"`
	Level       LogLevel          `json:"level"`
	Message     string            `json:"message"`
	Source      string            `json:"source"`
	ServiceName string            `json:"service_name"`
	TraceID     string            `json:"trace_id,omitempty"`
	SpanID      string            `json:"span_id,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
	Resource    map[string]string `json:"resource,omitempty"`
	// Parsed fields from log parsing rules
	ParsedFields map[string]interface{} `json:"parsed_fields,omitempty"`
	// Raw log line (before parsing)
	Raw       string    `json:"raw,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// NewLogEntry creates a new log entry.
func NewLogEntry(level LogLevel, message, source, serviceName string) *LogEntry {
	now := time.Now()
	return &LogEntry{
		ID:          uuid.Must(uuid.NewV7()),
		Timestamp:   now,
		Level:       level,
		Message:     message,
		Source:      source,
		ServiceName: serviceName,
		Attributes:  make(map[string]string),
		Resource:    make(map[string]string),
		CreatedAt:   now,
	}
}

// SetTraceContext sets the trace context for log correlation.
func (l *LogEntry) SetTraceContext(traceID, spanID string) {
	l.TraceID = traceID
	l.SpanID = spanID
}

// SetAttribute sets an attribute on the log entry.
func (l *LogEntry) SetAttribute(key, value string) {
	if l.Attributes == nil {
		l.Attributes = make(map[string]string)
	}
	l.Attributes[key] = value
}

// IsError returns true if the log level is error or fatal.
func (l *LogEntry) IsError() bool {
	return l.Level == LogLevelError || l.Level == LogLevelFatal
}

// LogParserType represents the type of log parser.
type LogParserType string

const (
	ParserTypeRegex   LogParserType = "regex"
	ParserTypeJSON    LogParserType = "json"
	ParserTypeGrok    LogParserType = "grok"
	ParserTypeKeyValue LogParserType = "key_value"
)

// LogParser defines a log parsing rule.
type LogParser struct {
	ID          uuid.UUID     `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	Type        LogParserType `json:"type"`
	Pattern     string        `json:"pattern"`
	// For regex parser: named capture groups map to fields
	// For JSON parser: field mappings
	FieldMappings map[string]string `json:"field_mappings,omitempty"`
	// Source filter: only apply to logs from these sources
	SourceFilter string `json:"source_filter,omitempty"`
	// Priority for ordering parsers
	Priority  int       `json:"priority"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Compiled regex (not serialized)
	compiledRegex *regexp.Regexp
}

// NewLogParser creates a new log parser.
func NewLogParser(name string, parserType LogParserType, pattern string) *LogParser {
	now := time.Now()
	return &LogParser{
		ID:            uuid.Must(uuid.NewV7()),
		Name:          name,
		Type:          parserType,
		Pattern:       pattern,
		FieldMappings: make(map[string]string),
		Enabled:       true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// Compile compiles the regex pattern for the parser.
func (p *LogParser) Compile() error {
	if p.Type == ParserTypeRegex || p.Type == ParserTypeGrok {
		r, err := regexp.Compile(p.Pattern)
		if err != nil {
			return err
		}
		p.compiledRegex = r
	}
	return nil
}

// GetCompiledRegex returns the compiled regex.
func (p *LogParser) GetCompiledRegex() *regexp.Regexp {
	return p.compiledRegex
}

// LogToMetricRule defines a rule for converting logs to metrics.
type LogToMetricRule struct {
	ID          uuid.UUID         `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	// Match condition
	MatchField    string   `json:"match_field"`    // Field to match (e.g., "level", "message")
	MatchPattern  string   `json:"match_pattern"`  // Regex pattern to match
	MatchValues   []string `json:"match_values,omitempty"` // Exact values to match
	// Metric configuration
	MetricName string            `json:"metric_name"`
	MetricType MetricType        `json:"metric_type"` // gauge, counter
	ValueField string            `json:"value_field,omitempty"` // Field to extract value from (for gauge)
	Tags       map[string]string `json:"tags,omitempty"`
	TagFields  []string          `json:"tag_fields,omitempty"` // Log fields to use as metric tags
	Enabled    bool              `json:"enabled"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

// NewLogToMetricRule creates a new log-to-metric rule.
func NewLogToMetricRule(name, matchField, matchPattern, metricName string, metricType MetricType) *LogToMetricRule {
	now := time.Now()
	return &LogToMetricRule{
		ID:           uuid.Must(uuid.NewV7()),
		Name:         name,
		MatchField:   matchField,
		MatchPattern: matchPattern,
		MetricName:   metricName,
		MetricType:   metricType,
		Tags:         make(map[string]string),
		TagFields:    []string{},
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// LogQuery represents a query for searching logs.
type LogQuery struct {
	StartTime   time.Time         `json:"start_time"`
	EndTime     time.Time         `json:"end_time"`
	Level       LogLevel          `json:"level,omitempty"`
	MinLevel    LogLevel          `json:"min_level,omitempty"`
	Source      string            `json:"source,omitempty"`
	ServiceName string            `json:"service_name,omitempty"`
	TraceID     string            `json:"trace_id,omitempty"`
	Search      string            `json:"search,omitempty"` // Full-text search
	Attributes  map[string]string `json:"attributes,omitempty"`
	Limit       int               `json:"limit,omitempty"`
	Offset      int               `json:"offset,omitempty"`
}

// LogStats represents statistics about logs.
type LogStats struct {
	TotalCount   int64            `json:"total_count"`
	ByLevel      map[string]int64 `json:"by_level"`
	ByService    map[string]int64 `json:"by_service"`
	BySource     map[string]int64 `json:"by_source"`
	ErrorRate    float64          `json:"error_rate"`
	TimeRange    time.Duration    `json:"time_range"`
	FirstLogTime time.Time        `json:"first_log_time"`
	LastLogTime  time.Time        `json:"last_log_time"`
}

// LogStream represents a log stream configuration.
type LogStream struct {
	ID          uuid.UUID         `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Source      string            `json:"source"`
	Filters     map[string]string `json:"filters,omitempty"`
	Parsers     []uuid.UUID       `json:"parsers,omitempty"` // Parser IDs to apply
	Retention   time.Duration     `json:"retention"`
	Enabled     bool              `json:"enabled"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// NewLogStream creates a new log stream.
func NewLogStream(name, source string, retention time.Duration) *LogStream {
	now := time.Now()
	return &LogStream{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      name,
		Source:    source,
		Filters:   make(map[string]string),
		Parsers:   []uuid.UUID{},
		Retention: retention,
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

