package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// CreateJSToolConfig 创建 JS 工具的配置
type CreateJSToolConfig struct {
	Workspace string
	ToolsDir  string
	Registry  *Registry
	mu        sync.RWMutex
}

// CreateJSTool 动态创建 JS 工具的工具
type CreateJSTool struct {
	config *CreateJSToolConfig
}

// NewCreateJSTool 创建工具
func NewCreateJSTool(config *CreateJSToolConfig) *CreateJSTool {
	if config.ToolsDir == "" {
		config.ToolsDir = "tools"
	}
	return &CreateJSTool{config: config}
}

// Name 获取工具名称
func (t *CreateJSTool) Name() string {
	return "create_tool"
}

// Description 获取工具描述
func (t *CreateJSTool) Description() string {
	return `动态创建新的 JavaScript 工具。使用此工具可以在运行时创建新的工具并立即使用。

创建的工具将保存在工作区的 tools 目录中，并在后续会话中自动加载。

使用示例:
- 创建一个简单的问候工具
- 创建一个数据转换工具
- 创建一个 API 调用包装工具`
}

// Parameters 获取参数定义
func (t *CreateJSTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "工具名称，只能包含字母、数字和下划线，如 'my_calculator'",
			},
			"description": map[string]interface{}{
				"type":        "string",
				"description": "工具功能描述，清晰说明工具的用途",
			},
			"parameters": map[string]interface{}{
				"type":        "object",
				"description": "工具参数定义（JSON Schema 格式）",
			},
			"code": map[string]interface{}{
				"type":        "string",
				"description": "JavaScript 执行函数代码。必须定义 execute(params) 函数，params 为参数对象，返回字符串结果。",
			},
			"overwrite": map[string]interface{}{
				"type":        "boolean",
				"description": "是否覆盖已存在的同名工具，默认 false",
			},
		},
		"required": []string{"name", "description", "code"},
	}
}

// Execute 执行工具
func (t *CreateJSTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	name, ok := params["name"].(string)
	if !ok || name == "" {
		return "", fmt.Errorf("name parameter is required")
	}

	description, ok := params["description"].(string)
	if !ok || description == "" {
		return "", fmt.Errorf("description parameter is required")
	}

	code, ok := params["code"].(string)
	if !ok || code == "" {
		return "", fmt.Errorf("code parameter is required")
	}

	overwrite := false
	if ow, ok := params["overwrite"].(bool); ok {
		overwrite = ow
	}

	// 验证名称
	if err := validateToolName(name); err != nil {
		return "", err
	}

	// 验证代码安全性
	if err := validateJSCode(code); err != nil {
		return "", fmt.Errorf("code validation failed: %w", err)
	}

	// 构建参数定义
	parameters := params["parameters"]
	if parameters == nil {
		parameters = map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}
	}

	// 生成完整的工具脚本
	script := generateToolScript(name, description, parameters, code)

	// 确保工具目录存在
	toolsDir := filepath.Join(t.config.Workspace, t.config.ToolsDir)
	if err := os.MkdirAll(toolsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create tools directory: %w", err)
	}

	// 检查是否已存在
	scriptPath := filepath.Join(toolsDir, name+".js")
	if _, err := os.Stat(scriptPath); err == nil && !overwrite {
		return "", fmt.Errorf("tool '%s' already exists, set overwrite=true to replace it", name)
	}

	// 写入文件
	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		return "", fmt.Errorf("failed to write tool file: %w", err)
	}

	// 动态注册工具
	if t.config.Registry != nil {
		jsConfig := &JSToolConfig{
			Workspace: t.config.Workspace,
			MaxMemory: 10 * 1024 * 1024,
			Timeout:   30,
		}

		loader := NewJSToolLoader(jsConfig, nil)
		tool, err := loader.LoadFromFile(scriptPath)
		if err != nil {
			// 删除文件
			os.Remove(scriptPath)
			return "", fmt.Errorf("failed to load created tool: %w", err)
		}

		t.config.mu.Lock()
		t.config.Registry.Register(tool)
		t.config.mu.Unlock()
	}

	result := map[string]interface{}{
		"success": true,
		"name":    name,
		"file":    scriptPath,
		"message": fmt.Sprintf("Tool '%s' created successfully and is now available for use", name),
		"usage":   fmt.Sprintf("You can now use the tool '%s' in subsequent requests", name),
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return string(resultJSON), nil
}

