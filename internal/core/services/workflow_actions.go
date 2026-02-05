// Package services implements core business logic services.
package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/forge-platform/forge/internal/core/domain"
	"github.com/forge-platform/forge/internal/core/ports"
)

// ShellAction executes shell commands.
type ShellAction struct {
	allowedCommands []string // Optional whitelist of allowed commands
	workDir         string
}

// NewShellAction creates a new shell action handler.
func NewShellAction(workDir string) *ShellAction {
	return &ShellAction{workDir: workDir}
}

// Execute runs a shell command.
func (a *ShellAction) Execute(ctx context.Context, step *domain.WorkflowStep, input map[string]interface{}) (map[string]interface{}, error) {
	command, ok := step.Config["command"].(string)
	if !ok || command == "" {
		return nil, fmt.Errorf("shell action requires 'command' config")
	}

	// Substitute variables from input
	for k, v := range input {
		placeholder := fmt.Sprintf("${%s}", k)
		if str, ok := v.(string); ok {
			command = strings.ReplaceAll(command, placeholder, str)
		}
	}

	// Parse shell and args
	shell := "/bin/sh"
	if s, ok := step.Config["shell"].(string); ok {
		shell = s
	}

	cmd := exec.CommandContext(ctx, shell, "-c", command)
	if a.workDir != "" {
		cmd.Dir = a.workDir
	}
	if dir, ok := step.Config["workdir"].(string); ok {
		cmd.Dir = dir
	}

	// Set environment variables
	cmd.Env = os.Environ()
	if envMap, ok := step.Config["env"].(map[string]interface{}); ok {
		for k, v := range envMap {
			if str, ok := v.(string); ok {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, str))
			}
		}
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("failed to execute command: %w", err)
		}
	}

	return map[string]interface{}{
		"stdout":    stdout.String(),
		"stderr":    stderr.String(),
		"exit_code": exitCode,
		"success":   exitCode == 0,
	}, nil
}

// HTTPAction performs HTTP requests.
type HTTPAction struct {
	client  *http.Client
	baseURL string
}

// NewHTTPAction creates a new HTTP action handler.
func NewHTTPAction(timeout time.Duration) *HTTPAction {
	return &HTTPAction{
		client: &http.Client{Timeout: timeout},
	}
}

// Execute performs an HTTP request.
func (a *HTTPAction) Execute(ctx context.Context, step *domain.WorkflowStep, input map[string]interface{}) (map[string]interface{}, error) {
	method, _ := step.Config["method"].(string)
	if method == "" {
		method = "GET"
	}

	url, ok := step.Config["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("http action requires 'url' config")
	}

	// Substitute variables
	for k, v := range input {
		placeholder := fmt.Sprintf("${%s}", k)
		if str, ok := v.(string); ok {
			url = strings.ReplaceAll(url, placeholder, str)
		}
	}

	var bodyReader io.Reader
	if body, ok := step.Config["body"]; ok {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(method), url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	if headers, ok := step.Config["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			if str, ok := v.(string); ok {
				req.Header.Set(k, str)
			}
		}
	}

	if bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Try to parse JSON response
	var jsonBody interface{}
	if err := json.Unmarshal(respBody, &jsonBody); err == nil {
		return map[string]interface{}{
			"status":      resp.StatusCode,
			"status_text": resp.Status,
			"body":        jsonBody,
			"headers":     headerToMap(resp.Header),
			"success":     resp.StatusCode >= 200 && resp.StatusCode < 300,
		}, nil
	}

	return map[string]interface{}{
		"status":      resp.StatusCode,
		"status_text": resp.Status,
		"body":        string(respBody),
		"headers":     headerToMap(resp.Header),
		"success":     resp.StatusCode >= 200 && resp.StatusCode < 300,
	}, nil
}

func headerToMap(h http.Header) map[string]string {
	result := make(map[string]string)
	for k, v := range h {
		if len(v) > 0 {
			result[k] = v[0]
		}
	}
	return result
}

// MetricAction queries metrics from the repository.
type MetricAction struct {
	metricRepo ports.MetricRepository
}

// NewMetricAction creates a new metric action handler.
func NewMetricAction(repo ports.MetricRepository) *MetricAction {
	return &MetricAction{metricRepo: repo}
}

// Execute queries metrics based on configuration.
func (a *MetricAction) Execute(ctx context.Context, step *domain.WorkflowStep, input map[string]interface{}) (map[string]interface{}, error) {
	if a.metricRepo == nil {
		return nil, fmt.Errorf("metric repository not configured")
	}

	name, ok := step.Config["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("metric action requires 'name' config")
	}

	// Build query
	query := ports.MetricQuery{
		Name:    name,
		EndTime: time.Now(),
	}

	// Parse duration for lookback
	if duration, ok := step.Config["duration"].(string); ok {
		d, err := time.ParseDuration(duration)
		if err == nil {
			query.StartTime = query.EndTime.Add(-d)
		}
	}
	if query.StartTime.IsZero() {
		query.StartTime = query.EndTime.Add(-1 * time.Hour)
	}

	// Parse aggregation
	if agg, ok := step.Config["aggregation"].(string); ok {
		query.Aggregation = ports.AggregationType(agg)
	}

	// Parse step/bucket size
	if stepStr, ok := step.Config["step"].(string); ok {
		d, err := time.ParseDuration(stepStr)
		if err == nil {
			query.Step = d
		}
	}

	// Parse tags filter
	if tags, ok := step.Config["tags"].(map[string]interface{}); ok {
		query.Tags = make(map[string]string)
		for k, v := range tags {
			if str, ok := v.(string); ok {
				query.Tags[k] = str
			}
		}
	}

	// Execute query
	series, err := a.metricRepo.QueryMultiple(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query metrics: %w", err)
	}

	// Convert to output format
	results := make([]map[string]interface{}, len(series))
	for i, s := range series {
		points := make([]map[string]interface{}, len(s.Points))
		for j, p := range s.Points {
			points[j] = map[string]interface{}{
				"timestamp": p.Timestamp,
				"value":     p.Value,
			}
		}
		results[i] = map[string]interface{}{
			"name":   s.Name,
			"tags":   s.Tags,
			"points": points,
		}
	}

	return map[string]interface{}{
		"series": results,
		"count":  len(series),
	}, nil
}

