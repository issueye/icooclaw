package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"icooclaw.ai/config"
	"icooclaw.ai/consts"
	"icooclaw.ai/hooks"
	"icooclaw.ai/memory"
	"icooclaw.ai/provider"
	"icooclaw.ai/skill"
	"icooclaw.ai/storage"
	"icooclaw.ai/tools"

	icooclawbus "icooclaw.bus"
)

type SessionMetadata struct {
	RolePrompt string `json:"role_prompt"`
}

type Agent struct {
	name      string
	provider  provider.Provider
	tools     *tools.Registry
	storage   *storage.Storage
	memory    *memory.MemoryStore
	skills    *skill.Loader
	config    config.AgentSettings
	bus       *icooclawbus.MessageBus
	logger    *slog.Logger
	workspace string
}

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
		memory: memory.NewMemoryStoreWithConfig(storage, logger, memory.MemoryConfig{
			ConsolidationThreshold: 50,
			SummaryEnabled:         true,
		}),
		skills:    skill.NewLoader(storage, logger),
		config:    config,
		logger:    logger,
		workspace: workspace,
	}
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

func (a *Agent) Tools() *tools.Registry {
	return a.tools
}

func (a *Agent) SetTools(registry *tools.Registry) {
	a.tools = registry
}

func (a *Agent) Storage() *storage.Storage {
	return a.storage
}

func (a *Agent) Config() config.AgentSettings {
	return a.config
}

func (a *Agent) Logger() *slog.Logger {
	return a.logger
}

func (a *Agent) Skills() *skill.Loader {
	return a.skills
}

func (a *Agent) Memory() *memory.MemoryStore {
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

func (a *Agent) Run(ctx context.Context, messageBus *icooclawbus.MessageBus) {
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

	reactCfg := NewReActConfig()
	reactCfg.Provider = a.Provider()
	reactCfg.Tools = a.Tools()
	reactCfg.Session = session
	reactCfg.Logger = a.logger
	reactCfg.Hooks = NewLoopHooks(a, onChunk, msg.ChatID, clientID, session)

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

type LoopHooks struct {
	agent    *Agent
	onChunk  hooks.OnChunkFunc
	chatID   string
	clientID string
	session  *storage.Session
}

func NewLoopHooks(agent *Agent, onChunk hooks.OnChunkFunc, chatID, clientID string, session *storage.Session) *LoopHooks {
	return &LoopHooks{
		agent:    agent,
		onChunk:  onChunk,
		chatID:   chatID,
		clientID: clientID,
		session:  session,
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
	if h.agent.storage != nil && h.session != nil {
		_, err := h.agent.storage.AddMessage(
			h.session.ID,
			consts.RoleToolCall.ToString(),
			"",
			arguments,
			toolCallID,
			toolName,
			"",
		)
		if err != nil {
			h.agent.logger.Error("Failed to save tool call message", "tool", toolName, "error", err)
		}
	}

	if h.agent.bus != nil {
		h.agent.bus.PublishOutbound(ctx, icooclawbus.OutboundMessage{
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
	if h.agent.storage != nil && h.session != nil {
		resultContent := result.Content
		if result.Error != nil {
			resultContent = fmt.Sprintf("Error: %v", result.Error)
		}
		_, err := h.agent.storage.AddMessage(
			h.session.ID,
			consts.RoleTool.ToString(),
			resultContent,
			"",
			toolCallID,
			toolName,
			"",
		)
		if err != nil {
			h.agent.logger.Error("Failed to save tool result message", "tool", toolName, "error", err)
		}
	}

	if h.agent.bus != nil {
		status := "completed"
		errorMsg := ""
		if result.Error != nil {
			status = "error"
			errorMsg = result.Error.Error()
		}
		h.agent.bus.PublishOutbound(ctx, icooclawbus.OutboundMessage{
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
