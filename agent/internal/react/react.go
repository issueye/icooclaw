package react

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/icooclaw/icooclaw/consts"
	"github.com/icooclaw/icooclaw/internal/agent"
	"github.com/icooclaw/icooclaw/internal/hooks"
	"github.com/icooclaw/icooclaw/internal/provider"
	"github.com/icooclaw/icooclaw/internal/storage"
	"github.com/icooclaw/icooclaw/internal/tools"
)

// ============ ReAct Agent ============

// ReActConfig ReAct 配置
type ReActConfig struct {
	MaxIterations int // 最大迭代次数，默认 10
	Provider      provider.Provider
	Tools         *tools.Registry
	Session       *storage.Session
	Logger        *slog.Logger
	Hooks         hooks.ReActHooks
}

// NewReActConfig 创建默认配置
func NewReActConfig() *ReActConfig {
	return &ReActConfig{
		MaxIterations: 10,
		Hooks:         &hooks.DefaultHooks{},
	}
}

// ReActAgent ReAct 模式的 Agent 实现
type ReActAgent struct {
	config *ReActConfig
	Agent  *agent.Agent
}

// NewReActAgent 创建 ReAct Agent
func NewReActAgent(config *ReActConfig) *ReActAgent {
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	if config.Hooks == nil {
		config.Hooks = &hooks.DefaultHooks{}
	}
	if config.MaxIterations <= 0 {
		config.MaxIterations = 20
	}
	return &ReActAgent{
		config: config,
	}
}

// Run 运行 ReAct 循环
// 返回: 内容, 思考内容, 工具调用, 错误
func (r *ReActAgent) Run(ctx context.Context, messages []provider.Message, systemPrompt string) (string, string, []provider.ToolCall, error) {
	cfg := r.config
	logger := cfg.Logger
	hooks := cfg.Hooks

	// 创建流式回调状态
	streamState := &streamCallbackState{
		content:          "",
		toolCalls:        nil,
		reasoningContent: "",
		hooks:            hooks,
		logger:           logger,
		toolCallsData: make(map[int]*struct {
			id   string
			typ  string
			name string
			args strings.Builder
		}),
	}

	// 创建流式回调 (在循环外创建，避免重复创建)
	streamCallback := r.createStreamCallback(streamState)

	for iteration := 0; iteration < cfg.MaxIterations; iteration++ {
		// 重置状态
		streamState.content = ""
		streamState.reasoningContent = ""
		streamState.toolCallsData = make(map[int]*struct {
			id   string
			typ  string
			name string
			args strings.Builder
		})

		// 触发迭代开始钩子
		if err := hooks.OnIterationStart(ctx, iteration, messages); err != nil {
			logger.Warn("迭代开始钩子报错", "error", err)
		}

		// 构建请求
		req := r.buildRequest(ctx, messages, systemPrompt)

		// 触发 LLM 请求钩子
		if err := hooks.OnLLMRequest(ctx, req, iteration); err != nil {
			logger.Warn("LLM 请求钩子报错", "error", err)
			return "", "", nil, err
		}

		logger.Debug("发送请求到 LLM", "iteration", iteration, "message_count", len(messages))

		// 调用 LLM
		err := cfg.Provider.ChatStream(ctx, *req, streamCallback)
		if err != nil {
			hooks.OnError(ctx, fmt.Errorf("LLM 流式调用报错: %w", err))
			return "", "", nil, err
		}

		content := streamState.content
		reasoningContent := streamState.reasoningContent

		// 将 toolCallsData 转换为 toolCalls
		toolCalls := make([]provider.ToolCall, 0, len(streamState.toolCallsData))
		for _, tc := range streamState.toolCallsData {
			toolCalls = append(toolCalls, provider.ToolCall{
				ID:   tc.id,
				Type: tc.typ,
				Function: provider.ToolCallFunction{
					Name:      tc.name,
					Arguments: tc.args.String(),
				},
			})
		}

		// 触发 LLM 响应钩子
		if err := hooks.OnLLMResponse(ctx, content, reasoningContent, toolCalls, iteration); err != nil {
			logger.Warn("LLM 响应钩子报错", "error", err)
		}

		logger.Info("LLM 响应", "iteration", iteration, "content_length", len(content), "tool_calls", len(toolCalls), "content_preview", content[:min(200, len(content))])

		// 调试：打印原始 toolCallsData
		logger.Debug("工具调用数据调试", "count", len(streamState.toolCallsData), "data", streamState.toolCallsData)

		// 检查是否需要执行工具
		if len(toolCalls) == 0 {
			// 没有工具调用，结束循环
			logger.Info("Agent 循环完成", "iterations", iteration+1)

			// 触发迭代结束钩子
			if err := hooks.OnIterationEnd(ctx, iteration, false); err != nil {
				logger.Warn("迭代结束钩子报错", "error", err)
			}

			// 触发完成钩子
			if err := hooks.OnComplete(ctx, content, reasoningContent, toolCalls, iteration+1); err != nil {
				logger.Warn("完成钩子报错", "error", err)
			}

			return content, reasoningContent, toolCalls, nil
		}

		// 执行工具调用
		for _, call := range toolCalls {
			toolName := call.Function.Name
			arguments := call.Function.Arguments

			logger.Info("执行工具", "tool", toolName, "call_id", call.ID)

			// 触发工具调用前钩子
			if err := hooks.OnToolCall(ctx, call.ID, toolName, arguments); err != nil {
				logger.Warn("工具调用前钩子报错", "error", err)
			}

			// 执行工具
			toolCall := tools.ToolCall{
				ID:   call.ID,
				Type: call.Type,
				Function: tools.ToolCallFunction{
					Name:      toolName,
					Arguments: arguments,
				},
			}
			result := cfg.Tools.Execute(ctx, toolCall)
			if result.Error != nil {
				logger.Error("工具执行报错", "tool", toolName, "error", result.Error)
			}

			logger.Debug(fmt.Sprintf("工具调用[%s]\n参数[%s]\n结果[%s]", toolName, arguments, result.Content))

			// 触发工具结果钩子
			if err := hooks.OnToolResult(ctx, call.ID, toolName, result); err != nil {
				logger.Warn("工具结果钩子报错", "error", err)
			}

			// 添加工具结果到消息
			var resultContent string
			if result.Error != nil {
				resultContent = fmt.Sprintf("Error: %v", result.Error)
			} else {
				resultContent = result.Content
			}

			// 添加 assistant 消息（包含 tool_calls）
			messages = append(messages, provider.Message{
				Role:      consts.RoleAssistant.ToString(),
				Content:   content,
				ToolCalls: []provider.ToolCall{call},
			})

			// 添加工具结果消息
			messages = append(messages, provider.Message{
				Role:       consts.RoleTool.ToString(),
				Content:    resultContent,
				ToolCallID: call.ID,
			})
		}

		// 触发迭代结束钩子
		if err := hooks.OnIterationEnd(ctx, iteration, true); err != nil {
			logger.Warn("迭代结束钩子报错", "error", err)
		}
	}

	return "", "", nil, fmt.Errorf("最大迭代次数 %d 已超过", cfg.MaxIterations)
}

