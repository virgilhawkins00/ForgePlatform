// Package services contains the application services implementing business logic.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/forge-platform/forge/internal/core/domain"
	"github.com/forge-platform/forge/internal/core/ports"
	"github.com/google/uuid"
)

// LogService provides log aggregation and analysis capabilities.
type LogService struct {
	logRepo         ports.LogRepository
	parserRepo      ports.LogParserRepository
	logToMetricRepo ports.LogToMetricRuleRepository
	metricRepo      ports.MetricRepository
	logger          ports.Logger

	// Cached parsers
	mu      sync.RWMutex
	parsers []*domain.LogParser

	// In-memory buffer for batch ingestion
	bufferMu      sync.Mutex
	buffer        []*domain.LogEntry
	bufferSize    int
	flushInterval time.Duration
}

// NewLogService creates a new log service.
func NewLogService(
	logRepo ports.LogRepository,
	parserRepo ports.LogParserRepository,
	logToMetricRepo ports.LogToMetricRuleRepository,
	metricRepo ports.MetricRepository,
	logger ports.Logger,
) *LogService {
	return &LogService{
		logRepo:         logRepo,
		parserRepo:      parserRepo,
		logToMetricRepo: logToMetricRepo,
		metricRepo:      metricRepo,
		logger:          logger,
		parsers:         []*domain.LogParser{},
		buffer:          []*domain.LogEntry{},
		bufferSize:      1000,
		flushInterval:   5 * time.Second,
	}
}

// RefreshParsers reloads parsers from the repository.
func (s *LogService) RefreshParsers(ctx context.Context) error {
	if s.parserRepo == nil {
		return nil
	}

	parsers, err := s.parserRepo.ListEnabled(ctx)
	if err != nil {
		return fmt.Errorf("failed to load parsers: %w", err)
	}

	// Compile regex patterns
	for _, p := range parsers {
		if err := p.Compile(); err != nil {
			s.logger.Warn("failed to compile parser", "parser", p.Name, "error", err)
		}
	}

	s.mu.Lock()
	s.parsers = parsers
	s.mu.Unlock()

	s.logger.Debug("refreshed log parsers", "count", len(parsers))
	return nil
}

// Ingest ingests a single log entry.
func (s *LogService) Ingest(ctx context.Context, entry *domain.LogEntry) error {
	// Parse the log entry
	s.parseEntry(entry)

	// Check log-to-metric rules
	if err := s.applyLogToMetricRules(ctx, entry); err != nil {
		s.logger.Warn("failed to apply log-to-metric rules", "error", err)
	}

	// Persist
	if s.logRepo != nil {
		if err := s.logRepo.Create(ctx, entry); err != nil {
			return fmt.Errorf("failed to persist log entry: %w", err)
		}
	}

	return nil
}

// IngestBatch ingests multiple log entries.
func (s *LogService) IngestBatch(ctx context.Context, entries []*domain.LogEntry) error {
	for _, entry := range entries {
		s.parseEntry(entry)
		if err := s.applyLogToMetricRules(ctx, entry); err != nil {
			s.logger.Warn("failed to apply log-to-metric rules", "error", err)
		}
	}

	if s.logRepo != nil {
		if err := s.logRepo.CreateBatch(ctx, entries); err != nil {
			return fmt.Errorf("failed to persist log entries: %w", err)
		}
	}

	return nil
}

// BufferEntry adds an entry to the buffer for batch processing.
func (s *LogService) BufferEntry(entry *domain.LogEntry) {
	s.bufferMu.Lock()
	s.buffer = append(s.buffer, entry)
	shouldFlush := len(s.buffer) >= s.bufferSize
	s.bufferMu.Unlock()

	if shouldFlush {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := s.FlushBuffer(ctx); err != nil {
				s.logger.Error("failed to flush log buffer", "error", err)
			}
		}()
	}
}

// FlushBuffer flushes the buffered log entries.
func (s *LogService) FlushBuffer(ctx context.Context) error {
	s.bufferMu.Lock()
	entries := s.buffer
	s.buffer = make([]*domain.LogEntry, 0, s.bufferSize)
	s.bufferMu.Unlock()

	if len(entries) == 0 {
		return nil
	}

	return s.IngestBatch(ctx, entries)
}

// parseEntry applies parsing rules to extract structured fields.
func (s *LogService) parseEntry(entry *domain.LogEntry) {
	s.mu.RLock()
	parsers := s.parsers
	s.mu.RUnlock()

	for _, parser := range parsers {
		if parser.SourceFilter != "" && !strings.Contains(entry.Source, parser.SourceFilter) {
			continue
		}

		switch parser.Type {
		case domain.ParserTypeJSON:
			s.parseJSON(entry)
		case domain.ParserTypeRegex:
			s.parseRegex(entry, parser)
		case domain.ParserTypeKeyValue:
			s.parseKeyValue(entry)
		}
	}
}

// parseJSON parses JSON log messages.
func (s *LogService) parseJSON(entry *domain.LogEntry) {
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(entry.Message), &parsed); err != nil {
		return
	}
	entry.ParsedFields = parsed
}

// parseRegex parses log messages using regex.
func (s *LogService) parseRegex(entry *domain.LogEntry, parser *domain.LogParser) {
	re := parser.GetCompiledRegex()
	if re == nil {
		return
	}

	matches := re.FindStringSubmatch(entry.Message)
	if matches == nil {
		return
	}

	names := re.SubexpNames()
	if entry.ParsedFields == nil {
		entry.ParsedFields = make(map[string]interface{})
	}

	for i, name := range names {
		if i > 0 && name != "" && i < len(matches) {
			entry.ParsedFields[name] = matches[i]
		}
	}
}

