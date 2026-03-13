// Package agent provides the core agent implementation for icooclaw.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"icooclaw/pkg/hooks"
	"icooclaw/pkg/providers"
	"icooclaw/pkg/tools"
)

// ReActStep 表示 ReAct 循环中的一个步骤。
type ReActStep struct {
	Thought     string         `json:"thought,omitempty"`      // 思考过程
	Action      string         `json:"action,omitempty"`       // 行动（工具名称）
	ActionInput map[string]any `json:"action_input,omitempty"` // 行动输入（工具参数）
	Observation string         `json:"observation,omitempty"`  // 观察（工具执行结果）
}

// ReActResult 表示 ReAct 循环的最终结果。
type ReActResult struct {
	FinalAnswer string        `json:"final_answer"`    // 最终答案
	Steps       []ReActStep   `json:"steps"`           // 执行步骤
	Iterations  int           `json:"iterations"`      // 迭代次数
	Success     bool          `json:"success"`         // 是否成功
	Error       error         `json:"error,omitempty"` // 错误信息
	Duration    time.Duration `json:"duration"`        // 执行时长
}

// ReActStreamChunk 表示流式响应的一个数据块。
type ReActStreamChunk struct {
	Type        string `json:"type"`                  // chunk 类型: thought, action, observation, content, done, error
	Content     string `json:"content,omitempty"`     // 内容
	Thought     string `json:"thought,omitempty"`     // 思考内容
	Action      string `json:"action,omitempty"`      // 行动名称
	Observation string `json:"observation,omitempty"` // 观察结果
	Done        bool   `json:"done,omitempty"`        // 是否完成
	Error       error  `json:"error,omitempty"`       // 错误信息
	Iteration   int    `json:"iteration,omitempty"`   // 当前迭代次数
}

// ReActStreamCallback 流式响应的回调函数。
type ReActStreamCallback func(chunk ReActStreamChunk) error

// ReActConfig ReAct 单元配置。
type ReActConfig struct {
	MaxIterations  int           `json:"max_iterations"`  // 最大迭代次数
	Timeout        time.Duration `json:"timeout"`         // 单次执行超时
	SystemPrompt   string        `json:"system_prompt"`   // 系统提示词
	ThoughtTag     string        `json:"thought_tag"`     // 思考标签
	ActionTag      string        `json:"action_tag"`      // 行动标签
	ObservationTag string        `json:"observation_tag"` // 观察标签
	AnswerTag      string        `json:"answer_tag"`      // 答案标签
}

// DefaultReActConfig 返回默认配置。
func DefaultReActConfig() ReActConfig {
	return ReActConfig{
		MaxIterations:  10,
		Timeout:        60 * time.Second,
		SystemPrompt:   defaultReActPrompt,
		ThoughtTag:     "Thought",
		ActionTag:      "Action",
		ObservationTag: "Observation",
		AnswerTag:      "Final Answer",
	}
}

// ReAct 是一个独立的 ReAct 推理单元。
// 它实现了 Reasoning + Acting 模式，通过迭代思考和行动来解决问题。
type ReAct struct {
	config        ReActConfig
	provider      providers.Provider
	tools         *tools.Registry
	logger        *slog.Logger
	reactHooks    hooks.ReActHooks
	providerHooks hooks.ProviderHooks
	agentHooks    hooks.AgentHooks
	mu            sync.Mutex
}

// NewReAct 创建一个新的 ReAct 单元。
func NewReAct(opts ...ReActOption) *ReAct {
	u := &ReAct{
		config: DefaultReActConfig(),
		tools:  tools.NewRegistry(),
		logger: slog.Default(),
	}

	for _, opt := range opts {
		opt(u)
	}

	return u
}

// ReActOption ReAct 单元的函数式选项。
type ReActOption func(*ReAct)

// WithReActProvider 设置提供商。
func WithReActProvider(p providers.Provider) ReActOption {
	return func(u *ReAct) { u.provider = p }
}

// WithReActTools 设置工具注册表。
func WithReActTools(t *tools.Registry) ReActOption {
	return func(u *ReAct) { u.tools = t }
}

// WithReActLogger 设置日志器。
func WithReActLogger(l *slog.Logger) ReActOption {
	return func(u *ReAct) { u.logger = l }
}

// WithReActConfig 设置配置。
func WithReActConfig(c ReActConfig) ReActOption {
	return func(u *ReAct) { u.config = c }
}

