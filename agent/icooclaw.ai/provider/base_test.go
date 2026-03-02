package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMessage tests for Message struct
func TestMessage_Structure(t *testing.T) {
	msg := Message{
		Role:             "user",
		Content:          "Hello",
		ReasoningContent: "thinking",
		ToolCalls:        nil,
		ToolCallID:       "",
		Name:             "",
	}

	assert.Equal(t, "user", msg.Role)
	assert.Equal(t, "Hello", msg.Content)
	assert.Equal(t, "thinking", msg.ReasoningContent)
}

func TestMessage_WithToolCalls(t *testing.T) {
	toolCall := ToolCall{
		ID:   "call_123",
		Type: "function",
		Function: ToolCallFunction{
			Name:      "test_func",
			Arguments: `{"key":"value"}`,
		},
	}

	msg := Message{
		Role:      "assistant",
		Content:   "",
		ToolCalls: []ToolCall{toolCall},
	}

	assert.Len(t, msg.ToolCalls, 1)
	assert.Equal(t, "call_123", msg.ToolCalls[0].ID)
}

func TestMessage_ToolRole(t *testing.T) {
	msg := Message{
		Role:       "tool",
		Content:    "result content",
		ToolCallID: "call_123",
		Name:       "test_func",
	}

	assert.Equal(t, "tool", msg.Role)
	assert.Equal(t, "call_123", msg.ToolCallID)
	assert.Equal(t, "test_func", msg.Name)
}

// TestToolCall tests for ToolCall struct
func TestToolCall_Structure(t *testing.T) {
	call := ToolCall{
		ID:   "call_123",
		Type: "function",
		Function: ToolCallFunction{
			Name:      "test_func",
			Arguments: `{"arg":123}`,
		},
	}

	assert.Equal(t, "call_123", call.ID)
	assert.Equal(t, "function", call.Type)
	assert.Equal(t, "test_func", call.Function.Name)
}

// TestToolDefinition tests for ToolDefinition struct
func TestToolDefinition_Structure(t *testing.T) {
	def := ToolDefinition{
		Type: "function",
		Function: FunctionDefinition{
			Name:        "my_tool",
			Description: "A test tool",
			Parameters: map[string]interface{}{
				"type": "object",
			},
		},
	}

	assert.Equal(t, "function", def.Type)
	assert.Equal(t, "my_tool", def.Function.Name)
	assert.Equal(t, "A test tool", def.Function.Description)
}

// TestUsage tests for Usage struct
func TestUsage_Structure(t *testing.T) {
	usage := Usage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}

	assert.Equal(t, 100, usage.PromptTokens)
	assert.Equal(t, 50, usage.CompletionTokens)
	assert.Equal(t, 150, usage.TotalTokens)
}

// TestChatRequest tests for ChatRequest struct
func TestChatRequest_Structure(t *testing.T) {
	req := ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		Tools: []ToolDefinition{
			{Type: "function", Function: FunctionDefinition{Name: "tool1"}},
		},
		Model:       "gpt-4",
		MaxTokens:   1000,
		Temperature: 0.7,
		TopP:        0.9,
		Stream:      false,
		Stop:        []string{"STOP"},
	}

	assert.Len(t, req.Messages, 1)
	assert.Len(t, req.Tools, 1)
	assert.Equal(t, "gpt-4", req.Model)
	assert.Equal(t, 1000, req.MaxTokens)
	assert.Equal(t, 0.7, req.Temperature)
	assert.Equal(t, 0.9, req.TopP)
	assert.False(t, req.Stream)
	assert.Contains(t, req.Stop, "STOP")
}

// TestChatResponse tests for ChatResponse struct
func TestChatResponse_Structure(t *testing.T) {
	resp := ChatResponse{
		ID:               "resp_123",
		Object:           "chat.completion",
		Created:          1234567890,
		Model:            "gpt-4",
		Choices:          []Choice{},
		Usage:            Usage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150},
		ReasoningContent: "thinking",
	}

	assert.Equal(t, "resp_123", resp.ID)
	assert.Equal(t, "chat.completion", resp.Object)
	assert.Equal(t, "gpt-4", resp.Model)
	assert.Equal(t, 150, resp.Usage.TotalTokens)
}

