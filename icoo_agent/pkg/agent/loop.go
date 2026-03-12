package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
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

// StreamChunk 表示流式响应的一个数据块。
type StreamChunk struct {
	Content   string `json:"content"`
	Reasoning string `json:"reasoning,omitempty"`
	Done      bool   `json:"done"`
	Error     error  `json:"error,omitempty"`
}

// StreamCallback 流式响应的回调函数，每个数据块都会调用一次。
type StreamCallback func(chunk StreamChunk) error

// Loop 表示主要的智能体处理循环。
type Loop struct {
	bus             *bus.MessageBus
	provider        providers.Provider
	providerFactory *providers.Factory
	fallbackChain   *providers.FallbackChain
	tools           *tools.Registry
	memory          memory.Loader
	skills          skill.Loader
	storage         *storage.Storage
	channelManager  *channels.Manager

	// Configuration 配置项
	maxToolIterations int
	systemPrompt      string

	running     atomic.Bool
	summarizing sync.Map

	logger *slog.Logger
}

// NewLoop 创建一个新的智能体循环。
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

// LoopOption 智能体循环的函数式选项。
type LoopOption func(*Loop)

// WithLoopBus 设置消息总线。
func WithLoopBus(b *bus.MessageBus) LoopOption {
	return func(l *Loop) { l.bus = b }
}

// WithLoopProvider 设置提供商。
func WithLoopProvider(p providers.Provider) LoopOption {
	return func(l *Loop) { l.provider = p }
}

// WithLoopProviderFactory 设置提供商工厂，用于动态获取提供商。
func WithLoopProviderFactory(f *providers.Factory) LoopOption {
	return func(l *Loop) { l.providerFactory = f }
}

// WithLoopFallbackChain 设置降级链。
func WithLoopFallbackChain(fc *providers.FallbackChain) LoopOption {
	return func(l *Loop) { l.fallbackChain = fc }
}

// WithLoopTools 设置工具注册表。
func WithLoopTools(t *tools.Registry) LoopOption {
	return func(l *Loop) { l.tools = t }
}

// WithLoopMemory 设置记忆加载器。
func WithLoopMemory(m memory.Loader) LoopOption {
	return func(l *Loop) { l.memory = m }
}

// WithLoopSkills 设置技能加载器。
func WithLoopSkills(s skill.Loader) LoopOption {
	return func(l *Loop) { l.skills = s }
}

// WithLoopStorage 设置存储。
func WithLoopStorage(s *storage.Storage) LoopOption {
	return func(l *Loop) { l.storage = s }
}

// WithLoopChannelManager 设置渠道管理器。
func WithLoopChannelManager(cm *channels.Manager) LoopOption {
	return func(l *Loop) { l.channelManager = cm }
}

// WithLoopLogger 设置日志器。
func WithLoopLogger(log *slog.Logger) LoopOption {
	return func(l *Loop) { l.logger = log }
}

// WithLoopMaxToolIterations 设置最大工具迭代次数。
func WithLoopMaxToolIterations(n int) LoopOption {
	return func(l *Loop) {
		if n > 0 {
			l.maxToolIterations = n
		}
	}
}

// WithLoopSystemPrompt 设置系统提示词。
func WithLoopSystemPrompt(prompt string) LoopOption {
	return func(l *Loop) { l.systemPrompt = prompt }
}

// GetDefaultProvider 获取默认提供商。
// 首先检查直接设置的提供商，然后尝试从工厂获取（使用存储中的默认模型配置）。
func (l *Loop) GetDefaultProvider() providers.Provider {
	// Return the provider if set
	if l.provider != nil {
		return l.provider
	}

	// Try to get provider from factory using default model config
	if l.providerFactory != nil && l.storage != nil {
		provider, model, err := l.GetDynamicProvider()
		if err == nil && provider != nil {
			l.logger.Debug("动态获取Provider成功", "provider", provider.GetName(), "model", model)
			return provider
		}
		l.logger.Debug("动态获取Provider失败，使用fallback", "error", err)
	}

	// Return the fallback chain
	return l.fallbackChain
}