// WithReActMaxIterations 设置最大迭代次数。
func WithReActMaxIterations(n int) ReActOption {
	return func(u *ReAct) {
		if n > 0 {
			u.config.MaxIterations = n
		}
	}
}

// WithReActTimeout 设置超时时间。
func WithReActTimeout(d time.Duration) ReActOption {
	return func(u *ReAct) {
		if d > 0 {
			u.config.Timeout = d
		}
	}
}

// WithReActSystemPrompt 设置系统提示词。
func WithReActSystemPrompt(prompt string) ReActOption {
	return func(u *ReAct) { u.config.SystemPrompt = prompt }
}

// Run 执行 ReAct 循环（非流式）。
func (u *ReAct) Run(ctx context.Context, query string) *ReActResult {
	start := time.Now()
	result := &ReActResult{
		Steps:   make([]ReActStep, 0),
		Success: false,
	}

	defer func() {
		result.Duration = time.Since(start)
	}()

	// 检查提供商
	if u.provider == nil {
		result.Error = fmt.Errorf("未配置 AI 提供商")
		return result
	}

	// 构建初始消息
	messages := u.buildMessages(query)

	if u.agentHooks != nil {
	}

	// 迭代执行
	for i := 0; i < u.config.MaxIterations; i++ {
		result.Iterations = i + 1

		// 设置超时
		ctx, cancel := context.WithTimeout(ctx, u.config.Timeout)
		defer cancel()

		// 调用 LLM
		response, err := u.callLLM(ctx, messages)
		if err != nil {
			result.Error = fmt.Errorf("LLM 调用失败: %w", err)
			return result
		}

		// 解析响应
		step, finalAnswer := u.parseResponse(response)

		// 检查是否有最终答案
		if finalAnswer != "" {
			result.FinalAnswer = finalAnswer
			result.Success = true
			u.logger.Info("ReAct 完成",
				"iterations", result.Iterations,
				"duration", result.Duration)
			return result
		}

		// 记录步骤
		result.Steps = append(result.Steps, step)

		// 执行工具
		if step.Action != "" {
			observation := u.executeTool(ctx, step.Action, step.ActionInput)
			step.Observation = observation

			// 将观察结果添加到消息中
			messages = append(messages,
				providers.ChatMessage{
					Role:    "assistant",
					Content: u.formatStep(step),
				},
				providers.ChatMessage{
					Role:    "user",
					Content: fmt.Sprintf("%s: %s", u.config.ObservationTag, observation),
				},
			)
		} else {
			// 没有行动，可能是格式问题
			result.Error = fmt.Errorf("无法解析行动")
			return result
		}
	}

	// 达到最大迭代次数
	result.Error = fmt.Errorf("达到最大迭代次数 (%d)", u.config.MaxIterations)
	return result
}

