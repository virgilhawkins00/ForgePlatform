package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/forge-platform/forge/internal/core/domain"
	"github.com/forge-platform/forge/internal/core/ports"
	"github.com/forge-platform/forge/internal/core/services"
	"github.com/google/uuid"
)

// Request represents a daemon RPC request.
type Request struct {
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params,omitempty"`
	ID     string                 `json:"id"`
}

// Response represents a daemon RPC response.
type Response struct {
	Result interface{} `json:"result,omitempty"`
	Error  string      `json:"error,omitempty"`
	ID     string      `json:"id"`
}

// acceptConnections accepts incoming connections.
func (s *Server) acceptConnections(ctx context.Context) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		default:
		}

		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.stopCh:
				return
			default:
				s.logger.Error("Failed to accept connection", "error", err)
				continue
			}
		}

		s.wg.Add(1)
		go s.handleConnection(ctx, conn)
	}
}

// handleConnection handles a single client connection.
func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		default:
		}

		// Read request
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				s.logger.Debug("Connection closed", "error", err)
			}
			return
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			s.sendError(conn, "", fmt.Sprintf("invalid request: %v", err))
			continue
		}

		// Handle request
		result, err := s.handleRequest(ctx, &req)
		resp := Response{ID: req.ID}
		if err != nil {
			resp.Error = err.Error()
		} else {
			resp.Result = result
		}

		// Send response
		respBytes, _ := json.Marshal(resp)
		respBytes = append(respBytes, '\n')
		conn.Write(respBytes)
	}
}

// handleRequest routes and handles a request.
func (s *Server) handleRequest(ctx context.Context, req *Request) (interface{}, error) {
	switch req.Method {
	case "status":
		return s.GetStatus(), nil

	case "task.list":
		// TODO: Implement task listing
		return []interface{}{}, nil

	case "task.create":
		// TODO: Implement task creation
		return map[string]string{"status": "created"}, nil

	case "metric.record":
		name, _ := req.Params["name"].(string)
		value, _ := req.Params["value"].(float64)
		tags, _ := req.Params["tags"].(map[string]string)
		if tags == nil {
			tags = make(map[string]string)
		}
		// TODO: Get metric type from params
		err := s.metricSvc.Record(ctx, name, "gauge", value, tags)
		if err != nil {
			return nil, err
		}
		return map[string]string{"status": "recorded"}, nil

	case "metric.query":
		// TODO: Implement metric query
		return map[string]interface{}{"points": []interface{}{}}, nil

	case "metric.stats":
		stats, err := s.metricSvc.GetStats(ctx)
		if err != nil {
			return nil, err
		}
		return stats, nil

	case "metric.series":
		series, err := s.metricSvc.GetDistinctSeries(ctx)
		if err != nil {
			return nil, err
		}
		return series, nil

	case "metric.downsample":
		olderThanSec, _ := req.Params["older_than_seconds"].(float64)
		resolution, _ := req.Params["resolution"].(string)
		if olderThanSec == 0 {
			olderThanSec = 7 * 24 * 3600 // 7 days default
		}
		if resolution == "" {
			resolution = "1m"
		}
		olderThan := time.Duration(olderThanSec) * time.Second
		err := s.metricSvc.Downsample(ctx, olderThan, resolution)
		if err != nil {
			return nil, err
		}
		return map[string]string{"status": "completed"}, nil

	case "plugin.list":
		// TODO: Implement plugin listing
		return map[string]interface{}{"plugins": []interface{}{}}, nil

	case "ai.chat":
		return s.handleAIChat(ctx, req.Params)

	case "ai.ask":
		return s.handleAIAsk(ctx, req.Params)

	case "ai.models":
		return s.handleAIModels(ctx)

	case "ai.analyze":
		return s.handleAIAnalyze(ctx, req.Params)

	case "ai.explain":
		return s.handleAIExplain(ctx, req.Params)

	case "ai.suggest":
		return s.handleAISuggest(ctx, req.Params)

	case "ai.automate":
		return s.handleAIAutomate(ctx, req.Params)

	case "workflow.run":
		return s.handleWorkflowRun(ctx, req.Params)

	case "workflow.list":
		return s.handleWorkflowList(ctx)

	case "workflow.status":
		return s.handleWorkflowStatus(ctx, req.Params)

	case "workflow.cancel":
		return s.handleWorkflowCancel(ctx, req.Params)

	case "workflow.history":
		return s.handleWorkflowHistory(ctx, req.Params)

	// Alert handlers
	case "alert.rule.list":
		return s.handleAlertRuleList(ctx)

	case "alert.rule.create":
		return s.handleAlertRuleCreate(ctx, req.Params)

	case "alert.rule.delete":
		return s.handleAlertRuleDelete(ctx, req.Params)

	case "alert.list.active":
		return s.handleAlertListActive(ctx)

	case "alert.history":
		return s.handleAlertHistory(ctx, req.Params)

	case "alert.ack":
		return s.handleAlertAck(ctx, req.Params)

	case "alert.silence.create":
		return s.handleAlertSilenceCreate(ctx, req.Params)

	case "alert.silence.list":
		return s.handleAlertSilenceList(ctx)

	case "alert.channel.list":
		return s.handleAlertChannelList(ctx)

	// Trace handlers
	case "trace.list":
		return s.handleTraceList(ctx, req.Params)

	case "trace.get":
		return s.handleTraceGet(ctx, req.Params)

	case "trace.spans":
		return s.handleTraceSpans(ctx, req.Params)

	case "trace.service-map":
		return s.handleTraceServiceMap(ctx, req.Params)

	case "trace.stats":
		return s.handleTraceStats(ctx)

	// Log handlers
	case "log.list":
		return s.handleLogList(ctx, req.Params)

	case "log.search":
		return s.handleLogSearch(ctx, req.Params)

	case "log.stats":
		return s.handleLogStats(ctx, req.Params)

	case "log.parser.list":
		return s.handleLogParserList(ctx)

	// Profile handlers
	case "profile.start.cpu":
		return s.handleProfileStartCPU(ctx, req.Params)

	case "profile.start.heap":
		return s.handleProfileStartHeap(ctx, req.Params)

	case "profile.start.goroutine":
		return s.handleProfileStartGoroutine(ctx, req.Params)

	case "profile.list":
		return s.handleProfileList(ctx, req.Params)

	case "profile.get":
		return s.handleProfileGet(ctx, req.Params)

	case "profile.stop":
		return s.handleProfileStop(ctx, req.Params)

	case "profile.delete":
		return s.handleProfileDelete(ctx, req.Params)

	case "profile.stats":
		return s.handleProfileStats(ctx)

	case "profile.memory":
		return s.handleProfileMemory(ctx)

	// User management
	case "user.create":
		return s.handleUserCreate(ctx, req.Params)

	case "user.list":
		return s.handleUserList(ctx, req.Params)

	case "user.get":
		return s.handleUserGet(ctx, req.Params)

	case "user.delete":
		return s.handleUserDelete(ctx, req.Params)

	case "apikey.create":
		return s.handleAPIKeyCreate(ctx, req.Params)

	case "apikey.list":
		return s.handleAPIKeyList(ctx, req.Params)

	case "apikey.revoke":
		return s.handleAPIKeyRevoke(ctx, req.Params)

	case "audit.list":
		return s.handleAuditList(ctx, req.Params)

	default:
		return nil, fmt.Errorf("unknown method: %s", req.Method)
	}
}

