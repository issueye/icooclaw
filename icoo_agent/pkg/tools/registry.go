// Package tools provides tool management for icooclaw.
package tools

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"icooclaw/pkg/errors"
)

// Parameter represents a tool parameter.
type Parameter struct {
	Type        string         `json:"type"`
	Description string         `json:"description"`
	Enum        []string       `json:"enum,omitempty"`
	Default     any            `json:"default,omitempty"`
	Properties  map[string]any `json:"properties,omitempty"`
	Required    []string       `json:"required,omitempty"`
}

// Result represents a tool execution result.
type Result struct {
	Success bool   `json:"success"`
	Content string `json:"content"`
	Error   error  `json:"error,omitempty"`
}

// ToolDefinition represents a tool definition for LLM providers.
type ToolDefinition struct {
	Type     string                 `json:"type"`
	Function ToolFunctionDefinition `json:"function"`
}

// ToolFunctionDefinition represents a function definition for LLM providers.
type ToolFunctionDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// Tool represents a tool that can be executed by the agent.
type Tool interface {
	// Name returns the tool name.
	Name() string

	// Description returns the tool description.
	Description() string

	// Parameters returns the tool parameters schema.
	Parameters() map[string]any

	// Execute executes the tool with given arguments.
	Execute(ctx context.Context, args map[string]any) *Result
}

// AsyncExecutor is an optional interface for async tool execution.
type AsyncExecutor interface {
	ExecuteAsync(ctx context.Context, args map[string]any, callback AsyncCallback) *Result
}

// AsyncCallback is called when async tool execution completes.
type AsyncCallback func(result *Result)

// Registry manages tool registration and execution.
type Registry struct {
	tools  map[string]Tool
	mu     sync.RWMutex
	logger *slog.Logger
}

// NewRegistry creates a new tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools:  make(map[string]Tool),
		logger: slog.Default(),
	}
}

// NewRegistryWithLogger creates a new tool registry with a custom logger.
func NewRegistryWithLogger(logger *slog.Logger) *Registry {
	if logger == nil {
		logger = slog.Default()
	}
	return &Registry{
		tools:  make(map[string]Tool),
		logger: logger,
	}
}

// Register registers a tool.
func (r *Registry) Register(tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := tool.Name()
	if _, exists := r.tools[name]; exists {
		r.logger.Warn("tool registration overwrites existing tool", "name", name)
	}
	r.tools[name] = tool
	r.logger.Debug("tool registered", "name", name)
}

// Unregister unregisters a tool.
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; exists {
		delete(r.tools, name)
		r.logger.Debug("tool unregistered", "name", name)
	}
}

// Get gets a tool by name.
func (r *Registry) Get(name string) (Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, ok := r.tools[name]
	if !ok {
		return nil, errors.ErrToolNotFound
	}
	return tool, nil
}

// GetOK gets a tool by name, returning (tool, true) if found or (nil, false) if not.
// This is useful for checking tool existence without error handling.
func (r *Registry) GetOK(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, ok := r.tools[name]
	return tool, ok
}

// List returns all registered tools.
func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := r.sortedToolNames()
	tools := make([]Tool, 0, len(names))
	for _, name := range names {
		tools = append(tools, r.tools[name])
	}
	return tools
}

// ListNames returns all registered tool names in sorted order.
func (r *Registry) ListNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.sortedToolNames()
}

// sortedToolNames returns sorted tool names for deterministic ordering.
// This is critical for KV cache stability: non-deterministic map iteration would
// produce different system prompts and tool definitions on each call, invalidating
// the LLM's prefix cache even when no tools have changed.
func (r *Registry) sortedToolNames() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Execute executes a tool by name.
func (r *Registry) Execute(ctx context.Context, name string, args map[string]any) *Result {
	return r.ExecuteWithContext(ctx, name, args, "", "", nil)
}

