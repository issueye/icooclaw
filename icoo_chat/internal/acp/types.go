package acp

import "time"

// AgentProfile Agent 描述文件
type AgentProfile struct {
	PublisherInfo string      `json:"publisherInfo"`
	Name          string      `json:"name"`
	Description   string      `json:"description"`
	Version       string      `json:"version"`
	Capabilities  Capabilities `json:"capabilities"`
	Input         IOConfig    `json:"input"`
	Output        IOConfig    `json:"output"`
	Authorization Authorization `json:"authorization"`
}

// Capabilities Agent 能力
type Capabilities struct {
	Core    []string `json:"core"`
	Extended []string `json:"extended"`
}

// IOConfig 输入输出配置
type IOConfig struct {
	Types   []string `json:"types"`
	Formats []string `json:"formats"`
}

// Authorization 授权配置
type Authorization struct {
	Modes      []string `json:"modes"`
	Description string  `json:"description"`
}

// AgentInfo Agent 信息
type AgentInfo struct {
	AID      string       `json:"aid"`
	Profile  *AgentProfile `json:"profile,omitempty"`
	Status   AgentStatus  `json:"status"`
	Endpoint string       `json:"endpoint"`
	ConnectedAt time.Time  `json:"connected_at"`
}

// AgentStatus Agent 连接状态
type AgentStatus string

const (
	AgentStatusDisconnected AgentStatus = "disconnected"
	AgentStatusConnecting   AgentStatus = "connecting"
	AgentStatusConnected    AgentStatus = "connected"
	AgentStatusError        AgentStatus = "error"
)

// Session 会话
type Session struct {
	ID        string    `json:"session_id"`
	AgentAID  string    `json:"agent_aid"`
	CreatedAt time.Time `json:"created_at"`
	Channel   string    `json:"channel"`
}

// ACPMessage ACP 协议消息
type ACPMessage struct {
	Type    string      `json:"type"`
	SessionID string    `json:"session_id,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ACPError   `json:"error,omitempty"`
}

// ACPError ACP 错误
type ACPError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// MessageContent 消息内容
type MessageContent struct {
	Type     string `json:"type"` // text, tool_call, tool_result, thinking, chunk, end
	Content  string `json:"content,omitempty"`
	ToolName string `json:"tool_name,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
	Arguments string `json:"arguments,omitempty"`
	Result   string `json:"result,omitempty"`
}

// APConfig 接入点配置
type APConfig struct {
	Endpoint  string `json:"endpoint"`  // AP 接入点地址
	APIKey    string `json:"api_key"`   // API 密钥
	AID       string `json:"aid"`       // 本地 Agent AID
}

// ConnectRequest 连接请求
type ConnectRequest struct {
	AID    string `json:"aid"`
	APIKey string `json:"api_key,omitempty"`
}

// SessionRequest 会话请求
type SessionRequest struct {
	SessionID string `json:"session_id"`
	Channel   string `json:"channel"`
	UserID    string `json:"user_id,omitempty"`
}

// ChatRequest 聊天请求
type ChatRequest struct {
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
}

// EventHandler 事件处理器
type EventHandler func(msg *ACPMessage)

// AgentEventHandler Agent 事件处理器
type AgentEventHandler func(aid string, msg *ACPMessage)