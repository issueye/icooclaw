package agent

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"icooclaw.ai/hooks"
	"icooclaw.ai/provider"
)

// mockReActHooks 简单的 mock hooks 用于测试流式输出
type mockReActHooks struct {
	hooks.DefaultHooks
	chunks          []string
	reasoningChunks []string
}

func (m *mockReActHooks) OnLLMChunk(ctx context.Context, content string, reasoning string) error {
	m.chunks = append(m.chunks, content)
	m.reasoningChunks = append(m.reasoningChunks, reasoning)
	return nil
}

func TestReActAgent_StreamCallback_DeepSeek(t *testing.T) {
	agent := &ReActAgent{}
	mockHooks := &mockReActHooks{}

	state := &streamCallbackState{
		content:          "",
		reasoningContent: "",
		hooks:            mockHooks,
		logger:           slog.Default(),
		toolCallsData:    make(map[int]*CallData),
	}

	cb := agent.createStreamCallback(state)

	// 测试包含完整 think 的 chunk
	err := cb(provider.StreamChunk{Content: "A <think>thought 1</think> B"})
	assert.NoError(t, err)
	assert.Equal(t, "A  B", state.content)
	assert.Equal(t, "thought 1", state.reasoningContent)
	assert.False(t, state.isThinking)

	// 测试流式的 think 标签截断 (模拟正常吐字，不截断标签本身)
	_ = cb(provider.StreamChunk{Content: "Start. "})
	_ = cb(provider.StreamChunk{Content: "<think>Deep "})
	_ = cb(provider.StreamChunk{Content: "thought "})
	_ = cb(provider.StreamChunk{Content: "</think> End."})

	assert.Equal(t, "A  BStart.  End.", state.content)
	assert.Equal(t, "thought 1Deep thought ", state.reasoningContent)

	// 重置 state
	state.content = ""
	state.reasoningContent = ""

	_ = cb(provider.StreamChunk{Content: "Hello "})
	_ = cb(provider.StreamChunk{Content: "<think>\nThinking process...\n"})
	_ = cb(provider.StreamChunk{Content: "More thinking..."})
	_ = cb(provider.StreamChunk{Content: "</think>\nWorld!"})

	assert.Equal(t, "Hello \nWorld!", state.content)
	assert.Equal(t, "\nThinking process...\nMore thinking...", state.reasoningContent)
	assert.False(t, state.isThinking)
}

func TestReActAgent_StreamCallback_Kimi(t *testing.T) {
	agent := &ReActAgent{}
	mockHooks := &mockReActHooks{}

	state := &streamCallbackState{
		content:          "",
		reasoningContent: "",
		hooks:            mockHooks,
		logger:           slog.Default(),
		toolCallsData:    make(map[int]*CallData),
	}

	cb := agent.createStreamCallback(state)

	_ = cb(provider.StreamChunk{Content: "Greeting. "})
	_ = cb(provider.StreamChunk{Content: "<|start_header_id|>reasoning<|end_header_id|>\nI am thinking.\n"})
	_ = cb(provider.StreamChunk{Content: "Still thinking."})
	_ = cb(provider.StreamChunk{Content: "<|start_header_id|>assistant<|end_header_id|>\nDone."})

	assert.Equal(t, "Greeting. \nDone.", state.content)
	assert.Equal(t, "\nI am thinking.\nStill thinking.", state.reasoningContent)
	assert.False(t, state.isKimiThinking)
}

func TestReActAgent_StreamCallback_DirectReasoning(t *testing.T) {
	agent := &ReActAgent{}
	mockHooks := &mockReActHooks{}

	state := &streamCallbackState{
		content:          "",
		reasoningContent: "",
		hooks:            mockHooks,
		logger:           slog.Default(),
		toolCallsData:    make(map[int]*CallData),
	}

	cb := agent.createStreamCallback(state)

	// O1 或其他直接支持 ReasoningContent 字段的形式
	_ = cb(provider.StreamChunk{ReasoningContent: "Direct reasoning 1"})
	_ = cb(provider.StreamChunk{ReasoningContent: " Direct reasoning 2"})
	_ = cb(provider.StreamChunk{Content: "Final Answer."})

	assert.Equal(t, "Final Answer.", state.content)
	assert.Equal(t, "Direct reasoning 1 Direct reasoning 2", state.reasoningContent)
}

func TestReActAgent_StreamCallback_ToolCalls(t *testing.T) {
	agent := &ReActAgent{}
	mockHooks := &mockReActHooks{}

	state := &streamCallbackState{
		content:          "",
		reasoningContent: "",
		hooks:            mockHooks,
		logger:           slog.Default(),
		toolCallsData:    make(map[int]*CallData),
	}

	cb := agent.createStreamCallback(state)

	_ = cb(provider.StreamChunk{
		ToolCalls: []provider.StreamToolCall{
			{Index: 0, ID: "call_1", Type: "function", Name: "get_weather", Arguments: "{\""},
		},
	})
	_ = cb(provider.StreamChunk{
		ToolCalls: []provider.StreamToolCall{
			{Index: 0, Arguments: "location\": \"Beijing\"}"},
		},
	})

	assert.Len(t, state.toolCallsData, 1)
	assert.Equal(t, "call_1", state.toolCallsData[0].id)
	assert.Equal(t, "get_weather", state.toolCallsData[0].name)
	assert.Equal(t, "{\"location\": \"Beijing\"}", state.toolCallsData[0].args.String())
}
