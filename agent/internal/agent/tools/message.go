package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/icooclaw/icooclaw/internal/channel"
)

// MessageConfig 消息工具配置
type MessageConfig struct {
	ChannelManager *channel.Manager
}

// MessageTool 跨通道消息工具
type MessageTool struct {
	baseTool *BaseTool
	config   *MessageConfig
}

// NewMessageTool 创建消息工具
func NewMessageTool(config *MessageConfig) *MessageTool {
	tool := NewBaseTool(
		"message",
		"向其他通道发送消息。支持向配置的通道发送消息。",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"channel": map[string]interface{}{
					"type":        "string",
					"description": "目标通道名称（如 websocket, webhook, telegram 等）",
				},
				"chat_id": map[string]interface{}{
					"type":        "string",
					"description": "目标聊天ID/房间ID",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "消息内容",
				},
				"parse_mode": map[string]interface{}{
					"type":        "string",
					"description": "解析模式（markdown, html, text），默认 text",
					"default":     "text",
				},
			},
			"required": []string{"channel", "content"},
		},
		nil,
	)

	return &MessageTool{
		baseTool: tool,
		config:   config,
	}
}

// Name 获取名称
func (t *MessageTool) Name() string {
	return t.baseTool.Name()
}

// Description 获取描述
func (t *MessageTool) Description() string {
	return t.baseTool.Description()
}

// Parameters 获取参数定义
func (t *MessageTool) Parameters() map[string]interface{} {
	return t.baseTool.Parameters()
}

// Execute 发送消息
func (t *MessageTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	if t.config.ChannelManager == nil {
		return "", fmt.Errorf("channel manager not configured")
	}

	channelName, ok := params["channel"].(string)
	if !ok || channelName == "" {
		return "", fmt.Errorf("invalid or missing channel")
	}

	chatID := ""
	if c, ok := params["chat_id"].(string); ok {
		chatID = c
	}

	content, ok := params["content"].(string)
	if !ok || content == "" {
		return "", fmt.Errorf("invalid or missing content")
	}

	parseMode := "text"
	if p, ok := params["parse_mode"].(string); ok {
		parseMode = p
	}

	// 获取通道
	ch, err := t.config.ChannelManager.Get(channelName)
	if err != nil {
		return "", fmt.Errorf("channel not found: %s", channelName)
	}

	// 检查通道是否运行
	if !ch.IsRunning() {
		return "", fmt.Errorf("channel is not running: %s", channelName)
	}

	// 构建消息
	msg := channel.OutboundMessage{
		Channel:   channelName,
		ChatID:    chatID,
		Content:   content,
		ParseMode: parseMode,
	}

	// 发送消息
	if err := ch.Send(ctx, msg); err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}

	// 构建结果
	result := map[string]interface{}{
		"channel":    channelName,
		"chat_id":    chatID,
		"status":     "sent",
		"content":    content,
		"parse_mode": parseMode,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// ToDefinition 转换为工具定义
func (t *MessageTool) ToDefinition() ToolDefinition {
	return t.baseTool.ToDefinition()
}
