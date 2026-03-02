package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	utils "icooclaw.utils"
)

// WorkspaceConfig 工作空间配置
type WorkspaceConfig struct {
	Path         string
	ToolsDir     string
	DataDir      string
	LogsDir      string
	MCPServers   map[string]MCPServerConfig
}

// DefaultWorkspaceDirs 工作空间默认子目录
var DefaultWorkspaceDirs = []string{
	"tools",
	"data",
	"logs",
	"memory",
	"cache",
}

// InitWorkspaceWithConfig 根据配置初始化工作空间
// 检查并创建所有必要的目录和文件
func InitWorkspaceWithConfig(cfg *Config) (*WorkspaceConfig, error) {
	// 解析工作空间路径
	workspacePath, err := utils.ExpandPath(cfg.Workspace)
	if err != nil {
		return nil, fmt.Errorf("failed to expand workspace path: %w", err)
	}

	if workspacePath == "" {
		workspacePath = "./workspace"
	}

	wsConfig := &WorkspaceConfig{
		Path:       workspacePath,
		ToolsDir:   cfg.Tools.JS.ToolsDir,
		MCPServers: cfg.MCP.Servers,
	}

	// 1. 创建工作空间根目录
	if err := ensureDir(workspacePath, "workspace"); err != nil {
		return nil, err
	}

	// 2. 创建默认子目录
	for _, dir := range DefaultWorkspaceDirs {
		dirPath := filepath.Join(workspacePath, dir)
		if err := ensureDir(dirPath, dir); err != nil {
			return nil, err
		}
	}

	// 3. 创建 JS 工具目录（如果配置了不同路径）
	if wsConfig.ToolsDir != "" && wsConfig.ToolsDir != "tools" {
		toolsPath := filepath.Join(workspacePath, wsConfig.ToolsDir)
		if err := ensureDir(toolsPath, "js tools"); err != nil {
			return nil, err
		}
	}

	// 4. 确保数据库目录存在
	dbPath, err := utils.ExpandPath(cfg.Database.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to expand database path: %w", err)
	}
	if dbPath != "" {
		dbDir := filepath.Dir(dbPath)
		if err := ensureDir(dbDir, "database"); err != nil {
			return nil, err
		}
	}

	// 5. 确保日志目录存在
	if cfg.Log.Output != "" {
		logPath, err := utils.ExpandPath(cfg.Log.Output)
		if err != nil {
			return nil, fmt.Errorf("failed to expand log path: %w", err)
		}
		if logPath != "" {
			logDir := filepath.Dir(logPath)
			if err := ensureDir(logDir, "logs"); err != nil {
				return nil, err
			}
		}
	}

	// 6. 创建默认工具文件（如果不存在）
	if err := createDefaultToolsFiles(workspacePath, cfg); err != nil {
		slog.Warn("Failed to create default tools files", "error", err)
		// 不返回错误，继续执行
	}

	// 7. 创建示例配置文件（如果不存在）
	if err := createExampleConfig(workspacePath); err != nil {
		slog.Debug("Example config creation skipped", "error", err)
	}

	wsConfig.DataDir = filepath.Join(workspacePath, "data")
	wsConfig.LogsDir = filepath.Join(workspacePath, "logs")

	slog.Info("Workspace initialized", "path", workspacePath)
	return wsConfig, nil
}

// ensureDir 确保目录存在，不存在则创建
func ensureDir(path, name string) error {
	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("%s path exists but is not a directory: %s", name, path)
		}
		return nil
	}

	if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check %s directory: %w", name, err)
	}

	// 创建目录
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create %s directory: %w", name, err)
	}

	slog.Debug("Created directory", "name", name, "path", path)
	return nil
}

// createDefaultToolsFiles 创建默认工具文件
func createDefaultToolsFiles(workspacePath string, cfg *Config) error {
	if !cfg.Tools.JS.Enabled {
		return nil
	}

	toolsDir := cfg.Tools.JS.ToolsDir
	if toolsDir == "" {
		toolsDir = "tools"
	}

	toolsPath := filepath.Join(workspacePath, toolsDir)

	// 创建 .gitkeep 文件
	gitkeepPath := filepath.Join(toolsPath, ".gitkeep")
	if _, err := os.Stat(gitkeepPath); os.IsNotExist(err) {
		if err := os.WriteFile(gitkeepPath, []byte{}, 0644); err != nil {
			return fmt.Errorf("failed to create .gitkeep: %w", err)
		}
	}

	// 创建 README.md
	readmePath := filepath.Join(toolsPath, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		readmeContent := `# JavaScript Tools

This directory contains custom JavaScript tools for icooclaw.

## Creating a Tool

Create a .js file with the following structure:

` + "```javascript" + `
// Tool: my_tool
// Description: My custom tool

function my_tool(args) {
    // Your tool logic here
    return "Result";
}
` + "```" + `

## Available APIs

- ` + "`file.read(path)`" + ` - Read file contents
- ` + "`file.write(path, content)`" + ` - Write file contents
- ` + "`http.get(url)`" + ` - HTTP GET request
- ` + "`http.post(url, body)`" + ` - HTTP POST request
`
		if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
			return fmt.Errorf("failed to create tools README: %w", err)
		}
	}

	return nil
}

