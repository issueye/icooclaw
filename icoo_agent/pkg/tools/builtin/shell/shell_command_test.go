package shell

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func TestShellCommandTool_Name(t *testing.T) {
	tool := NewShellCommandTool()
	if tool.Name() != "shell_command" {
		t.Errorf("Expected name 'shell_command', got '%s'", tool.Name())
	}
}

func TestShellCommandTool_Description(t *testing.T) {
	tool := NewShellCommandTool()
	if tool.Description() == "" {
		t.Error("Description should not be empty")
	}
}

func TestShellCommandTool_Parameters(t *testing.T) {
	tool := NewShellCommandTool()
	params := tool.Parameters()

	if _, ok := params["command"]; !ok {
		t.Error("Missing 'command' parameter")
	}
	if _, ok := params["timeout"]; !ok {
		t.Error("Missing 'timeout' parameter")
	}
	if _, ok := params["work_dir"]; !ok {
		t.Error("Missing 'work_dir' parameter")
	}
	if _, ok := params["env"]; !ok {
		t.Error("Missing 'env' parameter")
	}
}

func TestShellCommandTool_ExecuteSimpleCommand(t *testing.T) {
	tool := NewShellCommandTool()

	var command string
	if runtime.GOOS == "windows" {
		command = "echo hello"
	} else {
		command = "echo hello"
	}

	result := tool.Execute(context.Background(), map[string]any{
		"command": command,
	})

	if !result.Success {
		t.Errorf("Command should succeed, error: %v", result.Error)
	}
	if result.Content == "" {
		t.Error("Content should not be empty")
	}
}

func TestShellCommandTool_ExecuteWithTimeout(t *testing.T) {
	tool := NewShellCommandTool(WithTimeout(1))

	result := tool.Execute(context.Background(), map[string]any{
		"command": "echo test",
		"timeout": 5,
	})

	if !result.Success {
		t.Errorf("Command should succeed, error: %v", result.Error)
	}
}

func TestShellCommandTool_ExecuteMissingCommand(t *testing.T) {
	tool := NewShellCommandTool()

	result := tool.Execute(context.Background(), map[string]any{})

	if result.Success {
		t.Error("Should fail when command is missing")
	}
	if result.Error == nil {
		t.Error("Error should not be nil")
	}
}

func TestShellCommandTool_ExecuteBlockedCommand(t *testing.T) {
	tool := NewShellCommandTool()

	result := tool.Execute(context.Background(), map[string]any{
		"command": "rm -rf /",
	})

	if result.Success {
		t.Error("Blocked command should fail")
	}
	if result.Error == nil {
		t.Error("Error should not be nil for blocked command")
	}
}

func TestShellCommandTool_ExecuteWithAllowedCommands(t *testing.T) {
	tool := NewShellCommandTool(WithAllowedCommands([]string{"echo", "ls"}))

	// 允许的命令
	result := tool.Execute(context.Background(), map[string]any{
		"command": "echo hello",
	})
	if !result.Success {
		t.Errorf("Allowed command should succeed, error: %v", result.Error)
	}

	// 不允许的命令
	result = tool.Execute(context.Background(), map[string]any{
		"command": "pwd",
	})
	if result.Success {
		t.Error("Non-allowed command should fail")
	}
}

func TestShellCommandTool_ExecuteTimeout(t *testing.T) {
	tool := NewShellCommandTool()

	// 使用非常短的超时来测试超时场景
	var command string
	if runtime.GOOS == "windows" {
		command = "ping -n 10 127.0.0.1"
	} else {
		command = "sleep 10"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result := tool.Execute(ctx, map[string]any{
		"command": command,
		"timeout": 1,
	})

	// 由于超时，命令应该失败
	if result.Success {
		t.Error("Command should fail due to timeout")
	}
}

func TestShellCommandTool_Options(t *testing.T) {
	tool := NewShellCommandTool(
		WithWorkDir("/tmp"),
		WithTimeout(30),
		WithAllowedCommands([]string{"echo"}),
		WithBlockedCommands([]string{"dangerous"}),
	)

	if tool.WorkDir != "/tmp" {
		t.Errorf("Expected WorkDir '/tmp', got '%s'", tool.WorkDir)
	}
	if tool.Timeout != 30 {
		t.Errorf("Expected Timeout 30, got %d", tool.Timeout)
	}
	if len(tool.AllowedCommands) != 1 {
		t.Errorf("Expected 1 allowed command, got %d", len(tool.AllowedCommands))
	}
	if len(tool.BlockedCommands) == 0 {
		t.Error("BlockedCommands should not be empty")
	}
}

func TestShellCommandTool_ExecuteWithEnv(t *testing.T) {
	tool := NewShellCommandTool()

	var command string
	if runtime.GOOS == "windows" {
		command = "echo %TEST_VAR%"
	} else {
		command = "echo $TEST_VAR"
	}

	result := tool.Execute(context.Background(), map[string]any{
		"command": command,
		"env":     []any{"TEST_VAR=hello_world"},
	})

	if !result.Success {
		t.Errorf("Command should succeed, error: %v", result.Error)
	}
}