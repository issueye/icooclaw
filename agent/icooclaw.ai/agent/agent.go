package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"icooclaw.ai/hooks"
	"icooclaw.ai/provider"
	"icooclaw.ai/tools"
	icooclawbus "icooclaw.core/bus"
	"icooclaw.core/config"
	"icooclaw.core/consts"
	"icooclaw.core/storage"
)

// SessionMetadata 会话元数据
type SessionMetadata struct {
	RolePrompt string `json:"role_prompt"`
}

// Agent Agent 核心结构体
type Agent struct {
	name      string
	provider  provider.Provider
	tools     ToolRegistryInterface
	storage   StorageInterface
	memory    MemoryStoreInterface
	skills    SkillLoaderInterface
	config    config.AgentSettings
	bus       MessageBusInterface
	logger    *slog.Logger
	workspace string
}

// AgentOption Agent 选项函数
type AgentOption func(*Agent)

// WithTools 设置工具注册表
func WithTools(registry ToolRegistryInterface) AgentOption {
	return func(a *Agent) {
		a.tools = registry
	}
}

// WithMemoryStore 设置记忆存储
func WithMemoryStore(store MemoryStoreInterface) AgentOption {
	return func(a *Agent) {
		a.memory = store
	}
}

// WithSkillLoader 设置技能加载器
func WithSkillLoader(loader SkillLoaderInterface) AgentOption {
	return func(a *Agent) {
		a.skills = loader
	}
}

// WithMessageBus 设置消息总线
func WithMessageBus(bus MessageBusInterface) AgentOption {
	return func(a *Agent) {
		a.bus = bus
	}
}

// NewAgent 创建 Agent 实例（使用函数式选项模式）
func NewAgent(
	name string,
	provider provider.Provider,
	storageIntf StorageInterface,
	config config.AgentSettings,
	logger *slog.Logger,
	workspace string,
	opts ...AgentOption,
) *Agent {
	if logger == nil {
		logger = slog.Default()
	}

	agent := &Agent{
		name:      name,
		provider:  provider,
		tools:     tools.NewRegistry(), // 默认实现
		storage:   storageIntf,
		config:    config,
		logger:    logger,
		workspace: workspace,
	}

	// 应用选项
	for _, opt := range opts {
		opt(agent)
	}

	// 如果没有提供，使用默认实现（需要具体类型）
	// if agent.memory == nil {
	// 	if storageImpl, ok := storageIntf.(*storage.Storage); ok {
	// 		agent.memory = memory.NewMemoryStoreWithConfig(storageImpl, logger, memory.MemoryConfig{
	// 			ConsolidationThreshold: 50,
	// 			SummaryEnabled:         true,
	// 		})
	// 	} else {
	// 		logger.Warn("Storage is not *storage.Storage, skipping memory initialization")
	// 	}
	// }
	// if agent.skills == nil {
	// 	if storageImpl, ok := storageIntf.(*storage.Storage); ok {
	// 		agent.skills = skill.NewLoader(storageImpl, logger)
	// 	} else {
	// 		logger.Warn("Storage is not *storage.Storage, skipping skills initialization")
	// 	}
	// }

	return agent
}

func (a *Agent) Workspace() string {
	return a.workspace
}

func (a *Agent) Name() string {
	return a.name
}

func (a *Agent) Provider() provider.Provider {
	return a.provider
}

// SetProvider 设置 Provider（用于运行时切换模型）
func (a *Agent) SetProvider(p provider.Provider) {
	a.provider = p
	a.logger.Info("Provider switched", "name", p.GetName(), "model", p.GetDefaultModel())
}

func (a *Agent) Tools() ToolRegistryInterface {
	return a.tools
}

func (a *Agent) SetTools(registry ToolRegistryInterface) {
	a.tools = registry
}

func (a *Agent) Storage() StorageInterface {
	return a.storage
}

func (a *Agent) Config() config.AgentSettings {
	return a.config
}

func (a *Agent) Logger() *slog.Logger {
	return a.logger
}

func (a *Agent) Skills() SkillLoaderInterface {
	return a.skills
}

func (a *Agent) Memory() MemoryStoreInterface {
	return a.memory
}

func (a *Agent) RegisterTool(tool tools.Tool) {
	a.tools.Register(tool)
	a.logger.Info("Registered tool", "name", tool.Name())
}