// handleAIChat handles AI chat requests.
func (s *Server) handleAIChat(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if s.aiProvider == nil {
		return map[string]interface{}{"content": "AI provider not configured. Start Ollama and restart the daemon."}, nil
	}

	message, _ := params["message"].(string)
	model, _ := params["model"].(string)

	if model != "" && model != s.aiProvider.GetModel() {
		s.aiProvider.SetModel(model)
	}

	// Create a conversation with the user message
	conv := domain.NewConversation(s.aiProvider.GetModel(), "You are a helpful assistant for system administration and DevOps.")
	conv.AddMessage(domain.RoleUser, message)

	// Get response from AI
	response, err := s.aiProvider.Chat(ctx, conv)
	if err != nil {
		return nil, fmt.Errorf("AI error: %w", err)
	}

	return map[string]interface{}{
		"content": response.Content,
	}, nil
}

// handleAIAsk handles single AI question requests.
func (s *Server) handleAIAsk(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if s.aiProvider == nil {
		return map[string]interface{}{"content": "AI provider not configured"}, nil
	}

	question, _ := params["question"].(string)
	model, _ := params["model"].(string)

	if model != "" && model != s.aiProvider.GetModel() {
		s.aiProvider.SetModel(model)
	}

	conv := domain.NewConversation(s.aiProvider.GetModel(), "You are a helpful assistant for system administration and DevOps. Provide concise, actionable answers.")
	conv.AddMessage(domain.RoleUser, question)

	response, err := s.aiProvider.Chat(ctx, conv)
	if err != nil {
		return nil, fmt.Errorf("AI error: %w", err)
	}

	return map[string]interface{}{
		"content": response.Content,
	}, nil
}

// handleAIModels returns available AI models.
func (s *Server) handleAIModels(ctx context.Context) (interface{}, error) {
	if s.aiProvider == nil {
		return map[string]interface{}{"models": []string{}}, nil
	}

	models, err := s.aiProvider.ListModels(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"models":  models,
		"current": s.aiProvider.GetModel(),
	}, nil
}

// handleAIAnalyze performs AI analysis on system metrics.
func (s *Server) handleAIAnalyze(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	timeRangeStr, _ := params["time_range"].(string)
	if timeRangeStr == "" {
		timeRangeStr = "1h"
	}

	timeRange, err := time.ParseDuration(timeRangeStr)
	if err != nil {
		timeRange = time.Hour
	}

	// Use RAG service to analyze
	result, err := s.ragSvc.AnalyzeMetrics(ctx, timeRange)
	if err != nil {
		return nil, fmt.Errorf("analysis error: %w", err)
	}

	// Convert issues to interface slice
	issues := make([]interface{}, len(result.Issues))
	for i, issue := range result.Issues {
		issues[i] = map[string]interface{}{
			"severity":    issue.Severity,
			"component":   issue.Component,
			"description": issue.Description,
			"suggestion":  issue.Suggestion,
		}
	}

	return map[string]interface{}{
		"issues":  issues,
		"summary": result.Summary,
	}, nil
}

// handleAIExplain explains metric behavior using AI.
func (s *Server) handleAIExplain(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	metricName, _ := params["metric"].(string)
	timeRangeStr, _ := params["time_range"].(string)

	if timeRangeStr == "" {
		timeRangeStr = "1h"
	}

	timeRange, err := time.ParseDuration(timeRangeStr)
	if err != nil {
		timeRange = time.Hour
	}

	// Build context using RAG service
	contextReq := services.ContextRequest{
		TimeRange:      timeRange,
		IncludeMetrics: true,
		IncludeTasks:   false,
		IncludeLogs:    false,
	}

	if metricName != "" {
		contextReq.MetricNames = []string{metricName}
	}

	contextResult, err := s.ragSvc.BuildContext(ctx, contextReq)
	if err != nil {
		return nil, fmt.Errorf("context error: %w", err)
	}

	// If no AI provider, return RAG analysis only
	if s.aiProvider == nil {
		explanation := "## Metric Analysis\n\n"
		for _, m := range contextResult.Metrics {
			explanation += fmt.Sprintf("### %s\n", m.Name)
			explanation += fmt.Sprintf("- Current: %.2f\n", m.Latest)
			explanation += fmt.Sprintf("- Range: %.2f - %.2f\n", m.Min, m.Max)
			explanation += fmt.Sprintf("- Average: %.2f\n", m.Avg)
			explanation += fmt.Sprintf("- Trend: %s\n", m.Trend)
			if len(m.Anomalies) > 0 {
				explanation += fmt.Sprintf("- Anomalies: %v\n", m.Anomalies)
			}
			explanation += "\n"
		}
		return map[string]interface{}{"explanation": explanation}, nil
	}

	// Create conversation with context
	modelName := ""
	if s.aiProvider != nil {
		modelName = s.aiProvider.GetModel()
	}
	conv := domain.NewConversation(modelName, contextResult.SystemPrompt)
	conv.AddMessage(domain.RoleUser, fmt.Sprintf("Explain the behavior of the metric '%s' over the last %s. What patterns do you see?", metricName, timeRangeStr))

	response, err := s.aiProvider.Chat(ctx, conv)
	if err != nil {
		return nil, fmt.Errorf("AI error: %w", err)
	}

	return map[string]interface{}{
		"explanation": response.Content,
	}, nil
}

