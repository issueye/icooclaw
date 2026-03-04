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
	"icooclaw.ai/memory"
	"icooclaw.ai/provider"
	"icooclaw.ai/skill"
	"icooclaw.ai/tools"
	"icooclaw.core/bus"
	icooclawbus "icooclaw.core/bus"
	"icooclaw.core/config"
	"icooclaw.core/storage"
)

// SessionMetadata 会话元数据
type SessionMetadata struct {
	RolePrompt string `json:"role_prompt"`
}

// Agent Agent 核心结构体
type Agent struct {
	name           string                  // Agent 名称
	workspace      string                  // 工作空间
	sessionID      uint                    // 会话 ID
	config         *config.AgentSettings   // Agent 配置
	bus            *icooclawbus.MessageBus // 消息总线
	logger         *slog.Logger            // 日志记录器
	provider       provider.Provider       // 模型提供器
	tools          *tools.Registry         // 工具注册表
	storage        *storage.Storage        // 存储接口
	memory         memory.Loader           // 记忆加载器
	skills         skill.Loader            // 技能加载器
	contextBuilder *ContextBuilder         // 上下文构建器
}

// NewAgent 创建 Agent 实例（使用函数式选项模式）
func NewAgent(
	id uint,
	name string,
	provider provider.Provider,
	storage *storage.Storage,
	config *config.AgentSettings,
	logger *slog.Logger,
	workspace string,
	opts ...AgentOption,
) *Agent {
	if logger == nil {
		logger = slog.Default()
	}

	agent := &Agent{
		sessionID: id,
		name:      name,
		provider:  provider,
		tools:     tools.NewRegistry(), // 默认实现
		storage:   storage,
		config:    config,
		logger:    logger,
		workspace: workspace,
	}

	// 应用选项
	for _, opt := range opts {
		opt(agent)
	}

	// 初始化记忆加载器
	if agent.memory == nil {
		agent.memory = memory.NewStorage(storage, logger)
	}

	// 初始化技能加载器
	if agent.skills == nil {
		agent.skills = skill.NewLoader(storage, logger)
	}

	return agent
}

func (a *Agent) Workspace() string {
	return a.workspace
}

func (a *Agent) Name() string {
	return a.name
}

// Provider 获取 Provider
func (a *Agent) Provider() provider.Provider {
	return a.provider
}

// SetProvider 设置 Provider（用于运行时切换模型）
func (a *Agent) SetProvider(p provider.Provider) {
	a.provider = p
	a.logger.Info("切换供应商成功", "name", p.GetName(), "model", p.GetDefaultModel())
}

func (a *Agent) Tools() []tools.Tool {
	return a.tools.List()
}

func (a *Agent) SetTools(registry *tools.Registry) {
	a.tools = registry
}

func (a *Agent) Storage() *storage.Storage {
	return a.storage
}

func (a *Agent) Config() *config.AgentSettings {
	return a.config
}

func (a *Agent) Logger() *slog.Logger {
	return a.logger
}

func (a *Agent) Skills() skill.Loader {
	return a.skills
}

func (a *Agent) Memory() memory.Loader {
	return a.memory
}

func (a *Agent) RegisterTool(tool tools.Tool) {
	a.tools.Register(tool)
	a.logger.Info("注册工具成功", "name", tool.Name())
}

// Run 运行 Agent
func (a *Agent) Run(ctx context.Context, messageBus *icooclawbus.MessageBus) {
	a.bus = messageBus
	a.logger.Info("启动 Agent", "name", a.name)

	for {
		select {
		case <-ctx.Done():
			a.logger.Info("停止 Agent 成功", "name", a.name)
			return
		default:
			msg, err := messageBus.ConsumeInbound(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				a.logger.Error("消费消息失败", "error", err)
				continue
			}

			// 为每个消息处理创建独立的上下文，避免 goroutine 泄漏
			msgCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
			go func(ctx context.Context, msg bus.InboundMessage) {
				defer cancel()
				a.handleMessage(ctx, a.sessionID, msg)
			}(msgCtx, msg)
		}
	}
}

