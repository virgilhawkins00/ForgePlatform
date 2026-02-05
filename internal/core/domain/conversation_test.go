package domain

import (
	"testing"
)

func TestNewConversation(t *testing.T) {
	conv := NewConversation("gpt-4", "You are a helpful assistant")

	if conv.ID.String() == "" {
		t.Error("ID is empty")
	}
	if conv.Model != "gpt-4" {
		t.Errorf("Model = %v, want gpt-4", conv.Model)
	}
	if conv.Title != "New Conversation" {
		t.Errorf("Title = %v, want 'New Conversation'", conv.Title)
	}
	if len(conv.Messages) != 1 {
		t.Errorf("Messages count = %d, want 1 (system prompt)", len(conv.Messages))
	}
	if conv.Messages[0].Role != RoleSystem {
		t.Errorf("First message role = %v, want system", conv.Messages[0].Role)
	}
	if conv.Messages[0].Content != "You are a helpful assistant" {
		t.Error("System prompt not set correctly")
	}
}

func TestNewConversation_NoSystemPrompt(t *testing.T) {
	conv := NewConversation("gpt-4", "")

	if len(conv.Messages) != 0 {
		t.Errorf("Messages count = %d, want 0 for empty system prompt", len(conv.Messages))
	}
}

func TestConversation_AddMessage(t *testing.T) {
	conv := NewConversation("gpt-4", "")

	msg := conv.AddMessage(RoleUser, "Hello!")

	if msg == nil {
		t.Fatal("AddMessage returned nil")
	}
	if len(conv.Messages) != 1 {
		t.Errorf("Messages count = %d, want 1", len(conv.Messages))
	}
	if msg.Role != RoleUser {
		t.Errorf("Role = %v, want user", msg.Role)
	}
	if msg.Content != "Hello!" {
		t.Errorf("Content = %v, want 'Hello!'", msg.Content)
	}
	if msg.ID.String() == "" {
		t.Error("Message ID is empty")
	}
}

func TestConversation_AddToolCall(t *testing.T) {
	conv := NewConversation("gpt-4", "")
	conv.AddMessage(RoleAssistant, "Let me help you with that")

	toolCall := ToolCall{
		ID:        "call_123",
		Name:      "get_weather",
		Arguments: map[string]interface{}{"city": "London"},
	}
	conv.AddToolCall(toolCall)

	lastMsg := conv.GetLastMessage()
	if len(lastMsg.ToolCalls) != 1 {
		t.Errorf("ToolCalls count = %d, want 1", len(lastMsg.ToolCalls))
	}
	if lastMsg.ToolCalls[0].Name != "get_weather" {
		t.Errorf("ToolCall name = %v, want get_weather", lastMsg.ToolCalls[0].Name)
	}
}

func TestConversation_AddToolCall_NoMessages(t *testing.T) {
	conv := NewConversation("gpt-4", "")

	toolCall := ToolCall{ID: "call_123", Name: "test"}
	conv.AddToolCall(toolCall) // Should not panic

	if len(conv.Messages) != 0 {
		t.Error("AddToolCall should not add message when no messages exist")
	}
}

func TestConversation_AddToolCall_NotAssistant(t *testing.T) {
	conv := NewConversation("gpt-4", "")
	conv.AddMessage(RoleUser, "Hello")

	toolCall := ToolCall{ID: "call_123", Name: "test"}
	conv.AddToolCall(toolCall) // Should not add to user message

	lastMsg := conv.GetLastMessage()
	if len(lastMsg.ToolCalls) != 0 {
		t.Error("AddToolCall should not add tool call to non-assistant message")
	}
}

func TestConversation_GetLastMessage(t *testing.T) {
	conv := NewConversation("gpt-4", "")

	// Empty conversation
	if conv.GetLastMessage() != nil {
		t.Error("GetLastMessage should return nil for empty conversation")
	}

	conv.AddMessage(RoleUser, "First")
	conv.AddMessage(RoleAssistant, "Second")

	lastMsg := conv.GetLastMessage()
	if lastMsg == nil {
		t.Fatal("GetLastMessage returned nil")
	}
	if lastMsg.Content != "Second" {
		t.Errorf("Content = %v, want 'Second'", lastMsg.Content)
	}
}

func TestConversation_GetContextWindow(t *testing.T) {
	conv := NewConversation("gpt-4", "")
	conv.AddMessage(RoleUser, "Message 1")
	conv.AddMessage(RoleAssistant, "Message 2")
	conv.AddMessage(RoleUser, "Message 3")
	conv.AddMessage(RoleAssistant, "Message 4")

	// Request more than available
	window := conv.GetContextWindow(10)
	if len(window) != 4 {
		t.Errorf("GetContextWindow(10) count = %d, want 4", len(window))
	}

	// Request less than available
	window = conv.GetContextWindow(2)
	if len(window) != 2 {
		t.Errorf("GetContextWindow(2) count = %d, want 2", len(window))
	}
	if window[0].Content != "Message 3" {
		t.Errorf("First message in window = %v, want 'Message 3'", window[0].Content)
	}
}

func TestConversation_GenerateTitle(t *testing.T) {
	conv := NewConversation("gpt-4", "You are helpful")
	conv.AddMessage(RoleUser, "How do I sort a list in Python?")

	conv.GenerateTitle()

	if conv.Title != "How do I sort a list in Python?" {
		t.Errorf("Title = %v, want 'How do I sort a list in Python?'", conv.Title)
	}
}

func TestConversation_GenerateTitle_LongMessage(t *testing.T) {
	conv := NewConversation("gpt-4", "")
	conv.AddMessage(RoleUser, "This is a very long message that exceeds fifty characters and should be truncated")

	conv.GenerateTitle()

	if len(conv.Title) > 50 {
		t.Errorf("Title length = %d, should be <= 50", len(conv.Title))
	}
	if conv.Title != "This is a very long message that exceeds fifty ..." {
		t.Errorf("Title = %v, expected truncated version", conv.Title)
	}
}

func TestConversation_GenerateTitle_NoUserMessage(t *testing.T) {
	conv := NewConversation("gpt-4", "System prompt")

	originalTitle := conv.Title
	conv.GenerateTitle()

	// Title should remain unchanged if no user message
	if conv.Title != originalTitle {
		t.Errorf("Title changed when no user message exists")
	}
}

func TestMessageRoleConstants(t *testing.T) {
	if RoleUser != "user" {
		t.Errorf("RoleUser = %v, want user", RoleUser)
	}
	if RoleAssistant != "assistant" {
		t.Errorf("RoleAssistant = %v, want assistant", RoleAssistant)
	}
	if RoleSystem != "system" {
		t.Errorf("RoleSystem = %v, want system", RoleSystem)
	}
	if RoleTool != "tool" {
		t.Errorf("RoleTool = %v, want tool", RoleTool)
	}
}