// handleAISuggest generates optimization suggestions.
func (s *Server) handleAISuggest(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	timeRangeStr, _ := params["time_range"].(string)
	if timeRangeStr == "" {
		timeRangeStr = "1h"
	}

	timeRange, err := time.ParseDuration(timeRangeStr)
	if err != nil {
		timeRange = time.Hour
	}

	// Build context
	contextReq := services.ContextRequest{
		TimeRange:      timeRange,
		IncludeMetrics: true,
		IncludeTasks:   true,
		IncludeLogs:    false,
	}

	contextResult, err := s.ragSvc.BuildContext(ctx, contextReq)
	if err != nil {
		return nil, fmt.Errorf("context error: %w", err)
	}

	// If no AI provider, generate basic suggestions from analysis
	if s.aiProvider == nil {
		suggestions := []interface{}{}
		for _, m := range contextResult.Metrics {
			if m.Trend == "increasing" && m.Latest > m.Avg*1.5 {
				suggestions = append(suggestions, map[string]interface{}{
					"title":       fmt.Sprintf("Investigate %s increase", m.Name),
					"description": fmt.Sprintf("The metric %s is trending upward and currently %.2f (avg: %.2f)", m.Name, m.Latest, m.Avg),
					"impact":      "medium",
					"effort":      "low",
				})
			}
		}
		return map[string]interface{}{"suggestions": suggestions}, nil
	}

	// Use AI for suggestions
	conv := domain.NewConversation(s.aiProvider.GetModel(), contextResult.SystemPrompt)
	conv.AddMessage(domain.RoleUser, `Based on the system metrics and current state, provide optimization suggestions.
Format each suggestion as:
1. **Title**: Brief description
   - Description: Detailed explanation
   - Impact: high/medium/low
   - Effort: high/medium/low`)

	response, err := s.aiProvider.Chat(ctx, conv)
	if err != nil {
		return nil, fmt.Errorf("AI error: %w", err)
	}

	// Parse response into suggestions (simplified)
	return map[string]interface{}{
		"suggestions": []interface{}{
			map[string]interface{}{
				"title":       "AI Generated Suggestions",
				"description": response.Content,
				"impact":      "varies",
				"effort":      "varies",
			},
		},
	}, nil
}

// handleAIAutomate creates automation rules from natural language.
func (s *Server) handleAIAutomate(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	description, _ := params["description"].(string)

	if s.aiProvider == nil {
		return map[string]interface{}{
			"rule": map[string]interface{}{
				"name":      "generated-rule",
				"trigger":   "schedule: */5 * * * *",
				"condition": "true",
				"action":    "echo 'AI not connected - cannot parse: " + description + "'",
			},
		}, nil
	}

	systemPrompt := `You are an automation rule generator. Convert natural language descriptions into structured automation rules.
Output JSON with the following structure:
{
  "name": "rule-name",
  "trigger": "metric:cpu_usage > 80 | schedule:cron | event:type",
  "condition": "expression to evaluate",
  "action": "command or action to execute"
}`
	conv := domain.NewConversation(s.aiProvider.GetModel(), systemPrompt)
	conv.AddMessage(domain.RoleUser, description)

	response, err := s.aiProvider.Chat(ctx, conv)
	if err != nil {
		return nil, fmt.Errorf("AI error: %w", err)
	}

	// Try to parse as JSON, otherwise return as-is
	var rule map[string]interface{}
	if err := json.Unmarshal([]byte(response.Content), &rule); err != nil {
		rule = map[string]interface{}{
			"name":      "generated-rule",
			"trigger":   "manual",
			"condition": "true",
			"action":    response.Content,
		}
	}

	return map[string]interface{}{
		"rule": rule,
	}, nil
}

// handleWorkflowRun executes a workflow from a YAML file.
func (s *Server) handleWorkflowRun(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	filePath, ok := params["file"].(string)
	if !ok || filePath == "" {
		return nil, fmt.Errorf("file path is required")
	}

	input, _ := params["input"].(map[string]interface{})
	if input == nil {
		input = make(map[string]interface{})
	}

	async, _ := params["async"].(bool)

	// Load workflow from file
	workflow, err := s.workflowSvc.LoadFromFile(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load workflow: %w", err)
	}

	if async {
		// Run asynchronously
		go func() {
			bgCtx := context.Background()
			_, err := s.workflowSvc.Run(bgCtx, workflow, input)
			if err != nil {
				s.logger.Error("Async workflow failed", "workflow", workflow.Name, "error", err)
			}
		}()
		return map[string]interface{}{
			"execution_id":  "pending",
			"workflow_name": workflow.Name,
			"status":        "started",
		}, nil
	}

	// Run synchronously
	execution, err := s.workflowSvc.Run(ctx, workflow, input)
	if err != nil {
		return nil, err
	}

	return executionToMap(execution), nil
}

// handleWorkflowList lists all workflow definitions.
func (s *Server) handleWorkflowList(ctx context.Context) (interface{}, error) {
	// For now, return empty list since we don't persist definitions yet
	return map[string]interface{}{
		"workflows": []interface{}{},
	}, nil
}

// handleWorkflowStatus gets the status of a workflow execution.
func (s *Server) handleWorkflowStatus(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	executionID, ok := params["execution_id"].(string)
	if !ok || executionID == "" {
		return nil, fmt.Errorf("execution_id is required")
	}

	// TODO: Implement execution lookup
	return map[string]interface{}{
		"id":     executionID,
		"status": "unknown",
		"error":  "execution repository not implemented",
	}, nil
}

// handleWorkflowCancel cancels a running workflow.
func (s *Server) handleWorkflowCancel(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	executionID, ok := params["execution_id"].(string)
	if !ok || executionID == "" {
		return nil, fmt.Errorf("execution_id is required")
	}

	// TODO: Parse UUID and cancel
	return map[string]interface{}{
		"status": "cancelled",
	}, nil
}

// handleWorkflowHistory gets workflow execution history.
func (s *Server) handleWorkflowHistory(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// TODO: Implement history query
	return map[string]interface{}{
		"executions": []interface{}{},
	}, nil
}

