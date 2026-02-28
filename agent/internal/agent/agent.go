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
	"github.com/icooclaw/icooclaw/internal/skill"
	"github.com/icooclaw/icooclaw/internal/storage"
)

// SessionMetadata 会话元数据
type SessionMetadata struct {
	RolePrompt string `json:"role_prompt"` // 用户设定的角色提示词
}

// Agent Agent核心结构
type Agent struct {
	name      string
	provider  provider.Provider
	tools     *tools.Registry
	storage   *storage.Storage
	memory    *MemoryStore
	skills    *skill.Loader
	config    config.AgentSettings
	bus       *bus.MessageBus
	logger    *slog.Logger
	workspace string
}

// NewAgent 创建Agent实例
func NewAgent(
	name string,
	provider provider.Provider,
	storage *storage.Storage,
	config config.AgentSettings,
	logger *slog.Logger,
	workspace string,
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
		skills:    skill.NewLoader(storage, logger),
		config:    config,
		logger:    logger,
		workspace: workspace,
	}
}

// Workspace 获取 workspace 路径
func (a *Agent) Workspace() string {
	return a.workspace
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
	onChunk := func(chunk, thinking string) {
		if a.bus != nil {
			// 发送内容块
			if chunk != "" {
				a.bus.PublishOutbound(ctx, bus.OutboundMessage{
					Type:      bus.MessageTypeChunk,
					Channel:   msg.Channel,
					ChatID:    msg.ChatID,
					Content:   chunk,
					Timestamp: time.Now(),
					Metadata:  map[string]interface{}{"client_id": clientID},
				})
			}
			// 发送思考内容更新
			if thinking != "" {
				a.bus.PublishOutbound(ctx, bus.OutboundMessage{
					Type:      bus.MessageTypeThinking,
					Channel:   msg.Channel,
					ChatID:    msg.ChatID,
					Thinking:  thinking,
					Timestamp: time.Now(),
					Metadata:  map[string]interface{}{"client_id": clientID},
				})
			}
		}
	}

	// 创建 ReAct 配置和 Hooks
	config := NewReActConfig()
	config.Provider = a.Provider()
	config.Tools = a.Tools()
	config.Session = session
	config.Logger = a.logger
	config.Hooks = &loopHooks{agent: a, onChunk: onChunk, chatID: msg.ChatID, clientID: clientID, session: session}

	reactAgent := NewReActAgent(config)
	response, reasoningContent, toolCalls, err := reactAgent.Run(ctx, messages, systemPrompt)
	if err != nil {
		a.logger.Error("Agent loop failed", "error", err)
		if a.bus != nil {
			a.bus.PublishOutbound(ctx, bus.OutboundMessage{
				Type:      bus.MessageTypeError,
				Channel:   msg.Channel,
				ChatID:    msg.ChatID,
				Content:   fmt.Sprintf("处理消息时出错: %s", err.Error()),
				Timestamp: time.Now(),
				Metadata:  map[string]interface{}{"client_id": clientID},
			})
		}
		return
	}

	// 发送流式结束
	if a.bus != nil {
		a.bus.PublishOutbound(ctx, bus.OutboundMessage{
			Type:      bus.MessageTypeEnd,
			Channel:   msg.Channel,
			ChatID:    msg.ChatID,
			Timestamp: time.Now(),
			Metadata:  map[string]interface{}{"client_id": clientID},
		})
	}

	// 保存助手消息
	toolCallsJSON, _ := json.Marshal(toolCalls)
	_, err = a.storage.AddMessage(session.ID, "assistant", response, string(toolCallsJSON), "", "", reasoningContent)
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
	config := NewReActConfig()
	config.Provider = a.Provider()
	config.Tools = a.Tools()
	config.Session = session
	config.Logger = a.logger
	config.Hooks = &DefaultHooks{}

	reactAgent := NewReActAgent(config)
	response, reasoningContent, toolCalls, err := reactAgent.Run(ctx, messages, systemPrompt)
	if err != nil {
		return "", fmt.Errorf("agent loop failed: %w", err)
	}

	// 保存助手消息
	toolCallsJSON, _ := json.Marshal(toolCalls)
	_, err = a.storage.AddMessage(session.ID, "assistant", response, string(toolCallsJSON), "", "", reasoningContent)
	if err != nil {
		a.logger.Warn("Failed to save assistant message", "error", err)
	}

	return response, nil
}

// SetSessionRolePrompt 设置会话的角色提示词
func (a *Agent) SetSessionRolePrompt(sessionID uint, rolePrompt string) error {
	// 获取当前会话的元数据
	session, err := a.storage.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	var metadata SessionMetadata
	if session.Metadata != "" {
		if err := json.Unmarshal([]byte(session.Metadata), &metadata); err != nil {
			// 如果解析失败，创建新的
			metadata = SessionMetadata{}
		}
	}

	// 更新角色提示词
	metadata.RolePrompt = rolePrompt

	// 序列化和保存
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	return a.storage.UpdateSessionMetadata(sessionID, string(metadataJSON))
}

// GetSessionRolePrompt 获取会话的角色提示词
func (a *Agent) GetSessionRolePrompt(sessionID uint) (string, error) {
	session, err := a.storage.GetSession(sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to get session: %w", err)
	}

	if session.Metadata == "" {
		return "", nil
	}

	var metadata SessionMetadata
	if err := json.Unmarshal([]byte(session.Metadata), &metadata); err != nil {
		return "", fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return metadata.RolePrompt, nil
}
