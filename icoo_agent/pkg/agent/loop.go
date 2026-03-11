package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/channels"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/memory"
	"icooclaw/pkg/providers"
	"icooclaw/pkg/skill"
	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"
)

const (
	defaultMaxToolIterations = 10
	defaultResponse          = "处理完成，但没有响应内容。"
)

// Loop represents the main agent processing loop.
type Loop struct {
	bus            *bus.MessageBus
	provider       providers.Provider
	fallbackChain  *providers.FallbackChain
	tools          *tools.Registry
	memory         memory.Loader
	skills         skill.Loader
	storage        *storage.Storage
	channelManager *channels.Manager

	// Configuration
	maxToolIterations int
	systemPrompt      string

	running     atomic.Bool
	summarizing sync.Map

	logger *slog.Logger
}

// NewLoop creates a new agent loop.
func NewLoop(opts ...LoopOption) *Loop {
	l := &Loop{
		tools:             tools.NewRegistry(),
		maxToolIterations: defaultMaxToolIterations,
		logger:            slog.Default(),
	}

	for _, opt := range opts {
		opt(l)
	}

	return l
}

// LoopOption is a functional option for Loop.
type LoopOption func(*Loop)

// WithLoopBus sets the message bus.
func WithLoopBus(b *bus.MessageBus) LoopOption {
	return func(l *Loop) { l.bus = b }
}

// WithLoopProvider sets the provider.
func WithLoopProvider(p providers.Provider) LoopOption {
	return func(l *Loop) { l.provider = p }
}

// WithLoopFallbackChain sets the fallback chain.
func WithLoopFallbackChain(fc *providers.FallbackChain) LoopOption {
	return func(l *Loop) { l.fallbackChain = fc }
}

// WithLoopTools sets the tools registry.
func WithLoopTools(t *tools.Registry) LoopOption {
	return func(l *Loop) { l.tools = t }
}

// WithLoopMemory sets the memory loader.
func WithLoopMemory(m memory.Loader) LoopOption {
	return func(l *Loop) { l.memory = m }
}

// WithLoopSkills sets the skill loader.
func WithLoopSkills(s skill.Loader) LoopOption {
	return func(l *Loop) { l.skills = s }
}

// WithLoopStorage sets the storage.
func WithLoopStorage(s *storage.Storage) LoopOption {
	return func(l *Loop) { l.storage = s }
}

// WithLoopChannelManager sets the channel manager.
func WithLoopChannelManager(cm *channels.Manager) LoopOption {
	return func(l *Loop) { l.channelManager = cm }
}

// WithLoopLogger sets the logger.
func WithLoopLogger(log *slog.Logger) LoopOption {
	return func(l *Loop) { l.logger = log }
}

// WithLoopMaxToolIterations sets the maximum tool iterations.
func WithLoopMaxToolIterations(n int) LoopOption {
	return func(l *Loop) {
		if n > 0 {
			l.maxToolIterations = n
		}
	}
}

// WithLoopSystemPrompt sets the system prompt.
func WithLoopSystemPrompt(prompt string) LoopOption {
	return func(l *Loop) { l.systemPrompt = prompt }
}

// GetDefaultProvider returns the default provider.
func (l *Loop) GetDefaultProvider() providers.Provider {
	// Return the provider if set
	if l.provider != nil {
		return l.provider
	}

	// Return the fallback chain
	return l.fallbackChain
}

// Run starts the agent loop.
func (l *Loop) Run(ctx context.Context) error {
	l.running.Store(true)
	defer l.running.Store(false)

	l.logger.With("name", "【智能体】").Info("代理循环已启动")

	for l.running.Load() {
		select {
		case <-ctx.Done():
			l.logger.With("name", "【智能体】").Info("代理循环已停止", "reason", ctx.Err())
			return ctx.Err()
		default:
			msg, ok := l.bus.ConsumeInbound(ctx)
			if !ok {
				continue
			}

			// Process message in a separate goroutine
			go func() {
				response, err := l.processMessage(ctx, msg)
				if err != nil {
					l.logger.With("name", "【智能体】").Error("处理消息失败",
						"channel", msg.Channel,
						"session_id", msg.SessionID,
						"error", err)
					response = fmt.Sprintf("处理消息时出错: %v", err)
				}

				if response != "" {
					pubCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()

					if err := l.bus.PublishOutbound(pubCtx, bus.OutboundMessage{
						Channel:   msg.Channel,
						SessionID: msg.SessionID,
						Text:      response,
					}); err != nil {
						l.logger.With("name", "【智能体】").Error("发布响应失败",
							"channel", msg.Channel,
							"session_id", msg.SessionID,
							"error", err)
					}
				}
			}()
		}
	}

	return nil
}

