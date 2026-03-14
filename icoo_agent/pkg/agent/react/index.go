package react

import (
	"context"
	"encoding/json"
	"fmt"
	"icooclaw/pkg/agent/hooks"
	"icooclaw/pkg/bus"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/memory"
	"icooclaw/pkg/providers"
	"icooclaw/pkg/skill"
	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"
	"icooclaw/pkg/utils"
	"log/slog"
	"strings"
)

type ReActAgent struct {
	tools           *tools.Registry    // 工具注册表
	memory          memory.Loader      // 内存加载器
	skills          skill.Loader       // 工具加载器
	storage         *storage.Storage   // 存储管理
	bus             *bus.MessageBus    // 消息总线
	providerFactory *providers.Factory // 提供商工厂
	logger          *slog.Logger       // 日志记录器
	hooks           hooks.ReactHooks   // React钩子接口

	// Configuration 配置项
	maxToolIterations int // 最大工具迭代次数
}

type Option func(*ReActAgent)

func WithTools(r *tools.Registry) Option {
	return func(a *ReActAgent) {
		a.tools = r
	}
}

func WithMemory(m memory.Loader) Option {
	return func(a *ReActAgent) {
		a.memory = m
	}
}

func WithSkills(s skill.Loader) Option {
	return func(a *ReActAgent) {
		a.skills = s
	}
}

func WithBus(b *bus.MessageBus) Option {
	return func(a *ReActAgent) {
		a.bus = b
	}
}

func NewReActAgent() *ReActAgent {
	return &ReActAgent{}
}

// GetDynamicProvider 从存储配置动态获取提供商。
// 返回提供商、模型名称和错误。
func (a *ReActAgent) GetDynamicProvider(ctx context.Context) (providers.Provider, string, error) {
	if a.providerFactory == nil || a.storage == nil {
		return nil, "", fmt.Errorf("未配置提供商工厂或存储")
	}

	// 获取默认模型配置
	defaultModel, err := a.storage.Param().Get(consts.DEFAULT_MODEL_KEY)
	if err != nil || defaultModel == nil || defaultModel.Value == "" {
		return nil, "", fmt.Errorf("默认模型未配置")
	}

	// 分割模型字符串
	parts := utils.SplitProviderModel(defaultModel.Value)
	if len(parts) != 2 {
		return nil, "", fmt.Errorf("默认模型格式错误: %s", defaultModel.Value)
	}

	providerName, modelName := parts[0], parts[1]

	// 获取提供商实例
	provider, err := a.providerFactory.Get(providerName)
	if err != nil {
		return nil, "", fmt.Errorf("获取Provider失败: %w", err)
	}

	// 调用钩子获取提供商实例
	if a.hooks != nil {
		provider, modelName, err = a.hooks.GetProvider(ctx, providerName, a.storage.Provider())
		if err != nil {
			return nil, "", err
		}
	}

	// 返回提供商实例
	return provider, modelName, nil
}

// buildMessages 构建 LLM 请求的消息列表。
func (a *ReActAgent) buildMessages(ctx context.Context, sessionKey string, msg bus.InboundMessage) ([]providers.ChatMessage, error) {
	var (
		messages = make([]providers.ChatMessage, 0)
		err      error
	)

	// 1. Add hooks 添加钩子消息。
	if a.hooks != nil {
		messages, err = a.hooks.BuildMessagesBefore(ctx, sessionKey, msg, messages)
		if err != nil {
			return nil, err
		}
	}

	// 2. Add system prompt 添加系统提示词。
	// 加载 AGENTS.md、SOUL.md、USER.md 工作空间配置
	systemPrompt, err := a.storage.Workspace().LoadWorkspace()
	if err != nil {
		return nil, err
	}

	// 加载 SKILL 工具
	skills, err := a.skills.List(ctx)
	if err != nil {
		return nil, err
	}

	sb := strings.Builder{}
	sb.WriteString("\n\n## 技能列表\n")
	for _, skill := range skills {
		sb.WriteString(fmt.Sprintf("- 名称 %s\n", skill.Name))
		sb.WriteString(fmt.Sprintf("		描述 %s\n", skill.Description))
	}

	systemPrompt += sb.String()

	messages = append(messages, providers.ChatMessage{
		Role:    consts.RoleSystem.ToString(),
		Content: systemPrompt,
	})

	// 3. Load memory/history 加载记忆历史记录。
	var history []providers.ChatMessage
	if a.memory != nil {
		mem, err := a.memory.Load(ctx, sessionKey)
		if err != nil {
			a.logger.With("name", "【智能体】").Warn("加载记忆失败", "error", err, "session_key", sessionKey)
		} else {
			history = mem
		}
	}
	messages = append(messages, history...)

	// 4. Add user message 添加用户消息。
	messages = append(messages, providers.ChatMessage{
		Role:    consts.RoleUser.ToString(),
		Content: msg.Text,
	})

	// 5. Add hooks 添加钩子消息。
	if a.hooks != nil {
		messages, err = a.hooks.BuildMessagesAfter(ctx, sessionKey, msg, messages)
		if err != nil {
			return nil, err
		}
	}

	// 6. 保存用户消息到记忆历史记录。
	if a.memory != nil {
		err = a.memory.Save(ctx, sessionKey, consts.RoleUser.ToString(), msg.Text)
		if err != nil {
			return nil, err
		}
	}

	return messages, nil
}