// ExecuteWithContext executes a tool with context injection and optional async callback.
// If the tool implements AsyncExecutor and a non-nil callback is provided,
// ExecuteAsync is called instead of Execute.
func (r *Registry) ExecuteWithContext(
	ctx context.Context,
	name string,
	args map[string]any,
	channel, sessionID string,
	asyncCallback AsyncCallback,
) *Result {
	r.logger.With("name", "【智能体】").Info("工具执行开始",
		"tool", name,
		"channel", channel,
		"session_id", sessionID)

	tool, err := r.Get(name)
	if err != nil {
		r.logger.With("name", "【智能体】").Error("工具不存在",
			"tool", name)
		return &Result{
			Success: false,
			Error:   fmt.Errorf("tool %q not found", name),
		}
	}

	// Inject context
	ctx = WithToolContext(ctx, channel, sessionID)

	// Execute with timing
	start := time.Now()
	var result *Result

	// Check for async execution
	if asyncExec, ok := tool.(AsyncExecutor); ok && asyncCallback != nil {
		r.logger.With("name", "【智能体】").Info("异步执行工具",
			"tool", name)
		result = asyncExec.ExecuteAsync(ctx, args, asyncCallback)
	} else {
		result = tool.Execute(ctx, args)
	}
	duration := time.Since(start)

	// Log based on result type
	if result.Error != nil {
		r.logger.With("name", "【智能体】").Error("工具执行失败",
			"tool", name,
			"duration_ms", duration.Milliseconds(),
			"error", result.Error)
	} else {
		r.logger.With("name", "【智能体】").Info("工具执行完成",
			"tool", name,
			"duration_ms", duration.Milliseconds(),
			"result_length", len(result.Content))
	}

	return result
}

// GetToolDefinitions returns tool definitions for LLM.
// Deprecated: Use ToProviderDefs for provider-compatible format.
func (r *Registry) GetToolDefinitions() []map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := r.sortedToolNames()
	definitions := make([]map[string]any, 0, len(names))

	for _, name := range names {
		tool := r.tools[name]
		def := map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        tool.Name(),
				"description": tool.Description(),
				"parameters": map[string]any{
					"type":       "object",
					"properties": tool.Parameters(),
				},
			},
		}
		definitions = append(definitions, def)
	}

	return definitions
}

// ToProviderDefs converts tool definitions to provider-compatible format.
// This is the format expected by LLM provider APIs.
func (r *Registry) ToProviderDefs() []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := r.sortedToolNames()
	definitions := make([]ToolDefinition, 0, len(names))

	for _, name := range names {
		tool := r.tools[name]
		definitions = append(definitions, ToolDefinition{
			Type: "function",
			Function: ToolFunctionDefinition{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters: map[string]any{
					"type":       "object",
					"properties": tool.Parameters(),
				},
			},
		})
	}

	return definitions
}

// GetSummaries returns human-readable summaries of all registered tools.
// Returns a slice of "name - description" strings.
func (r *Registry) GetSummaries() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := r.sortedToolNames()
	summaries := make([]string, 0, len(names))
	for _, name := range names {
		tool := r.tools[name]
		summaries = append(summaries, fmt.Sprintf("- `%s` - %s", tool.Name(), tool.Description()))
	}
	return summaries
}

// HasTool checks if a tool exists.
func (r *Registry) HasTool(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.tools[name]
	return ok
}

// Count returns the number of registered tools.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}

// ToolToSchema converts a Tool to a JSON schema map.
func ToolToSchema(tool Tool) map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        tool.Name(),
			"description": tool.Description(),
			"parameters": map[string]any{
				"type":       "object",
				"properties": tool.Parameters(),
			},
		},
	}
}

// ErrorResult creates a Result with an error message.
func ErrorResult(content string) *Result {
	return &Result{
		Success: false,
		Content: content,
		Error:   fmt.Errorf("%s", content),
	}
}

// SuccessResult creates a successful Result.
func SuccessResult(content string) *Result {
	return &Result{
		Success: true,
		Content: content,
	}
}
