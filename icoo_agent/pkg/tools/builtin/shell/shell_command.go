package shell

import (
	"context"
	"encoding/json"
	"fmt"
	"icooclaw/pkg/tools"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// ShellCommandTool 提供 shell 命令执行功能。
type ShellCommandTool struct {
	// WorkDir 工作目录，命令执行的基础目录
	WorkDir string
	// Timeout 默认超时时间（秒）
	Timeout int
	// AllowedCommands 允许执行的命令列表（为空表示允许所有）
	AllowedCommands []string
	// BlockedCommands 禁止执行的命令列表
	BlockedCommands []string
}

// ShellCommandOption 配置选项。
type ShellCommandOption func(*ShellCommandTool)

// WithWorkDir 设置工作目录。
func WithWorkDir(dir string) ShellCommandOption {
	return func(t *ShellCommandTool) {
		t.WorkDir = dir
	}
}

// WithTimeout 设置默认超时时间。
func WithTimeout(seconds int) ShellCommandOption {
	return func(t *ShellCommandTool) {
		t.Timeout = seconds
	}
}

// WithAllowedCommands 设置允许的命令列表。
func WithAllowedCommands(commands []string) ShellCommandOption {
	return func(t *ShellCommandTool) {
		t.AllowedCommands = commands
	}
}

// WithBlockedCommands 设置禁止的命令列表。
func WithBlockedCommands(commands []string) ShellCommandOption {
	return func(t *ShellCommandTool) {
		t.BlockedCommands = commands
	}
}

// NewShellCommandTool 创建一个新的 shell 命令工具。
func NewShellCommandTool(opts ...ShellCommandOption) *ShellCommandTool {
	t := &ShellCommandTool{
		Timeout:         60, // 默认 60 秒超时
		AllowedCommands: []string{},
		BlockedCommands: []string{
			"rm -rf /",
			"mkfs",
			"dd if=/dev/zero",
			":(){ :|:& };:", // Fork bomb
		},
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

// Name 返回工具名称。
func (t *ShellCommandTool) Name() string {
	return "shell_command"
}

// Description 返回工具描述。
func (t *ShellCommandTool) Description() string {
	return "执行 shell 命令并返回输出结果。支持设置超时时间和工作目录。"
}

// Parameters 返回工具参数定义。
func (t *ShellCommandTool) Parameters() map[string]any {
	return map[string]any{
		"command": map[string]any{
			"type":        "string",
			"description": "要执行的 shell 命令",
		},
		"timeout": map[string]any{
			"type":        "integer",
			"description": "超时时间（秒），默认 60 秒",
		},
		"work_dir": map[string]any{
			"type":        "string",
			"description": "工作目录（可选）",
		},
		"env": map[string]any{
			"type":        "array",
			"description": "环境变量列表，格式为 'KEY=value'",
			"items": map[string]any{
				"type": "string",
			},
		},
	}
}

// Execute 执行 shell 命令。
func (t *ShellCommandTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	command, ok := args["command"].(string)
	if !ok || command == "" {
		return &tools.Result{Success: false, Error: fmt.Errorf("需要提供 command 参数")}
	}

	// 安全检查
	if err := t.checkCommand(command); err != nil {
		return &tools.Result{Success: false, Error: err}
	}

	// 获取超时时间
	timeout := t.Timeout
	if t, ok := args["timeout"].(float64); ok {
		timeout = int(t)
	}
	if timeout <= 0 {
		timeout = 60
	}

	// 获取工作目录
	workDir := t.WorkDir
	if wd, ok := args["work_dir"].(string); ok && wd != "" {
		workDir = wd
	}

	// 获取环境变量
	var env []string
	if e, ok := args["env"].([]any); ok {
		for _, v := range e {
			if s, ok := v.(string); ok {
				env = append(env, s)
			}
		}
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// 执行命令
	result := t.runCommand(ctx, command, workDir, env)
	return result
}

// checkCommand 检查命令是否被允许执行。
func (t *ShellCommandTool) checkCommand(command string) error {
	// 检查禁止的命令
	for _, blocked := range t.BlockedCommands {
		if strings.Contains(command, blocked) {
			return fmt.Errorf("命令被禁止执行: 包含危险操作 '%s'", blocked)
		}
	}

	// 如果设置了允许列表，检查命令是否在允许列表中
	if len(t.AllowedCommands) > 0 {
		allowed := false
		for _, a := range t.AllowedCommands {
			if strings.HasPrefix(command, a) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("命令不在允许列表中")
		}
	}

	return nil
}

// runCommand 执行命令并返回结果。
func (t *ShellCommandTool) runCommand(ctx context.Context, command, workDir string, env []string) *tools.Result {
	var cmd *exec.Cmd

	// 根据操作系统选择 shell
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd.exe", "/c", command)
	} else {
		cmd = exec.CommandContext(ctx, "/bin/sh", "-c", command)
	}

	// 设置工作目录
	if workDir != "" {
		cmd.Dir = workDir
	}

	// 设置环境变量
	if len(env) > 0 {
		cmd.Env = append(cmd.Env, env...)
	}

	// 执行命令并获取输出
	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	// 构建结果
	result := map[string]any{
		"command":      command,
		"duration_ms":  duration.Milliseconds(),
		"output":       string(output),
		"success":      err == nil,
		"exit_code":    0,
		"timed_out":    false,
		"work_dir":     workDir,
		"platform":     runtime.GOOS,
	}

	// 处理错误
	if err != nil {
		// 检查是否超时
		if ctx.Err() == context.DeadlineExceeded {
			result["timed_out"] = true
			result["error"] = fmt.Sprintf("命令执行超时（%d秒）", int(duration.Seconds()))
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			result["exit_code"] = exitErr.ExitCode()
			// 命令执行失败但输出仍然有效，不设置 error 字段
			// 让调用者根据 exit_code 和 output 判断
		} else {
			result["error"] = err.Error()
		}
	}

	// 截断过长的输出
	outputStr := string(output)
	if len(outputStr) > 10000 {
		result["output"] = outputStr[:10000] + "\n... (输出已截断)"
		result["truncated"] = true
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")

	// 工具执行成功（即使命令返回非零退出码）
	// 只有超时或无法执行命令时才返回错误
	toolErr := error(nil)
	if ctx.Err() == context.DeadlineExceeded {
		toolErr = fmt.Errorf("命令执行超时")
	}

	return &tools.Result{
		Success: toolErr == nil,
		Content: string(resultJSON),
		Error:   toolErr,
	}
}