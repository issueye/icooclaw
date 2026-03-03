package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"icooclaw.ai/provider"
	"icooclaw.core/storage"
)

// ContextBuilder 上下文构建器
type ContextBuilder struct {
	agent   AgentContext
	session *storage.Session
	logger  *slog.Logger
	fs      FileSystemInterface
}

// NewContextBuilder 创建上下文构建器
func NewContextBuilder(agent AgentContext, session *storage.Session) *ContextBuilder {
	return &ContextBuilder{
		agent:   agent,
		session: session,
		logger:  agent.Logger(),
		fs:      OsFileSystem{},
	}
}

// ContextBuilderOption 上下文构建器选项
type ContextBuilderOption func(*ContextBuilder)

// WithFileSystem 设置文件系统实现
func WithFileSystem(fs FileSystemInterface) ContextBuilderOption {
	return func(cb *ContextBuilder) {
		cb.fs = fs
	}
}

// Build 构建上下文
func (cb *ContextBuilder) Build(ctx context.Context) ([]provider.Message, string, error) {
	// 1. 获取系统提示词
	systemPrompt := cb.buildSystemPrompt()

	// 记录系统提示词
	cb.logger.Debug("=== System Prompt ===", "content", systemPrompt)

	// 2. 获取历史消息
	messages, err := cb.buildMessages()
	if err != nil {
		return nil, "", fmt.Errorf("failed to build messages: %w", err)
	}

	// 记录历史消息
	messagesJSON, _ := json.MarshalIndent(messages, "", "  ")
	cb.logger.Debug("=== Context Messages ===", "count", len(messages), "messages", string(messagesJSON))

	return messages, systemPrompt, nil
}

// buildSystemPrompt 构建系统提示词
func (cb *ContextBuilder) buildSystemPrompt() string {
	var parts []string

	// 读取 SOUL.md 文件（AI 身份和人格）
	soulContent := cb.readTemplateFile("SOUL.md")
	if soulContent != "" {
		parts = append(parts, "## 身份与人格")
		parts = append(parts, soulContent)
	}

	// 读取 USER.md 文件（用户信息）
	userContent := cb.readTemplateFile("USER.md")
	if userContent != "" {
		parts = append(parts, "", "## 用户信息")
		parts = append(parts, userContent)
	}

	// 添加用户设定的角色提示词（优先级最高）
	rolePrompt, err := cb.agent.GetSessionRolePrompt(cb.session.ID)
	if err == nil && rolePrompt != "" {
		parts = append(parts, "", "## 角色设定")
		parts = append(parts, rolePrompt)
	}

	// 添加默认系统提示词
	defaultPrompt := cb.agent.GetSystemPrompt()
	if defaultPrompt != "" {
		parts = append(parts, "", "## 系统指令")
		parts = append(parts, defaultPrompt)
	}

	// 添加环境信息
	parts = append(parts, "", "## 环境信息")
	parts = append(parts, fmt.Sprintf("- 操作系统: %s", runtime.GOOS))
	parts = append(parts, fmt.Sprintf("- 工作目录: %s", cb.agent.Workspace()))

	// 读取 workspace 下的记忆文件
	memoryContent := cb.readMemoryFile()
	if memoryContent != "" {
		parts = append(parts, "", "## 用户记忆 (来自 memory/MEMORY.md)")
		parts = append(parts, memoryContent)
	}

	// 添加技能提示词
	skills := cb.agent.Skills().GetLoaded()
	for _, sk := range skills {
		parts = append(parts, "", fmt.Sprintf("## Skill: %s", sk.Name))
		parts = append(parts, sk.Content)
	}

	// 添加记忆
	memories, err := cb.agent.Memory().GetAll()
	if err == nil && len(memories) > 0 {
		parts = append(parts, "", "## Long-term Memory")
		for _, mem := range memories {
			parts = append(parts, mem.Content)
		}
	}

	return strings.Join(parts, "\n")
}

