// Package ai implements the AI provider adapters.
package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/forge-platform/forge/internal/core/domain"
	"github.com/forge-platform/forge/internal/core/ports"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

// OllamaProvider implements the AI provider using Ollama.
type OllamaProvider struct {
	llm         *ollama.LLM
	model       string
	endpoint    string
	temperature float64
	tools       ports.AIToolRegistry
}

// OllamaConfig holds Ollama configuration.
type OllamaConfig struct {
	Model       string
	Endpoint    string
	Temperature float64
}

// DefaultOllamaConfig returns the default Ollama configuration.
func DefaultOllamaConfig() OllamaConfig {
	return OllamaConfig{
		Model:       "llama3.2",
		Endpoint:    "http://localhost:11434",
		Temperature: 0.7,
	}
}

// NewOllamaProvider creates a new Ollama AI provider.
func NewOllamaProvider(config OllamaConfig, tools ports.AIToolRegistry) (*OllamaProvider, error) {
	llm, err := ollama.New(
		ollama.WithModel(config.Model),
		ollama.WithServerURL(config.Endpoint),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama client: %w", err)
	}

	return &OllamaProvider{
		llm:         llm,
		model:       config.Model,
		endpoint:    config.Endpoint,
		temperature: config.Temperature,
		tools:       tools,
	}, nil
}

// Chat sends a conversation to the LLM and returns the response.
func (p *OllamaProvider) Chat(ctx context.Context, conv *domain.Conversation) (*domain.Message, error) {
	// Convert conversation to LangChain messages
	messages := p.convertMessages(conv.Messages)

	// Generate response
	response, err := p.llm.GenerateContent(ctx, messages,
		llms.WithTemperature(p.temperature),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no response generated")
	}

	content := response.Choices[0].Content

	// Add response to conversation
	msg := conv.AddMessage(domain.RoleAssistant, content)

	return msg, nil
}

// ChatStream sends a conversation and streams the response.
func (p *OllamaProvider) ChatStream(ctx context.Context, conv *domain.Conversation, callback func(chunk string)) (*domain.Message, error) {
	messages := p.convertMessages(conv.Messages)

	var fullResponse strings.Builder

	_, err := p.llm.GenerateContent(ctx, messages,
		llms.WithTemperature(p.temperature),
		llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			text := string(chunk)
			fullResponse.WriteString(text)
			callback(text)
			return nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to stream response: %w", err)
	}

	msg := conv.AddMessage(domain.RoleAssistant, fullResponse.String())
	return msg, nil
}

// ListModels returns available models.
func (p *OllamaProvider) ListModels(ctx context.Context) ([]string, error) {
	// Ollama doesn't have a direct API for this in langchaingo
	// Return common models as a fallback
	return []string{
		"llama3.2",
		"llama3.2:1b",
		"llama3.1",
		"mistral",
		"codellama",
		"gemma2",
		"phi3",
	}, nil
}

// GetModel returns the current model name.
func (p *OllamaProvider) GetModel() string {
	return p.model
}

// SetModel sets the model to use.
func (p *OllamaProvider) SetModel(model string) {
	p.model = model
	// Recreate the LLM with the new model
	llm, err := ollama.New(
		ollama.WithModel(model),
		ollama.WithServerURL(p.endpoint),
	)
	if err == nil {
		p.llm = llm
	}
}

// convertMessages converts domain messages to LangChain format.
func (p *OllamaProvider) convertMessages(messages []domain.Message) []llms.MessageContent {
	var result []llms.MessageContent

	for _, msg := range messages {
		var role llms.ChatMessageType
		switch msg.Role {
		case domain.RoleUser:
			role = llms.ChatMessageTypeHuman
		case domain.RoleAssistant:
			role = llms.ChatMessageTypeAI
		case domain.RoleSystem:
			role = llms.ChatMessageTypeSystem
		case domain.RoleTool:
			role = llms.ChatMessageTypeTool
		default:
			role = llms.ChatMessageTypeHuman
		}

		result = append(result, llms.MessageContent{
			Role: role,
			Parts: []llms.ContentPart{
				llms.TextContent{Text: msg.Content},
			},
		})
	}

	return result
}

var _ ports.AIProvider = (*OllamaProvider)(nil)