// Stop stops the agent loop.
func (l *Loop) Stop() {
	l.running.Store(false)
}

// processMessage processes an inbound message.
func (l *Loop) processMessage(ctx context.Context, msg bus.InboundMessage) (string, error) {
	l.logger.With("name", "【智能体】").Info("正在处理消息",
		"channel", msg.Channel,
		"session_id", msg.SessionID,
		"sender", msg.Sender.ID)

	// Get binding
	binding, err := l.storage.Binding().GetBinding(msg.Channel, msg.SessionID)
	if err != nil {
		l.logger.With("name", "【智能体】").Debug("未找到绑定，使用默认代理",
			"channel", msg.Channel,
			"session_id", msg.SessionID)
		// Use default agent name
		return l.processWithAgent(ctx, "default", msg)
	}

	// Process with agent
	return l.processWithAgent(ctx, binding.AgentName, msg)
}

// processWithAgent processes a message with a specific agent.
// This is the core message processing logic, similar to PicoClaw's runAgentLoop.
func (l *Loop) processWithAgent(ctx context.Context, agentName string, msg bus.InboundMessage) (string, error) {
	l.logger.With("name", "【智能体】").Info("使用代理处理",
		"agent", agentName,
		"channel", msg.Channel,
		"session_id", msg.SessionID)

	// 1. Build session key (format: channel:sessionID)
	sessionKey := consts.GetSessionKey(msg.Channel, msg.SessionID)

	// 2. Load memory/history
	var history []providers.ChatMessage
	if l.memory != nil {
		mem, err := l.memory.Load(ctx, sessionKey)
		if err != nil {
			l.logger.With("name", "【智能体】").Warn("加载记忆失败", "error", err, "session_key", sessionKey)
		} else {
			history = mem
		}
	}

	// 3. Build messages
	messages := l.buildMessages(history, msg)

	// 4. Save user message to memory
	if l.memory != nil {
		if err := l.memory.Save(ctx, sessionKey, "user", msg.Text); err != nil {
			l.logger.With("name", "【智能体】").Warn("保存用户消息失败", "error", err)
		}
	}

	// 5. Run LLM iteration loop with tool calling
	finalContent, iteration, err := l.runLLMIteration(ctx, messages, msg)
	if err != nil {
		return "", err
	}

	// 6. Handle empty response
	if finalContent == "" {
		finalContent = defaultResponse
	}

	// 7. Save final assistant message to memory
	if l.memory != nil {
		if err := l.memory.Save(ctx, sessionKey, "assistant", finalContent); err != nil {
			l.logger.With("name", "【智能体】").Warn("保存助手消息失败", "error", err)
		}
	}

	// 8. Log response
	l.logger.With("name", "【智能体】").Info("响应已生成",
		"agent", agentName,
		"session_key", sessionKey,
		"iterations", iteration,
		"content_length", len(finalContent))

	return finalContent, nil
}

// buildMessages builds the message list for the LLM request.
func (l *Loop) buildMessages(history []providers.ChatMessage, msg bus.InboundMessage) []providers.ChatMessage {
	messages := make([]providers.ChatMessage, 0, len(history)+2)

	// Add system prompt
	systemPrompt := l.systemPrompt
	if systemPrompt == "" {
		systemPrompt = "你是一个有帮助的AI助手。"
	}
	messages = append(messages, providers.ChatMessage{
		Role:    "system",
		Content: systemPrompt,
	})

	// Add history
	messages = append(messages, history...)

	// Add user message
	messages = append(messages, providers.ChatMessage{
		Role:    "user",
		Content: msg.Text,
	})

	return messages
}