// executionToMap converts a WorkflowExecution to a map.
func executionToMap(e *domain.WorkflowExecution) map[string]interface{} {
	steps := make([]map[string]interface{}, len(e.Steps))
	for i, s := range e.Steps {
		steps[i] = map[string]interface{}{
			"step_id":      s.StepID,
			"step_name":    s.StepName,
			"status":       string(s.Status),
			"retry_count":  s.RetryCount,
			"error":        s.Error,
			"started_at":   s.StartedAt,
			"completed_at": s.CompletedAt,
			"duration":     s.Duration,
		}
	}

	return map[string]interface{}{
		"id":            e.ID.String(),
		"workflow_id":   e.WorkflowID.String(),
		"workflow_name": e.WorkflowName,
		"status":        string(e.Status),
		"steps":         steps,
		"input":         e.Input,
		"output":        e.Output,
		"error":         e.Error,
		"started_at":    e.StartedAt,
		"completed_at":  e.CompletedAt,
		"duration":      e.Duration,
	}
}

// sendError sends an error response.
func (s *Server) sendError(conn net.Conn, id, errMsg string) {
	resp := Response{ID: id, Error: errMsg}
	respBytes, _ := json.Marshal(resp)
	respBytes = append(respBytes, '\n')
	conn.Write(respBytes)
}

// handleAlertRuleList lists all alert rules.
func (s *Server) handleAlertRuleList(ctx context.Context) (interface{}, error) {
	if s.alertSvc == nil {
		return map[string]interface{}{"rules": []interface{}{}}, nil
	}

	rules, err := s.alertSvc.ListRules(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]interface{}, len(rules))
	for i, r := range rules {
		result[i] = map[string]interface{}{
			"id":          r.ID.String(),
			"name":        r.Name,
			"metric_name": r.MetricName,
			"condition":   string(r.Condition),
			"threshold":   r.Threshold,
			"severity":    string(r.Severity),
			"duration":    r.Duration.String(),
			"interval":    r.Interval.String(),
			"enabled":     r.Enabled,
			"channels":    r.Channels,
			"labels":      r.Labels,
		}
	}
	return map[string]interface{}{"rules": result}, nil
}

// handleAlertRuleCreate creates a new alert rule.
func (s *Server) handleAlertRuleCreate(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if s.alertSvc == nil {
		return nil, fmt.Errorf("alert service not available")
	}

	name, _ := params["name"].(string)
	metricName, _ := params["metric_name"].(string)
	conditionStr, _ := params["condition"].(string)
	threshold, _ := params["threshold"].(float64)
	severityStr, _ := params["severity"].(string)
	durationStr, _ := params["duration"].(string)
	intervalStr, _ := params["interval"].(string)

	if name == "" || metricName == "" {
		return nil, fmt.Errorf("name and metric_name are required")
	}

	duration, _ := time.ParseDuration(durationStr)
	if duration == 0 {
		duration = time.Minute
	}

	interval, _ := time.ParseDuration(intervalStr)
	if interval == 0 {
		interval = time.Minute
	}

	condition := domain.RuleConditionType(conditionStr)
	if condition == "" {
		condition = domain.ConditionThresholdAbove
	}

	severity := domain.AlertSeverity(severityStr)
	if severity == "" {
		severity = domain.AlertSeverityWarning
	}

	rule := domain.NewAlertRule(name, metricName, condition, threshold, severity)
	rule.Duration = duration
	rule.Interval = interval

	err := s.alertSvc.CreateRule(ctx, rule)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":   rule.ID.String(),
		"name": rule.Name,
	}, nil
}

// handleAlertRuleDelete deletes an alert rule.
func (s *Server) handleAlertRuleDelete(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if s.alertSvc == nil {
		return nil, fmt.Errorf("alert service not available")
	}

	idStr, _ := params["id"].(string)
	if idStr == "" {
		return nil, fmt.Errorf("id is required")
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", err)
	}

	err = s.alertSvc.DeleteRule(ctx, id)
	if err != nil {
		return nil, err
	}

	return map[string]string{"status": "deleted"}, nil
}


// handleAlertListActive lists active alerts.
func (s *Server) handleAlertListActive(ctx context.Context) (interface{}, error) {
	if s.alertSvc == nil {
		return map[string]interface{}{"alerts": []interface{}{}}, nil
	}

	alerts, err := s.alertSvc.ListActiveAlerts(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]interface{}, len(alerts))
	for i, a := range alerts {
		result[i] = s.alertToMap(a)
	}
	return map[string]interface{}{"alerts": result}, nil
}

// handleAlertHistory returns alert history.
func (s *Server) handleAlertHistory(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if s.alertSvc == nil {
		return map[string]interface{}{"alerts": []interface{}{}}, nil
	}

	limit, _ := params["limit"].(float64)
	if limit == 0 {
		limit = 50
	}
	stateStr, _ := params["state"].(string)
	severityStr, _ := params["severity"].(string)

	filter := ports.AlertFilter{
		Limit: int(limit),
	}
	if stateStr != "" {
		filter.State = (*domain.AlertState)(&stateStr)
	}
	if severityStr != "" {
		filter.Severity = (*domain.AlertSeverity)(&severityStr)
	}

	alerts, err := s.alertSvc.ListAlerts(ctx, filter)
	if err != nil {
		return nil, err
	}

	result := make([]interface{}, len(alerts))
	for i, a := range alerts {
		result[i] = s.alertToMap(a)
	}
	return map[string]interface{}{"alerts": result}, nil
}

// handleAlertAck acknowledges an alert.
func (s *Server) handleAlertAck(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if s.alertSvc == nil {
		return nil, fmt.Errorf("alert service not available")
	}

	idStr, _ := params["id"].(string)
	comment, _ := params["comment"].(string)
	if idStr == "" {
		return nil, fmt.Errorf("id is required")
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", err)
	}

	err = s.alertSvc.AcknowledgeAlert(ctx, id, "daemon-user", comment)
	if err != nil {
		return nil, err
	}

	return map[string]string{"status": "acknowledged"}, nil
}

// handleAlertSilenceCreate creates a new silence.
func (s *Server) handleAlertSilenceCreate(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if s.alertSvc == nil {
		return nil, fmt.Errorf("alert service not available")
	}

	matchersRaw, _ := params["matchers"].(map[string]interface{})
	durationStr, _ := params["duration"].(string)
	comment, _ := params["comment"].(string)

	matchers := make(map[string]string)
	for k, v := range matchersRaw {
		matchers[k] = fmt.Sprintf("%v", v)
	}

	duration, _ := time.ParseDuration(durationStr)
	if duration == 0 {
		duration = time.Hour
	}

	now := time.Now()
	silence := &domain.Silence{
		ID:        uuid.New(),
		Matchers:  matchers,
		StartsAt:  now,
		EndsAt:    now.Add(duration),
		Comment:   comment,
		CreatedBy: "daemon-user",
		CreatedAt: now,
	}

	err := s.alertSvc.CreateSilence(ctx, silence)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":        silence.ID.String(),
		"starts_at": silence.StartsAt.Format(time.RFC3339),
		"ends_at":   silence.EndsAt.Format(time.RFC3339),
	}, nil
}

