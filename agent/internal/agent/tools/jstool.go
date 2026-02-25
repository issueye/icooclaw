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
)

// JSToolConfig JavaScript 工具配置
type JSToolConfig struct {
	Workspace string
	MaxMemory int64
	Timeout   int
}

// JSTool JavaScript 工具
type JSTool struct {
	name        string
	description string
	parameters  map[string]interface{}
	script      string
	vm          *goja.Runtime
	config      *JSToolConfig
	logger      *slog.Logger
}

// JSToolDefinition JS 工具定义（从脚本中解析）
type JSToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Execute     string                 `json:"execute"`
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

	vm := goja.New()
	vm.SetMaxCallStackSize(100)
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

	tool := &JSTool{
		name:        def.Name,
		description: def.Description,
		parameters:  def.Parameters,
		script:      def.Execute,
		vm:          vm,
		config:      config,
		logger:      logger,
	}

	tool.setupEnvironment()

	if _, err := vm.RunString(tool.script); err != nil {
		return nil, fmt.Errorf("failed to compile script: %w", err)
	}

	return tool, nil
}

// setupEnvironment 设置 JS 运行环境
func (t *JSTool) setupEnvironment() {
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
			return execute(params);
		})()
	`, string(paramsJSON)))
	if err != nil {
		return "", fmt.Errorf("script execution failed: %w", err)
	}

	if result == nil || goja.IsUndefined(result) {
		return "", nil
	}

	switch v := result.Export().(type) {
	case string:
		return v, nil
	case map[string]interface{}:
		b, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("failed to marshal result: %w", err)
		}
		return string(b), nil
	case nil:
		return "", nil
	default:
		return fmt.Sprintf("%v", v), nil
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