// parseKeyValue parses key=value formatted logs.
func (s *LogService) parseKeyValue(entry *domain.LogEntry) {
	re := regexp.MustCompile(`(\w+)=("[^"]*"|\S+)`)
	matches := re.FindAllStringSubmatch(entry.Message, -1)

	if entry.ParsedFields == nil {
		entry.ParsedFields = make(map[string]interface{})
	}

	for _, match := range matches {
		if len(match) >= 3 {
			key := match[1]
			value := strings.Trim(match[2], `"`)
			entry.ParsedFields[key] = value
		}
	}
}

// applyLogToMetricRules applies log-to-metric conversion rules.
func (s *LogService) applyLogToMetricRules(ctx context.Context, entry *domain.LogEntry) error {
	if s.logToMetricRepo == nil || s.metricRepo == nil {
		return nil
	}

	rules, err := s.logToMetricRepo.ListEnabled(ctx)
	if err != nil {
		return err
	}

	for _, rule := range rules {
		if s.matchesRule(entry, rule) {
			metric := s.createMetricFromLog(entry, rule)
			if metric != nil {
				if err := s.metricRepo.Record(ctx, metric); err != nil {
					s.logger.Warn("failed to record log-to-metric", "rule", rule.Name, "error", err)
				}
			}
		}
	}

	return nil
}

// matchesRule checks if a log entry matches a log-to-metric rule.
func (s *LogService) matchesRule(entry *domain.LogEntry, rule *domain.LogToMetricRule) bool {
	var fieldValue string

	switch rule.MatchField {
	case "level":
		fieldValue = string(entry.Level)
	case "message":
		fieldValue = entry.Message
	case "source":
		fieldValue = entry.Source
	case "service_name":
		fieldValue = entry.ServiceName
	default:
		if entry.Attributes != nil {
			fieldValue = entry.Attributes[rule.MatchField]
		}
	}

	// Check exact match values
	for _, v := range rule.MatchValues {
		if fieldValue == v {
			return true
		}
	}

	// Check regex pattern
	if rule.MatchPattern != "" {
		matched, _ := regexp.MatchString(rule.MatchPattern, fieldValue)
		return matched
	}

	return false
}

// createMetricFromLog creates a metric from a log entry.
func (s *LogService) createMetricFromLog(entry *domain.LogEntry, rule *domain.LogToMetricRule) *domain.Metric {
	var value float64 = 1 // Default: count

	// Extract value from field if specified
	if rule.ValueField != "" && entry.ParsedFields != nil {
		if v, ok := entry.ParsedFields[rule.ValueField]; ok {
			switch t := v.(type) {
			case float64:
				value = t
			case int:
				value = float64(t)
			case int64:
				value = float64(t)
			}
		}
	}

	// Build tags
	tags := make(map[string]string)
	for k, v := range rule.Tags {
		tags[k] = v
	}
	for _, field := range rule.TagFields {
		if entry.Attributes != nil {
			if v, ok := entry.Attributes[field]; ok {
				tags[field] = v
			}
		}
	}
	tags["source"] = entry.Source
	tags["service"] = entry.ServiceName

	return domain.NewMetric(rule.MetricName, rule.MetricType, value, tags)
}

// Query searches for log entries.
func (s *LogService) Query(ctx context.Context, filter ports.LogFilter) ([]*domain.LogEntry, error) {
	if s.logRepo == nil {
		return []*domain.LogEntry{}, nil
	}
	return s.logRepo.List(ctx, filter)
}

// Search performs full-text search on logs.
func (s *LogService) Search(ctx context.Context, query string, filter ports.LogFilter) ([]*domain.LogEntry, error) {
	if s.logRepo == nil {
		return []*domain.LogEntry{}, nil
	}
	return s.logRepo.Search(ctx, query, filter)
}

// GetStats returns log statistics.
func (s *LogService) GetStats(ctx context.Context, startTime, endTime time.Time) (*domain.LogStats, error) {
	if s.logRepo == nil {
		return &domain.LogStats{
			ByLevel:   make(map[string]int64),
			ByService: make(map[string]int64),
			BySource:  make(map[string]int64),
		}, nil
	}
	return s.logRepo.GetStats(ctx, startTime, endTime)
}

// GetByID retrieves a log entry by ID.
func (s *LogService) GetByID(ctx context.Context, id uuid.UUID) (*domain.LogEntry, error) {
	if s.logRepo == nil {
		return nil, fmt.Errorf("log repository not configured")
	}
	return s.logRepo.GetByID(ctx, id)
}

// GetLogsByTraceID retrieves logs correlated with a trace.
func (s *LogService) GetLogsByTraceID(ctx context.Context, traceID string) ([]*domain.LogEntry, error) {
	filter := ports.LogFilter{
		TraceID: traceID,
		Limit:   1000,
	}
	return s.Query(ctx, filter)
}

// CreateParser creates a new log parser.
func (s *LogService) CreateParser(ctx context.Context, parser *domain.LogParser) error {
	if s.parserRepo == nil {
		return fmt.Errorf("parser repository not configured")
	}
	if err := parser.Compile(); err != nil {
		return fmt.Errorf("invalid parser pattern: %w", err)
	}
	return s.parserRepo.Create(ctx, parser)
}

// ListParsers lists all log parsers.
func (s *LogService) ListParsers(ctx context.Context) ([]*domain.LogParser, error) {
	if s.parserRepo == nil {
		return []*domain.LogParser{}, nil
	}
	return s.parserRepo.List(ctx)
}

// DeleteParser deletes a log parser.
func (s *LogService) DeleteParser(ctx context.Context, id uuid.UUID) error {
	if s.parserRepo == nil {
		return fmt.Errorf("parser repository not configured")
	}
	return s.parserRepo.Delete(ctx, id)
}