// handleAlertSilenceList lists active silences.
func (s *Server) handleAlertSilenceList(ctx context.Context) (interface{}, error) {
	if s.alertSvc == nil {
		return map[string]interface{}{"silences": []interface{}{}}, nil
	}

	silences, err := s.alertSvc.ListSilences(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]interface{}, len(silences))
	for i, sil := range silences {
		result[i] = map[string]interface{}{
			"id":         sil.ID.String(),
			"matchers":   sil.Matchers,
			"starts_at":  sil.StartsAt.Format(time.RFC3339),
			"ends_at":    sil.EndsAt.Format(time.RFC3339),
			"comment":    sil.Comment,
			"created_by": sil.CreatedBy,
		}
	}
	return map[string]interface{}{"silences": result}, nil
}

// handleAlertChannelList lists notification channels.
func (s *Server) handleAlertChannelList(ctx context.Context) (interface{}, error) {
	if s.alertSvc == nil {
		return map[string]interface{}{"channels": []interface{}{}}, nil
	}

	channels, err := s.alertSvc.ListChannels(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]interface{}, len(channels))
	for i, ch := range channels {
		result[i] = map[string]interface{}{
			"id":      ch.ID.String(),
			"name":    ch.Name,
			"type":    string(ch.Type),
			"enabled": ch.Enabled,
		}
	}
	return map[string]interface{}{"channels": result}, nil
}

// alertToMap converts an alert to a map for JSON serialization.
func (s *Server) alertToMap(a *domain.Alert) map[string]interface{} {
	result := map[string]interface{}{
		"id":          a.ID.String(),
		"rule_id":     a.RuleID.String(),
		"rule_name":   a.RuleName,
		"state":       string(a.State),
		"severity":    string(a.Severity),
		"message":     a.Message,
		"value":       a.Value,
		"threshold":   a.Threshold,
		"starts_at":   a.StartsAt.Format(time.RFC3339),
		"fingerprint": a.Fingerprint,
		"labels":      a.Labels,
	}
	if a.EndsAt != nil {
		result["ends_at"] = a.EndsAt.Format(time.RFC3339)
	}
	if a.AcknowledgedAt != nil {
		result["acknowledged_at"] = a.AcknowledgedAt.Format(time.RFC3339)
		result["acknowledged_by"] = a.AcknowledgedBy
	}
	return result
}

// ============================================================================
// Trace Handlers
// ============================================================================

// handleTraceList lists traces with optional filtering.
func (s *Server) handleTraceList(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if s.traceSvc == nil {
		return map[string]interface{}{"traces": []interface{}{}}, nil
	}

	filter := ports.TraceFilter{
		Limit: 20,
	}

	if service, ok := params["service_name"].(string); ok && service != "" {
		filter.ServiceName = service
	}
	if status, ok := params["status"].(string); ok && status != "" {
		filter.Status = status
	}
	if startTime, ok := params["start_time"].(string); ok && startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filter.StartTime = t
		}
	}
	if limit, ok := params["limit"].(float64); ok && limit > 0 {
		filter.Limit = int(limit)
	}

	traces, err := s.traceSvc.ListTraces(ctx, filter)
	if err != nil {
		return nil, err
	}

	result := make([]interface{}, len(traces))
	for i, t := range traces {
		result[i] = s.traceToMap(t)
	}
	return map[string]interface{}{"traces": result}, nil
}

// handleTraceGet gets a trace by ID.
func (s *Server) handleTraceGet(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if s.traceSvc == nil {
		return nil, fmt.Errorf("trace service not configured")
	}

	traceIDStr, _ := params["trace_id"].(string)
	if traceIDStr == "" {
		return nil, fmt.Errorf("trace_id is required")
	}

	traceID, err := domain.ParseTraceID(traceIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid trace_id: %w", err)
	}

	trace, err := s.traceSvc.GetTraceByTraceID(ctx, traceID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{"trace": s.traceToMap(trace)}, nil
}

// handleTraceSpans gets spans for a trace.
func (s *Server) handleTraceSpans(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if s.traceSvc == nil {
		return map[string]interface{}{"spans": []interface{}{}}, nil
	}

	traceIDStr, _ := params["trace_id"].(string)
	if traceIDStr == "" {
		return nil, fmt.Errorf("trace_id is required")
	}

	traceID, err := domain.ParseTraceID(traceIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid trace_id: %w", err)
	}

	spans, err := s.traceSvc.GetSpansByTraceID(ctx, traceID)
	if err != nil {
		return nil, err
	}

	result := make([]interface{}, len(spans))
	for i, sp := range spans {
		result[i] = s.spanToMap(sp)
	}
	return map[string]interface{}{"spans": result}, nil
}

// handleTraceServiceMap gets the service dependency map.
func (s *Server) handleTraceServiceMap(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if s.traceSvc == nil {
		return map[string]interface{}{"nodes": []interface{}{}}, nil
	}

	startTime := time.Now().Add(-24 * time.Hour)
	endTime := time.Now()

	if st, ok := params["start_time"].(string); ok && st != "" {
		if t, err := time.Parse(time.RFC3339, st); err == nil {
			startTime = t
		}
	}
	if et, ok := params["end_time"].(string); ok && et != "" {
		if t, err := time.Parse(time.RFC3339, et); err == nil {
			endTime = t
		}
	}

	serviceMap, err := s.traceSvc.GetServiceMap(ctx, startTime, endTime)
	if err != nil {
		return nil, err
	}

	nodes := make([]interface{}, len(serviceMap.Nodes))
	for i, n := range serviceMap.Nodes {
		nodes[i] = map[string]interface{}{
			"service_name":    n.ServiceName,
			"span_count":      n.SpanCount,
			"error_count":     n.ErrorCount,
			"avg_duration_ms": n.AvgDuration,
			"dependencies":    n.Dependencies,
		}
	}
	return map[string]interface{}{"nodes": nodes}, nil
}

// handleTraceStats gets trace statistics.
func (s *Server) handleTraceStats(ctx context.Context) (interface{}, error) {
	if s.traceSvc == nil {
		return map[string]interface{}{"active_traces": 0}, nil
	}

	stats, err := s.traceSvc.GetTraceStats(ctx)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

// traceToMap converts a trace to a map for JSON serialization.
func (s *Server) traceToMap(t *domain.Trace) map[string]interface{} {
	return map[string]interface{}{
		"id":           t.ID.String(),
		"trace_id":     t.TraceID.String(),
		"service_name": t.ServiceName,
		"name":         t.Name,
		"status":       string(t.Status),
		"duration":     t.Duration.String(),
		"span_count":   t.SpanCount,
		"error_count":  t.ErrorCount,
		"start_time":   t.StartTime.Format(time.RFC3339),
		"end_time":     t.EndTime.Format(time.RFC3339),
	}
}

// spanToMap converts a span to a map for JSON serialization.
func (s *Server) spanToMap(sp *domain.Span) map[string]interface{} {
	return map[string]interface{}{
		"id":           sp.ID.String(),
		"trace_id":     sp.TraceID.String(),
		"span_id":      sp.SpanID.String(),
		"name":         sp.Name,
		"kind":         string(sp.Kind),
		"status":       string(sp.Status),
		"duration":     sp.Duration.String(),
		"service_name": sp.ServiceName,
		"start_time":   sp.StartTime.Format(time.RFC3339),
		"end_time":     sp.EndTime.Format(time.RFC3339),
		"attributes":   sp.Attributes,
	}
}

// ============================================================================
// Log Handlers
// ============================================================================

// handleLogList lists log entries with optional filtering.
func (s *Server) handleLogList(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if s.logSvc == nil {
		return map[string]interface{}{"logs": []interface{}{}}, nil
	}

	filter := ports.LogFilter{
		Limit: 50,
	}

	if level, ok := params["level"].(string); ok && level != "" {
		filter.Level = domain.LogLevel(level)
	}
	if service, ok := params["service_name"].(string); ok && service != "" {
		filter.ServiceName = service
	}
	if source, ok := params["source"].(string); ok && source != "" {
		filter.Source = source
	}
	if traceID, ok := params["trace_id"].(string); ok && traceID != "" {
		filter.TraceID = traceID
	}
	if startTime, ok := params["start_time"].(string); ok && startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filter.StartTime = t
		}
	}
	if limit, ok := params["limit"].(float64); ok && limit > 0 {
		filter.Limit = int(limit)
	}

	logs, err := s.logSvc.Query(ctx, filter)
	if err != nil {
		return nil, err
	}

	result := make([]interface{}, len(logs))
	for i, l := range logs {
		result[i] = s.logEntryToMap(l)
	}
	return map[string]interface{}{"logs": result}, nil
}

