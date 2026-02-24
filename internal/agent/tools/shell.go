package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// ShellExecConfig Shell执行配置
type ShellExecConfig struct {
	Allowed   bool
	Timeout   int // 秒
	Workspace string
}

// ShellExecTool Shell执行工具
type ShellExecTool struct {
	baseTool *BaseTool
	config   *ShellExecConfig
}

// NewShellExecTool 创建Shell执行工具
func NewShellExecTool(config *ShellExecConfig) *ShellExecTool {
	tool := NewBaseTool(
		"exec",
		"执行Shell命令。根据配置决定是否启用，默认关闭以保证安全。",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "要执行的Shell命令",
				},
				"timeout": map[string]interface{}{
					"type":        "integer",
					"description": "超时时间（秒），默认30秒",
					"default":     30,
				},
			},
			"required": []string{"command"},
		},
		nil,
	)

	return &ShellExecTool{
		baseTool: tool,
		config:   config,
	}
}

// Name 获取名称
func (t *ShellExecTool) Name() string {
	return t.baseTool.Name()
}

// Description 获取描述
func (t *ShellExecTool) Description() string {
	return t.baseTool.Description()
}

// Parameters 获取参数定义
func (t *ShellExecTool) Parameters() map[string]interface{} {
	return t.baseTool.Parameters()
}

// Execute 执行Shell命令
func (t *ShellExecTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	if !t.config.Allowed {
		return "", fmt.Errorf("shell execution is not allowed")
	}

	command, ok := params["command"].(string)
	if !ok || command == "" {
		return "", fmt.Errorf("invalid or missing command")
	}

	// 获取超时时间
	timeout := t.config.Timeout
	if t, ok := params["timeout"].(float64); ok {
		timeout = int(t)
	}
	if timeout <= 0 {
		timeout = 30
	}

	// 确定使用的shell
	shell := "sh"
	shellArgs := []string{"-c", command}
	if isWindows() {
		shell = "cmd"
		shellArgs = []string{"/c", command}
	}

	// 创建命令
	cmd := exec.CommandContext(ctx, shell, shellArgs...)
	cmd.Dir = t.config.Workspace

	// 设置超时
	timeoutDuration := time.Duration(timeout) * time.Second

	// 执行命令
	done := make(chan struct{})
	var stdout, stderr strings.Builder

	go func() {
		defer close(done)
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		_ = cmd.Run()
	}()

	select {
	case <-done:
		// 命令执行完成
	case <-time.After(timeoutDuration):
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return "", fmt.Errorf("command timed out after %d seconds", timeout)
	}

	// 构建结果
	result := map[string]interface{}{
		"command":  command,
		"exitCode": cmd.ProcessState.ExitCode(),
		"stdout":   stdout.String(),
		"stderr":   stderr.String(),
	}

	// 如果命令执行失败，包含错误信息
	if cmd.ProcessState.ExitCode() != 0 {
		result["error"] = fmt.Sprintf("command exited with code %d", cmd.ProcessState.ExitCode())
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// isWindows 检查是否为Windows系统
func isWindows() bool {
	return strings.Contains(strings.ToLower(runtime.GOOS), "windows")
}

// ToDefinition 转换为工具定义
func (t *ShellExecTool) ToDefinition() ToolDefinition {
	return t.baseTool.ToDefinition()
}