// GetDynamicProvider 从存储配置动态获取提供商。
// 返回提供商、模型名称和错误。
func (l *Loop) GetDynamicProvider() (providers.Provider, string, error) {
	if l.providerFactory == nil || l.storage == nil {
		return nil, "", fmt.Errorf("未配置提供商工厂或存储")
	}

	// Get default model from storage
	defaultModel, err := l.storage.Param().Get(consts.DEFAULT_MODEL_KEY)
	if err != nil || defaultModel == nil || defaultModel.Value == "" {
		return nil, "", fmt.Errorf("默认模型未配置")
	}

	// Parse provider/model format (e.g., "openai/gpt-4o")
	parts := splitProviderModel(defaultModel.Value)
	if len(parts) != 2 {
		return nil, "", fmt.Errorf("默认模型格式错误: %s", defaultModel.Value)
	}

	providerName, modelName := parts[0], parts[1]

	// Get provider from factory
	provider, err := l.providerFactory.Get(providerName)
	if err != nil {
		return nil, "", fmt.Errorf("获取Provider失败: %w", err)
	}

	return provider, modelName, nil
}

// splitProviderModel 分割模型字符串，格式为 "provider/model"。
func splitProviderModel(modelStr string) []string {
	idx := -1
	for i := 0; i < len(modelStr); i++ {
		if modelStr[i] == '/' {
			idx = i
			break
		}
	}
	if idx == -1 {
		return nil
	}
	return []string{modelStr[:idx], modelStr[idx+1:]}
}

// Run 启动智能体循环。
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

// Stop 停止智能体循环。
func (l *Loop) Stop() {
	l.running.Store(false)
}

// processMessage 处理入站消息。
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

// processWithAgent 使用指定智能体处理消息。
// 这是核心消息处理逻辑，类似于 PicoClaw 的 runAgentLoop。
func (l *Loop) processWithAgent(ctx context.Context, agentName string, msg bus.InboundMessage) (string, error) {
	l.logger.With("name", "【智能体】").Info("使用代理处理",
		"agent", agentName,
		"channel", msg.Channel,
		"session_id", msg.SessionID)

	// 1. Build session key (format: channel:sessionID) 构建会话键。
	sessionKey := consts.GetSessionKey(msg.Channel, msg.SessionID)

	// 2. Load memory/history 加载记忆历史记录。
	var history []providers.ChatMessage
	if l.memory != nil {
		mem, err := l.memory.Load(ctx, sessionKey)
		if err != nil {
			l.logger.With("name", "【智能体】").Warn("加载记忆失败", "error", err, "session_key", sessionKey)
		} else {
			history = mem
		}
	}

	// 3. Build messages 构建消息列表。
	messages := l.buildMessages(history, msg)

	// 4. Save user message to memory 保存用户消息到记忆中。
	if l.memory != nil {
		if err := l.memory.Save(ctx, sessionKey, consts.RoleUser.ToString(), msg.Text); err != nil {
			l.logger.With("name", "【智能体】").Warn("保存用户消息失败", "error", err)
		}
	}

	// 5. Run LLM iteration loop with tool calling 运行 LLM 迭代循环，支持工具调用。
	finalContent, iteration, err := l.runLLMIteration(ctx, messages, msg)
	if err != nil {
		return "", err
	}

	// 6. Handle empty response 处理空响应。
	if finalContent == "" {
		finalContent = defaultResponse
	}

	// 7. Save final assistant message to memory 保存助手助手消息到记忆中。
	if l.memory != nil {
		if err := l.memory.Save(ctx, sessionKey, consts.RoleAssistant.ToString(), finalContent); err != nil {
			l.logger.With("name", "【智能体】").Warn("保存助手消息失败", "error", err)
		}
	}

	// 8. Log response 记录响应。
	l.logger.With("name", "【智能体】").Info("响应已生成",
		"agent", agentName,
		"session_key", sessionKey,
		"iterations", iteration,
		"content_length", len(finalContent))

	return finalContent, nil
}