// AIAction executes AI-powered analysis.
type AIAction struct {
	aiProvider ports.AIProvider
}

// NewAIAction creates a new AI action handler.
func NewAIAction(provider ports.AIProvider) *AIAction {
	return &AIAction{aiProvider: provider}
}

// Execute runs an AI prompt or analysis.
func (a *AIAction) Execute(ctx context.Context, step *domain.WorkflowStep, input map[string]interface{}) (map[string]interface{}, error) {
	if a.aiProvider == nil {
		return nil, fmt.Errorf("AI provider not configured")
	}

	prompt, ok := step.Config["prompt"].(string)
	if !ok || prompt == "" {
		return nil, fmt.Errorf("ai action requires 'prompt' config")
	}

	// Substitute variables from input
	for k, v := range input {
		placeholder := fmt.Sprintf("${%s}", k)
		if str, ok := v.(string); ok {
			prompt = strings.ReplaceAll(prompt, placeholder, str)
		} else {
			// Try JSON encoding for complex values
			if jsonBytes, err := json.Marshal(v); err == nil {
				prompt = strings.ReplaceAll(prompt, placeholder, string(jsonBytes))
			}
		}
	}

	// Get model from config or use default
	model := "llama3.2"
	if m, ok := step.Config["model"].(string); ok {
		model = m
	}

	// Get system prompt if provided
	systemPrompt := "You are a helpful AI assistant for infrastructure and DevOps tasks."
	if sys, ok := step.Config["system"].(string); ok {
		systemPrompt = sys
	}

	// Create conversation for this request
	conv := domain.NewConversation(model, systemPrompt)
	conv.AddMessage(domain.RoleUser, prompt)

	// Generate response using Chat
	response, err := a.aiProvider.Chat(ctx, conv)
	if err != nil {
		return nil, fmt.Errorf("AI generation failed: %w", err)
	}

	return map[string]interface{}{
		"response": response.Content,
		"model":    model,
		"prompt":   prompt,
	}, nil
}

// TaskAction creates or updates tasks.
type TaskAction struct {
	taskRepo ports.TaskRepository
}

// NewTaskAction creates a new task action handler.
func NewTaskAction(repo ports.TaskRepository) *TaskAction {
	return &TaskAction{taskRepo: repo}
}

// Execute creates or updates a task.
func (a *TaskAction) Execute(ctx context.Context, step *domain.WorkflowStep, input map[string]interface{}) (map[string]interface{}, error) {
	if a.taskRepo == nil {
		return nil, fmt.Errorf("task repository not configured")
	}

	action, _ := step.Config["action"].(string)
	if action == "" {
		action = "create"
	}

	switch action {
	case "create":
		title, _ := step.Config["title"].(string)
		if title == "" {
			return nil, fmt.Errorf("task action requires 'title' config")
		}

		taskType := domain.TaskTypeMaintenance
		if t, ok := step.Config["type"].(string); ok {
			taskType = domain.TaskType(t)
		}

		payload := map[string]interface{}{
			"title":       title,
			"description": step.Config["description"],
		}

		task := domain.NewTask(taskType, payload)
		if err := a.taskRepo.Create(ctx, task); err != nil {
			return nil, fmt.Errorf("failed to create task: %w", err)
		}

		return map[string]interface{}{
			"task_id": task.ID.String(),
			"type":    string(task.Type),
			"status":  string(task.Status),
		}, nil

	default:
		return nil, fmt.Errorf("unsupported task action: %s", action)
	}
}

// PluginAction invokes plugin functionality.
type PluginAction struct {
	pluginRunner func(ctx context.Context, pluginName string, input map[string]interface{}) (map[string]interface{}, error)
}

// NewPluginAction creates a new plugin action handler.
func NewPluginAction(runner func(ctx context.Context, pluginName string, input map[string]interface{}) (map[string]interface{}, error)) *PluginAction {
	return &PluginAction{pluginRunner: runner}
}

// Execute runs a plugin.
func (a *PluginAction) Execute(ctx context.Context, step *domain.WorkflowStep, input map[string]interface{}) (map[string]interface{}, error) {
	if a.pluginRunner == nil {
		return nil, fmt.Errorf("plugin runner not configured")
	}

	pluginName, ok := step.Config["plugin"].(string)
	if !ok || pluginName == "" {
		return nil, fmt.Errorf("plugin action requires 'plugin' config")
	}

	// Merge step config into input
	pluginInput := make(map[string]interface{})
	for k, v := range input {
		pluginInput[k] = v
	}
	for k, v := range step.Config {
		if k != "plugin" {
			pluginInput[k] = v
		}
	}

	return a.pluginRunner(ctx, pluginName, pluginInput)
}