// ToDefinition 转换为工具定义
func (t *CreateJSTool) ToDefinition() ToolDefinition {
	return ToolDefinition{
		Type: "function",
		Function: FunctionDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Parameters(),
		},
	}
}

// validateToolName 验证工具名称
func validateToolName(name string) error {
	if len(name) < 2 || len(name) > 50 {
		return fmt.Errorf("tool name must be between 2 and 50 characters")
	}

	for i, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return fmt.Errorf("tool name can only contain letters, numbers, and underscores")
		}
		if i == 0 && (c >= '0' && c <= '9') {
			return fmt.Errorf("tool name cannot start with a number")
		}
	}

	// 检查保留名称
	reserved := map[string]bool{
		"create_tool":  true,
		"delete_tool":  true,
		"list_tools":   true,
		"update_tool":  true,
		"file_read":    true,
		"file_write":   true,
		"file_edit":    true,
		"file_delete":  true,
		"file_list":    true,
		"http_request": true,
		"web_search":   true,
		"web_fetch":    true,
		"calculator":   true,
		"exec":         true,
		"message":      true,
		"grep":         true,
		"find":         true,
		"tree":         true,
		"read_part":    true,
		"wc":           true,
	}

	if reserved[name] {
		return fmt.Errorf("'%s' is a reserved tool name", name)
	}

	return nil
}

// validateJSCode 验证 JavaScript 代码安全性
func validateJSCode(code string) error {
	// 检查是否包含 execute 函数
	if !strings.Contains(code, "function execute") {
		return fmt.Errorf("code must define an 'execute(params)' function")
	}

	// 危险模式检查
	dangerousPatterns := []string{
		"require(",
		"import ",
		"eval(",
		"Function(",
		"process.",
		"global.",
		"__dirname",
		"__filename",
		"child_process",
		"fs.",
		"os.",
		"net.",
		"http.",
		"https.",
		"fetch(",
		"XMLHttpRequest",
		"WebSocket",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(code, pattern) {
			return fmt.Errorf("code contains potentially unsafe pattern: %s", pattern)
		}
	}

	return nil
}

// generateToolScript 生成完整的工具脚本
func generateToolScript(name, description string, parameters interface{}, code string) string {
	paramsJSON, _ := json.MarshalIndent(parameters, "    ", "    ")

	return fmt.Sprintf(`/**
 * 工具名称: %s
 * 描述: %s
 * 自动生成时间: 由 AI 动态创建
 */

var tool = {
    name: "%s",
    description: "%s",
    parameters: %s
};

%s
`, name, description, name, description, string(paramsJSON), code)
}

// ListJSToolsConfig 列出 JS 工具的配置
type ListJSToolsConfig struct {
	Workspace string
	ToolsDir  string
}

// ListJSTools 列出所有 JS 工具
type ListJSTools struct {
	config *ListJSToolsConfig
}

// NewListJSTools 创建工具
func NewListJSTools(config *ListJSToolsConfig) *ListJSTools {
	if config.ToolsDir == "" {
		config.ToolsDir = "tools"
	}
	return &ListJSTools{config: config}
}

// Name 获取工具名称
func (t *ListJSTools) Name() string {
	return "list_tools"
}

// Description 获取工具描述
func (t *ListJSTools) Description() string {
	return "列出所有可用的工具，包括内置工具和动态创建的 JavaScript 工具"
}

// Parameters 获取参数定义
func (t *ListJSTools) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