// buildMessages 构建 LLM 请求的消息列表。
func (l *Loop) buildMessages(history []providers.ChatMessage, msg bus.InboundMessage) []providers.ChatMessage {
	messages := make([]providers.ChatMessage, 0, len(history)+2)

	// Add system prompt 添加系统提示词。
	systemPrompt := l.systemPrompt
	if systemPrompt == "" {
		systemPrompt = "你是一个有帮助的AI助手。"
	}
	messages = append(messages, providers.ChatMessage{
		Role:    consts.RoleSystem.ToString(),
		Content: systemPrompt,
	})

	// Add history 添加会话历史记录。
	messages = append(messages, history...)

	// Add user message 添加用户消息。
	messages = append(messages, providers.ChatMessage{
		Role:    consts.RoleUser.ToString(),
		Content: msg.Text,
	})

	return messages
}

// runLLMIteration 运行 LLM 迭代循环，支持工具调用。
func (l *Loop) runLLMIteration(
	ctx context.Context,
	messages []providers.ChatMessage,
	msg bus.InboundMessage,
) (string, int, error) {
	// Check if provider is configured
	if l.provider == nil && l.fallbackChain == nil && l.providerFactory == nil {
		l.logger.With("name", "【智能体】").Error("未配置AI提供商")
		return "", 0, fmt.Errorf("未配置AI提供商，请在设置中配置默认模型")
	}

	// Get the provider to use
	provider := l.GetDefaultProvider()
	if provider == nil {
		l.logger.With("name", "【智能体】").Error("无法获取有效的AI提供商")
		return "", 0, fmt.Errorf("无法获取有效的AI提供商")
	}

	// Get model name (try dynamic first, then fallback to provider default)
	modelName := ""
	if l.providerFactory != nil && l.storage != nil {
		_, model, err := l.GetDynamicProvider()
		if err == nil {
			modelName = model
		}
	}
	if modelName == "" {
		modelName = provider.GetDefaultModel()
	}

	iteration := 0
	currentMessages := messages

	for iteration < l.maxToolIterations {
		iteration++

		// Build request
		req := providers.ChatRequest{
			Model:    modelName,
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
				Role:      consts.RoleAssistant.ToString(),
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
					Role:       consts.RoleTool.ToString(),
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

// executeToolCall 执行工具调用并返回结果。
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

	// 打印工具执行结果
	l.logger.With("name", "【智能体】").Info("工具执行结果",
		"tool", toolName,
		"result", result.Content)

	return result.Content, nil
}

// convertToolDefinitions 将 tools.ToolDefinition 转换为 providers.Tool。
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

// IsRunning 返回循环是否正在运行。
func (l *Loop) IsRunning() bool {
	return l.running.Load()
}

// ProcessDirect 直接处理消息，不经过消息总线。
// 适用于 CLI 或 API 交互。
func (l *Loop) ProcessDirect(
	ctx context.Context,
	content, sessionID string,
) (string, error) {
	return l.ProcessDirectWithChannel(ctx, content, sessionID, "cli", sessionID)
}

// ProcessDirectWithChannel 直接处理消息，指定渠道和会话ID。
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

// RegisterTool 向循环的工具注册表注册工具。
func (l *Loop) RegisterTool(tool tools.Tool) {
	l.tools.Register(tool)
}

// SetChannelManager 设置渠道管理器。
func (l *Loop) SetChannelManager(cm *channels.Manager) {
	l.channelManager = cm
}

// ProcessStreamWithChannel 处理消息并返回流式响应。
func (l *Loop) ProcessStreamWithChannel(
	ctx context.Context,
	content, sessionID, channel, _ string,
	callback StreamCallback,
) error {
	msg := bus.InboundMessage{
		Channel:   channel,
		SessionID: sessionID,
		Sender: bus.SenderInfo{
			ID: "stream",
		},
		Text:      content,
		Timestamp: time.Now(),
	}

	return l.processStreamWithAgent(ctx, "default", msg, callback)
}

// processStreamWithAgent 使用流式响应处理消息。
func (l *Loop) processStreamWithAgent(ctx context.Context, agentName string, msg bus.InboundMessage, callback StreamCallback) error {
	l.logger.With("name", "【智能体】").Info("使用代理处理(流式)",
		"agent", agentName,
		"channel", msg.Channel,
		"session_id", msg.SessionID)

	// 1. Build session key (format: channel:sessionID) 会话键用于在会话中唯一标识一个会话，用于存储和检索会话状态。
	sessionKey := consts.GetSessionKey(msg.Channel, msg.SessionID)

	// 2. Load memory/history 从记忆中加载会话历史记录。
	var history []providers.ChatMessage
	if l.memory != nil {
		mem, err := l.memory.Load(ctx, sessionKey)
		if err != nil {
			l.logger.With("name", "【智能体】").Warn("加载记忆失败", "error", err, "session_key", sessionKey)
		} else {
			history = mem
		}
	}

	// 3. Build messages 构建 LLM 请求的消息列表。
	messages := l.buildMessages(history, msg)

	// 4. Save user message to memory 保存用户消息到记忆中。
	if l.memory != nil {
		if err := l.memory.Save(ctx, sessionKey, consts.RoleUser.ToString(), msg.Text); err != nil {
			l.logger.With("name", "【智能体】").Warn("保存用户消息失败", "error", err)
		}
	}

	// 5. Run LLM iteration loop with streaming 运行 LLM 迭代循环，支持流式响应。
	finalContent, iteration, err := l.runLLMIterationStream(ctx, messages, msg, callback)
	if err != nil {
		return err
	}

	// 6. Handle empty response 处理空响应。
	if finalContent == "" {
		finalContent = defaultResponse
	}

	// 7. Save final assistant message to memory 保存助手消息到记忆中。
	if l.memory != nil {
		if err := l.memory.Save(ctx, sessionKey, consts.RoleAssistant.ToString(), finalContent); err != nil {
			l.logger.With("name", "【智能体】").Warn("保存助手消息失败", "error", err)
		}
	}

	// 8. Send final done chunk 发送最终完成块。
	if callback != nil {
		if err := callback(StreamChunk{Content: "", Done: true}); err != nil {
			return err
		}
	}

	l.logger.With("name", "【智能体】").Info("流式响应已完成",
		"agent", agentName,
		"session_key", sessionKey,
		"iterations", iteration,
		"content_length", len(finalContent))

	return nil
}

// runLLMIterationStream 运行 LLM 迭代循环，支持流式响应。
func (l *Loop) runLLMIterationStream(
	ctx context.Context,
	messages []providers.ChatMessage,
	msg bus.InboundMessage,
	callback StreamCallback,
) (string, int, error) {
	// Check if provider is configured
	if l.provider == nil && l.fallbackChain == nil && l.providerFactory == nil {
		l.logger.With("name", "【智能体】").Error("未配置AI提供商")
		return "", 0, fmt.Errorf("未配置AI提供商，请在设置中配置默认模型")
	}

	// Get the provider to use
	provider := l.GetDefaultProvider()
	if provider == nil {
		l.logger.With("name", "【智能体】").Error("无法获取有效的AI提供商")
		return "", 0, fmt.Errorf("无法获取有效的AI提供商")
	}

	// Get model name (try dynamic first, then fallback to provider default)
	modelName := ""
	if l.providerFactory != nil && l.storage != nil {
		_, model, err := l.GetDynamicProvider()
		if err == nil {
			modelName = model
		}
	}
	if modelName == "" {
		modelName = provider.GetDefaultModel()
	}

	iteration := 0
	currentMessages := messages
	var finalContent string

	for iteration < l.maxToolIterations {
		iteration++

		// Build request
		req := providers.ChatRequest{
			Model:    modelName,
			Messages: currentMessages,
		}

		// Add tools if available
		if toolDefs := l.tools.ToProviderDefs(); len(toolDefs) > 0 {
			req.Tools = l.convertToolDefinitions(toolDefs)
		}

		l.logger.With("name", "【智能体】").Debug("正在发送流式请求到LLM",
			"iteration", iteration,
			"message_count", len(currentMessages))

		// Collect tool calls during streaming
		var collectedToolCalls []providers.ToolCall
		var collectedContent string
		var collectedReasoning string

		// Send streaming request
		err := provider.ChatStream(ctx, req, func(chunk string, reasoning string, toolCalls []providers.ToolCall, done bool) error {
			// Collect content
			collectedContent += chunk
			collectedReasoning += reasoning

			// Collect tool calls
			if len(toolCalls) > 0 {
				collectedToolCalls = append(collectedToolCalls, toolCalls...)
			}

			// Send chunk to callback (only content chunks, not done signal)
			if callback != nil && chunk != "" {
				if err := callback(StreamChunk{
					Content:   chunk,
					Reasoning: reasoning,
					Done:      false,
				}); err != nil {
					return err
				}
			}

			return nil
		})

		if err != nil {
			l.logger.With("name", "【智能体】").Error("LLM流式请求失败", "error", err, "iteration", iteration)
			errMsg := fmt.Sprintf("LLM请求失败: %v", err)
			if errors.Is(err, context.DeadlineExceeded) {
				errMsg = "请求超时，AI服务响应时间过长，请稍后重试"
			}
			return "", iteration, fmt.Errorf("%s", errMsg)
		}

		// Handle tool calls
		if len(collectedToolCalls) > 0 {
			l.logger.With("name", "【智能体】").Info("正在处理工具调用(流式)",
				"count", len(collectedToolCalls),
				"iteration", iteration)

			// Merge tool calls with same ID
			mergedToolCalls := l.mergeToolCalls(collectedToolCalls)

			// Validate tool calls before adding to messages
			validToolCalls := make([]providers.ToolCall, 0, len(mergedToolCalls))
			for _, tc := range mergedToolCalls {
				// Skip tool calls with empty name
				if tc.Function.Name == "" {
					l.logger.Warn("跳过无效工具调用：缺少工具名称", "id", tc.ID)
					continue
				}
				// Ensure arguments is valid JSON or empty object
				if tc.Function.Arguments == "" {
					tc.Function.Arguments = "{}"
				} else if !isValidJSON(tc.Function.Arguments) {
					l.logger.Warn("工具调用参数不是有效JSON，尝试修复",
						"name", tc.Function.Name,
						"arguments", tc.Function.Arguments)
					// Try to fix common issues
					tc.Function.Arguments = fixJSONArguments(tc.Function.Arguments)
				}
				validToolCalls = append(validToolCalls, tc)
			}

			if len(validToolCalls) == 0 {
				// No valid tool calls, treat as normal response
				finalContent = collectedContent
				return finalContent, iteration, nil
			}

			// Add assistant message with tool calls
			assistantMsg := providers.ChatMessage{
				Role:      consts.RoleAssistant.ToString(),
				Content:   collectedContent,
				ToolCalls: validToolCalls,
			}
			currentMessages = append(currentMessages, assistantMsg)

			// Execute each tool call
			for _, tc := range mergedToolCalls {
				toolResult, err := l.executeToolCall(ctx, tc, msg)
				if err != nil {
					toolResult = fmt.Sprintf("错误: %v", err)
				}

				// Add tool result message
				currentMessages = append(currentMessages, providers.ChatMessage{
					Role:       consts.RoleTool.ToString(),
					Content:    toolResult,
					ToolCallID: tc.ID,
				})
			}

			// Continue to next iteration for LLM to process tool results
			continue
		}

		// No tool calls, we have the final response
		finalContent = collectedContent
		l.logger.With("name", "【智能体】").Debug("LLM流式响应已完成",
			"iteration", iteration,
			"content_length", len(finalContent))

		return finalContent, iteration, nil
	}

	// Max iterations reached
	l.logger.With("name", "【智能体】").Warn("已达到最大工具迭代次数",
		"iterations", l.maxToolIterations)

	return "", iteration, fmt.Errorf("已达到最大工具迭代次数 (%d)", l.maxToolIterations)
}

// mergeToolCalls 合并流式响应中的工具调用。
// 在流式响应中，工具调用分块传输，使用 index 作为标识符（存储在 ID 中为 "stream_index:N"）。
// 此函数按流式 ID 合并它们，并在需要时生成正确的 ID。
func (l *Loop) mergeToolCalls(toolCalls []providers.ToolCall) []providers.ToolCall {
	if len(toolCalls) == 0 {
		return nil
	}

	// Merge by ID (which contains stream_index or real ID)
	merged := make(map[string]*providers.ToolCall)
	for _, tc := range toolCalls {
		key := tc.ID
		if key == "" {
			continue
		}

		if existing, ok := merged[key]; ok {
			// Merge pieces
			if tc.Function.Name != "" {
				existing.Function.Name = tc.Function.Name
			}
			if tc.ID != "" && !isStreamIndexID(tc.ID) {
				// Real ID, update it
				existing.ID = tc.ID
			}
			existing.Function.Arguments += tc.Function.Arguments
		} else {
			// Create new entry
			copy := tc
			merged[key] = &copy
		}
	}

	// Convert to result and fix IDs
	result := make([]providers.ToolCall, 0, len(merged))
	for _, tc := range merged {
		// Skip tool calls without a name (invalid/incomplete)
		if tc.Function.Name == "" {
			l.logger.Warn("跳过无效工具调用：缺少工具名称", "id", tc.ID)
			continue
		}

		// Generate a proper ID if it's still a stream index ID
		id := tc.ID
		if isStreamIndexID(id) {
			id = fmt.Sprintf("call_%s_%d", tc.Function.Name, time.Now().UnixNano())
		}

		result = append(result, providers.ToolCall{
			ID:   id,
			Type: "function",
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		})
	}

	l.logger.Debug("合并工具调用完成",
		"input_count", len(toolCalls),
		"output_count", len(result))

	return result
}

// isStreamIndexID 检查 ID 是否为临时的流式索引 ID。
func isStreamIndexID(id string) bool {
	return len(id) > 12 && id[:12] == "stream_index"
}

// isValidJSON 检查字符串是否为有效的 JSON。
func isValidJSON(s string) bool {
	var js interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}

// fixJSONArguments 尝试修复流式响应中不完整的 JSON 参数。
func fixJSONArguments(s string) string {
	s = strings.TrimSpace(s)

	// If empty, return empty object
	if s == "" {
		return "{}"
	}

	// Try to parse as-is first
	if isValidJSON(s) {
		return s
	}

	// Common fixes for incomplete streaming JSON

	// Fix unclosed string
	if strings.Count(s, "\"")%2 == 1 {
		s += "\""
	}

	// Fix unclosed braces
	openBraces := strings.Count(s, "{") - strings.Count(s, "}")
	for i := 0; i < openBraces; i++ {
		s += "}"
	}

	// Fix unclosed brackets
	openBrackets := strings.Count(s, "[") - strings.Count(s, "]")
	for i := 0; i < openBrackets; i++ {
		s += "]"
	}

	// Try again
	if isValidJSON(s) {
		return s
	}

	// Last resort: return empty object
	return "{}"
}