// runLLMIteration runs the LLM iteration loop with tool calling.
func (l *Loop) runLLMIteration(
	ctx context.Context,
	messages []providers.ChatMessage,
	msg bus.InboundMessage,
) (string, int, error) {
	// Check if provider is configured
	if l.provider == nil && l.fallbackChain == nil {
		l.logger.With("name", "【智能体】").Error("未配置AI提供商")
		return "", 0, fmt.Errorf("未配置AI提供商，请在设置中配置默认模型")
	}

	// Get the provider to use
	provider := l.GetDefaultProvider()
	if provider == nil {
		l.logger.With("name", "【智能体】").Error("无法获取有效的AI提供商")
		return "", 0, fmt.Errorf("无法获取有效的AI提供商")
	}

	iteration := 0
	currentMessages := messages

	for iteration < l.maxToolIterations {
		iteration++

		// Build request
		req := providers.ChatRequest{
			Model:    provider.GetDefaultModel(),
			Messages: currentMessages,
		}

		// Add tools if available
		if toolDefs := l.tools.ToProviderDefs(); len(toolDefs) > 0 {
			req.Tools = l.convertToolDefinitions(toolDefs)
		}

		l.logger.With("name", "【智能体】").Debug("正在发送请求到LLM",
			"iteration", iteration,
			"message_count", len(currentMessages))

		// Send to provider
		resp, err := provider.Chat(ctx, req)
		if err != nil {
			l.logger.With("name", "【智能体】").Error("LLM请求失败", "error", err, "iteration", iteration)
			// Provide user-friendly error message
			errMsg := fmt.Sprintf("LLM请求失败: %v", err)
			if errors.Is(err, context.DeadlineExceeded) {
				errMsg = "请求超时，AI服务响应时间过长，请稍后重试"
			}
			return "", iteration, fmt.Errorf("%s", errMsg)
		}

		// Handle tool calls
		if len(resp.ToolCalls) > 0 {
			l.logger.With("name", "【智能体】").Info("正在处理工具调用",
				"count", len(resp.ToolCalls),
				"iteration", iteration)

			// Add assistant message with tool calls
			assistantMsg := providers.ChatMessage{
				Role:      "assistant",
				Content:   resp.Content,
				ToolCalls: resp.ToolCalls,
			}
			currentMessages = append(currentMessages, assistantMsg)

			// Execute each tool call
			for _, tc := range resp.ToolCalls {
				toolResult, err := l.executeToolCall(ctx, tc, msg)
				if err != nil {
					toolResult = fmt.Sprintf("错误: %v", err)
				}

				// Add tool result message
				currentMessages = append(currentMessages, providers.ChatMessage{
					Role:       "tool",
					Content:    toolResult,
					ToolCallID: tc.ID,
				})
			}

			continue
		}

		// No tool calls, return the response
		l.logger.With("name", "【智能体】").Debug("LLM响应已接收",
			"iteration", iteration,
			"content_length", len(resp.Content))

		return resp.Content, iteration, nil
	}

	// Max iterations reached
	l.logger.With("name", "【智能体】").Warn("已达到最大工具迭代次数",
		"iterations", l.maxToolIterations)

	return "", iteration, fmt.Errorf("已达到最大工具迭代次数 (%d)", l.maxToolIterations)
}

// executeToolCall executes a tool call and returns the result.
func (l *Loop) executeToolCall(
	ctx context.Context,
	tc providers.ToolCall,
	msg bus.InboundMessage,
) (string, error) {
	toolName := tc.Function.Name
	l.logger.With("name", "【智能体】").Info("正在执行工具",
		"tool", toolName,
		"channel", msg.Channel,
		"session_id", msg.SessionID)

	// Parse arguments
	var args map[string]any
	if tc.Function.Arguments != "" {
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			l.logger.With("name", "【智能体】").Error("解析工具参数失败",
				"tool", toolName,
				"error", err)
			return "", fmt.Errorf("解析参数失败: %w", err)
		}
	}

	// Execute tool
	result := l.tools.ExecuteWithContext(ctx, toolName, args, msg.Channel, msg.SessionID, nil)
	if result.Error != nil {
		return "", result.Error
	}

	return result.Content, nil
}

// convertToolDefinitions converts tools.ToolDefinition to providers.Tool.
func (l *Loop) convertToolDefinitions(defs []tools.ToolDefinition) []providers.Tool {
	tools := make([]providers.Tool, 0, len(defs))
	for _, def := range defs {
		tools = append(tools, providers.Tool{
			Type: def.Type,
			Function: providers.Function{
				Name:        def.Function.Name,
				Description: def.Function.Description,
				Parameters:  def.Function.Parameters,
			},
		})
	}
	return tools
}

// IsRunning returns true if the loop is running.
func (l *Loop) IsRunning() bool {
	return l.running.Load()
}

// ProcessDirect processes a message directly without going through the bus.
// This is useful for CLI or API interactions.
func (l *Loop) ProcessDirect(
	ctx context.Context,
	content, sessionID string,
) (string, error) {
	return l.ProcessDirectWithChannel(ctx, content, sessionID, "cli", sessionID)
}

// ProcessDirectWithChannel processes a message directly with specific channel/sessionID.
func (l *Loop) ProcessDirectWithChannel(
	ctx context.Context,
	content, sessionID, channel, _ string,
) (string, error) {
	msg := bus.InboundMessage{
		Channel:   channel,
		SessionID: sessionID,
		Sender: bus.SenderInfo{
			ID: "direct",
		},
		Text:      content,
		Timestamp: time.Now(),
	}

	return l.processWithAgent(ctx, "default", msg)
}

// RegisterTool registers a tool with the loop's tool registry.
func (l *Loop) RegisterTool(tool tools.Tool) {
	l.tools.Register(tool)
}

// SetChannelManager sets the channel manager.
func (l *Loop) SetChannelManager(cm *channels.Manager) {
	l.channelManager = cm
}
