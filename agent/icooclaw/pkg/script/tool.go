// Package script provides JavaScript scripting engine for icooclaw.
package script

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dop251/goja"
	"icooclaw/pkg/tools"
)

// ScriptTool wraps the script engine as a tool.
type ScriptTool struct {
	engine *Engine
	logger *slog.Logger
}

// NewScriptTool creates a new script tool.
func NewScriptTool(cfg *Config, logger *slog.Logger) *ScriptTool {
	return &ScriptTool{
		engine: NewEngine(cfg, logger),
		logger: logger,
	}
}

// Name returns the tool name.
func (t *ScriptTool) Name() string {
	return "script"
}

// Description returns the tool description.
func (t *ScriptTool) Description() string {
	return "Execute JavaScript code. Use for data processing, calculations, and automation tasks."
}

// Parameters returns the tool parameters.
func (t *ScriptTool) Parameters() map[string]any {
	return map[string]any{
		"code": map[string]any{
			"type":        "string",
			"description": "JavaScript code to execute",
		},
		"timeout": map[string]any{
			"type":        "integer",
			"description": "Execution timeout in seconds (default: 30)",
		},
	}
}

// Execute executes the script.
func (t *ScriptTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	code, ok := args["code"].(string)
	if !ok {
		return &tools.Result{
			Success: false,
			Error:   fmt.Errorf("code parameter is required"),
		}
	}

	// Set context for the engine
	t.engine.SetContext(ctx)

	// Run the script
	value, err := t.engine.Run(code)
	if err != nil {
		return &tools.Result{
			Success: false,
			Error:   err,
		}
	}

	// Convert result to string
	var result string
	if value != nil && !goja.IsUndefined(value) {
		result = value.String()
	} else {
		result = "undefined"
	}

	return &tools.Result{
		Success: true,
		Content: result,
	}
}

// ScriptFileTool executes script files.
type ScriptFileTool struct {
	engine *Engine
	logger *slog.Logger
}

// NewScriptFileTool creates a new script file tool.
func NewScriptFileTool(cfg *Config, logger *slog.Logger) *ScriptFileTool {
	return &ScriptFileTool{
		engine: NewEngine(cfg, logger),
		logger: logger,
	}
}

// Name returns the tool name.
func (t *ScriptFileTool) Name() string {
	return "script_file"
}

// Description returns the tool description.
func (t *ScriptFileTool) Description() string {
	return "Execute a JavaScript file from the workspace."
}

// Parameters returns the tool parameters.
func (t *ScriptFileTool) Parameters() map[string]any {
	return map[string]any{
		"path": map[string]any{
			"type":        "string",
			"description": "Path to the JavaScript file (relative to workspace)",
		},
		"args": map[string]any{
			"type":        "object",
			"description": "Arguments to pass to the script (available as global 'args' variable)",
		},
	}
}

// Execute executes the script file.
func (t *ScriptFileTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	path, ok := args["path"].(string)
	if !ok {
		return &tools.Result{
			Success: false,
			Error:   fmt.Errorf("path parameter is required"),
		}
	}

	// Set context for the engine
	t.engine.SetContext(ctx)

	// Set args as global variable
	if scriptArgs, ok := args["args"].(map[string]any); ok {
		t.engine.SetGlobal("args", scriptArgs)
	}

	// Run the script file
	value, err := t.engine.RunFile(path)
	if err != nil {
		return &tools.Result{
			Success: false,
			Error:   err,
		}
	}

	// Convert result to string
	var result string
	if value != nil && !goja.IsUndefined(value) {
		result = value.String()
	} else {
		result = "undefined"
	}

	return &tools.Result{
		Success: true,
		Content: result,
	}
}

// RegisterScriptTools registers script tools to the registry.
func RegisterScriptTools(registry *tools.Registry, cfg *Config, logger *slog.Logger) {
	registry.Register(NewScriptTool(cfg, logger))
	registry.Register(NewScriptFileTool(cfg, logger))
}