// RunStream 执行 ReAct 循环（流式）。
func (u *ReAct) RunStream(ctx context.Context, query string, callback ReActStreamCallback) *ReActResult {
	start := time.Now()
	result := &ReActResult{
		Steps:   make([]ReActStep, 0),
		Success: false,
	}

	defer func() {
		result.Duration = time.Since(start)
	}()

	// 检查提供商
	if u.provider == nil {
		result.Error = fmt.Errorf("未配置 AI 提供商")
		if callback != nil {
			callback(ReActStreamChunk{Type: "error", Error: result.Error})
		}
		return result
	}

	// 构建初始消息
	messages := u.buildMessages(query)

	// 迭代执行
	for i := 0; i < u.config.MaxIterations; i++ {
		result.Iterations = i + 1

		// 设置超时
		ctx, cancel := context.WithTimeout(ctx, u.config.Timeout)
		defer cancel()

		// 调用 LLM（流式）
		response, err := u.callLLMStream(ctx, messages, callback, result.Iterations)
		if err != nil {
			result.Error = fmt.Errorf("LLM 调用失败: %w", err)
			if callback != nil {
				callback(ReActStreamChunk{Type: "error", Error: result.Error})
			}
			return result
		}

		// 解析响应
		step, finalAnswer := u.parseResponse(response)

		// 检查是否有最终答案
		if finalAnswer != "" {
			result.FinalAnswer = finalAnswer
			result.Success = true
			u.logger.Info("ReAct 完成",
				"iterations", result.Iterations,
				"duration", result.Duration)
			// 发送最终答案
			if callback != nil {
				callback(ReActStreamChunk{
					Type:    "content",
					Content: finalAnswer,
				})
				callback(ReActStreamChunk{
					Type:      "done",
					Done:      true,
					Iteration: result.Iterations,
				})
			}
			return result
		}

		// 记录步骤
		result.Steps = append(result.Steps, step)

		// 执行工具
		if step.Action != "" {
			// 发送行动信息
			if callback != nil {
				callback(ReActStreamChunk{
					Type:      "action",
					Action:    step.Action,
					Iteration: result.Iterations,
				})
			}

			observation := u.executeTool(ctx, step.Action, step.ActionInput)
			step.Observation = observation

			// 发送观察结果
			if callback != nil {
				callback(ReActStreamChunk{
					Type:        "observation",
					Observation: observation,
					Iteration:   result.Iterations,
				})
			}

			// 将观察结果添加到消息中
			messages = append(messages,
				providers.ChatMessage{
					Role:    "assistant",
					Content: u.formatStep(step),
				},
				providers.ChatMessage{
					Role:    "user",
					Content: fmt.Sprintf("%s: %s", u.config.ObservationTag, observation),
				},
			)
		} else {
			// 没有行动，可能是格式问题
			result.Error = fmt.Errorf("无法解析行动")
			if callback != nil {
				callback(ReActStreamChunk{Type: "error", Error: result.Error})
			}
			return result
		}
	}

	// 达到最大迭代次数
	result.Error = fmt.Errorf("达到最大迭代次数 (%d)", u.config.MaxIterations)
	if callback != nil {
		callback(ReActStreamChunk{Type: "error", Error: result.Error})
	}
	return result
}

// RunWithHistory 使用历史记录执行 ReAct 循环。
func (u *ReAct) RunWithHistory(ctx context.Context, query string, history []providers.ChatMessage) *ReActResult {
	// 将历史记录添加到消息中
	messages := u.buildMessagesWithHistory(query, history)

	start := time.Now()
	result := &ReActResult{
		Steps:   make([]ReActStep, 0),
		Success: false,
	}

	defer func() {
		result.Duration = time.Since(start)
	}()

	if u.provider == nil {
		result.Error = fmt.Errorf("未配置 AI 提供商")
		return result
	}

	for i := 0; i < u.config.MaxIterations; i++ {
		result.Iterations = i + 1

		ctx, cancel := context.WithTimeout(ctx, u.config.Timeout)
		defer cancel()

		response, err := u.callLLM(ctx, messages)
		if err != nil {
			result.Error = fmt.Errorf("LLM 调用失败: %w", err)
			return result
		}

		step, finalAnswer := u.parseResponse(response)

		if finalAnswer != "" {
			result.FinalAnswer = finalAnswer
			result.Success = true
			return result
		}

		result.Steps = append(result.Steps, step)

		if step.Action != "" {
			observation := u.executeTool(ctx, step.Action, step.ActionInput)
			step.Observation = observation

			messages = append(messages,
				providers.ChatMessage{
					Role:    "assistant",
					Content: u.formatStep(step),
				},
				providers.ChatMessage{
					Role:    "user",
					Content: fmt.Sprintf("%s: %s", u.config.ObservationTag, observation),
				},
			)
		} else {
			result.Error = fmt.Errorf("无法解析行动")
			return result
		}
	}

	result.Error = fmt.Errorf("达到最大迭代次数 (%d)", u.config.MaxIterations)
	return result
}

