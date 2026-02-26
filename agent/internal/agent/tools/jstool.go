package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/dop251/goja"
	"github.com/icooclaw/icooclaw/internal/script"
)

// JSToolConfig JavaScript 工具配置
type JSToolConfig struct {
	Workspace       string
	MaxMemory       int64
	Timeout         int
	AllowFileRead   bool
	AllowFileWrite  bool
	AllowFileDelete bool
	AllowExec       bool
	AllowNetwork    bool
	ExecTimeout     int
	HTTPTimeout     int
	AllowedDomains  []string
}

// JSTool JavaScript 工具
type JSTool struct {
	name        string
	description string
	parameters  map[string]interface{}
	script      string
	vm          *goja.Runtime
	engine      *script.Engine
	config      *JSToolConfig
	logger      *slog.Logger
	useEngine   bool
}

// JSToolDefinition JS 工具定义（从脚本中解析）
type JSToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Permissions *JSToolPermissions     `json:"permissions"`
	Execute     string                 `json:"execute"`
}

// JSToolPermissions JS 工具权限
type JSToolPermissions struct {
	FileRead   bool `json:"fileRead"`
	FileWrite  bool `json:"fileWrite"`
	FileDelete bool `json:"fileDelete"`
	Network    bool `json:"network"`
	Exec       bool `json:"exec"`
}

// NewJSTool 创建 JavaScript 工具
func NewJSTool(def JSToolDefinition, config *JSToolConfig, logger *slog.Logger) (*JSTool, error) {
	if config == nil {
		config = &JSToolConfig{
			MaxMemory: 10 * 1024 * 1024,
			Timeout:   30,
		}
	}
	if logger == nil {
		logger = slog.Default()
	}

	// 从工具定义中获取权限配置
	if def.Permissions != nil {
		config.AllowFileRead = def.Permissions.FileRead
		config.AllowFileWrite = def.Permissions.FileWrite
		config.AllowFileDelete = def.Permissions.FileDelete
		config.AllowNetwork = def.Permissions.Network
		config.AllowExec = def.Permissions.Exec
	}

	// 检查是否需要增强功能
	useEngine := config.AllowFileRead || config.AllowFileWrite || config.AllowFileDelete ||
		config.AllowExec || config.AllowNetwork

	tool := &JSTool{
		name:        def.Name,
		description: def.Description,
		parameters:  def.Parameters,
		script:      def.Execute,
		config:      config,
		logger:      logger,
		useEngine:   useEngine,
	}

	if useEngine {
		// 使用增强脚本引擎
		engineConfig := &script.Config{
			Workspace:       config.Workspace,
			AllowFileRead:   config.AllowFileRead,
			AllowFileWrite:  config.AllowFileWrite,
			AllowFileDelete: config.AllowFileDelete,
			AllowExec:       config.AllowExec,
			AllowNetwork:    config.AllowNetwork,
			ExecTimeout:     config.ExecTimeout,
			HTTPTimeout:     config.HTTPTimeout,
			MaxMemory:       config.MaxMemory,
			AllowedDomains:  config.AllowedDomains,
		}
		engine := script.NewEngine(engineConfig, logger.With("tool", def.Name))
		tool.engine = engine
		tool.vm = engine.GetVM()
	} else {
		// 使用基础模式
		vm := goja.New()
		vm.SetMaxCallStackSize(100)
		vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
		tool.vm = vm
		tool.setupBasicEnvironment()
	}

	if _, err := tool.vm.RunString(tool.script); err != nil {
		return nil, fmt.Errorf("failed to compile script: %w", err)
	}

	return tool, nil
}

// setupBasicEnvironment 设置基础 JS 运行环境（无增强功能）
func (t *JSTool) setupBasicEnvironment() {
	console := &jsConsole{logger: t.logger.With("tool", t.name)}
	t.vm.Set("console", console)

	t.vm.Set("JSON", map[string]interface{}{
		"stringify": func(v interface{}) string {
			b, _ := json.Marshal(v)
			return string(b)
		},
		"parse": func(s string) (interface{}, error) {
			var v interface{}
			err := json.Unmarshal([]byte(s), &v)
			return v, err
		},
	})
}

// Name 获取工具名称
func (t *JSTool) Name() string {
	return t.name
}

// Description 获取工具描述
func (t *JSTool) Description() string {
	return t.description
}

// Parameters 获取参数定义
func (t *JSTool) Parameters() map[string]interface{} {
	return t.parameters
}

// Execute 执行工具
func (t *JSTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return "", fmt.Errorf("failed to marshal params: %w", err)
	}

	result, err := t.vm.RunString(fmt.Sprintf(`
		(function() {
			var params = %s;
			var result = execute(params);
			// Ensure result is serializable
			if (typeof result === 'object' && result !== null) {
				return JSON.stringify(result);
			}
			return result;
		})()
	`, string(paramsJSON)))
	if err != nil {
		return "", fmt.Errorf("script execution failed: %w", err)
	}

	if result == nil || goja.IsUndefined(result) {
		return "", nil
	}

	exported := result.Export()
	if exported == nil {
		return "", nil
	}

	switch v := exported.(type) {
	case string:
		return v, nil
	default:
		return fmt.Sprintf("%v", exported), nil
	}
}

