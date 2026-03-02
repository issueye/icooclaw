package builtons

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/icooclaw/icooclaw/internal/script/config"
	"github.com/icooclaw/icooclaw/pkg/utils"
)

// === shell 命令执行对象 ===

type shellExec struct {
	cfg    *config.Config
	logger *slog.Logger
	ctx    context.Context
}

func NewShellExec(ctx context.Context, cfg *config.Config, logger *slog.Logger) *shellExec {
	return &shellExec{
		cfg:    cfg,
		logger: logger,
		ctx:    ctx,
	}
}

// Exec 执行命令
func (s *shellExec) Exec(command string) (map[string]interface{}, error) {
	return s.ExecWithTimeout(command, s.cfg.ExecTimeout)
}

// ExecWithTimeout 执行命令（带超时）
func (s *shellExec) ExecWithTimeout(command string, timeoutSeconds int) (map[string]interface{}, error) {
	if !s.cfg.AllowExec {
		return nil, fmt.Errorf("shell execution is not allowed")
	}

	if timeoutSeconds <= 0 {
		timeoutSeconds = s.cfg.ExecTimeout
	}
	if timeoutSeconds <= 0 {
		timeoutSeconds = 30
	}

	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	// 确定使用的 shell
	shell := "sh"
	shellArgs := []string{"-c", command}
	if utils.IsWindows() {
		shell = "cmd"
		shellArgs = []string{"/c", command}
	}

	cmd := exec.CommandContext(ctx, shell, shellArgs...)
	cmd.Dir = s.cfg.Workspace

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime).String()

	result := map[string]interface{}{
		"stdout":    stdout.String(),
		"stderr":    stderr.String(),
		"duration":  duration,
		"success":   err == nil,
		"timed_out": ctx.Err() == context.DeadlineExceeded,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result["exit_code"] = exitErr.ExitCode()
		} else {
			result["exit_code"] = -1
			result["error"] = err.Error()
		}
	} else {
		result["exit_code"] = 0
	}

	return result, nil
}

// ExecInDir 在指定目录执行命令
func (s *shellExec) ExecInDir(command string, workDir string) (map[string]interface{}, error) {
	if !s.cfg.AllowExec {
		return nil, fmt.Errorf("shell execution is not allowed")
	}

	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	timeout := time.Duration(s.cfg.ExecTimeout) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	shell := "sh"
	shellArgs := []string{"-c", command}
	if utils.IsWindows() {
		shell = "cmd"
		shellArgs = []string{"/c", command}
	}

	cmd := exec.CommandContext(ctx, shell, shellArgs...)
	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime).String()

	result := map[string]interface{}{
		"stdout":    stdout.String(),
		"stderr":    stderr.String(),
		"duration":  duration,
		"success":   err == nil,
		"timed_out": ctx.Err() == context.DeadlineExceeded,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result["exit_code"] = exitErr.ExitCode()
		} else {
			result["exit_code"] = -1
			result["error"] = err.Error()
		}
	} else {
		result["exit_code"] = 0
	}

	return result, nil
}

// Pipe 执行管道命令
func (s *shellExec) Pipe(commands []string) (map[string]interface{}, error) {
	if !s.cfg.AllowExec {
		return nil, fmt.Errorf("shell execution is not allowed")
	}

	// 简化实现：将命令用管道连接
	combinedCommand := strings.Join(commands, " | ")
	return s.Exec(combinedCommand)
}