// handleLogSearch searches log entries.
func (s *Server) handleLogSearch(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if s.logSvc == nil {
		return map[string]interface{}{"logs": []interface{}{}}, nil
	}

	query, _ := params["query"].(string)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}

	filter := ports.LogFilter{
		Limit: 50,
	}
	if startTime, ok := params["start_time"].(string); ok && startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filter.StartTime = t
		}
	}
	if limit, ok := params["limit"].(float64); ok && limit > 0 {
		filter.Limit = int(limit)
	}

	logs, err := s.logSvc.Search(ctx, query, filter)
	if err != nil {
		return nil, err
	}

	result := make([]interface{}, len(logs))
	for i, l := range logs {
		result[i] = s.logEntryToMap(l)
	}
	return map[string]interface{}{"logs": result}, nil
}

// handleLogStats gets log statistics.
func (s *Server) handleLogStats(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if s.logSvc == nil {
		return map[string]interface{}{
			"total_count": 0,
			"by_level":    map[string]int64{},
			"by_service":  map[string]int64{},
		}, nil
	}

	startTime := time.Now().Add(-time.Hour)
	endTime := time.Now()

	if st, ok := params["start_time"].(string); ok && st != "" {
		if t, err := time.Parse(time.RFC3339, st); err == nil {
			startTime = t
		}
	}
	if et, ok := params["end_time"].(string); ok && et != "" {
		if t, err := time.Parse(time.RFC3339, et); err == nil {
			endTime = t
		}
	}

	stats, err := s.logSvc.GetStats(ctx, startTime, endTime)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_count":    stats.TotalCount,
		"by_level":       stats.ByLevel,
		"by_service":     stats.ByService,
		"by_source":      stats.BySource,
		"first_log_time": stats.FirstLogTime.Format(time.RFC3339),
		"last_log_time":  stats.LastLogTime.Format(time.RFC3339),
	}, nil
}

// handleLogParserList lists log parsers.
func (s *Server) handleLogParserList(ctx context.Context) (interface{}, error) {
	if s.logSvc == nil {
		return map[string]interface{}{"parsers": []interface{}{}}, nil
	}

	parsers, err := s.logSvc.ListParsers(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]interface{}, len(parsers))
	for i, p := range parsers {
		result[i] = map[string]interface{}{
			"id":            p.ID.String(),
			"name":          p.Name,
			"type":          string(p.Type),
			"enabled":       p.Enabled,
			"source_filter": p.SourceFilter,
		}
	}
	return map[string]interface{}{"parsers": result}, nil
}

// logEntryToMap converts a log entry to a map for JSON serialization.
func (s *Server) logEntryToMap(l *domain.LogEntry) map[string]interface{} {
	return map[string]interface{}{
		"id":           l.ID.String(),
		"timestamp":    l.Timestamp.Format(time.RFC3339),
		"level":        string(l.Level),
		"message":      l.Message,
		"source":       l.Source,
		"service_name": l.ServiceName,
		"trace_id":     l.TraceID,
		"span_id":      l.SpanID,
		"attributes":   l.Attributes,
	}
}

// ============================================================================
// Profile Handlers
// ============================================================================

// handleProfileStartCPU starts CPU profiling.
func (s *Server) handleProfileStartCPU(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if s.profileSvc == nil {
		return nil, fmt.Errorf("profile service not configured")
	}

	name, _ := params["name"].(string)
	serviceName, _ := params["service_name"].(string)
	durationStr, _ := params["duration"].(string)

	duration := 30 * time.Second
	if durationStr != "" {
		if d, err := time.ParseDuration(durationStr); err == nil {
			duration = d
		}
	}

	profile, err := s.profileSvc.StartCPUProfile(ctx, name, serviceName, duration)
	if err != nil {
		return nil, err
	}

	return s.profileToMap(profile), nil
}

