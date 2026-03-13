package agent

import (
	"context"
	"icooclaw/pkg/bus"
	"icooclaw/pkg/hooks"
	"log/slog"
	"sync/atomic"
)

type Manager interface {
	// 启动智能体循环
	Start() error
	// 停止智能体循环
	Stop() error
}

// AgentManager 智能体管理器
type AgentManager struct {
	// 上下文
	ctx context.Context
	// 是否正在运行
	running atomic.Bool
	// 消息总线
	bus *bus.MessageBus
	// 日志记录器
	logger *slog.Logger
	// hook 钩子接口
	agentHooks hooks.AgentHooks
	// 提供程序钩子接口
	provider hooks.ProviderHooks
	// ReAct钩子接口
	react hooks.ReActHooks
}

// NewAgentManager 创建智能体管理器
func NewAgentManager(
	ctx context.Context,
	bus *bus.MessageBus,
	logger *slog.Logger,
	agent hooks.AgentHooks,
	provider hooks.ProviderHooks,
	react hooks.ReActHooks,
) *AgentManager {
	return &AgentManager{ctx: ctx, bus: bus, logger: logger, agentHooks: agent, provider: provider, react: react}
}

// Start 启动智能体循环
func (m *AgentManager) Start() error {
	if m.running.Load() == true {
		return nil
	}

	go m.start()
	m.running.Store(true)
	return nil
}

// IsRunning 是否正在运行
func (m *AgentManager) IsRunning() bool {
	return m.running.Load()
}

// start 启动智能体循环
func (m *AgentManager) start() error {
	// 监听消息总线
	for m.running.Load() {
		select {
		case <-m.ctx.Done():
			m.logger.With("name", "【智能体】").Info("代理循环已停止", "reason", m.ctx.Err())
			return m.ctx.Err()
		case msg := <-m.bus.Inbound():
			// 处理消息
			err := m.RunAgent(msg)
			if err != nil {
				m.logger.With("name", "【智能体】").Error("处理消息失败", "reason", err)
				continue
			}

		}
	}

	return nil
}

// Stop 停止智能体循环
func (m *AgentManager) Stop() error {
	if m.running.Load() == false {
		return nil
	}
	m.running.Store(false)
	return nil
}

func (m *AgentManager) RunAgent(msg bus.InboundMessage) error {
	// 触发 OnMessageReceived 钩子事件
	if m.agentHooks != nil {
		err := m.agentHooks.OnMessageReceived(m.ctx, msg.Channel, msg.SessionID, msg.Text)
		if err != nil {
			m.logger.With("name", "【智能体】").Error("处理消息失败", "reason", err)
			return err
		}
	}

	// 调用 agent
	return nil
}