// convertToolDefinitions 转换工具定义为提供商工具
func (a *ReActAgent) convertToolDefinitions(defs []tools.ToolDefinition) []providers.Tool {
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

// executeToolCall 执行工具调用
func (a *ReActAgent) executeToolCall(ctx context.Context, tc providers.ToolCall, msg bus.InboundMessage) (string, error) {
	toolName := tc.Function.Name
	var err error

	// 调用钩子工具调用前
	if a.hooks != nil {
		tc, err = a.hooks.ToolCallBefore(ctx, toolName, tc, msg)
		if err != nil {
			return "", err
		}
	}

	// 解析工具参数
	var args map[string]any
	if tc.Function.Arguments != "" {
		// 调用钩子工具参数解析
		if a.hooks != nil {
			args, err = a.hooks.ToolParseArguments(ctx, toolName, tc, msg)
			if err != nil {
				return "", err
			}
		} else {
			err = json.Unmarshal([]byte(tc.Function.Arguments), &args)
			if err != nil {
				a.logger.With("name", "【智能体】").Error("解析工具参数失败",
					"tool", toolName,
					"error", err)
				return "", fmt.Errorf("解析参数失败: %w", err)
			}
		}
	}

	// 执行工具
	result := a.tools.ExecuteWithContext(ctx, toolName, args, msg.Channel, msg.SessionID, nil)
	if result.Error != nil {
		return "", result.Error
	}

	// 调用钩子工具调用后
	if a.hooks != nil {
		err = a.hooks.ToolCallAfter(ctx, toolName, msg, result)
		if err != nil {
			return "", err
		}
	}

	return result.Content, nil
}

// mergeToolCalls 合并流式响应中的工具调用
func (a *ReActAgent) mergeToolCalls(toolCalls []providers.ToolCall) []providers.ToolCall {
	if len(toolCalls) == 0 {
		return nil
	}

	// 按 index 分组
	indexToCall := make(map[int]*providers.ToolCall)
	realIDToIndex := make(map[string]int)
	nextIndex := 0

	for _, tc := range toolCalls {
		var idx int
		var found bool

		// 检查是否为流式索引ID
		if isStreamIndexID(tc.ID) {
			fmt.Sscanf(tc.ID, "stream_index:%d", &idx)
			found = true
		} else if tc.ID != "" {
			if i, ok := realIDToIndex[tc.ID]; ok {
				idx = i
				found = true
			} else {
				idx = nextIndex
				nextIndex++
				realIDToIndex[tc.ID] = idx
				found = true
			}
		}

		if !found {
			continue
		}

		if existing, ok := indexToCall[idx]; ok {
			// 合并内容
			if tc.Function.Name != "" {
				existing.Function.Name = tc.Function.Name
			}
			if tc.Function.Arguments != "" {
				existing.Function.Arguments += tc.Function.Arguments
			}
			if tc.ID != "" && !isStreamIndexID(tc.ID) {
				existing.ID = tc.ID
			}
		} else {
			// 创建新条目
			copy := tc
			indexToCall[idx] = &copy
		}
	}

	// 转换为结果
	result := make([]providers.ToolCall, 0, len(indexToCall))
	for _, tc := range indexToCall {
		if tc.Function.Name == "" {
			continue
		}
		result = append(result, providers.ToolCall{
			ID:   tc.ID,
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

	return result
}

// validateToolCalls 验证工具调用是否有效
func (a *ReActAgent) validateToolCalls(toolCalls []providers.ToolCall) []providers.ToolCall {
	valid := make([]providers.ToolCall, 0, len(toolCalls))

	for _, tc := range toolCalls {
		// 跳过空名称的工具调用
		if tc.Function.Name == "" {
			a.logger.Warn("跳过无效工具调用：缺少工具名称", "id", tc.ID)
			continue
		}

		// 确保参数是有效的JSON或空对象
		if tc.Function.Arguments == "" {
			tc.Function.Arguments = "{}"
		}

		valid = append(valid, tc)
	}

	return valid
}

// isStreamIndexID 检查 ID 是否为临时的流式索引 ID
func isStreamIndexID(id string) bool {
	return len(id) > 12 && id[:12] == "stream_index"
}
