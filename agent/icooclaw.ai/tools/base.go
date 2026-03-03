package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"icooclaw.ai/provider"
)

type ToolIntf interface {
	Name() string
	Description() string
	ToDefinition() ToolDefinition
	Execute(ctx context.Context, params map[string]any) (string, error)
	Parameters() map[string]any
}

// Tool 工具接口
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]any
	Execute(ctx context.Context, params map[string]any) (string, error)
	ToDefinition() ToolDefinition
}

// BaseTool 基础工具
type BaseTool struct {
	name        string
	description string
	parameters  map[string]any
	executor    func(ctx context.Context, params map[string]any) (string, error)
}

// NewBaseTool 创建基础工具
func NewBaseTool(
	name string,
	description string,
	parameters map[string]any,
	executor func(ctx context.Context, params map[string]any) (string, error),
) *BaseTool {
	return &BaseTool{
		name:        name,
		description: description,
		parameters:  parameters,
		executor:    executor,
	}
}

// Name 获取名称
func (t *BaseTool) Name() string {
	return t.name
}

// Description 获取描述
func (t *BaseTool) Description() string {
	return t.description
}

// Parameters 获取参数定义
func (t *BaseTool) Parameters() map[string]any {
	return t.parameters
}

// Execute 执行工具
func (t *BaseTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	return t.executor(ctx, params)
}

// ToDefinition 转换为工具定义
func (t *BaseTool) ToDefinition() ToolDefinition {
	return ToolDefinition{
		Type: "function",
		Function: FunctionDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Parameters(),
		},
	}
}

// ToolDefinition 工具定义
type ToolDefinition struct {
	Type     string             `json:"type"`
	Function FunctionDefinition `json:"function"`
}

// FunctionDefinition 函数定义
type FunctionDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// ToolResult 工具结果
type ToolResult struct {
	ToolCallID string
	Content    string
	Error      error
}

// Registry 工具注册表
type Registry struct {
	tools map[string]ToolIntf
}

// NewRegistry 创建注册表
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]ToolIntf),
	}
}

// Register 注册工具
func (r *Registry) Register(tool ToolIntf) {
	r.tools[tool.Name()] = tool
}

// Get 获取工具
func (r *Registry) Get(name string) (Tool, error) {
	tool, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	return tool, nil
}

// List 列出所有工具
func (r *Registry) List() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// ToDefinitions 转换为工具定义列表
func (r *Registry) ToDefinitions() []ToolDefinition {
	definitions := make([]ToolDefinition, 0, len(r.tools))
	for _, tool := range r.tools {
		definitions = append(definitions, tool.ToDefinition())
	}
	return definitions
}

// Count 获取工具数量
func (r *Registry) Count() int {
	return len(r.tools)
}

// Execute 执行工具调用
// call 必须是 provider.ToolCall 类型
func (r *Registry) Execute(ctx context.Context, call any) ToolResult {
	var toolCallID, toolName string
	var arguments json.RawMessage

	switch c := call.(type) {
	case provider.ToolCall:
		toolCallID = c.ID
		toolName = c.Function.Name
		arguments = json.RawMessage(c.Function.Arguments)
	default:
		return ToolResult{
			ToolCallID: "",
			Content:    "",
			Error:      fmt.Errorf("unsupported tool call type: %T, expected provider.ToolCall", call),
		}
	}

	tool, err := r.Get(toolName)
	if err != nil {
		return ToolResult{
			ToolCallID: toolCallID,
			Content:    "",
			Error:      err,
		}
	}

	// 解析参数
	var params map[string]any
	if err := json.Unmarshal(arguments, &params); err != nil {
		return ToolResult{
			ToolCallID: toolCallID,
			Content:    "",
			Error:      fmt.Errorf("failed to parse arguments: %w", err),
		}
	}

	// 执行工具
	result, err := tool.Execute(ctx, params)
	return ToolResult{
		ToolCallID: toolCallID,
		Content:    result,
		Error:      err,
	}
}
