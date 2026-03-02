package hooks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"icooclaw.ai/provider"
	"icooclaw.ai/tools"
)

func TestDefaultHooks_OnLLMRequest(t *testing.T) {
	h := &DefaultHooks{}
	req := &provider.ChatRequest{
		Model:    "gpt-4",
		Messages: []provider.Message{{Role: "user", Content: "hello"}},
	}

	err := h.OnLLMRequest(context.Background(), req, 1)
	assert.NoError(t, err)
}

func TestDefaultHooks_OnLLMChunk(t *testing.T) {
	h := &DefaultHooks{}

	err := h.OnLLMChunk(context.Background(), "test content", "thinking")
	assert.NoError(t, err)
}

func TestDefaultHooks_OnLLMResponse(t *testing.T) {
	h := &DefaultHooks{}
	toolCalls := []provider.ToolCall{
		{ID: "call_1", Type: "function", Function: provider.ToolCallFunction{Name: "test_tool", Arguments: "{}"}},
	}

	err := h.OnLLMResponse(context.Background(), "response", "reasoning", toolCalls, 1)
	assert.NoError(t, err)
}

func TestDefaultHooks_OnToolCall(t *testing.T) {
	h := &DefaultHooks{}

	err := h.OnToolCall(context.Background(), "call_123", "file_read", `{"path": "/test"}`)
	assert.NoError(t, err)
}

func TestDefaultHooks_OnToolResult(t *testing.T) {
	h := &DefaultHooks{}
	result := tools.ToolResult{
		Content: "file content",
		Error:   nil,
	}

	err := h.OnToolResult(context.Background(), "call_123", "file_read", result)
	assert.NoError(t, err)
}

func TestDefaultHooks_OnToolResultWithError(t *testing.T) {
	h := &DefaultHooks{}
	result := tools.ToolResult{
		Content: "",
		Error:   assert.AnError,
	}

	err := h.OnToolResult(context.Background(), "call_123", "file_read", result)
	assert.NoError(t, err)
}

func TestDefaultHooks_OnIterationStart(t *testing.T) {
	h := &DefaultHooks{}
	messages := []provider.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	}

	err := h.OnIterationStart(context.Background(), 1, messages)
	assert.NoError(t, err)
}

func TestDefaultHooks_OnIterationEnd(t *testing.T) {
	h := &DefaultHooks{}

	err := h.OnIterationEnd(context.Background(), 1, true)
	assert.NoError(t, err)

	err = h.OnIterationEnd(context.Background(), 2, false)
	assert.NoError(t, err)
}

func TestDefaultHooks_OnError(t *testing.T) {
	h := &DefaultHooks{}

	err := h.OnError(context.Background(), assert.AnError)
	assert.NoError(t, err)
}

func TestDefaultHooks_OnComplete(t *testing.T) {
	h := &DefaultHooks{}
	toolCalls := []provider.ToolCall{
		{ID: "call_1", Type: "function", Function: provider.ToolCallFunction{Name: "test_tool", Arguments: "{}"}},
	}

	err := h.OnComplete(context.Background(), "final response", "reasoning", toolCalls, 3)
	assert.NoError(t, err)
}

func TestDefaultHooks_AllMethodsReturnNil(t *testing.T) {
	h := &DefaultHooks{}
	ctx := context.Background()

	// 确保所有方法都返回 nil
	assert.Nil(t, h.OnLLMRequest(ctx, nil, 0))
	assert.Nil(t, h.OnLLMChunk(ctx, "", ""))
	assert.Nil(t, h.OnLLMResponse(ctx, "", "", nil, 0))
	assert.Nil(t, h.OnToolCall(ctx, "", "", ""))
	assert.Nil(t, h.OnToolResult(ctx, "", "", tools.ToolResult{}))
	assert.Nil(t, h.OnIterationStart(ctx, 0, nil))
	assert.Nil(t, h.OnIterationEnd(ctx, 0, false))
	assert.Nil(t, h.OnError(ctx, nil))
	assert.Nil(t, h.OnComplete(ctx, "", "", nil, 0))
}

// MockHooks 用于测试的模拟实现
type MockHooks struct {
	Called map[string]int
	Last   map[string]interface{}
}

func NewMockHooks() *MockHooks {
	return &MockHooks{
		Called: make(map[string]int),
		Last:   make(map[string]interface{}),
	}
}

func (m *MockHooks) record(name string, data interface{}) error {
	m.Called[name]++
	m.Last[name] = data
	return nil
}

func (m *MockHooks) OnLLMRequest(ctx context.Context, req *provider.ChatRequest, iteration int) error {
	return m.record("OnLLMRequest", map[string]interface{}{"request": req, "iteration": iteration})
}

func (m *MockHooks) OnLLMChunk(ctx context.Context, content, thinking string) error {
	return m.record("OnLLMChunk", map[string]string{"content": content, "thinking": thinking})
}

func (m *MockHooks) OnLLMResponse(ctx context.Context, content, reasoningContent string, toolCalls []provider.ToolCall, iteration int) error {
	return m.record("OnLLMResponse", map[string]interface{}{"content": content, "toolCalls": len(toolCalls), "iteration": iteration})
}

func (m *MockHooks) OnToolCall(ctx context.Context, toolCallID string, toolName string, arguments string) error {
	return m.record("OnToolCall", map[string]string{"id": toolCallID, "name": toolName, "args": arguments})
}

func (m *MockHooks) OnToolResult(ctx context.Context, toolCallID string, toolName string, result tools.ToolResult) error {
	return m.record("OnToolResult", map[string]interface{}{"id": toolCallID, "name": toolName, "result": result})
}

func (m *MockHooks) OnIterationStart(ctx context.Context, iteration int, messages []provider.Message) error {
	return m.record("OnIterationStart", map[string]interface{}{"iteration": iteration, "messages": len(messages)})
}

func (m *MockHooks) OnIterationEnd(ctx context.Context, iteration int, hasToolCalls bool) error {
	return m.record("OnIterationEnd", map[string]interface{}{"iteration": iteration, "hasToolCalls": hasToolCalls})
}

func (m *MockHooks) OnError(ctx context.Context, err error) error {
	return m.record("OnError", map[string]interface{}{"error": err})
}

func (m *MockHooks) OnComplete(ctx context.Context, content, reasoningContent string, toolCalls []provider.ToolCall, iterations int) error {
	return m.record("OnComplete", map[string]interface{}{"content": content, "iterations": iterations})
}

func TestMockHooks(t *testing.T) {
	m := NewMockHooks()
	ctx := context.Background()

	// 测试 MockHooks 记录调用
	m.OnLLMRequest(ctx, &provider.ChatRequest{Model: "gpt-4"}, 1)
	assert.Equal(t, 1, m.Called["OnLLMRequest"])

	m.OnLLMChunk(ctx, "hello", "thinking")
	assert.Equal(t, 1, m.Called["OnLLMChunk"])

	m.OnToolCall(ctx, "call_1", "test", "{}")
	assert.Equal(t, 1, m.Called["OnToolCall"])

	// 验证记录的数据
	data := m.Last["OnToolCall"].(map[string]string)
	assert.Equal(t, "call_1", data["id"])
	assert.Equal(t, "test", data["name"])
}

func TestReActHooksInterface(t *testing.T) {
	// 确保 DefaultHooks 和 MockHooks 都实现了 ReActHooks 接口
	var _ ReActHooks = (*DefaultHooks)(nil)
	var _ ReActHooks = (*MockHooks)(nil)
}