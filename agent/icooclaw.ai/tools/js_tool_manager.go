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
	// 从 embed.FS 读取工具描述
	desc, err := toolDescs.ReadFile(filepath.Join("tools_desc", "js_tool_create.md"))
	if err != nil {
		return fmt.Sprintf("读取工具描述失败: %v", err)
	}

	return string(desc)
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
			"permissions": map[string]interface{}{
				"type":        "object",
				"description": "工具权限配置，控制脚本可以访问的功能",
				"properties": map[string]interface{}{
					"fileRead": map[string]interface{}{
						"type":        "boolean",
						"description": "允许读取文件，可使用 fs.readFile、fs.exists 等",
					},
					"fileWrite": map[string]interface{}{
						"type":        "boolean",
						"description": "允许写入文件，可使用 fs.writeFile、fs.appendFile 等",
					},
					"fileDelete": map[string]interface{}{
						"type":        "boolean",
						"description": "允许删除文件，可使用 fs.deleteFile、fs.rmdir 等",
					},
					"network": map[string]interface{}{
						"type":        "boolean",
						"description": "允许网络访问，可使用 http.get、http.post 等",
					},
					"exec": map[string]interface{}{
						"type":        "boolean",
						"description": "允许执行命令，可使用 shell.exec 等",
					},
				},
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
		return "", fmt.Errorf("name 参数是必填项")
	}

	description, ok := params["description"].(string)
	if !ok || description == "" {
		return "", fmt.Errorf("description 参数是必填项")
	}

	code, ok := params["code"].(string)
	if !ok || code == "" {
		return "", fmt.Errorf("code 参数是必填项")
	}

	overwrite := false
	if ow, ok := params["overwrite"].(bool); ok {
		overwrite = ow
	}

	// 解析权限配置
	permissions := parsePermissions(params["permissions"])

	// 验证名称
	if err := validateToolName(name); err != nil {
		return "", err
	}

	// 验证代码安全性
	if err := validateJSCode(code); err != nil {
		return "", fmt.Errorf("code 参数验证失败: %w", err)
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
	script := generateToolScript(name, description, parameters, code, permissions)

	// 确保工具目录存在
	toolsDir := filepath.Join(t.config.Workspace, t.config.ToolsDir)
	if err := os.MkdirAll(toolsDir, 0755); err != nil {
		return "", fmt.Errorf("创建工具目录失败: %w", err)
	}

	// 检查是否已存在
	scriptPath := filepath.Join(toolsDir, name+".js")
	if _, err := os.Stat(scriptPath); err == nil && !overwrite {
		return "", fmt.Errorf("工具 '%s' 已存在，设置 overwrite=true 可覆盖", name)
	}

	// 写入文件
	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		return "", fmt.Errorf("写入工具文件失败: %w", err)
	}

	// 动态注册工具
	if t.config.Registry != nil {
		jsConfig := &JSToolConfig{
			Workspace:       t.config.Workspace,
			MaxMemory:       10 * 1024 * 1024,
			Timeout:         30,
			AllowFileRead:   permissions.FileRead,
			AllowFileWrite:  permissions.FileWrite,
			AllowFileDelete: permissions.FileDelete,
			AllowNetwork:    permissions.Network,
			AllowExec:       permissions.Exec,
			ExecTimeout:     30,
			HTTPTimeout:     30,
		}

		loader := NewJSToolLoader(jsConfig, nil)
		tool, err := loader.LoadFromFile(scriptPath)
		if err != nil {
			// 删除文件
			os.Remove(scriptPath)
			return "", fmt.Errorf("加载创建的工具失败: %w", err)
		}

		t.config.mu.Lock()
		t.config.Registry.Register(tool)
		t.config.mu.Unlock()
	}

	// 构建权限说明
	var permDesc []string
	if permissions.FileRead {
		permDesc = append(permDesc, "fileRead")
	}
	if permissions.FileWrite {
		permDesc = append(permDesc, "fileWrite")
	}
	if permissions.FileDelete {
		permDesc = append(permDesc, "fileDelete")
	}
	if permissions.Network {
		permDesc = append(permDesc, "network")
	}
	if permissions.Exec {
		permDesc = append(permDesc, "exec")
	}

	result := map[string]interface{}{
		"success":     true,
		"name":        name,
		"file":        scriptPath,
		"message":     fmt.Sprintf("工具 '%s' 创建成功，可在后续请求中使用", name),
		"usage":       fmt.Sprintf("你可以在后续请求中使用工具 '%s'", name),
		"permissions": permDesc,
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return string(resultJSON), nil
}

