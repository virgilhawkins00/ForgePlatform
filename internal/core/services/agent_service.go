// Package services implements core business logic services.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/forge-platform/forge/internal/core/domain"
	"github.com/forge-platform/forge/internal/core/ports"
)

// AgentService provides autonomous agent capabilities.
type AgentService struct {
	aiProvider   ports.AIProvider
	toolRegistry ports.AIToolRegistry
	logger       ports.Logger
	maxSteps     int
	confirmFn    func(action string) bool
}

// AgentConfig holds agent service configuration.
type AgentConfig struct {
	MaxSteps         int
	RequireConfirm   bool
	ConfirmFn        func(action string) bool
}

// DefaultAgentConfig returns default agent configuration.
func DefaultAgentConfig() AgentConfig {
	return AgentConfig{
		MaxSteps:       10,
		RequireConfirm: true,
		ConfirmFn:      nil, // Must be set if RequireConfirm is true
	}
}

// NewAgentService creates a new agent service.
func NewAgentService(
	aiProvider ports.AIProvider,
	toolRegistry ports.AIToolRegistry,
	logger ports.Logger,
	config AgentConfig,
) *AgentService {
	if config.MaxSteps == 0 {
		config.MaxSteps = 10
	}
	return &AgentService{
		aiProvider:   aiProvider,
		toolRegistry: toolRegistry,
		logger:       logger,
		maxSteps:     config.MaxSteps,
		confirmFn:    config.ConfirmFn,
	}
}

// AgentRequest represents a request to the agent.
type AgentRequest struct {
	Goal          string
	Context       string
	AllowedTools  []string
	RequireConfirm bool
}

// AgentResult represents the result of agent execution.
type AgentResult struct {
	Success     bool
	FinalAnswer string
	Steps       []AgentStep
	TotalTime   time.Duration
}

// AgentStep represents a single step in agent execution.
type AgentStep struct {
	StepNumber int
	Thought    string
	Action     string
	ActionArgs map[string]interface{}
	Result     string
	Error      string
	Duration   time.Duration
}

// Run executes the agent loop to accomplish a goal.
func (s *AgentService) Run(ctx context.Context, req AgentRequest) (*AgentResult, error) {
	if s.aiProvider == nil {
		return nil, fmt.Errorf("AI provider not configured")
	}

	startTime := time.Now()
	result := &AgentResult{
		Steps: make([]AgentStep, 0, s.maxSteps),
	}

	// Build system prompt with available tools
	systemPrompt := s.buildAgentPrompt(req)
	conv := domain.NewConversation(s.aiProvider.GetModel(), systemPrompt)
	conv.AddMessage(domain.RoleUser, req.Goal)

	for step := 1; step <= s.maxSteps; step++ {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		stepStart := time.Now()
		agentStep := AgentStep{StepNumber: step}

		// Get AI response
		response, err := s.aiProvider.Chat(ctx, conv)
		if err != nil {
			agentStep.Error = fmt.Sprintf("AI error: %v", err)
			result.Steps = append(result.Steps, agentStep)
			return result, err
		}

		// Parse response for thought, action, and answer
		thought, action, args, finalAnswer := s.parseResponse(response.Content)
		agentStep.Thought = thought
		agentStep.Action = action
		agentStep.ActionArgs = args

		// Check for final answer
		if finalAnswer != "" {
			agentStep.Duration = time.Since(stepStart)
			result.Steps = append(result.Steps, agentStep)
			result.Success = true
			result.FinalAnswer = finalAnswer
			result.TotalTime = time.Since(startTime)
			return result, nil
		}

		// Execute action if specified
		if action != "" {
			// Confirm action if required
			if req.RequireConfirm && s.confirmFn != nil {
				actionDesc := fmt.Sprintf("%s(%v)", action, args)
				if !s.confirmFn(actionDesc) {
					agentStep.Error = "Action cancelled by user"
					agentStep.Duration = time.Since(stepStart)
					result.Steps = append(result.Steps, agentStep)
					result.FinalAnswer = "Execution cancelled by user"
					result.TotalTime = time.Since(startTime)
					return result, nil
				}
			}

			// Execute the tool
			toolResult, err := s.toolRegistry.ExecuteTool(ctx, action, args)
			if err != nil {
				agentStep.Error = fmt.Sprintf("Tool error: %v", err)
				agentStep.Result = ""
			} else {
				agentStep.Result = toolResult
			}

			// Add result to conversation
			conv.AddMessage(domain.RoleTool, fmt.Sprintf("Result of %s: %s", action, agentStep.Result))
		}

		agentStep.Duration = time.Since(stepStart)
		result.Steps = append(result.Steps, agentStep)

		s.logger.Debug("Agent step completed", "step", step, "action", action, "duration", agentStep.Duration)
	}

	result.TotalTime = time.Since(startTime)
	result.FinalAnswer = "Max steps reached without finding answer"
	return result, nil
}