// readTemplateFile 读取模板文件内容
// 优先读取 workspace 目录下的文件，如果不存在则回退到 templates 目录
func (cb *ContextBuilder) readTemplateFile(filename string) string {
	workspace := cb.agent.Workspace()
	if workspace == "" {
		return ""
	}

	// 优先读取 workspace 目录下的文件
	workspacePath := filepath.Join(workspace, filename)
	if _, err := cb.fs.Stat(workspacePath); err == nil {
		content, err := cb.fs.ReadFile(workspacePath)
		if err == nil {
			cb.logger.Debug("Read template from workspace", "file", filename, "path", workspacePath)
			return string(content)
		}
	}

	// 回退到 templates 目录
	templatePath := filepath.Join(workspace, "..", "templates", filename)
	if _, err := cb.fs.Stat(templatePath); os.IsNotExist(err) {
		templatePath = filepath.Join("templates", filename)
		if _, err := cb.fs.Stat(templatePath); os.IsNotExist(err) {
			cb.logger.Debug("Template file not found", "file", filename)
			return ""
		}
	}

	content, err := cb.fs.ReadFile(templatePath)
	if err != nil {
		cb.logger.Warn("Failed to read template file", "file", templatePath, "error", err)
		return ""
	}

	return string(content)
}

// readMemoryFile 读取 workspace/memory/MEMORY.md 文件
func (cb *ContextBuilder) readMemoryFile() string {
	workspace := cb.agent.Workspace()
	if workspace == "" {
		return ""
	}

	memoryPath := filepath.Join(workspace, "memory", "MEMORY.md")
	data, err := cb.fs.ReadFile(memoryPath)
	if err != nil {
		if !os.IsNotExist(err) {
			cb.logger.Warn("Failed to read memory file", "error", err)
		}
		return ""
	}

	content := string(data)

	// 检查用户偏好部分是否有实际内容（不是注释）
	hasUserPreference := cb.hasUserPreference(content)

	// 如果没有用户设定，标记需要提醒用户
	if !hasUserPreference {
		cb.logger.Debug("No user preference set in MEMORY.md")
	}

	return content
}

// hasUserPreference 检查用户偏好部分是否有实际内容
func (cb *ContextBuilder) hasUserPreference(content string) bool {
	// 查找 "## 用户偏好" 部分
	lines := strings.Split(content, "\n")
	inUserPreference := false
	userPrefContent := strings.Builder{}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "##") {
			if inUserPreference {
				// 到达下一个 ## 标题，停止收集
				break
			}
			if strings.Contains(trimmed, "用户偏好") {
				inUserPreference = true
				continue
			}
		}
		if inUserPreference {
			userPrefContent.WriteString(line)
			userPrefContent.WriteString("\n")
		}
	}

	// 检查是否有非注释内容
	userPrefStr := userPrefContent.String()
	lines = strings.Split(userPrefStr, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// 跳过空行和 HTML 注释
		if trimmed == "" || strings.HasPrefix(trimmed, "<!--") {
			continue
		}
		// 发现非注释内容，说明有用户设定
		if !strings.Contains(trimmed, "-->") {
			return true
		}
	}

	return false
}

// CheckUserPreferenceSet 检查用户是否已设置偏好
func (cb *ContextBuilder) CheckUserPreferenceSet() bool {
	workspace := cb.agent.Workspace()
	if workspace == "" {
		return false
	}

	memoryPath := filepath.Join(workspace, "memory", "MEMORY.md")
	data, err := cb.fs.ReadFile(memoryPath)
	if err != nil {
		return false
	}

	return cb.hasUserPreference(string(data))
}

// UpdateMemoryFile 更新 memory/MEMORY.md 文件
func (cb *ContextBuilder) UpdateMemoryFile(section, content string) error {
	workspace := cb.agent.Workspace()
	if workspace == "" {
		return fmt.Errorf("workspace not set")
	}

	memoryPath := filepath.Join(workspace, "memory", "MEMORY.md")

	// 读取现有内容
	var fileContent string
	if data, err := cb.fs.ReadFile(memoryPath); err == nil {
		fileContent = string(data)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read memory file: %w", err)
	} else {
		// 文件不存在，创建默认内容
		fileContent = `# 记忆

此文件存储长期记忆和重要信息。

## 重要事实

<!-- 重要事实和信息将存储在这里 -->

## 用户偏好

<!-- 用户偏好和设置 -->

## 学到的知识

<!-- 从对话中学习的知识 -->

## 最后更新

<!-- 最后记忆更新的时间戳 -->
`
	}

	// 更新指定部分的内容
	updatedContent, err := updateMarkdownSection(fileContent, section, content)
	if err != nil {
		return err
	}

	// 更新最后更新时间
	updatedContent = updateLastUpdated(updatedContent)

	// 写入文件
	if err := cb.fs.WriteFile(memoryPath, []byte(updatedContent), 0644); err != nil {
		return fmt.Errorf("failed to write memory file: %w", err)
	}

	cb.logger.Info("Memory file updated", "section", section)
	return nil
}

