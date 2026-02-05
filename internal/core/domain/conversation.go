package domain

import (
	"time"

	"github.com/google/uuid"
)

// MessageRole represents the role of a message sender.
type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleSystem    MessageRole = "system"
	RoleTool      MessageRole = "tool"
)

// Message represents a single message in a conversation.
type Message struct {
	ID        uuid.UUID   `json:"id"`
	Role      MessageRole `json:"role"`
	Content   string      `json:"content"`
	ToolCalls []ToolCall  `json:"tool_calls,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// ToolCall represents a function call made by the AI.
type ToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
	Result    string                 `json:"result,omitempty"`
}

// Conversation represents a chat session with the AI.
type Conversation struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Messages  []Message `json:"messages"`
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewConversation creates a new conversation with a system prompt.
func NewConversation(model, systemPrompt string) *Conversation {
	now := time.Now()
	conv := &Conversation{
		ID:        uuid.Must(uuid.NewV7()),
		Title:     "New Conversation",
		Messages:  []Message{},
		Model:     model,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if systemPrompt != "" {
		conv.AddMessage(RoleSystem, systemPrompt)
	}

	return conv
}

// AddMessage adds a new message to the conversation.
func (c *Conversation) AddMessage(role MessageRole, content string) *Message {
	msg := Message{
		ID:        uuid.Must(uuid.NewV7()),
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}
	c.Messages = append(c.Messages, msg)
	c.UpdatedAt = time.Now()
	return &msg
}

// AddToolCall adds a tool call to the last assistant message.
func (c *Conversation) AddToolCall(toolCall ToolCall) {
	if len(c.Messages) == 0 {
		return
	}
	lastIdx := len(c.Messages) - 1
	if c.Messages[lastIdx].Role == RoleAssistant {
		c.Messages[lastIdx].ToolCalls = append(c.Messages[lastIdx].ToolCalls, toolCall)
		c.UpdatedAt = time.Now()
	}
}

// GetLastMessage returns the last message in the conversation.
func (c *Conversation) GetLastMessage() *Message {
	if len(c.Messages) == 0 {
		return nil
	}
	return &c.Messages[len(c.Messages)-1]
}

// GetContextWindow returns the last n messages for context.
func (c *Conversation) GetContextWindow(n int) []Message {
	if len(c.Messages) <= n {
		return c.Messages
	}
	return c.Messages[len(c.Messages)-n:]
}

// GenerateTitle generates a title from the first user message.
func (c *Conversation) GenerateTitle() {
	for _, msg := range c.Messages {
		if msg.Role == RoleUser {
			title := msg.Content
			if len(title) > 50 {
				title = title[:47] + "..."
			}
			c.Title = title
			return
		}
	}
}