func (a *Agent) Init(ctx context.Context) error {
	if err := a.skills.Load(ctx); err != nil {
		return fmt.Errorf("failed to load skills: %w", err)
	}

	for _, skill := range a.skills.GetLoaded() {
		a.logger.Debug("Loaded skill", "name", skill.Name)
	}

	if err := a.memory.Load(ctx); err != nil {
		a.logger.Warn("Failed to load memory", "error", err)
	}

	a.logger.Info("Agent initialized", "name", a.name)
	return nil
}

func (a *Agent) Run(ctx context.Context, messageBus MessageBusInterface) {
	a.bus = messageBus
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

			// 为每个消息处理创建独立的上下文，避免 goroutine 泄漏
			msgCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
			go func(ctx context.Context, msg icooclawbus.InboundMessage) {
				defer cancel()
				a.handleMessage(ctx, msg)
			}(msgCtx, msg)
		}
	}
}

func (a *Agent) handleMessage(ctx context.Context, msg icooclawbus.InboundMessage) {
	a.logger.Info("Handling message",
		"channel", msg.Channel,
		"chat_id", msg.ChatID,
		"user_id", msg.UserID,
		"content", msg.Content)

	session, err := a.storage.GetOrCreateSession(msg.Channel, msg.ChatID, msg.UserID)
	if err != nil {
		a.logger.Error("Failed to get or create session", "error", err)
		return
	}

	_, err = a.storage.AddMessage(session.ID, "user", msg.Content, "", "", "", "")
	if err != nil {
		a.logger.Error("Failed to add user message", "error", err)
		return
	}

	contextBuilder := NewContextBuilder(a, session)
	messages, systemPrompt, err := contextBuilder.Build(ctx)
	if err != nil {
		a.logger.Error("Failed to build context", "error", err)
		return
	}

	clientID, _ := msg.Metadata["client_id"].(string)

	onChunk := func(chunk, thinking string) {
		if a.bus != nil {
			if chunk != "" {
				a.bus.PublishOutbound(ctx, icooclawbus.OutboundMessage{
					Type:      icooclawbus.MessageTypeChunk,
					Channel:   msg.Channel,
					ChatID:    msg.ChatID,
					Content:   chunk,
					Timestamp: time.Now(),
					Metadata:  map[string]interface{}{"client_id": clientID},
				})
			}
			if thinking != "" {
				a.bus.PublishOutbound(ctx, icooclawbus.OutboundMessage{
					Type:      icooclawbus.MessageTypeThinking,
					Channel:   msg.Channel,
					ChatID:    msg.ChatID,
					Thinking:  thinking,
					Timestamp: time.Now(),
					Metadata:  map[string]interface{}{"client_id": clientID},
				})
			}
		}
	}

	// 使用解耦的 LoopHooks
	reactCfg := NewReActConfig()
	reactCfg.Provider = a.Provider()
	reactCfg.Tools = a.Tools()
	reactCfg.Session = session
	reactCfg.Logger = a.logger
	reactCfg.Hooks = NewLoopHooks(
		a.storage,
		a.bus,
		onChunk,
		msg.ChatID,
		clientID,
		session,
		a.logger,
	)

	reactAgent := NewReActAgent(reactCfg)
	response, reasoningContent, toolCalls, err := reactAgent.Run(ctx, messages, systemPrompt)
	if err != nil {
		a.logger.Error("Agent loop failed", "error", err)
		if a.bus != nil {
			a.bus.PublishOutbound(ctx, icooclawbus.OutboundMessage{
				Type:      icooclawbus.MessageTypeError,
				Channel:   msg.Channel,
				ChatID:    msg.ChatID,
				Content:   fmt.Sprintf("处理消息时出错: %s", err.Error()),
				Timestamp: time.Now(),
				Metadata:  map[string]interface{}{"client_id": clientID},
			})
		}
		return
	}

	if a.bus != nil {
		a.bus.PublishOutbound(ctx, icooclawbus.OutboundMessage{
			Type:      icooclawbus.MessageTypeEnd,
			Channel:   msg.Channel,
			ChatID:    msg.ChatID,
			Timestamp: time.Now(),
			Metadata:  map[string]interface{}{"client_id": clientID},
		})
	}

	toolCallsJSON, err := json.Marshal(toolCalls)
	if err != nil {
		a.logger.Warn("Failed to marshal tool calls", "error", err)
	}
	_, err = a.storage.AddMessage(session.ID, "assistant", response, string(toolCallsJSON), "", "", reasoningContent)
	if err != nil {
		a.logger.Error("Failed to save assistant message", "error", err)
	}

	a.logger.Info("Message handled", "response_length", len(response))
}

