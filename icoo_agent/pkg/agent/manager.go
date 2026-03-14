package agent

import (
	"context"
	"icooclaw/pkg/agent/react"
	"icooclaw/pkg/bus"
	channelschannels "icooclaw/pkg/channels/consts"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/memory"
	"icooclaw/pkg/providers"
	"icooclaw/pkg/skill"
	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"
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
	// 内存加载器
	memory memory.Loader
	// 技能加载器
	skills skill.Loader
	// 工具注册器
	tools *tools.Registry
	// 日志记录器
	logger *slog.Logger
	// 钩子函数
	hooks react.ReactHooks
	// 提供商工厂
	providerFactory *providers.Factory
	// 存储加载器
	storage *storage.Storage
	// 智能体示例map
	agentsMap map[string]*react.ReActAgent
}

// NewAgentManager 创建智能体管理器
func NewAgentManager(
	ctx context.Context,
	logger *slog.Logger,
) *AgentManager {
	manager := AgentManager{
		ctx:     ctx,
		running: atomic.Bool{},
		logger:  logger,
	}

	manager.agentsMap = make(map[string]*react.ReActAgent)
	return &manager
}

func (m *AgentManager) WithProviderFactory(f *providers.Factory) *AgentManager {
	m.providerFactory = f
	return m
}

func (m *AgentManager) WithHooks(hooks react.ReactHooks) *AgentManager {
	m.hooks = hooks
	return m
}

func (m *AgentManager) WithBus(b *bus.MessageBus) *AgentManager {
	m.bus = b
	return m
}

func (m *AgentManager) WithMemory(mem memory.Loader) *AgentManager {
	m.memory = mem
	return m
}

func (m *AgentManager) WithTools(reg *tools.Registry) *AgentManager {
	m.tools = reg
	return m
}

func (m *AgentManager) WithSkills(skills skill.Loader) *AgentManager {
	m.skills = skills
	return m
}

func (m *AgentManager) WithStorage(s *storage.Storage) *AgentManager {
	m.storage = s
	return m
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
			switch msg.Channel {
			case channelschannels.WEBSOCKET:
				// 处理消息
				err := m.RunAgentStream(msg, m.callback(msg))
				if err != nil {
					m.logger.With("name", "【智能体】").Error("处理消息失败", "reason", err)
					continue
				}
			case channelschannels.FEISHU:
				// 处理消息
				finallyContent, err := m.RunAgent(msg)
				if err != nil {
					m.logger.With("name", "【智能体】").Error("处理消息失败", "reason", err)
					continue
				}

				// 发送消息到bus
				out := bus.OutboundMessage{
					Channel:   msg.Channel,
					SessionID: msg.SessionID,
					Text:      finallyContent,
				}
				m.bus.PublishOutbound(m.ctx, out)
			}
		}
	}

	return nil
}

func (m *AgentManager) callback(inbound bus.InboundMessage) react.StreamCallback {
	return func(chunk react.StreamChunk) error {
		// 发送消息到bus
		out := bus.OutboundMessage{
			Channel:   inbound.Channel,
			SessionID: inbound.SessionID,
			Text:      chunk.Content,
		}
		m.bus.PublishOutbound(m.ctx, out)
		return nil
	}
}

// Stop 停止智能体循环
func (m *AgentManager) Stop() error {
	if m.running.Load() == false {
		return nil
	}
	m.running.Store(false)
	return nil
}

func (m *AgentManager) RunAgent(msg bus.InboundMessage) (string, error) {
	// 生成智能体实例
	var (
		agent *react.ReActAgent
		err   error
	)
	agent, ok := m.agentsMap[msg.SessionID]
	if !ok {
		agent, err = react.NewReActAgent(
			m.ctx,
			m.hooks,
			react.WithBus(m.bus),
			react.WithMaxToolIterations(consts.DEFAULT_TOOL_ITERATIONS),
			react.WithMemory(m.memory),
			react.WithSkills(m.skills),
			react.WithTools(m.tools),
			react.WithProviderFactory(m.providerFactory),
			react.WithStorage(m.storage),
		)
	}

	m.agentsMap[msg.SessionID] = agent

	finallyContent, finallyIteration, err := agent.Chat(m.ctx, msg)
	if err != nil {
		m.logger.With("name", "【智能体】").Error("处理消息失败", "reason", err)
		return "", err
	}

	// 将消息发送到消息总线
	out := bus.OutboundMessage{
		Channel:   msg.Channel,
		SessionID: msg.SessionID,
		Text:      finallyContent,
		Metadata: map[string]any{
			"iteration": finallyIteration, // 迭代次数
		},
	}
	m.bus.PublishOutbound(m.ctx, out)

	// 调用 agent
	return finallyContent, nil
}

func (m *AgentManager) RunAgentStream(msg bus.InboundMessage, callback react.StreamCallback) error {
	// 生成智能体实例
	var (
		agent *react.ReActAgent
		err   error
	)
	agent, ok := m.agentsMap[msg.SessionID]
	if !ok {
		agent, err = react.NewReActAgent(
			m.ctx,
			m.hooks,
			react.WithBus(m.bus),
			react.WithMaxToolIterations(consts.DEFAULT_TOOL_ITERATIONS),
			react.WithMemory(m.memory),
			react.WithSkills(m.skills),
			react.WithTools(m.tools),
			react.WithProviderFactory(m.providerFactory),
			react.WithStorage(m.storage),
		)
	}

	m.agentsMap[msg.SessionID] = agent

	finallyContent, finallyIteration, err := agent.ChatStream(m.ctx, msg, callback)
	if err != nil {
		m.logger.With("name", "【智能体】").Error("处理消息失败", "reason", err)
		return err
	}

	// 将消息发送到消息总线
	out := bus.OutboundMessage{
		Channel:   msg.Channel,
		SessionID: msg.SessionID,
		Text:      finallyContent,
		Metadata: map[string]any{
			"iteration": finallyIteration, // 迭代次数
		},
	}
	m.bus.PublishOutbound(m.ctx, out)

	// 调用 agent
	return nil
}
