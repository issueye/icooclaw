package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"icooclaw.ai/hooks"
	"icooclaw.ai/provider"
	"icooclaw.ai/tools"
	"icooclaw.core/consts"
	"icooclaw.core/storage"
)

// ============ ReAct Agent ============

// ReActConfig ReAct 配置
type ReActConfig struct {
	MaxIterations int               // 最大迭代次数，默认 10
	Provider      provider.Provider // LLM 提供者
	Tools         *tools.Registry   // 工具注册表
	Session       *storage.Session  // 会话
	Logger        *slog.Logger      // 日志
	Hooks         hooks.ReActHooks  // 钩子接口
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
		reasoningContent: "",
		hooks:            hooks,
		logger:           logger,
		toolCallsData:    make(map[int]*CallData),
	}

	// 创建流式回调 (在循环外创建，避免重复创建)
	streamCallback := r.createStreamCallback(streamState)

	for iteration := 0; iteration < cfg.MaxIterations; iteration++ {
		// 重置状态
		streamState.content = ""
		streamState.reasoningContent = ""
		streamState.isThinking = false
		streamState.isKimiThinking = false
		// 清空 map 而非重新创建，减少内存分配
		for key := range streamState.toolCallsData {
			delete(streamState.toolCallsData, key)
		}

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
		if len(toolCalls) > 0 {
			// 添加 assistant 消息（包含所有 tool_calls 且无空 content，符合 OpenAI 标准形式）
			messages = append(messages, provider.Message{
				Role:      consts.RoleAssistant,
				Content:   content,
				ToolCalls: toolCalls,
			})

			// 并发执行所有的工具
			var wg sync.WaitGroup
			results := make([]provider.Message, len(toolCalls))

			for i, call := range toolCalls {
				wg.Add(1)
				go func(index int, c provider.ToolCall) {
					defer wg.Done()
					toolName := c.Function.Name
					arguments := c.Function.Arguments

					logger.Info("执行工具", "tool", toolName, "call_id", c.ID)

					// 触发工具调用前钩子
					if err := hooks.OnToolCall(ctx, c.ID, toolName, arguments); err != nil {
						logger.Warn("工具调用前钩子报错", "error", err)
					}

					// 执行工具
					result := cfg.Tools.Execute(ctx, c)
					if result.Error != nil {
						logger.Error("工具执行报错", "tool", toolName, "error", result.Error)
					}

					logger.Debug(fmt.Sprintf("工具调用[%s]\n参数[%s]\n结果[%s]", toolName, arguments, result.Content))

					// 触发工具结果钩子
					if err := hooks.OnToolResult(ctx, c.ID, toolName, result); err != nil {
						logger.Warn("工具结果钩子报错", "error", err)
					}

					// 添加工具结果到消息
					var resultContent string
					if result.Error != nil {
						resultContent = fmt.Sprintf("Error: %v", result.Error)
					} else {
						resultContent = result.Content
					}

					results[index] = provider.Message{
						Role:       consts.RoleToolResult,
						Content:    resultContent,
						ToolCallID: c.ID,
					}
				}(i, call)
			}
			wg.Wait()

			// 按顺序把结果追加到 messages
			messages = append(messages, results...)
		}

		// 触发迭代结束钩子
		if err := hooks.OnIterationEnd(ctx, iteration, true); err != nil {
			logger.Warn("迭代结束钩子报错", "error", err)
		}
	}

	return "", "", nil, fmt.Errorf("最大迭代次数 %d 已超过", cfg.MaxIterations)
}

// ============ 流式回调状态管理 ============

type CallData struct {
	id   string
	typ  string
	name string
	args strings.Builder
}

// streamCallbackState 流式回调状态
type streamCallbackState struct {
	content          string
	reasoningContent string
	hooks            hooks.ReActHooks
	logger           *slog.Logger
	// 工具调用累积（按 index 分组）
	toolCallsData map[int]*CallData

	// 思考状态追踪
	isThinking     bool
	isKimiThinking bool
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
			cleanContent := chunk.Content
			var thinking string

			// 处理 DeepSeek 的 <think> 标签流式拦截
			if state.isThinking {
				if endIdx := strings.Index(cleanContent, "</think>"); endIdx != -1 {
					state.isThinking = false
					thinking += cleanContent[:endIdx]
					cleanContent = cleanContent[endIdx+len("</think>"):]
				} else {
					thinking += cleanContent
					cleanContent = ""
				}
			} else if startIdx := strings.Index(cleanContent, "<think>"); startIdx != -1 {
				state.isThinking = true
				searchStr := cleanContent[startIdx+len("<think>"):]
				if endIdx := strings.Index(searchStr, "</think>"); endIdx != -1 {
					state.isThinking = false
					thinking += searchStr[:endIdx]
					cleanContent = cleanContent[:startIdx] + searchStr[endIdx+len("</think>"):]
				} else {
					thinking += searchStr
					cleanContent = cleanContent[:startIdx]
				}
			}

			// 处理 Kimi 的 reasoning 标签流式拦截
			kimiStart := "<|start_header_id|>reasoning<|end_header_id|>"
			kimiEnd := "<|start_header_id|>assistant<|end_header_id|>"
			if state.isKimiThinking {
				if endIdx := strings.Index(cleanContent, kimiEnd); endIdx != -1 {
					state.isKimiThinking = false
					thinking += cleanContent[:endIdx]
					cleanContent = cleanContent[endIdx+len(kimiEnd):]
				} else {
					thinking += cleanContent
					cleanContent = ""
				}
			} else if startIdx := strings.Index(cleanContent, kimiStart); startIdx != -1 {
				state.isKimiThinking = true
				searchStr := cleanContent[startIdx+len(kimiStart):]
				if endIdx := strings.Index(searchStr, kimiEnd); endIdx != -1 {
					state.isKimiThinking = false
					thinking += searchStr[:endIdx]
					cleanContent = cleanContent[:startIdx] + searchStr[endIdx+len(kimiEnd):]
				} else {
					thinking += searchStr
					cleanContent = cleanContent[:startIdx]
				}
			}

			// 累积思考内容
			if thinking != "" {
				state.reasoningContent += thinking
				// 触发思考内容更新钩子
				if err := state.hooks.OnLLMChunk(context.Background(), "", state.reasoningContent); err != nil {
					state.logger.Warn("思考内容更新钩子报错", "error", err)
				}
			}

			// 累积正式内容
			if cleanContent != "" {
				state.content += cleanContent
				// 触发流式输出钩子
				if err := state.hooks.OnLLMChunk(context.Background(), cleanContent, state.reasoningContent); err != nil {
					state.logger.Warn("流式输出钩子报错", "error", err)
				}
			}
		}

		// 处理工具调用 - StreamToolCall 需要按 index 分组累积
		for _, tc := range chunk.ToolCalls {
			index := tc.Index
			tcData, exists := state.toolCallsData[index]
			if exists {
				// 累积 arguments
				if tc.Arguments != "" {
					tcData.args.WriteString(tc.Arguments)
				}
				// 更新 ID 和 Type
				if tc.ID != "" {
					tcData.id = tc.ID
				}
				if tc.Type != "" {
					tcData.typ = tc.Type
				}
				if tc.Name != "" {
					tcData.name = tc.Name
				}
			} else {
				tcData := &CallData{
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
			{Role: consts.RoleSystem, Content: systemPrompt},
		}, messages...)
	}
	req.Messages = messages

	return req
}