// buildAgentPrompt constructs the system prompt with available tools.
func (s *AgentService) buildAgentPrompt(req AgentRequest) string {
	var sb strings.Builder

	sb.WriteString(`You are an autonomous AI agent that can use tools to accomplish tasks.
You should think step by step and use the available tools to achieve the goal.

RESPONSE FORMAT:
Always respond in one of these two formats:

1. When you need to use a tool:
Thought: [your reasoning about what to do next]
Action: [tool_name]
Action Input: [JSON object with tool arguments]

2. When you have the final answer:
Thought: [your final reasoning]
Final Answer: [your complete answer to the user's goal]

AVAILABLE TOOLS:
`)

	// List available tools
	tools := s.toolRegistry.ListTools()
	filteredTools := tools

	// Filter tools if AllowedTools is specified
	if len(req.AllowedTools) > 0 {
		allowedSet := make(map[string]bool)
		for _, t := range req.AllowedTools {
			allowedSet[t] = true
		}
		filteredTools = nil
		for _, tool := range tools {
			if allowedSet[tool.Name] {
				filteredTools = append(filteredTools, tool)
			}
		}
	}

	for _, tool := range filteredTools {
		sb.WriteString(fmt.Sprintf("\n- %s: %s\n", tool.Name, tool.Description))
		if len(tool.Parameters) > 0 {
			sb.WriteString("  Parameters:\n")
			for name, param := range tool.Parameters {
				required := ""
				if param.Required {
					required = " (required)"
				}
				sb.WriteString(fmt.Sprintf("    - %s: %s%s\n", name, param.Description, required))
			}
		}
	}

	// Add context if provided
	if req.Context != "" {
		sb.WriteString("\nCONTEXT:\n")
		sb.WriteString(req.Context)
		sb.WriteString("\n")
	}

	sb.WriteString("\nIMPORTANT:\n")
	sb.WriteString("- Always use the exact tool names as listed above\n")
	sb.WriteString("- Action Input must be valid JSON\n")
	sb.WriteString("- Provide Final Answer only when you have completed the task\n")

	return sb.String()
}

// parseResponse extracts thought, action, args, and final answer from AI response.
func (s *AgentService) parseResponse(content string) (thought, action string, args map[string]interface{}, finalAnswer string) {
	lines := strings.Split(content, "\n")
	args = make(map[string]interface{})

	var thoughtBuilder strings.Builder
	var actionInputBuilder strings.Builder
	inActionInput := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for Thought
		if strings.HasPrefix(trimmed, "Thought:") {
			thoughtBuilder.WriteString(strings.TrimSpace(strings.TrimPrefix(trimmed, "Thought:")))
			continue
		}

		// Check for Action
		if strings.HasPrefix(trimmed, "Action:") {
			action = strings.TrimSpace(strings.TrimPrefix(trimmed, "Action:"))
			continue
		}

		// Check for Action Input
		if strings.HasPrefix(trimmed, "Action Input:") {
			inActionInput = true
			remainder := strings.TrimSpace(strings.TrimPrefix(trimmed, "Action Input:"))
			if remainder != "" {
				actionInputBuilder.WriteString(remainder)
			}
			continue
		}

		// Check for Final Answer
		if strings.HasPrefix(trimmed, "Final Answer:") {
			finalAnswer = strings.TrimSpace(strings.TrimPrefix(trimmed, "Final Answer:"))
			continue
		}

		// Continue building action input if we're in that section
		if inActionInput && trimmed != "" {
			actionInputBuilder.WriteString(line)
		}
	}

	thought = thoughtBuilder.String()

	// Parse action input as JSON
	actionInputStr := strings.TrimSpace(actionInputBuilder.String())
	if actionInputStr != "" {
		if err := json.Unmarshal([]byte(actionInputStr), &args); err != nil {
			// Try to extract simple key-value if JSON parsing fails
			s.logger.Debug("Failed to parse action input as JSON", "input", actionInputStr, "error", err)
		}
	}

	return thought, action, args, finalAnswer
}