// RunStreamWithHistory 使用历史记录执行 ReAct 循环（流式）。
func (u *ReAct) RunStreamWithHistory(ctx context.Context, query string, history []providers.ChatMessage, callback ReActStreamCallback) *ReActResult {
	messages := u.buildMessagesWithHistory(query, history)

	start := time.Now()
	result := &ReActResult{
		Steps:   make([]ReActStep, 0),
		Success: false,
	}

	defer func() {
		result.Duration = time.Since(start)
	}()

	if u.provider == nil {
		result.Error = fmt.Errorf("未配置 AI 提供商")
		if callback != nil {
			callback(ReActStreamChunk{Type: "error", Error: result.Error})
		}
		return result
	}

	for i := 0; i < u.config.MaxIterations; i++ {
		result.Iterations = i + 1

		ctx, cancel := context.WithTimeout(ctx, u.config.Timeout)
		defer cancel()

		response, err := u.callLLMStream(ctx, messages, callback, result.Iterations)
		if err != nil {
			result.Error = fmt.Errorf("LLM 调用失败: %w", err)
			if callback != nil {
				callback(ReActStreamChunk{Type: "error", Error: result.Error})
			}
			return result
		}

		step, finalAnswer := u.parseResponse(response)

		if finalAnswer != "" {
			result.FinalAnswer = finalAnswer
			result.Success = true
			if callback != nil {
				callback(ReActStreamChunk{
					Type:    "content",
					Content: finalAnswer,
				})
				callback(ReActStreamChunk{
					Type:      "done",
					Done:      true,
					Iteration: result.Iterations,
				})
			}
			return result
		}

		result.Steps = append(result.Steps, step)

		if step.Action != "" {
			if callback != nil {
				callback(ReActStreamChunk{
					Type:      "action",
					Action:    step.Action,
					Iteration: result.Iterations,
				})
			}

			observation := u.executeTool(ctx, step.Action, step.ActionInput)
			step.Observation = observation

			if callback != nil {
				callback(ReActStreamChunk{
					Type:        "observation",
					Observation: observation,
					Iteration:   result.Iterations,
				})
			}

			messages = append(messages,
				providers.ChatMessage{
					Role:    "assistant",
					Content: u.formatStep(step),
				},
				providers.ChatMessage{
					Role:    "user",
					Content: fmt.Sprintf("%s: %s", u.config.ObservationTag, observation),
				},
			)
		} else {
			result.Error = fmt.Errorf("无法解析行动")
			if callback != nil {
				callback(ReActStreamChunk{Type: "error", Error: result.Error})
			}
			return result
		}
	}

	result.Error = fmt.Errorf("达到最大迭代次数 (%d)", u.config.MaxIterations)
	if callback != nil {
		callback(ReActStreamChunk{Type: "error", Error: result.Error})
	}
	return result
}

// buildMessages 构建初始消息列表。
func (u *ReAct) buildMessages(query string) []providers.ChatMessage {
	systemPrompt := u.config.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = defaultReActPrompt
	}

	// 添加工具描述
	toolDescriptions := u.getToolDescriptions()
	if toolDescriptions != "" {
		systemPrompt += "\n\n" + toolDescriptions
	}

	return []providers.ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: query},
	}
}

// buildMessagesWithHistory 构建包含历史记录的消息列表。
func (u *ReAct) buildMessagesWithHistory(query string, history []providers.ChatMessage) []providers.ChatMessage {
	messages := u.buildMessages(query)
	// 在用户消息之前插入历史记录
	if len(history) > 0 {
		newMessages := make([]providers.ChatMessage, 0, len(messages)+len(history))
		newMessages = append(newMessages, messages[0]) // system
		newMessages = append(newMessages, history...)  // history
		newMessages = append(newMessages, messages[1]) // user
		return newMessages
	}
	return messages
}

// getToolDescriptions 获取工具描述。
func (u *ReAct) getToolDescriptions() string {
	if u.tools == nil {
		return ""
	}

	toolList := u.tools.List()
	if len(toolList) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("可用工具:\n")
	for _, tool := range toolList {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name(), tool.Description()))
	}
	return sb.String()
}

// callLLM 调用 LLM（非流式）。
func (u *ReAct) callLLM(ctx context.Context, messages []providers.ChatMessage) (string, error) {
	req := providers.ChatRequest{
		Model:    u.provider.GetModel(),
		Messages: messages,
	}

	resp, err := u.provider.Chat(ctx, req)
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}

