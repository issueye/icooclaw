package ws

import (
	"time"
)

// MessageType WebSocket 消息类型
type MessageType string

const (
	// 客户端 -> 服务端
	MessageTypeCreateSession MessageType = "create_session" // 创建会话
	MessageTypeChat          MessageType = "chat"           // 聊天消息
	MessageTypePing          MessageType = "ping"           // 心跳

	// 服务端 -> 客户端
	MessageTypeSessionCreated MessageType = "session_created" // 会话创建成功
	MessageTypeChunk          MessageType = "chunk"           // 流式响应块
	MessageTypeThinking       MessageType = "thinking"        // 思考内容
	MessageTypeToolCall       MessageType = "tool_call"       // 工具调用
	MessageTypeToolResult     MessageType = "tool_result"     // 工具结果
	MessageTypeEnd            MessageType = "end"             // 响应结束
	MessageTypeError          MessageType = "error"           // 错误
	MessageTypePong           MessageType = "pong"            // 心跳响应
	MessageTypeQueueStatus    MessageType = "queue_status"    // 队列状态
)

// Message WebSocket 消息结构
type Message struct {
	Type      MessageType `json:"type"`
	SessionID string      `json:"session_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	Error     *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo 错误信息
type ErrorInfo struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// CreateSessionRequest 创建会话请求
type CreateSessionRequest struct {
	Channel string `json:"channel,omitempty"` // 渠道 (默认为 "websocket")
	UserID  string `json:"user_id,omitempty"` // 用户ID
}

// CreateSessionResponse 创建会话响应
type CreateSessionResponse struct {
	SessionID string `json:"session_id"`
	Channel   string `json:"channel"`
	ChatID    string `json:"chat_id"`
	UserID    string `json:"user_id"`
}

// ChatRequest 聊天请求
type ChatRequest struct {
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
}

// ChunkData 流式响应块数据
type ChunkData struct {
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
}

// ThinkingData 思考内容数据
type ThinkingData struct {
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
}

// ToolCallData 工具调用数据
type ToolCallData struct {
	SessionID  string `json:"session_id"`
	ToolCallID string `json:"tool_call_id"`
	ToolName   string `json:"tool_name"`
	Arguments  string `json:"arguments"`
}

// ToolResultData 工具结果数据
type ToolResultData struct {
	SessionID  string `json:"session_id"`
	ToolCallID string `json:"tool_call_id"`
	Result     string `json:"result"`
}

// EndData 结束数据
type EndData struct {
	SessionID string `json:"session_id"`
}

// ErrorData 错误数据
type ErrorData struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

// QueueStatusData 队列状态数据
type QueueStatusData struct {
	ActiveCount   int `json:"active_count"`
	WaitingCount  int `json:"waiting_count"`
	MaxConcurrent int `json:"max_concurrent"`
	Position      int `json:"position,omitempty"` // 当前会话在队列中的位置，0表示正在处理
}

// NewMessage 创建新消息
func NewMessage(msgType MessageType, data interface{}) *Message {
	return &Message{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// NewErrorMessage 创建错误消息
func NewErrorMessage(sessionID string, code int, message string) *Message {
	return &Message{
		Type:      MessageTypeError,
		SessionID: sessionID,
		Timestamp: time.Now(),
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	}
}