func (a *Agent) SetSystemPrompt(prompt string) {
	a.config.SystemPrompt = prompt
}

func (a *Agent) GetSystemPrompt() string {
	return a.config.SystemPrompt
}

func (a *Agent) ProcessMessage(ctx context.Context, content string) (string, error) {
	session, err := a.storage.GetOrCreateSession("cli", "cli-session", "cli-user")
	if err != nil {
		return "", fmt.Errorf("failed to get or create session: %w", err)
	}

	_, err = a.storage.AddMessage(session.ID, "user", content, "", "", "", "")
	if err != nil {
		return "", fmt.Errorf("failed to add user message: %w", err)
	}

	contextBuilder := NewContextBuilder(a, session)
	messages, systemPrompt, err := contextBuilder.Build(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to build context: %w", err)
	}

	reactCfg := NewReActConfig()
	reactCfg.Provider = a.Provider()
	reactCfg.Tools = a.Tools()
	reactCfg.Session = session
	reactCfg.Logger = a.logger
	reactCfg.Hooks = &hooks.DefaultHooks{}

	reactAgent := NewReActAgent(reactCfg)
	response, reasoningContent, toolCalls, err := reactAgent.Run(ctx, messages, systemPrompt)
	if err != nil {
		return "", fmt.Errorf("agent loop failed: %w", err)
	}

	toolCallsJSON, err := json.Marshal(toolCalls)
	if err != nil {
		a.logger.Warn("Failed to marshal tool calls", "error", err)
	}
	_, err = a.storage.AddMessage(session.ID, "assistant", response, string(toolCallsJSON), "", "", reasoningContent)
	if err != nil {
		a.logger.Warn("Failed to save assistant message", "error", err)
	}

	return response, nil
}

func (a *Agent) SetSessionRolePrompt(sessionID uint, rolePrompt string) error {
	session, err := a.storage.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	var metadata SessionMetadata
	if session.Metadata != "" {
		if err := json.Unmarshal([]byte(session.Metadata), &metadata); err != nil {
			metadata = SessionMetadata{}
		}
	}

	metadata.RolePrompt = rolePrompt

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	return a.storage.UpdateSessionMetadata(sessionID, string(metadataJSON))
}

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

// GetMemoryFile 读取 memory/MEMORY.md 文件内容
func (a *Agent) GetMemoryFile() (string, error) {
	if a.workspace == "" {
		return "", fmt.Errorf("workspace not set")
	}

	memoryPath := filepath.Join(a.workspace, "memory", "MEMORY.md")
	data, err := os.ReadFile(memoryPath)
	if err != nil {
		return "", fmt.Errorf("failed to read memory file: %w", err)
	}

	return string(data), nil
}

