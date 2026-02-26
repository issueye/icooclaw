package tools

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// MemoryUpdateConfig 记忆更新工具配置
type MemoryUpdateConfig struct {
	Agent interface {
		UpdateSoulFile(section, content string) error
		UpdateUserFile(section, content string) error
	}
	Logger *slog.Logger
}

// MemoryUpdateTool 记忆更新工具
// 用于更新 SOUL.md 和 USER.md 文件
type MemoryUpdateTool struct {
	config *MemoryUpdateConfig
}

// NewMemoryUpdateTool 创建记忆更新工具
func NewMemoryUpdateTool(config *MemoryUpdateConfig) *MemoryUpdateTool {
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	return &MemoryUpdateTool{
		config: config,
	}
}

// Name 获取工具名称
func (t *MemoryUpdateTool) Name() string {
	return "memory_update"
}

// Description 获取工具描述
func (t *MemoryUpdateTool) Description() string {
	return "更新记忆文件，包括设置AI名字、用户称呼等。用于：当用户告诉你他的名字时设置AI名字，或当用户告诉你希望如何称呼时设置用户称呼。"
}

// Parameters 获取参数定义
func (t *MemoryUpdateTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file": map[string]interface{}{
				"type":        "string",
				"description": "要更新的文件：soul(SOUL.md) 或 user(USER.md)",
				"enum":        []string{"soul", "user"},
			},
			"section": map[string]interface{}{
				"type":        "string",
				"description": "要更新的部分：如'身份'、'用户称呼'等",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "要设置的内容",
			},
		},
		"required": []string{"file", "section", "content"},
	}
}

// Execute 执行工具
func (t *MemoryUpdateTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	if t.config == nil || t.config.Agent == nil {
		return "", fmt.Errorf("memory update tool not initialized properly")
	}

	file, _ := params["file"].(string)
	section, _ := params["section"].(string)
	content, _ := params["content"].(string)

	// 添加调试日志
	if t.config.Logger != nil {
		t.config.Logger.Debug("memory_update called",
			"file", file,
			"section", section,
			"content", content)
	}

	// 验证参数
	if file == "" {
		return "", fmt.Errorf("file parameter is required")
	}
	if section == "" {
		return "", fmt.Errorf("section parameter is required")
	}
	if content == "" {
		return "", fmt.Errorf("content parameter is required")
	}

	// 清理内容中的引号
	content = strings.Trim(content, "\"")

	// 根据文件类型调用不同的更新方法
	switch strings.ToLower(file) {
	case "soul":
		err := t.config.Agent.UpdateSoulFile(section, content)
		if err != nil {
			if t.config.Logger != nil {
				t.config.Logger.Error("failed to update soul file", "error", err)
			}
			return "", fmt.Errorf("failed to update soul file: %w", err)
		}
		if t.config.Logger != nil {
			t.config.Logger.Debug("soul file updated successfully", "section", section, "content", content)
		}
		return fmt.Sprintf("已成功更新 SOUL.md 的 '%s' 部分为: %s", section, content), nil

	case "user":
		err := t.config.Agent.UpdateUserFile(section, content)
		if err != nil {
			if t.config.Logger != nil {
				t.config.Logger.Error("failed to update user file", "error", err)
			}
			return "", fmt.Errorf("failed to update user file: %w", err)
		}
		if t.config.Logger != nil {
			t.config.Logger.Debug("user file updated successfully", "section", section, "content", content)
		}
		return fmt.Sprintf("已成功更新 USER.md 的 '%s' 部分为: %s", section, content), nil

	default:
		return "", fmt.Errorf("unknown file type: %s, valid values are: soul, user", file)
	}
}

// ToDefinition 转换为工具定义
func (t *MemoryUpdateTool) ToDefinition() ToolDefinition {
	return ToolDefinition{
		Type: "function",
		Function: FunctionDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Parameters(),
		},
	}
}