// Execute 执行工具
func (t *ListJSTools) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	toolsDir := filepath.Join(t.config.Workspace, t.config.ToolsDir)

	result := map[string]interface{}{
		"builtin_tools": []map[string]interface{}{
			{"name": "file_read", "description": "读取文件内容"},
			{"name": "file_write", "description": "写入文件"},
			{"name": "file_edit", "description": "编辑文件"},
			{"name": "file_list", "description": "列出目录内容"},
			{"name": "http_request", "description": "发送 HTTP 请求"},
			{"name": "web_search", "description": "搜索网页"},
			{"name": "web_fetch", "description": "获取网页内容"},
			{"name": "calculator", "description": "数学计算"},
			{"name": "grep", "description": "搜索文件内容"},
			{"name": "find", "description": "查找文件"},
			{"name": "exec", "description": "执行 shell 命令"},
		},
		"dynamic_tools": []map[string]interface{}{},
		"tools_dir":     toolsDir,
		"can_create":    true,
		"create_example": `create_tool example:
{
  "name": "my_tool",
  "description": "My custom tool",
  "code": "function execute(params) { return JSON.stringify({result: params.input}); }",
  "parameters": {
    "type": "object",
    "properties": {
      "input": {"type": "string", "description": "Input value"}
    }
  }
}`,
	}

	// 列出动态工具
	entries, err := os.ReadDir(toolsDir)
	if err == nil {
		dynamicTools := []map[string]interface{}{}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if strings.HasSuffix(name, ".js") || strings.HasSuffix(name, ".tool") {
				toolName := strings.TrimSuffix(name, filepath.Ext(name))
				dynamicTools = append(dynamicTools, map[string]interface{}{
					"name": toolName,
					"file": name,
				})
			}
		}
		result["dynamic_tools"] = dynamicTools
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return string(resultJSON), nil
}

// ToDefinition 转换为工具定义
func (t *ListJSTools) ToDefinition() ToolDefinition {
	return ToolDefinition{
		Type: "function",
		Function: FunctionDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Parameters(),
		},
	}
}

// DeleteJSToolConfig 删除 JS 工具的配置
type DeleteJSToolConfig struct {
	Workspace string
	ToolsDir  string
	Registry  *Registry
	mu        *sync.RWMutex
}

// DeleteJSTool 删除 JS 工具
type DeleteJSTool struct {
	config *DeleteJSToolConfig
}

// NewDeleteJSTool 创建工具
func NewDeleteJSTool(config *DeleteJSToolConfig) *DeleteJSTool {
	if config.ToolsDir == "" {
		config.ToolsDir = "tools"
	}
	return &DeleteJSTool{config: config}
}

// Name 获取工具名称
func (t *DeleteJSTool) Name() string {
	return "delete_tool"
}

// Description 获取工具描述
func (t *DeleteJSTool) Description() string {
	return "删除动态创建的 JavaScript 工具。只能删除通过 create_tool 创建的工具，不能删除内置工具。"
}

// Parameters 获取参数定义
func (t *DeleteJSTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "要删除的工具名称",
			},
		},
		"required": []string{"name"},
	}
}

// Execute 执行工具
func (t *DeleteJSTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	name, ok := params["name"].(string)
	if !ok || name == "" {
		return "", fmt.Errorf("name parameter is required")
	}

	// 验证不能删除内置工具
	builtinTools := map[string]bool{
		"file_read": true, "file_write": true, "file_edit": true,
		"file_delete": true, "file_list": true, "http_request": true,
		"web_search": true, "web_fetch": true, "calculator": true,
		"exec": true, "message": true, "grep": true, "find": true,
		"tree": true, "read_part": true, "wc": true,
		"create_tool": true, "delete_tool": true, "list_tools": true,
	}

	if builtinTools[name] {
		return "", fmt.Errorf("cannot delete built-in tool '%s'", name)
	}

	// 查找并删除文件
	toolsDir := filepath.Join(t.config.Workspace, t.config.ToolsDir)
	scriptPath := filepath.Join(toolsDir, name+".js")

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return "", fmt.Errorf("tool '%s' not found", name)
	}

	if err := os.Remove(scriptPath); err != nil {
		return "", fmt.Errorf("failed to delete tool file: %w", err)
	}

	result := map[string]interface{}{
		"success": true,
		"name":    name,
		"message": fmt.Sprintf("Tool '%s' has been deleted", name),
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return string(resultJSON), nil
}

// ToDefinition 转换为工具定义
func (t *DeleteJSTool) ToDefinition() ToolDefinition {
	return ToolDefinition{
		Type: "function",
		Function: FunctionDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Parameters(),
		},
	}
}