// ToDefinition 转换为工具定义
func (t *JSTool) ToDefinition() ToolDefinition {
	return ToolDefinition{
		Type: "function",
		Function: FunctionDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Parameters(),
		},
	}
}

// jsConsole JS 控制台实现
type jsConsole struct {
	logger *slog.Logger
}

func (c *jsConsole) Log(args ...interface{}) {
	msg := fmt.Sprint(args...)
	c.logger.Info(msg)
}

func (c *jsConsole) Info(args ...interface{}) {
	msg := fmt.Sprint(args...)
	c.logger.Info(msg)
}

func (c *jsConsole) Debug(args ...interface{}) {
	msg := fmt.Sprint(args...)
	c.logger.Debug(msg)
}

func (c *jsConsole) Warn(args ...interface{}) {
	msg := fmt.Sprint(args...)
	c.logger.Warn(msg)
}

func (c *jsConsole) Error(args ...interface{}) {
	msg := fmt.Sprint(args...)
	c.logger.Error(msg)
}

func (c *jsConsole) Table(data interface{}) {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		c.logger.Error(fmt.Sprintf("table: %v", err))
		return
	}
	c.logger.Info(string(b))
}

// JSToolLoader JavaScript 工具加载器
type JSToolLoader struct {
	config *JSToolConfig
	logger *slog.Logger
}

// NewJSToolLoader 创建 JS 工具加载器
func NewJSToolLoader(config *JSToolConfig, logger *slog.Logger) *JSToolLoader {
	if logger == nil {
		logger = slog.Default()
	}
	return &JSToolLoader{
		config: config,
		logger: logger,
	}
}

// LoadFromFile 从文件加载 JS 工具
func (l *JSToolLoader) LoadFromFile(path string) (*JSTool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	def, err := l.parseScript(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse script: %w", err)
	}

	if def.Name == "" {
		def.Name = strings.TrimSuffix(filepath.Base(path), ".js")
	}

	return NewJSTool(def, l.config, l.logger)
}

// LoadFromDirectory 从目录加载所有 JS 工具
func (l *JSToolLoader) LoadFromDirectory(dir string) ([]Tool, error) {
	var tools []Tool

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return tools, nil
		}
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".js") && !strings.HasSuffix(name, ".tool") {
			continue
		}

		path := filepath.Join(dir, name)
		tool, err := l.LoadFromFile(path)
		if err != nil {
			l.logger.Warn("Failed to load JS tool", "path", path, "error", err)
			continue
		}

		tools = append(tools, tool)
		l.logger.Info("Loaded JS tool", "name", tool.Name(), "path", path)
	}

	return tools, nil
}

// parseScript 解析 JS 脚本获取工具定义
func (l *JSToolLoader) parseScript(script string) (JSToolDefinition, error) {
	var def JSToolDefinition

	vm := goja.New()
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

	if _, err := vm.RunString(script); err != nil {
		return def, fmt.Errorf("failed to run script: %w", err)
	}

	toolDef := vm.Get("tool")
	if toolDef == nil || goja.IsUndefined(toolDef) {
		return def, fmt.Errorf("script must define a 'tool' object")
	}

	exported := toolDef.Export()
	if m, ok := exported.(map[string]interface{}); ok {
		if name, ok := m["name"].(string); ok {
			def.Name = name
		}
		if desc, ok := m["description"].(string); ok {
			def.Description = desc
		}
		if params, ok := m["parameters"].(map[string]interface{}); ok {
			def.Parameters = params
		}
		// 解析权限配置
		if perms, ok := m["permissions"].(map[string]interface{}); ok {
			def.Permissions = &JSToolPermissions{}
			if v, ok := perms["fileRead"].(bool); ok {
				def.Permissions.FileRead = v
			}
			if v, ok := perms["fileWrite"].(bool); ok {
				def.Permissions.FileWrite = v
			}
			if v, ok := perms["fileDelete"].(bool); ok {
				def.Permissions.FileDelete = v
			}
			if v, ok := perms["network"].(bool); ok {
				def.Permissions.Network = v
			}
			if v, ok := perms["exec"].(bool); ok {
				def.Permissions.Exec = v
			}
		}
	}

	if def.Name == "" {
		return def, fmt.Errorf("tool definition must have a 'name' property")
	}

	executeFunc := vm.Get("execute")
	if executeFunc == nil || goja.IsUndefined(executeFunc) {
		return def, fmt.Errorf("tool must define an 'execute' function")
	}

	def.Execute = script

	return def, nil
}

// RegisterJSTools 从目录加载并注册 JS 工具到注册表
func RegisterJSTools(registry *Registry, dir string, config *JSToolConfig, logger *slog.Logger) error {
	loader := NewJSToolLoader(config, logger)
	tools, err := loader.LoadFromDirectory(dir)
	if err != nil {
		return err
	}

	for _, tool := range tools {
		registry.Register(tool)
	}

	return nil
}