// createExampleConfig 创建示例配置文件
func createExampleConfig(workspacePath string) error {
	configPath := filepath.Join(workspacePath, "config.toml.example")
	if _, err := os.Stat(configPath); err == nil {
		return nil // 已存在
	}

	exampleConfig := `# icooclaw 配置文件示例
# 复制此文件为 config.toml 并修改配置

[database]
path = "./data/icooclaw.db"

[log]
level = "info"
format = "text"
output = "./logs/icooclaw.log"

[providers.openrouter]
enabled = true
api_key = "your-api-key-here"
model = "anthropic/claude-sonnet-4-20250514"

[agents.defaults]
name = "icooclaw"
model = "anthropic/claude-sonnet-4-20250514"
temperature = 0.7
max_tokens = 4096
memory_window = 10
system_prompt = "You are a helpful AI assistant."

[channels.websocket]
enabled = false
host = "0.0.0.0"
port = 8080

[tools]
enabled = true

[tools.js]
enabled = true
tools_dir = "tools"

[memory]
consolidation_threshold = 50
summary_enabled = true

[scheduler]
enabled = false
heartbeat_interval = 30
`

	return os.WriteFile(configPath, []byte(exampleConfig), 0644)
}

// ValidateWorkspace 验证工作空间是否有效
func ValidateWorkspace(cfg *Config) []error {
	var errors []error

	// 检查工作空间目录
	workspacePath, err := utils.ExpandPath(cfg.Workspace)
	if err != nil {
		errors = append(errors, fmt.Errorf("invalid workspace path: %w", err))
		return errors
	}

	if workspacePath == "" {
		workspacePath = "./workspace"
	}

	// 检查工作空间是否存在
	if info, err := os.Stat(workspacePath); err != nil {
		if os.IsNotExist(err) {
			errors = append(errors, fmt.Errorf("workspace directory does not exist: %s", workspacePath))
		} else {
			errors = append(errors, fmt.Errorf("cannot access workspace: %w", err))
		}
	} else if !info.IsDir() {
		errors = append(errors, fmt.Errorf("workspace path is not a directory: %s", workspacePath))
	}

	// 检查数据库路径
	dbPath, err := utils.ExpandPath(cfg.Database.Path)
	if err != nil {
		errors = append(errors, fmt.Errorf("invalid database path: %w", err))
	} else if dbPath != "" {
		dbDir := filepath.Dir(dbPath)
		if _, err := os.Stat(dbDir); os.IsNotExist(err) {
			errors = append(errors, fmt.Errorf("database directory does not exist: %s", dbDir))
		}
	}

	// 检查日志路径
	if cfg.Log.Output != "" {
		logPath, err := utils.ExpandPath(cfg.Log.Output)
		if err != nil {
			errors = append(errors, fmt.Errorf("invalid log path: %w", err))
		} else {
			logDir := filepath.Dir(logPath)
			if _, err := os.Stat(logDir); os.IsNotExist(err) {
				errors = append(errors, fmt.Errorf("log directory does not exist: %s", logDir))
			}
		}
	}

	return errors
}

// GetWorkspacePaths 获取工作空间相关路径
func GetWorkspacePaths(cfg *Config) map[string]string {
	workspacePath, _ := utils.ExpandPath(cfg.Workspace)
	if workspacePath == "" {
		workspacePath = "./workspace"
	}

	paths := map[string]string{
		"workspace": workspacePath,
		"tools":     filepath.Join(workspacePath, "tools"),
		"data":      filepath.Join(workspacePath, "data"),
		"logs":      filepath.Join(workspacePath, "logs"),
		"memory":    filepath.Join(workspacePath, "memory"),
		"cache":     filepath.Join(workspacePath, "cache"),
	}

	// 添加配置中指定的路径
	if cfg.Database.Path != "" {
		dbPath, _ := utils.ExpandPath(cfg.Database.Path)
		paths["database"] = dbPath
	}

	if cfg.Log.Output != "" {
		logPath, _ := utils.ExpandPath(cfg.Log.Output)
		paths["log_file"] = logPath
	}

	if cfg.Tools.JS.ToolsDir != "" {
		paths["js_tools"] = filepath.Join(workspacePath, cfg.Tools.JS.ToolsDir)
	}

	return paths
}