func (a *Agent) handleMessage(ctx context.Context, sessionID uint, msg bus.InboundMessage) {
	a.logger.Info("开始处理消息",
		"channel", msg.Channel,
		"chat_id", msg.ChatID,
		"user_id", msg.UserID,
		"content", msg.Content)

	// 获取会话
	session, err := a.storage.Session().GetByID(sessionID)
	if err != nil || (session != nil && session.ID == 0) {
		// 如果会话不存在，创建一个新的会话
		session, err = a.storage.Session().GetOrCreateSession(msg.Channel, msg.ChatID, msg.UserID)
		if err != nil {
			a.logger.Error("获取或创建会话失败", "error", err)
			return
		}
	}

	// 创建用户消息
	userMsg := storage.NewUserMessage(session.ID, msg.Content)
	err = a.storage.Message().CreateOrUpdate(userMsg)
	if err != nil {
		a.logger.Error("添加用户消息失败", "error", err)
		return
	}

	// 构建上下文
	contextBuilder := NewContextBuilder(session.ID, a.workspace, a.logger, a.skills, a.memory)
	messages, systemPrompt, err := contextBuilder.Build(ctx)
	if err != nil {
		a.logger.Error("构建上下文失败", "error", err)
		return
	}

	// 获取客户端 ID
	clientID, _ := msg.Metadata["client_id"].(string)

	// 处理 OnLLMChunk 钩子
	onChunk := func(chunk, thinking string) {
		if a.bus != nil {
			if chunk != "" {
				a.bus.PublishOutbound(ctx, bus.OutboundMessage{
					Type:      bus.MessageTypeChunk,
					Channel:   msg.Channel,
					ChatID:    msg.ChatID,
					Content:   chunk,
					Timestamp: time.Now(),
					Metadata:  map[string]any{"client_id": clientID},
				})
			}
			if thinking != "" {
				a.bus.PublishOutbound(ctx, bus.OutboundMessage{
					Type:      bus.MessageTypeThinking,
					Channel:   msg.Channel,
					ChatID:    msg.ChatID,
					Thinking:  thinking,
					Timestamp: time.Now(),
					Metadata:  map[string]any{"client_id": clientID},
				})
			}
		}
	}

	// 使用解耦的 LoopHooks
	reactCfg := NewReActConfig()
	reactCfg.Provider = a.Provider()
	reactCfg.Tools = a.tools
	reactCfg.Session = session
	reactCfg.Logger = a.logger
	reactCfg.Hooks = NewLoopHooks(a.storage, a.bus, onChunk, msg.ChatID, clientID, session, a.logger)

	reactAgent := NewReActAgent(reactCfg)
	response, reasoningContent, toolCalls, err := reactAgent.Run(ctx, messages, systemPrompt)
	if err != nil {
		a.logger.Error("处理消息时出错失败", "error", err)
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
		a.logger.Warn("序列化工具调用失败失败", "error", err)
	}

	assistantMsg := storage.NewAssistantMessage(session.ID, response, reasoningContent)
	assistantMsg.ToolResult = string(toolCallsJSON)
	err = a.storage.Message().CreateOrUpdate(assistantMsg)
	if err != nil {
		a.logger.Error("添加助手消息失败", "error", err)
	}

	a.logger.Info("处理消息成功", "response_length", len(response))
}

func (a *Agent) SetSystemPrompt(prompt string) {
	a.config.SystemPrompt = prompt
}

func (a *Agent) GetSystemPrompt() string {
	return a.config.SystemPrompt
}

func (a *Agent) ProcessMessage(ctx context.Context, content string) (uint, string, error) {
	session, err := a.storage.Session().GetOrCreateSession("cli", "cli-session", "cli-user")
	if err != nil {
		return 0, "", fmt.Errorf("创建会话失败: %w", err)
	}

	// 创建用户消息
	userMsg := storage.NewUserMessage(session.ID, content)
	err = a.storage.Message().CreateOrUpdate(userMsg)
	if err != nil {
		return 0, "", fmt.Errorf("添加用户消息失败: %w", err)
	}

	// 构建上下文
	contextBuilder := NewContextBuilder(session.ID, a.workspace, a.logger, a.skills, a.memory)
	messages, systemPrompt, err := contextBuilder.Build(ctx)
	if err != nil {
		return 0, "", fmt.Errorf("构建上下文失败: %w", err)
	}

	// 运行 ReactActAgent
	reactCfg := NewReActConfig()
	reactCfg.Provider = a.Provider()
	reactCfg.Tools = a.tools
	reactCfg.Session = session
	reactCfg.Logger = a.logger
	reactCfg.Hooks = &hooks.DefaultHooks{}

	// 运行 ReactActAgent
	reactAgent := NewReActAgent(reactCfg)
	response, reasoningContent, toolCalls, err := reactAgent.Run(ctx, messages, systemPrompt)
	if err != nil {
		return 0, "", fmt.Errorf("处理消息时出错: %w", err)
	}

	// 序列化工具调用
	toolCallsJSON, err := json.Marshal(toolCalls)
	if err != nil {
		a.logger.Warn("序列化工具调用失败", slog.Any("error", err))
	}

	// 创建助手消息
	assistantMsg := storage.NewAssistantMessage(session.ID, response, reasoningContent)
	assistantMsg.ToolResult = string(toolCallsJSON)
	err = a.storage.Message().CreateOrUpdate(assistantMsg)
	if err != nil {
		return 0, "", fmt.Errorf("添加助手消息失败: %w", err)
	}

	return session.ID, response, nil
}

func (a *Agent) SetSessionRolePrompt(sessionID uint, rolePrompt string) error {
	session, err := a.storage.Session().GetByID(sessionID)
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

	return a.storage.Session().UpdateSessionMetadata(sessionID, string(metadataJSON))
}

func (a *Agent) GetSessionRolePrompt(sessionID uint) (string, error) {
	session, err := a.storage.Session().GetByID(sessionID)
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