// TestChoice tests for Choice struct
func TestChoice_Structure(t *testing.T) {
	choice := Choice{
		Index:        0,
		Message:      Message{Role: "assistant", Content: "Response"},
		FinishReason: "stop",
	}

	assert.Equal(t, 0, choice.Index)
	assert.Equal(t, "assistant", choice.Message.Role)
	assert.Equal(t, "Response", choice.Message.Content)
	assert.Equal(t, "stop", choice.FinishReason)
}

// TestBaseProvider tests for BaseProvider
func TestBaseProvider_NewBaseProvider(t *testing.T) {
	provider := NewBaseProvider(
		"test_provider",
		"api_key_123",
		"https://api.test.com",
		"gpt-4",
	)

	assert.Equal(t, "test_provider", provider.Name)
	assert.Equal(t, "api_key_123", provider.APIKey)
	assert.Equal(t, "https://api.test.com", provider.APIBase)
	assert.Equal(t, "gpt-4", provider.Model)
	assert.NotNil(t, provider.httpClient)
}

func TestBaseProvider_GetDefaultModel(t *testing.T) {
	provider := NewBaseProvider("test", "key", "url", "gpt-4")
	assert.Equal(t, "gpt-4", provider.GetDefaultModel())
}

func TestBaseProvider_GetName(t *testing.T) {
	provider := NewBaseProvider("my_provider", "key", "url", "model")
	assert.Equal(t, "my_provider", provider.GetName())
}

func TestBaseProvider_Chat_NotImplemented(t *testing.T) {
	provider := NewBaseProvider("test", "key", "url", "model")

	_, err := provider.Chat(nil, ChatRequest{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
}

// TestExtractThinkingContent tests for ExtractThinkingContent function
func TestExtractThinkingContent_NoThinking(t *testing.T) {
	content := "Hello, this is a normal response."
	cleanContent, thinking := ExtractThinkingContent(content, "")

	assert.Equal(t, "Hello, this is a normal response.", cleanContent)
	assert.Equal(t, "", thinking)
}

func TestExtractThinkingContent_WithReasoningContent(t *testing.T) {
	content := "This is the response."
	reasoningContent := "This is my reasoning."
	cleanContent, thinking := ExtractThinkingContent(content, reasoningContent)

	// 当有独立的 reasoning_content 字段时，应该直接使用，不修改 content
	assert.Equal(t, "This is the response.", cleanContent)
	assert.Equal(t, "This is my reasoning.", thinking)
}

func TestExtractThinkingContent_DeepSeekFormat(t *testing.T) {
	content := "<think>Let me think about this...</think>This is the answer."
	cleanContent, thinking := ExtractThinkingContent(content, "")

	assert.Equal(t, "This is the answer.", cleanContent)
	assert.Equal(t, "Let me think about this...", thinking)
}

func TestExtractThinkingContent_DeepSeekFormat_Multiline(t *testing.T) {
	content := `<think>First, I need to analyze the problem.
Then I will provide a solution.
</think>Here is my final answer.`
	cleanContent, thinking := ExtractThinkingContent(content, "")

	assert.Equal(t, "Here is my final answer.", cleanContent)
	assert.Contains(t, thinking, "First, I need to analyze the problem.")
	assert.Contains(t, thinking, "Then I will provide a solution.")
}

func TestExtractThinkingContent_KimiFormat(t *testing.T) {
	content := "<|start_header_id|>reasoning<|end_header_id|>Let me reason through this.<|start_header_id|>assistant<|end_header_id|>This is the answer."
	cleanContent, thinking := ExtractThinkingContent(content, "")

	assert.Equal(t, "This is the answer.", cleanContent)
	assert.Equal(t, "Let me reason through this.", thinking)
}

func TestExtractThinkingContent_OnlyThinking(t *testing.T) {
	content := "<think>Just thinking, no answer yet.</think>"
	cleanContent, thinking := ExtractThinkingContent(content, "")

	assert.Equal(t, "", cleanContent)
	assert.Equal(t, "Just thinking, no answer yet.", thinking)
}

func TestExtractThinkingContent_ThinkingWithWhitespace(t *testing.T) {
	content := "<think>  Thinking with spaces  </think>  Answer here.  "
	cleanContent, thinking := ExtractThinkingContent(content, "")

	assert.Equal(t, "Answer here.", cleanContent)
	assert.Equal(t, "Thinking with spaces", thinking)
}
