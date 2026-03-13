package agent

import (
	"context"
	"testing"
	"time"
)

func TestReActConfig_Default(t *testing.T) {
	config := DefaultReActConfig()

	if config.MaxIterations <= 0 {
		t.Error("MaxIterations should be positive")
	}
	if config.Timeout <= 0 {
		t.Error("Timeout should be positive")
	}
	if config.ThoughtTag == "" {
		t.Error("ThoughtTag should not be empty")
	}
	if config.ActionTag == "" {
		t.Error("ActionTag should not be empty")
	}
	if config.ObservationTag == "" {
		t.Error("ObservationTag should not be empty")
	}
	if config.AnswerTag == "" {
		t.Error("AnswerTag should not be empty")
	}
}

func TestReAct_New(t *testing.T) {
	unit := NewReAct()

	if unit == nil {
		t.Fatal("ReAct should not be nil")
	}
	if unit.tools == nil {
		t.Error("tools registry should be initialized")
	}
	if unit.config.MaxIterations != DefaultReActConfig().MaxIterations {
		t.Error("should use default config")
	}
}

func TestReAct_Options(t *testing.T) {
	unit := NewReAct(
		WithReActMaxIterations(5),
		WithReActTimeout(30*time.Second),
		WithReActSystemPrompt("custom prompt"),
	)

	if unit.config.MaxIterations != 5 {
		t.Errorf("expected MaxIterations 5, got %d", unit.config.MaxIterations)
	}
	if unit.config.Timeout != 30*time.Second {
		t.Errorf("expected Timeout 30s, got %v", unit.config.Timeout)
	}
	if unit.config.SystemPrompt != "custom prompt" {
		t.Errorf("expected custom prompt, got %s", unit.config.SystemPrompt)
	}
}

func TestReAct_Run_NoProvider(t *testing.T) {
	unit := NewReAct()

	result := unit.Run(context.Background(), "test query")

	if result.Success {
		t.Error("should fail without provider")
	}
	if result.Error == nil {
		t.Error("should have error")
	}
}

func TestReAct_RunStream_NoProvider(t *testing.T) {
	unit := NewReAct()

	var receivedChunks []ReActStreamChunk
	callback := func(chunk ReActStreamChunk) error {
		receivedChunks = append(receivedChunks, chunk)
		return nil
	}

	result := unit.RunStream(context.Background(), "test query", callback)

	if result.Success {
		t.Error("should fail without provider")
	}
	if result.Error == nil {
		t.Error("should have error")
	}
	// Should receive error chunk
	if len(receivedChunks) == 0 {
		t.Error("should receive error chunk")
	}
	if receivedChunks[0].Type != "error" {
		t.Errorf("expected error chunk, got %s", receivedChunks[0].Type)
	}
}

func TestReAct_parseResponse_FinalAnswer(t *testing.T) {
	unit := NewReAct()

	response := "Thought: I have the answer\nFinal Answer: The result is 42"

	step, finalAnswer := unit.parseResponse(response)

	if finalAnswer != "The result is 42" {
		t.Errorf("expected 'The result is 42', got '%s'", finalAnswer)
	}
	if step.Thought != "" {
		t.Error("thought should be empty when final answer is present")
	}
}

func TestReAct_parseResponse_WithAction(t *testing.T) {
	unit := NewReAct()

	response := `Thought: I need to search for information
Action: search
Action Input: {"query": "golang"}`

	step, finalAnswer := unit.parseResponse(response)

	if finalAnswer != "" {
		t.Error("finalAnswer should be empty")
	}
	if step.Thought == "" {
		t.Error("thought should not be empty")
	}
	if step.Action != "search" {
		t.Errorf("expected action 'search', got '%s'", step.Action)
	}
	if step.ActionInput == nil {
		t.Error("ActionInput should not be nil")
	}
	if step.ActionInput["query"] != "golang" {
		t.Errorf("expected query 'golang', got '%v'", step.ActionInput["query"])
	}
}

func TestReAct_parseResponse_WithSimpleInput(t *testing.T) {
	unit := NewReAct()

	response := `Thought: I need to check the time
Action: datetime
Action Input: timezone=Asia/Shanghai`

	step, _ := unit.parseResponse(response)

	if step.Action != "datetime" {
		t.Errorf("expected action 'datetime', got '%s'", step.Action)
	}
	// Non-JSON input should be wrapped
	if step.ActionInput == nil {
		t.Error("ActionInput should not be nil")
	}
}

func TestReAct_formatStep(t *testing.T) {
	unit := NewReAct()

	step := ReActStep{
		Thought: "I need to search",
		Action:  "search",
		ActionInput: map[string]any{
			"query": "test",
		},
	}

	formatted := unit.formatStep(step)

	if formatted == "" {
		t.Error("formatted step should not be empty")
	}
	if !contains(formatted, "Thought:") {
		t.Error("should contain Thought tag")
	}
	if !contains(formatted, "Action:") {
		t.Error("should contain Action tag")
	}
}

func TestReAct_buildMessages(t *testing.T) {
	unit := NewReAct()

	messages := unit.buildMessages("test query")

	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(messages))
	}
	if messages[0].Role != "system" {
		t.Error("first message should be system")
	}
	if messages[1].Role != "user" {
		t.Error("second message should be user")
	}
	if messages[1].Content != "test query" {
		t.Errorf("expected 'test query', got '%s'", messages[1].Content)
	}
}

func TestReActResult_JSON(t *testing.T) {
	result := &ReActResult{
		FinalAnswer: "The answer is 42",
		Steps: []ReActStep{
			{
				Thought: "First thought",
				Action:  "search",
				ActionInput: map[string]any{
					"query": "test",
				},
				Observation: "Search results",
			},
		},
		Iterations: 2,
		Success:    true,
		Duration:   time.Second,
	}

	// Test that result can be serialized
	// This is mainly to ensure the struct is properly defined
	if result.FinalAnswer != "The answer is 42" {
		t.Error("FinalAnswer mismatch")
	}
	if len(result.Steps) != 1 {
		t.Error("Steps count mismatch")
	}
	if !result.Success {
		t.Error("Success should be true")
	}
}

func TestReActStreamChunk_Types(t *testing.T) {
	// Test different chunk types
	chunks := []ReActStreamChunk{
		{Type: "thought", Thought: "thinking..."},
		{Type: "action", Action: "search"},
		{Type: "observation", Observation: "result"},
		{Type: "content", Content: "some content"},
		{Type: "done", Done: true},
		{Type: "error", Error: context.Canceled},
	}

	for _, chunk := range chunks {
		if chunk.Type == "" {
			t.Error("chunk type should not be empty")
		}
	}
}

func TestReActStreamCallback(t *testing.T) {
	var collectedChunks []ReActStreamChunk
	callback := func(chunk ReActStreamChunk) error {
		collectedChunks = append(collectedChunks, chunk)
		return nil
	}

	// Test that callback is called correctly
	_ = callback(ReActStreamChunk{Type: "content", Content: "test"})
	_ = callback(ReActStreamChunk{Type: "done", Done: true})

	if len(collectedChunks) != 2 {
		t.Errorf("expected 2 chunks, got %d", len(collectedChunks))
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}