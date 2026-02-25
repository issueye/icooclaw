package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/icooclaw/icooclaw/internal/provider"
	"github.com/icooclaw/icooclaw/internal/storage"
)

// ContextBuilder 上下文构建器
type ContextBuilder struct {
	agent   *Agent
	session *storage.Session
	logger  *slog.Logger
}

// NewContextBuilder 创建上下文构建器
func NewContextBuilder(agent *Agent, session *storage.Session) *ContextBuilder {
	return &ContextBuilder{
		agent:   agent,
		session: session,
		logger:  agent.logger,
	}
}

// Build 构建上下文
func (cb *ContextBuilder) Build(ctx context.Context) ([]provider.Message, string, error) {
	// 1. 获取系统提示词
	systemPrompt := cb.buildSystemPrompt()

	// 记录系统提示词
	cb.logger.Debug("=== System Prompt ===", "content", systemPrompt)

	// 2. 获取历史消息
	messages, err := cb.buildMessages()
	if err != nil {
		return nil, "", fmt.Errorf("failed to build messages: %w", err)
	}

	// 记录历史消息
	messagesJSON, _ := json.MarshalIndent(messages, "", "  ")
	cb.logger.Debug("=== Context Messages ===", "count", len(messages), "messages", string(messagesJSON))

	return messages, systemPrompt, nil
}

// buildSystemPrompt 构建系统提示词
func (cb *ContextBuilder) buildSystemPrompt() string {
	var parts []string

	// 添加默认系统提示词
	parts = append(parts, cb.agent.GetSystemPrompt())

	// 添加技能提示词
	skills := cb.agent.skills.GetLoaded()
	for _, skill := range skills {
		parts = append(parts, "", fmt.Sprintf("## Skill: %s", skill.Name))
		parts = append(parts, skill.Content)
	}

	// 添加记忆
	memories, err := cb.agent.memory.GetAll()
	if err == nil && len(memories) > 0 {
		parts = append(parts, "", "## Long-term Memory")
		for _, mem := range memories {
			parts = append(parts, mem.Content)
		}
	}

	return strings.Join(parts, "\n")
}

// buildMessages 构建消息列表
func (cb *ContextBuilder) buildMessages() ([]provider.Message, error) {
	// 获取会话消息
	messages, err := cb.session.GetMessages(cb.agent.Config().MemoryWindow)
	if err != nil {
		return nil, err
	}

	// 转换为provider.Message
	result := make([]provider.Message, len(messages))
	for i, msg := range messages {
		result[i] = provider.Message{
			Role:             msg.Role,
			Content:          msg.Content,
			ReasoningContent: msg.ReasoningContent,
		}

		// 处理tool_calls
		if msg.ToolCalls != "" {
			var calls []provider.ToolCall
			if err := json.Unmarshal([]byte(msg.ToolCalls), &calls); err == nil {
				result[i].ToolCalls = calls
			}
		}

		// 处理tool角色
		if msg.Role == "tool" {
			result[i].ToolCallID = msg.ToolCallID
			result[i].Name = msg.ToolName
		}
	}

	return result, nil
}

// AddContext 添加额外上下文
func (cb *ContextBuilder) AddContext(contextType, content string) {
	// 可以扩展为添加额外的上下文信息
	cb.logger.Debug("Added context", "type", contextType)
}
