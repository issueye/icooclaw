// Package tools provides tool management for icooclaw.
package tools

import (
	"context"
	"sort"
	"sync"

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

// Tool represents a tool that can be executed by the agent.
type Tool interface {
	// Name returns the tool name.
	Name() string

	// Description returns the tool description.
	Description() string

	// Parameters returns the tool parameters schema.
	Parameters() map[string]Parameter

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
	tools map[string]Tool
	mu    sync.RWMutex
}

// NewRegistry creates a new tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register registers a tool.
func (r *Registry) Register(tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name()] = tool
}

// Unregister unregisters a tool.
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tools, name)
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

// sortedToolNames returns sorted tool names for deterministic ordering.
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
	tool, err := r.Get(name)
	if err != nil {
		return &Result{
			Success: false,
			Error:   err,
		}
	}
	return tool.Execute(ctx, args)
}

// ExecuteWithContext executes a tool with context injection.
func (r *Registry) ExecuteWithContext(
	ctx context.Context,
	name string,
	args map[string]any,
	channel, chatID string,
	asyncCallback AsyncCallback,
) *Result {
	tool, err := r.Get(name)
	if err != nil {
		return &Result{
			Success: false,
			Error:   err,
		}
	}

	// Inject context
	ctx = WithToolContext(ctx, channel, chatID)

	// Check for async execution
	if asyncExec, ok := tool.(AsyncExecutor); ok && asyncCallback != nil {
		return asyncExec.ExecuteAsync(ctx, args, asyncCallback)
	}

	return tool.Execute(ctx, args)
}

// GetToolDefinitions returns tool definitions for LLM.
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