// callLLMStream 调用 LLM（流式）。
func (u *ReAct) callLLMStream(ctx context.Context, messages []providers.ChatMessage, callback ReActStreamCallback, iteration int) (string, error) {
	req := providers.ChatRequest{
		Model:    u.provider.GetModel(),
		Messages: messages,
	}

	var fullContent strings.Builder
	var fullReasoning strings.Builder

	err := u.provider.ChatStream(ctx, req, func(chunk string, reasoning string, toolCalls []providers.ToolCall, done bool) error {
		// 收集内容
		fullContent.WriteString(chunk)
		if reasoning != "" {
			fullReasoning.WriteString(reasoning)
		}

		// 发送思考内容（reasoning）
		if callback != nil && reasoning != "" {
			callback(ReActStreamChunk{
				Type:      "thought",
				Thought:   reasoning,
				Iteration: iteration,
			})
		}

		// 发送内容块
		if callback != nil && chunk != "" {
			callback(ReActStreamChunk{
				Type:      "content",
				Content:   chunk,
				Iteration: iteration,
			})
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	// 如果有 reasoning，将其添加到内容前面
	result := fullContent.String()
	if fullReasoning.Len() > 0 {
		result = fmt.Sprintf("Thought: %s\n%s", fullReasoning.String(), result)
	}

	return result, nil
}

// parseResponse 解析 LLM 响应。
func (u *ReAct) parseResponse(response string) (ReActStep, string) {
	step := ReActStep{}

	// 检查是否有最终答案
	if idx := strings.Index(response, u.config.AnswerTag+":"); idx != -1 {
		finalAnswer := strings.TrimSpace(response[idx+len(u.config.AnswerTag)+1:])
		return step, finalAnswer
	}

	// 解析思考
	if idx := strings.Index(response, u.config.ThoughtTag+":"); idx != -1 {
		rest := response[idx+len(u.config.ThoughtTag)+1:]
		if endIdx := strings.Index(rest, "\n"+u.config.ActionTag+":"); endIdx != -1 {
			step.Thought = strings.TrimSpace(rest[:endIdx])
		} else {
			step.Thought = strings.TrimSpace(rest)
		}
	}

	// 解析行动
	if idx := strings.Index(response, u.config.ActionTag+":"); idx != -1 {
		rest := response[idx+len(u.config.ActionTag)+1:]
		// 提取工具名称
		rest = strings.TrimSpace(rest)
		if endIdx := strings.Index(rest, "\n"); endIdx != -1 {
			step.Action = strings.TrimSpace(rest[:endIdx])
		} else {
			step.Action = rest
		}

		// 解析参数（如果有的话）
		// 格式: Action Input: {"key": "value"}
		if inputIdx := strings.Index(response, "Action Input:"); inputIdx != -1 {
			inputRest := response[inputIdx+len("Action Input:"):]
			inputRest = strings.TrimSpace(inputRest)
			if endIdx := strings.Index(inputRest, "\n"); endIdx != -1 {
				inputRest = inputRest[:endIdx]
			}
			// 尝试解析 JSON
			if err := json.Unmarshal([]byte(inputRest), &step.ActionInput); err != nil {
				// 如果不是 JSON，作为简单字符串参数
				step.ActionInput = map[string]any{"input": inputRest}
			}
		}
	}

	return step, ""
}

// formatStep 格式化步骤为字符串。
func (u *ReAct) formatStep(step ReActStep) string {
	var sb strings.Builder
	if step.Thought != "" {
		sb.WriteString(fmt.Sprintf("%s: %s\n", u.config.ThoughtTag, step.Thought))
	}
	if step.Action != "" {
		sb.WriteString(fmt.Sprintf("%s: %s\n", u.config.ActionTag, step.Action))
		if step.ActionInput != nil {
			inputJSON, _ := json.Marshal(step.ActionInput)
			sb.WriteString(fmt.Sprintf("Action Input: %s\n", string(inputJSON)))
		}
	}
	return sb.String()
}

// executeTool 执行工具。
func (u *ReAct) executeTool(ctx context.Context, action string, input map[string]any) string {
	if u.tools == nil {
		return "错误: 工具注册表未配置"
	}

	result := u.tools.ExecuteWithContext(ctx, action, input, "", "", nil)
	if result.Error != nil {
		return fmt.Sprintf("错误: %v", result.Error)
	}

	// 截断过长的输出
	observation := result.Content
	if len(observation) > 2000 {
		observation = observation[:2000] + "\n... (输出已截断)"
	}

	return observation
}

// RegisterTool 注册工具。
func (u *ReAct) RegisterTool(tool tools.Tool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.tools.Register(tool)
}

// SetProvider 设置提供商。
func (u *ReAct) SetProvider(p providers.Provider) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.provider = p
}

// SetTools 设置工具注册表。
func (u *ReAct) SetTools(t *tools.Registry) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.tools = t
}

// 默认 ReAct 提示词
const defaultReActPrompt = `你是一个智能助手，使用 ReAct (Reasoning + Acting) 模式来回答问题。

对于每一步，你需要：
1. 思考 (Thought): 分析当前情况，决定下一步行动
2. 行动 (Action): 选择一个工具来执行
3. 观察 (Observation): 查看工具执行结果

重复以上步骤直到找到答案。

开始!`
