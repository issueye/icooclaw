package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/icooclaw/icooclaw/internal/agent/tools"
	"github.com/icooclaw/icooclaw/internal/bus"
	"github.com/icooclaw/icooclaw/internal/config"
	"github.com/icooclaw/icooclaw/internal/provider"
	"github.com/icooclaw/icooclaw/internal/storage"
)

// Agent Agent核心结构
type Agent struct {
	name     string
	provider provider.Provider
	tools    *tools.Registry
	storage  *storage.Storage
	memory   *MemoryStore
	skills   *SkillsLoader
	config   config.AgentSettings
	bus      *bus.MessageBus
	logger   *slog.Logger
}

// NewAgent 创建Agent实例
func NewAgent(
	name string,
	provider provider.Provider,
	storage *storage.Storage,
	config config.AgentSettings,
	logger *slog.Logger,
) *Agent {
	if logger == nil {
		logger = slog.Default()
	}

	return &Agent{
		name:     name,
		provider: provider,
		tools:    tools.NewRegistry(),
		storage:  storage,
		memory: NewMemoryStoreWithConfig(storage, logger, MemoryConfig{
			ConsolidationThreshold: 50,
			SummaryEnabled:         true,
		}),
		skills: NewSkillsLoader(storage, logger),
		config: config,
		logger: logger,
	}
}

// Name 获取Agent名称
func (a *Agent) Name() string {
	return a.name
}

// Provider 获取Provider
func (a *Agent) Provider() provider.Provider {
	return a.provider
}

// Tools 获取工具注册表
func (a *Agent) Tools() *tools.Registry {
	return a.tools
}

// SetTools 设置工具注册表
func (a *Agent) SetTools(registry *tools.Registry) {
	a.tools = registry
}

// Storage 获取存储
func (a *Agent) Storage() *storage.Storage {
	return a.storage
}

// Config 获取配置
func (a *Agent) Config() config.AgentSettings {
	return a.config
}

// Logger 获取日志
func (a *Agent) Logger() *slog.Logger {
	return a.logger
}

// RegisterTool 注册工具
func (a *Agent) RegisterTool(tool tools.Tool) {
	a.tools.Register(tool)
	a.logger.Info("Registered tool", "name", tool.Name())
}

// Init 初始化Agent
func (a *Agent) Init(ctx context.Context) error {
	// 加载技能
	if err := a.skills.Load(ctx); err != nil {
		return fmt.Errorf("failed to load skills: %w", err)
	}

	// 注册技能工具
	for _, skill := range a.skills.GetLoaded() {
		a.logger.Debug("Loaded skill", "name", skill.Name)
	}

	// 加载记忆
	if err := a.memory.Load(ctx); err != nil {
		a.logger.Warn("Failed to load memory", "error", err)
	}

	a.logger.Info("Agent initialized", "name", a.name)
	return nil
}

// Run 运行Agent
func (a *Agent) Run(ctx context.Context, messageBus *bus.MessageBus) {
	a.bus = messageBus
	// 初始化
	if err := a.Init(ctx); err != nil {
		a.logger.Error("Failed to initialize agent", "error", err)
		return
	}

	a.logger.Info("Agent started", "name", a.name)

	for {
		select {
		case <-ctx.Done():
			a.logger.Info("Agent stopped", "name", a.name)
			return
		default:
			msg, err := messageBus.ConsumeInbound(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				a.logger.Error("Failed to consume message", "error", err)
				continue
			}

			// 处理消息
			go a.handleMessage(ctx, msg)
		}
	}
}

// handleMessage 处理消息
func (a *Agent) handleMessage(ctx context.Context, msg bus.InboundMessage) {
	a.logger.Info("Handling message",
		"channel", msg.Channel,
		"chat_id", msg.ChatID,
		"user_id", msg.UserID,
		"content", msg.Content)

	// 获取或创建会话
	session, err := a.storage.GetOrCreateSession(msg.Channel, msg.ChatID, msg.UserID)
	if err != nil {
		a.logger.Error("Failed to get or create session", "error", err)
		return
	}

	// 添加用户消息
	_, err = a.storage.AddMessage(session.ID, "user", msg.Content, "", "", "", "")
	if err != nil {
		a.logger.Error("Failed to add user message", "error", err)
		return
	}

	// 构建上下文
	contextBuilder := NewContextBuilder(a, session)
	messages, systemPrompt, err := contextBuilder.Build(ctx)
	if err != nil {
		a.logger.Error("Failed to build context", "error", err)
		return
	}

	// 获取 client_id (如果是 WebSocket)
	clientID, _ := msg.Metadata["client_id"].(string)

	// 运行 Agent Loop (带流式回调)
	onChunk := func(chunk string) {
		if a.bus != nil {
			a.bus.PublishOutbound(ctx, bus.OutboundMessage{
				Type:      "chunk",
				Channel:   msg.Channel,
				ChatID:    msg.ChatID,
				Content:   chunk,
				Timestamp: time.Now(),
				Metadata:  map[string]interface{}{"client_id": clientID},
			})
		}
	}

	loop := NewLoopWithStream(a, session, a.logger, onChunk)
	response, toolCalls, err := loop.Run(ctx, messages, systemPrompt)
	if err != nil {
		a.logger.Error("Agent loop failed", "error", err)
		return
	}

	// 发送流式结束
	if a.bus != nil {
		a.bus.PublishOutbound(ctx, bus.OutboundMessage{
			Type:      "chunk_end",
			Channel:   msg.Channel,
			ChatID:    msg.ChatID,
			Timestamp: time.Now(),
			Metadata:  map[string]interface{}{"client_id": clientID},
		})
	}

	// 保存助手消息
	toolCallsJSON, _ := json.Marshal(toolCalls)
	_, err = a.storage.AddMessage(session.ID, "assistant", response, string(toolCallsJSON), "", "", "")
	if err != nil {
		a.logger.Error("Failed to save assistant message", "error", err)
	}

	a.logger.Info("Message handled", "response_length", len(response))
}

// SetSystemPrompt 设置系统提示词
func (a *Agent) SetSystemPrompt(prompt string) {
	a.config.SystemPrompt = prompt
}

// GetSystemPrompt 获取系统提示词
func (a *Agent) GetSystemPrompt() string {
	return a.config.SystemPrompt
}

// ProcessMessage 处理单条消息（用于 CLI）
func (a *Agent) ProcessMessage(ctx context.Context, content string) (string, error) {
	// 创建虚拟会话
	session, err := a.storage.GetOrCreateSession("cli", "cli-session", "cli-user")
	if err != nil {
		return "", fmt.Errorf("failed to get or create session: %w", err)
	}

	// 添加用户消息
	_, err = a.storage.AddMessage(session.ID, "user", content, "", "", "", "")
	if err != nil {
		return "", fmt.Errorf("failed to add user message: %w", err)
	}

	// 构建上下文
	contextBuilder := NewContextBuilder(a, session)
	messages, systemPrompt, err := contextBuilder.Build(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to build context: %w", err)
	}

	// 运行 Agent Loop
	loop := NewLoop(a, session, a.logger)
	response, toolCalls, err := loop.Run(ctx, messages, systemPrompt)
	if err != nil {
		return "", fmt.Errorf("agent loop failed: %w", err)
	}

	// 保存助手消息
	toolCallsJSON, _ := json.Marshal(toolCalls)
	_, err = a.storage.AddMessage(session.ID, "assistant", response, string(toolCallsJSON), "", "", "")
	if err != nil {
		a.logger.Warn("Failed to save assistant message", "error", err)
	}

	return response, nil
}