// ============ 流式回调状态管理 ============

// streamCallbackState 流式回调状态
type streamCallbackState struct {
	content          string
	toolCalls        []provider.ToolCall
	reasoningContent string
	hooks            hooks.ReActHooks
	logger           *slog.Logger
	// 工具调用累积（按 index 分组）
	toolCallsData map[int]*struct {
		id   string
		typ  string
		name string
		args strings.Builder
	}
}

// createStreamCallback 创建流式回调函数 (在循环外创建)
func (r *ReActAgent) createStreamCallback(state *streamCallbackState) provider.StreamCallback {
	return func(chunk provider.StreamChunk) error {
		// 处理独立的 reasoning_content 字段（如 OpenAI o1）
		if chunk.ReasoningContent != "" {
			state.reasoningContent += chunk.ReasoningContent
			// 触发思考内容更新钩子
			if err := state.hooks.OnLLMChunk(context.Background(), "", state.reasoningContent); err != nil {
				state.logger.Warn("思考内容更新钩子报错", "error", err)
			}
		}

		if chunk.Content != "" {
			// 使用统一的解析器提取思考内容（如 DeepSeek 的 <think> 标签）
			cleanContent, thinking := provider.ExtractThinkingContent(chunk.Content, "")

			// 累积思考内容
			if thinking != "" {
				state.reasoningContent += thinking
			}

			// 触发流式输出钩子（发送累积的思考内容）
			if err := state.hooks.OnLLMChunk(context.Background(), cleanContent, state.reasoningContent); err != nil {
				state.logger.Warn("流式输出钩子报错", "error", err)
			}

			state.content += cleanContent
		}

		// 处理工具调用 - StreamToolCall 需要按 index 分组累积
		for _, tc := range chunk.ToolCalls {
			index := tc.Index
			if existing, ok := state.toolCallsData[index]; ok {
				// 累积 arguments
				if tc.Arguments != "" {
					existing.args.WriteString(tc.Arguments)
				}
				// 更新 ID 和 Type
				if tc.ID != "" {
					existing.id = tc.ID
				}
				if tc.Type != "" {
					existing.typ = tc.Type
				}
				if tc.Name != "" {
					existing.name = tc.Name
				}
			} else {
				tcData := &struct {
					id   string
					typ  string
					name string
					args strings.Builder
				}{
					id:   tc.ID,
					typ:  tc.Type,
					name: tc.Name,
				}
				tcData.args.WriteString(tc.Arguments)
				state.toolCallsData[index] = tcData
			}
		}

		return nil
	}
}

// buildRequest 构建 LLM 请求
func (r *ReActAgent) buildRequest(ctx context.Context, messages []provider.Message, systemPrompt string) *provider.ChatRequest {
	cfg := r.config
	toolDefs := cfg.Tools.ToDefinitions()

	req := &provider.ChatRequest{
		Model:       cfg.Provider.GetDefaultModel(),
		Temperature: 0.7,
		MaxTokens:   4096,
	}

	// 添加工具定义
	if len(toolDefs) > 0 {
		tools := make([]provider.ToolDefinition, len(toolDefs))
		for i, t := range toolDefs {
			tools[i] = provider.ToolDefinition{
				Type: t.Type,
				Function: provider.FunctionDefinition{
					Name:        t.Function.Name,
					Description: t.Function.Description,
					Parameters:  t.Function.Parameters,
				},
			}
		}
		req.Tools = tools
	}

	// 添加系统提示词
	if systemPrompt != "" {
		messages = append([]provider.Message{
			{Role: consts.RoleSystem.ToString(), Content: systemPrompt},
		}, messages...)
	}
	req.Messages = messages

	return req
}