// updateMarkdownSection 更新 Markdown 文件中指定部分的内容
func updateMarkdownSection(content, section, newContent string) (string, error) {
	lines := strings.Split(content, "\n")
	result := make([]string, 0, len(lines)+10)
	inSection := false
	sectionFound := false
	sectionWritten := false

	sectionHeader := "## " + section

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 检查是否是 ## 标题
		if strings.HasPrefix(trimmed, "##") {
			// 如果之前在目标部分内且还未写入新内容，先写入新内容
			if inSection && !sectionWritten {
				result = append(result, strings.TrimRight(newContent, "\n"))
				sectionWritten = true
			}

			// 检查是否是目标部分
			if strings.HasPrefix(strings.TrimSpace(line), sectionHeader) ||
				strings.Contains(trimmed, section) {
				inSection = true
				sectionFound = true
				// 写入标题行
				result = append(result, line)
				continue
			} else {
				inSection = false
			}
		}

		// 如果在目标部分内，跳过原有内容（保留注释和空行）
		if inSection {
			if trimmed == "" || strings.HasPrefix(trimmed, "<!") {
				result = append(result, line)
			}
			continue
		}

		result = append(result, line)

		// 如果是最后一个元素且部分未找到，追加新部分
		if i == len(lines)-1 && !sectionFound {
			result = append(result, "", fmt.Sprintf("## %s", section), "", newContent)
		}
	}

	// 如果部分已找到但未写入内容（在文件末尾的情况）
	if sectionFound && !sectionWritten {
		result = append(result, strings.TrimRight(newContent, "\n"))
	}

	return strings.Join(result, "\n"), nil
}

// updateLastUpdated 更新最后更新时间
func updateLastUpdated(content string) string {
	lines := strings.Split(content, "\n")
	result := strings.Builder{}
	inLastUpdated := false
	updated := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "##") {
			if inLastUpdated && !updated {
				result.WriteString(time.Now().Format("2006-01-02 15:04:05"))
				result.WriteString("\n")
				updated = true
			}
			inLastUpdated = false

			if strings.Contains(trimmed, "最后更新") {
				inLastUpdated = true
			}
		}

		if inLastUpdated && strings.Contains(trimmed, "<!--") {
			continue
		}

		result.WriteString(line)
		result.WriteString("\n")
	}

	if !updated {
		result.WriteString(fmt.Sprintf("\n最后更新: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	}

	return result.String()
}

// GetMemoryFile 读取 memory/MEMORY.md 文件内容
func (cb *ContextBuilder) GetMemoryFile() (string, error) {
	workspace := cb.agent.Workspace()
	if workspace == "" {
		return "", fmt.Errorf("workspace not set")
	}

	memoryPath := filepath.Join(workspace, "memory", "MEMORY.md")
	data, err := cb.fs.ReadFile(memoryPath)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// GetMemoryFilePath 获取 memory/MEMORY.md 文件路径
func (cb *ContextBuilder) GetMemoryFilePath() string {
	workspace := cb.agent.Workspace()
	if workspace == "" {
		return ""
	}
	return filepath.Join(workspace, "memory", "MEMORY.md")
}

// buildMessages 构建消息列表
func (cb *ContextBuilder) buildMessages() ([]provider.Message, error) {
	// 获取会话消息
	// messages, err := cb.session.GetMessages(cb.agent.Config().MemoryWindow)
	// if err != nil {
	// 	return nil, err
	// }

	// // 转换为provider.Message
	// result := make([]provider.Message, len(messages))
	// for i, msg := range messages {
	// 	result[i] = provider.Message{
	// 		Role:             msg.Role,
	// 		Content:          msg.Content,
	// 		ReasoningContent: msg.ReasoningContent,
	// 	}

	// 	// 处理tool_calls
	// 	if msg.ToolCalls != "" {
	// 		var calls []provider.ToolCall
	// 		if err := json.Unmarshal([]byte(msg.ToolCalls), &calls); err == nil {
	// 			result[i].ToolCalls = calls
	// 		}
	// 	}

	// 	// 处理tool角色
	// 	if msg.Role == "tool" {
	// 		result[i].ToolCallID = msg.ToolCallID
	// 		result[i].Name = msg.ToolName
	// 	}
	// }

	// return result, nil
	return nil, nil
}

// AddContext 添加额外上下文
func (cb *ContextBuilder) AddContext(contextType, content string) {
	// 可以扩展为添加额外的上下文信息
	cb.logger.Debug("Added context", "type", contextType)
}
