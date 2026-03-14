package react

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/providers"
)

func TestStreamChunk_Types(t *testing.T) {
	chunks := []StreamChunk{
		{Content: "test content"},
		{Reasoning: "thinking..."},
		{ToolName: "search"},
		{ToolResult: "result"},
		{Iteration: 1},
		{Done: true},
		{Error: errors.New("test error")},
	}

	for i, chunk := range chunks {
		if chunk.Content == "" && chunk.Reasoning == "" && chunk.ToolName == "" && chunk.ToolResult == "" && !chunk.Done && chunk.Error == nil && chunk.Iteration == 0 {
			t.Errorf("chunk %d should have some data", i)
		}
	}
}

func TestStreamCallback(t *testing.T) {
	var receivedChunks []StreamChunk
	callback := func(chunk StreamChunk) error {
		receivedChunks = append(receivedChunks, chunk)
		return nil
	}

	// Test callback invocation
	_ = callback(StreamChunk{Content: "hello"})
	_ = callback(StreamChunk{Done: true})

	if len(receivedChunks) != 2 {
		t.Errorf("expected 2 chunks, got %d", len(receivedChunks))
	}
	if receivedChunks[0].Content != "hello" {
		t.Errorf("expected 'hello', got '%s'", receivedChunks[0].Content)
	}
	if !receivedChunks[1].Done {
		t.Error("second chunk should be done")
	}
}

func TestIsStreamIndexID(t *testing.T) {
	tests := []struct {
		id       string
		expected bool
	}{
		{"stream_index:0", true},
		{"stream_index:1", true},
		{"stream_index:10", true},
		{"call_abc123", false},
		{"", false},
		{"short", false},
		{"regular_id", false},
	}

	for _, tt := range tests {
		result := isStreamIndexID(tt.id)
		if result != tt.expected {
			t.Errorf("isStreamIndexID(%q) = %v, expected %v", tt.id, result, tt.expected)
		}
	}
}

func TestMergeToolCalls(t *testing.T) {
	agent := &ReActAgent{logger: slog.Default()}

	// Test empty input
	result := agent.mergeToolCalls(nil)
	if result != nil {
		t.Error("expected nil for empty input")
	}

	// Test single tool call
	toolCalls := []providers.ToolCall{
		{
			ID:   "call_1",
			Type: "function",
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      "test_tool",
				Arguments: `{"arg": "value"}`,
			},
		},
	}

	result = agent.mergeToolCalls(toolCalls)
	if len(result) != 1 {
		t.Errorf("expected 1 tool call, got %d", len(result))
	}
	if result[0].Function.Name != "test_tool" {
		t.Errorf("expected 'test_tool', got '%s'", result[0].Function.Name)
	}
}

func TestMergeToolCalls_StreamIndex(t *testing.T) {
	agent := &ReActAgent{logger: slog.Default()}

	// Test merging stream index tool calls
	toolCalls := []providers.ToolCall{
		{
			ID:   "stream_index:0",
			Type: "function",
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      "test",
				Arguments: `{"a":`,
			},
		},
		{
			ID:   "stream_index:0",
			Type: "function",
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Arguments: ` "1"}`,
			},
		},
	}

	result := agent.mergeToolCalls(toolCalls)
	if len(result) != 1 {
		t.Errorf("expected 1 merged tool call, got %d", len(result))
	}
	if result[0].Function.Name != "test" {
		t.Errorf("expected 'test', got '%s'", result[0].Function.Name)
	}
	if result[0].Function.Arguments != `{"a": "1"}` {
		t.Errorf("expected merged arguments, got '%s'", result[0].Function.Arguments)
	}
}

func TestValidateToolCalls(t *testing.T) {
	agent := &ReActAgent{logger: slog.Default()}

	// Test empty input
	result := agent.validateToolCalls(nil)
	if len(result) != 0 {
		t.Error("expected empty slice for nil input")
	}

	// Test valid tool calls
	toolCalls := []providers.ToolCall{
		{
			ID:   "call_1",
			Type: "function",
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      "valid_tool",
				Arguments: `{"arg": "value"}`,
			},
		},
		{
			ID:   "call_2",
			Type: "function",
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      "", // Empty name - should be filtered
				Arguments: `{}`,
			},
		},
		{
			ID:   "call_3",
			Type: "function",
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      "no_args_tool",
				Arguments: "", // Empty args - should be set to "{}"
			},
		},
	}

	result = agent.validateToolCalls(toolCalls)
	if len(result) != 2 {
		t.Errorf("expected 2 valid tool calls, got %d", len(result))
	}

	// Check that empty args was set to "{}"
	for _, tc := range result {
		if tc.Function.Name == "no_args_tool" && tc.Function.Arguments != "{}" {
			t.Errorf("expected empty args to be '{}', got '%s'", tc.Function.Arguments)
		}
	}
}

func TestReActAgent_ChatStream_NoProvider(t *testing.T) {
	agent := NewReActAgent()

	var receivedChunks []StreamChunk
	callback := func(chunk StreamChunk) error {
		receivedChunks = append(receivedChunks, chunk)
		return nil
	}

	msg := bus.InboundMessage{
		Channel:   "test",
		SessionID: "session-1",
		Text:      "hello",
	}

	_, _, err := agent.ChatStream(context.Background(), msg, callback)
	if err == nil {
		t.Error("expected error without provider")
	}

	// Should receive error chunk
	if len(receivedChunks) == 0 {
		t.Error("expected error chunk")
	}
	if receivedChunks[0].Error == nil {
		t.Error("chunk should have error")
	}
}

// Mock provider for testing
type mockProvider struct {
	name          string
	defaultModel  string
	streamContent []string
	streamErr     error
}

func (m *mockProvider) GetName() string {
	return m.name
}

func (m *mockProvider) GetModel() string {
	return m.defaultModel
}

func (m *mockProvider) SetModel(model string) {
	m.defaultModel = model
}

func (m *mockProvider) Chat(ctx context.Context, req providers.ChatRequest) (*providers.ChatResponse, error) {
	return &providers.ChatResponse{
		Content: "mock response",
	}, nil
}

func (m *mockProvider) ChatStream(ctx context.Context, req providers.ChatRequest, callback providers.StreamCallback) error {
	if m.streamErr != nil {
		return m.streamErr
	}
	for _, content := range m.streamContent {
		callback(content, "", nil, false)
	}
	callback("", "", nil, true)
	return nil
}