// handleProfileStartHeap captures a heap profile.
func (s *Server) handleProfileStartHeap(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if s.profileSvc == nil {
		return nil, fmt.Errorf("profile service not configured")
	}

	name, _ := params["name"].(string)
	serviceName, _ := params["service_name"].(string)

	profile, err := s.profileSvc.CaptureHeapProfile(ctx, name, serviceName)
	if err != nil {
		return nil, err
	}

	return s.profileToMap(profile), nil
}

// handleProfileStartGoroutine captures a goroutine profile.
func (s *Server) handleProfileStartGoroutine(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if s.profileSvc == nil {
		return nil, fmt.Errorf("profile service not configured")
	}

	name, _ := params["name"].(string)
	serviceName, _ := params["service_name"].(string)

	profile, err := s.profileSvc.CaptureGoroutineProfile(ctx, name, serviceName)
	if err != nil {
		return nil, err
	}

	return s.profileToMap(profile), nil
}

// handleProfileList lists profiles.
func (s *Server) handleProfileList(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if s.profileSvc == nil {
		return map[string]interface{}{"profiles": []interface{}{}}, nil
	}

	filter := ports.ProfileFilter{
		Limit: 20,
	}

	if profileType, ok := params["type"].(string); ok && profileType != "" {
		filter.Type = domain.ProfileType(profileType)
	}
	if limit, ok := params["limit"].(float64); ok && limit > 0 {
		filter.Limit = int(limit)
	}

	profiles, err := s.profileSvc.ListProfiles(ctx, filter)
	if err != nil {
		return nil, err
	}

	result := make([]interface{}, len(profiles))
	for i, p := range profiles {
		result[i] = s.profileToMap(p)
	}
	return map[string]interface{}{"profiles": result}, nil
}

// handleProfileGet gets a profile by ID.
func (s *Server) handleProfileGet(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if s.profileSvc == nil {
		return nil, fmt.Errorf("profile service not configured")
	}

	idStr, _ := params["id"].(string)
	if idStr == "" {
		return nil, fmt.Errorf("id is required")
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", err)
	}

	profile, err := s.profileSvc.GetProfile(ctx, id)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{"profile": s.profileToMap(profile)}, nil
}

// handleProfileStop stops an active profile.
func (s *Server) handleProfileStop(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if s.profileSvc == nil {
		return nil, fmt.Errorf("profile service not configured")
	}

	idStr, _ := params["id"].(string)
	if idStr == "" {
		return nil, fmt.Errorf("id is required")
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", err)
	}

	profile, err := s.profileSvc.StopProfile(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.profileToMap(profile), nil
}

// handleProfileDelete deletes a profile.
func (s *Server) handleProfileDelete(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if s.profileSvc == nil {
		return nil, fmt.Errorf("profile service not configured")
	}

	idStr, _ := params["id"].(string)
	if idStr == "" {
		return nil, fmt.Errorf("id is required")
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", err)
	}

	if err := s.profileSvc.DeleteProfile(ctx, id); err != nil {
		return nil, err
	}

	return map[string]string{"status": "deleted"}, nil
}

// handleProfileStats gets profile statistics.
func (s *Server) handleProfileStats(ctx context.Context) (interface{}, error) {
	if s.profileSvc == nil {
		return map[string]interface{}{
			"active_profiles": 0,
			"num_goroutine":   0,
		}, nil
	}

	stats, err := s.profileSvc.GetProfileStats(ctx)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

// handleProfileMemory gets current memory statistics.
func (s *Server) handleProfileMemory(ctx context.Context) (interface{}, error) {
	if s.profileSvc == nil {
		return nil, fmt.Errorf("profile service not configured")
	}

	stats := s.profileSvc.GetMemoryStats()
	return map[string]interface{}{
		"alloc":         stats.Alloc,
		"total_alloc":   stats.TotalAlloc,
		"sys":           stats.Sys,
		"heap_alloc":    stats.HeapAlloc,
		"heap_sys":      stats.HeapSys,
		"heap_idle":     stats.HeapIdle,
		"heap_inuse":    stats.HeapInuse,
		"heap_released": stats.HeapReleased,
		"heap_objects":  stats.HeapObjects,
		"stack_inuse":   stats.StackInuse,
		"stack_sys":     stats.StackSys,
		"num_gc":        stats.NumGC,
		"num_goroutine": stats.NumGoroutine,
		"captured_at":   stats.CapturedAt.Format(time.RFC3339),
	}, nil
}

// profileToMap converts a profile to a map for JSON serialization.
func (s *Server) profileToMap(p *domain.Profile) map[string]interface{} {
	return map[string]interface{}{
		"id":           p.ID.String(),
		"name":         p.Name,
		"type":         string(p.Type),
		"status":       string(p.Status),
		"service_name": p.ServiceName,
		"duration":     p.Duration.String(),
		"data_size":    p.DataSize,
		"file_path":    p.FilePath,
		"created_at":   p.CreatedAt.Format(time.RFC3339),
	}
}

// ============================================================================
// User Management Handlers
// ============================================================================

// handleUserCreate creates a new user.
func (s *Server) handleUserCreate(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if s.authSvc == nil {
		return nil, fmt.Errorf("auth service not configured")
	}

	username, _ := params["username"].(string)
	email, _ := params["email"].(string)
	password, _ := params["password"].(string)
	roleStr, _ := params["role"].(string)

	if username == "" || email == "" || password == "" {
		return nil, fmt.Errorf("username, email, and password are required")
	}

	role := domain.UserRole(roleStr)
	if role == "" {
		role = domain.RoleViewer
	}

	user, err := s.authSvc.CreateUser(ctx, username, email, password, role)
	if err != nil {
		return nil, err
	}

	return s.userToMap(user), nil
}

// handleUserList lists all users.
func (s *Server) handleUserList(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if s.authSvc == nil {
		return map[string]interface{}{"users": []interface{}{}}, nil
	}

	filter := ports.UserFilter{
		Limit: 100,
	}

	if role, ok := params["role"].(string); ok && role != "" {
		filter.Role = domain.UserRole(role)
	}
	if status, ok := params["status"].(string); ok && status != "" {
		filter.Status = domain.UserStatus(status)
	}

	users, err := s.authSvc.ListUsers(ctx, filter)
	if err != nil {
		return nil, err
	}

	result := make([]interface{}, len(users))
	for i, u := range users {
		result[i] = s.userToMap(u)
	}
	return map[string]interface{}{"users": result}, nil
}

// handleUserGet retrieves a user by username.
func (s *Server) handleUserGet(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if s.authSvc == nil {
		return nil, fmt.Errorf("auth service not configured")
	}

	username, _ := params["username"].(string)
	if username == "" {
		return nil, fmt.Errorf("username is required")
	}

	// For now, we need to list and filter since we don't have userRepo directly
	users, err := s.authSvc.ListUsers(ctx, ports.UserFilter{Username: username, Limit: 1})
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("user not found")
	}

	return s.userToMap(users[0]), nil
}

// handleUserDelete deletes a user.
func (s *Server) handleUserDelete(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if s.authSvc == nil {
		return nil, fmt.Errorf("auth service not configured")
	}

	username, _ := params["username"].(string)
	if username == "" {
		return nil, fmt.Errorf("username is required")
	}

	// Get user first
	users, err := s.authSvc.ListUsers(ctx, ports.UserFilter{Username: username, Limit: 1})
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("user not found")
	}

	if err := s.authSvc.DeleteUser(ctx, users[0].ID); err != nil {
		return nil, err
	}

	return map[string]interface{}{"status": "deleted", "username": username}, nil
}