// toolPermissions 工具权限
type toolPermissions struct {
	FileRead   bool
	FileWrite  bool
	FileDelete bool
	Network    bool
	Exec       bool
}

// parsePermissions 解析权限配置
func parsePermissions(v interface{}) toolPermissions {
	perms := toolPermissions{}
	if v == nil {
		return perms
	}

	if m, ok := v.(map[string]interface{}); ok {
		if val, ok := m["fileRead"].(bool); ok {
			perms.FileRead = val
		}
		if val, ok := m["fileWrite"].(bool); ok {
			perms.FileWrite = val
		}
		if val, ok := m["fileDelete"].(bool); ok {
			perms.FileDelete = val
		}
		if val, ok := m["network"].(bool); ok {
			perms.Network = val
		}
		if val, ok := m["exec"].(bool); ok {
			perms.Exec = val
		}
	}

	return perms
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
		return fmt.Errorf("工具名称长度必须在 2 到 50 个字符之间")
	}

	for i, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return fmt.Errorf("工具名称只能包含字母、数字和下划线")
		}
		if i == 0 && (c >= '0' && c <= '9') {
			return fmt.Errorf("工具名称不能以数字开头")
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
		return fmt.Errorf("工具名称 '%s' 是保留名称", name)
	}

	return nil
}

// validateJSCode 验证 JavaScript 代码安全性
func validateJSCode(code string) error {
	// 检查是否包含 execute 函数
	if !strings.Contains(code, "function execute") {
		return fmt.Errorf("代码必须定义一个 'execute(params)' 函数")
	}

	// 检查是否使用了 async/await（goja 不支持）
	if strings.Contains(code, "async ") || strings.Contains(code, "await ") {
		return fmt.Errorf("代码不能使用 'async' 或 'await' 关键字（JavaScript 引擎不支持）")
	}

	// 危险模式检查（移除 fs.、http. 等，因为这些现在由脚本引擎安全提供）
	dangerousPatterns := []string{
		"require(",
		"import ",
		"eval(",
		"new Function(",
		"process.",
		"global.",
		"__dirname",
		"__filename",
		"child_process",
		"os.Exit",
		"os.Remove",
		"os.Create",
		"os.Open",
		"os.Read",
		"os.Write",
		"net.Listen",
		"net.Dial",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(code, pattern) {
			return fmt.Errorf("代码包含潜在不安全模式: %s", pattern)
		}
	}

	return nil
}

// generateToolScript 生成完整的工具脚本
func generateToolScript(name, description string, parameters interface{}, code string, permissions toolPermissions) string {
	paramsJSON, _ := json.MarshalIndent(parameters, "    ", "    ")

	// 构建权限注释
	var permLines []string
	if permissions.FileRead {
		permLines = append(permLines, " *   - fileRead: 允许读取文件")
	}
	if permissions.FileWrite {
		permLines = append(permLines, " *   - fileWrite: 允许写入文件")
	}
	if permissions.FileDelete {
		permLines = append(permLines, " *   - fileDelete: 允许删除文件")
	}
	if permissions.Network {
		permLines = append(permLines, " *   - network: 允许网络访问")
	}
	if permissions.Exec {
		permLines = append(permLines, " *   - exec: 允许执行命令")
	}

	permissionsComment := ""
	if len(permLines) > 0 {
		permissionsComment = "\n * 权限:\n" + strings.Join(permLines, "\n")
	}

	return fmt.Sprintf(`/**
 * 工具名称: %s
 * 描述: %s
 * 自动生成时间: 由 AI 动态创建%s
 */

var tool = {
    name: "%s",
    description: "%s",
    parameters: %s,
    permissions: {
        fileRead: %v,
        fileWrite: %v,
        fileDelete: %v,
        network: %v,
        exec: %v
    }
};

%s
`, name, description, permissionsComment, name, description, string(paramsJSON),
		permissions.FileRead, permissions.FileWrite, permissions.FileDelete,
		permissions.Network, permissions.Exec, code)
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
		return "", fmt.Errorf("参数 'name' 是必填项")
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
		return "", fmt.Errorf("不能删除内置工具 '%s'", name)
	}

	// 查找并删除文件
	toolsDir := filepath.Join(t.config.Workspace, t.config.ToolsDir)
	scriptPath := filepath.Join(toolsDir, name+".js")

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return "", fmt.Errorf("工具 '%s' 不存在", name)
	}

	if err := os.Remove(scriptPath); err != nil {
		return "", fmt.Errorf("删除工具文件失败: %w", err)
	}

	result := map[string]interface{}{
		"success": true,
		"name":    name,
		"message": fmt.Sprintf("工具 '%s' 已删除", name),
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