// UpdateMemoryFile 更新 memory/MEMORY.md 文件内容
func (a *Agent) UpdateMemoryFile(section, content string) error {
	if a.workspace == "" {
		return fmt.Errorf("workspace not set")
	}

	memoryPath := filepath.Join(a.workspace, "memory", "MEMORY.md")

	// 读取现有内容
	var fileContent string
	if data, err := os.ReadFile(memoryPath); err == nil {
		fileContent = string(data)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read memory file: %w", err)
	} else {
		// 文件不存在，创建默认内容
		fileContent = `# 记忆

此文件存储长期记忆和重要信息。

## 重要事实

<!-- 重要事实和信息将存储在这里 -->

## 用户偏好

<!-- 用户偏好和设置 -->

## 学到的知识

<!-- 从对话中学习的知识 -->

## 最后更新

<!-- 最后记忆更新的时间戳 -->`
	}

	// 更新指定部分
	updated := false
	lines := strings.Split(fileContent, "\n")
	var result strings.Builder
	currentSection := ""

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			currentSection = strings.TrimSpace(strings.TrimPrefix(line, "## "))
		}

		if currentSection == section && strings.Contains(line, "<!--") && !updated {
			// 找到要更新的部分，替换内容
			result.WriteString(fmt.Sprintf("%s\n", content))
			updated = true
			continue
		}

		result.WriteString(line + "\n")
	}

	if !updated {
		result.WriteString(fmt.Sprintf("\n最后更新: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	}

	// 确保目录存在
	memoryDir := filepath.Dir(memoryPath)
	if err := os.MkdirAll(memoryDir, 0755); err != nil {
		return fmt.Errorf("failed to create memory directory: %w", err)
	}

	return os.WriteFile(memoryPath, []byte(result.String()), 0644)
}

// LoopHooks ReAct 钩子实现（解耦版本）
type LoopHooks struct {
	storage  StorageInterface
	bus      MessageBusInterface
	onChunk  hooks.OnChunkFunc
	chatID   string
	clientID string
	session  *storage.Session
	logger   *slog.Logger
}

// NewLoopHooks 创建解耦的 LoopHooks
func NewLoopHooks(
	storage StorageInterface,
	bus MessageBusInterface,
	onChunk hooks.OnChunkFunc,
	chatID, clientID string,
	session *storage.Session,
	logger *slog.Logger,
) *LoopHooks {
	return &LoopHooks{
		storage:  storage,
		bus:      bus,
		onChunk:  onChunk,
		chatID:   chatID,
		clientID: clientID,
		session:  session,
		logger:   logger,
	}
}

func (h *LoopHooks) OnLLMRequest(ctx context.Context, req *provider.ChatRequest, iteration int) error {
	return nil
}

func (h *LoopHooks) OnLLMChunk(ctx context.Context, content, thinking string) error {
	if h.onChunk != nil {
		h.onChunk(content, thinking)
	}
	return nil
}

func (h *LoopHooks) OnLLMResponse(ctx context.Context, content, reasoningContent string, toolCalls []provider.ToolCall, iteration int) error {
	return nil
}

func (h *LoopHooks) OnToolCall(ctx context.Context, toolCallID string, toolName string, arguments string) error {
	if h.storage != nil && h.session != nil {
		_, err := h.storage.AddMessage(
			h.session.ID,
			consts.RoleToolCall.ToString(),
			"",
			arguments,
			toolCallID,
			toolName,
			"",
		)
		if err != nil {
			h.logger.Error("Failed to save tool call message", "tool", toolName, "error", err)
		}
	}

	if h.bus != nil {
		h.bus.PublishOutbound(ctx, icooclawbus.OutboundMessage{
			Type:       "tool_call",
			ID:         toolCallID,
			ToolCallID: toolCallID,
			ToolName:   toolName,
			Arguments:  arguments,
			Status:     "running",
			ChatID:     h.chatID,
			Timestamp:  time.Now(),
			Metadata:   map[string]interface{}{"client_id": h.clientID},
		})
	}
	return nil
}

func (h *LoopHooks) OnToolResult(ctx context.Context, toolCallID string, toolName string, result tools.ToolResult) error {
	if h.storage != nil && h.session != nil {
		resultContent := result.Content
		if result.Error != nil {
			resultContent = fmt.Sprintf("Error: %v", result.Error)
		}
		_, err := h.storage.AddMessage(
			h.session.ID,
			consts.RoleTool.ToString(),
			resultContent,
			"",
			toolCallID,
			toolName,
			"",
		)
		if err != nil {
			h.logger.Error("Failed to save tool result message", "tool", toolName, "error", err)
		}
	}

	if h.bus != nil {
		status := "completed"
		errorMsg := ""
		if result.Error != nil {
			status = "error"
			errorMsg = result.Error.Error()
		}
		h.bus.PublishOutbound(ctx, icooclawbus.OutboundMessage{
			Type:       "tool_result",
			ID:         toolCallID,
			ToolCallID: toolCallID,
			ToolName:   toolName,
			Content:    result.Content,
			Error:      errorMsg,
			Status:     status,
			ChatID:     h.chatID,
			Timestamp:  time.Now(),
			Metadata:   map[string]interface{}{"client_id": h.clientID},
		})
	}
	return nil
}

func (h *LoopHooks) OnIterationStart(ctx context.Context, iteration int, messages []provider.Message) error {
	return nil
}

func (h *LoopHooks) OnIterationEnd(ctx context.Context, iteration int, hasToolCalls bool) error {
	return nil
}

func (h *LoopHooks) OnError(ctx context.Context, err error) error { return nil }

func (h *LoopHooks) OnComplete(ctx context.Context, content, reasoningContent string, toolCalls []provider.ToolCall, iterations int) error {
	return nil
}