// handleAPIKeyCreate creates a new API key.
func (s *Server) handleAPIKeyCreate(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if s.authSvc == nil {
		return nil, fmt.Errorf("auth service not configured")
	}

	name, _ := params["name"].(string)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	// Get permissions
	var permissions []string
	if perms, ok := params["permissions"].([]interface{}); ok {
		for _, p := range perms {
			if ps, ok := p.(string); ok {
				permissions = append(permissions, ps)
			}
		}
	}
	if len(permissions) == 0 {
		permissions = []string{"*"}
	}

	// For now, we'll use a default user ID (in production, get from session)
	// This is a placeholder - real implementation would use authenticated user
	apiKey, key, err := s.authSvc.CreateAPIKey(ctx, [16]byte{}, name, permissions, nil)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":          apiKey.ID.String(),
		"name":        apiKey.Name,
		"key":         key, // Only returned once!
		"key_prefix":  apiKey.KeyPrefix,
		"permissions": apiKey.Permissions,
		"created_at":  apiKey.CreatedAt.Format(time.RFC3339),
	}, nil
}

// handleAPIKeyList lists API keys.
func (s *Server) handleAPIKeyList(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if s.authSvc == nil {
		return map[string]interface{}{"keys": []interface{}{}}, nil
	}

	// For now, return empty list - real implementation would use authenticated user
	keys, err := s.authSvc.ListAPIKeys(ctx, [16]byte{})
	if err != nil {
		return nil, err
	}

	result := make([]interface{}, len(keys))
	for i, k := range keys {
		result[i] = s.apiKeyToMap(k)
	}
	return map[string]interface{}{"keys": result}, nil
}

// handleAPIKeyRevoke revokes an API key.
func (s *Server) handleAPIKeyRevoke(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if s.authSvc == nil {
		return nil, fmt.Errorf("auth service not configured")
	}

	idStr, _ := params["id"].(string)
	if idStr == "" {
		return nil, fmt.Errorf("id is required")
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", err)
	}

	if err := s.authSvc.RevokeAPIKey(ctx, id); err != nil {
		return nil, err
	}

	return map[string]interface{}{"status": "revoked", "id": idStr}, nil
}

// handleAuditList lists audit logs.
func (s *Server) handleAuditList(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if s.authSvc == nil {
		return map[string]interface{}{"logs": []interface{}{}}, nil
	}

	filter := ports.AuditLogFilter{
		Limit: 50,
	}

	if limit, ok := params["limit"].(float64); ok && limit > 0 {
		filter.Limit = int(limit)
	}
	if action, ok := params["action"].(string); ok && action != "" {
		filter.Action = action
	}

	logs, err := s.authSvc.GetAuditLogs(ctx, filter)
	if err != nil {
		return nil, err
	}

	result := make([]interface{}, len(logs))
	for i, l := range logs {
		result[i] = s.auditLogToMap(l)
	}
	return map[string]interface{}{"logs": result}, nil
}

// userToMap converts a user to a map for JSON serialization.
func (s *Server) userToMap(u *domain.User) map[string]interface{} {
	m := map[string]interface{}{
		"id":            u.ID.String(),
		"username":      u.Username,
		"email":         u.Email,
		"role":          string(u.Role),
		"status":        string(u.Status),
		"display_name":  u.DisplayName,
		"failed_logins": u.FailedLogins,
		"created_at":    u.CreatedAt.Format(time.RFC3339),
		"updated_at":    u.UpdatedAt.Format(time.RFC3339),
	}
	if u.LastLoginAt != nil {
		m["last_login_at"] = u.LastLoginAt.Format(time.RFC3339)
	}
	if u.LockedUntil != nil {
		m["locked_until"] = u.LockedUntil.Format(time.RFC3339)
	}
	return m
}

// apiKeyToMap converts an API key to a map for JSON serialization.
func (s *Server) apiKeyToMap(k *domain.APIKey) map[string]interface{} {
	m := map[string]interface{}{
		"id":          k.ID.String(),
		"user_id":     k.UserID.String(),
		"name":        k.Name,
		"key_prefix":  k.KeyPrefix,
		"permissions": k.Permissions,
		"created_at":  k.CreatedAt.Format(time.RFC3339),
	}
	if k.ExpiresAt != nil {
		m["expires_at"] = k.ExpiresAt.Format(time.RFC3339)
	}
	if k.LastUsedAt != nil {
		m["last_used_at"] = k.LastUsedAt.Format(time.RFC3339)
	}
	if k.RevokedAt != nil {
		m["revoked_at"] = k.RevokedAt.Format(time.RFC3339)
	}
	return m
}

// auditLogToMap converts an audit log to a map for JSON serialization.
func (s *Server) auditLogToMap(l *domain.AuditLog) map[string]interface{} {
	m := map[string]interface{}{
		"id":          l.ID.String(),
		"action":      l.Action,
		"resource":    l.Resource,
		"resource_id": l.ResourceID,
		"success":     l.Success,
		"timestamp":   l.Timestamp.Format(time.RFC3339),
	}
	if l.UserID != nil {
		m["user_id"] = l.UserID.String()
	}
	if l.Error != "" {
		m["error"] = l.Error
	}
	if len(l.Details) > 0 {
		m["details"] = l.Details
	}
	if l.IPAddress != "" {
		m["ip_address"] = l.IPAddress
	}
	return m
}