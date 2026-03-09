package services

import (
	"context"
	"encoding/json"
	"fmt"
)

// App struct
type App struct {
	ctx        context.Context
	acpService *ACPService
}

// NewApp creates a new App application struct
func NewApp() *App {
	app := &App{}
	app.acpService = NewACPService(app)
	return app
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	// 启动 ACP 消息监听
	a.acpService.StartMessageListener(ctx)
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// ===== ACP 协议客户端接口 =====

// InitACP 初始化 ACP 客户端
func (a *App) InitACP(endpoint, apiKey, aid string) string {
	err := a.acpService.InitACP(endpoint, apiKey, aid)
	if err != nil {
		return fmt.Sprintf("初始化失败: %v", err)
	}
	return "OK"
}

// ConnectACP 连接到 AP 接入点
func (a *App) ConnectACP() string {
	err := a.acpService.Connect()
	if err != nil {
		return fmt.Sprintf("连接失败: %v", err)
	}
	return "OK"
}

// DisconnectACP 断开连接
func (a *App) DisconnectACP() string {
	err := a.acpService.Disconnect()
	if err != nil {
		return fmt.Sprintf("断开连接失败: %v", err)
	}
	return "OK"
}

// GetACPStatus 获取 ACP 连接状态
func (a *App) GetACPStatus() map[string]interface{} {
	connected := a.acpService.IsConnected()
	config := a.acpService.GetConfig()

	return map[string]interface{}{
		"connected": connected,
		"config":    config,
	}
}

// ConnectAgent 连接 Agent
func (a *App) ConnectAgent(aid string) string {
	agent, err := a.acpService.ConnectAgent(aid)
	if err != nil {
		return fmt.Sprintf("连接 Agent 失败: %v", err)
	}
	if agent == nil {
		return "Agent 不存在或无法连接"
	}
	data, _ := json.Marshal(agent)
	return string(data)
}

// DisconnectAgent 断开 Agent 连接
func (a *App) DisconnectAgent(aid string) string {
	err := a.acpService.DisconnectAgent(aid)
	if err != nil {
		return fmt.Sprintf("断开 Agent 失败: %v", err)
	}
	return "OK"
}

// GetAgentInfo 获取 Agent 信息
func (a *App) GetAgentInfo(aid string) string {
	agent, err := a.acpService.GetAgentInfo(aid)
	if err != nil {
		return fmt.Sprintf("获取 Agent 信息失败: %v", err)
	}
	if agent == nil {
		return "{}"
	}
	data, _ := json.Marshal(agent)
	return string(data)
}

// ListConnectedAgents 列出已连接 Agent
func (a *App) ListConnectedAgents() string {
	agents := a.acpService.ListConnectedAgents()
	if agents == nil {
		return "[]"
	}
	data, _ := json.Marshal(agents)
	return string(data)
}

// CreateACPSession 创建会话
func (a *App) CreateACPSession(agentAID string) string {
	sessionID, err := a.acpService.CreateSession(agentAID)
	if err != nil {
		return fmt.Sprintf("创建会话失败: %v", err)
	}
	return sessionID
}

// SendACPMessage 发送消息
func (a *App) SendACPMessage(sessionID, content string) string {
	err := a.acpService.SendMessage(sessionID, content)
	if err != nil {
		return fmt.Sprintf("发送消息失败: %v", err)
	}
	return "OK"
}

// CloseACPSession 关闭会话
func (a *App) CloseACPSession(sessionID string) string {
	err := a.acpService.CloseSession(sessionID)
	if err != nil {
		return fmt.Sprintf("关闭会话失败: %v", err)
	}
	return "OK"
}
