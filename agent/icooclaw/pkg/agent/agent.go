// Package agent provides the core agent implementation for icooclaw.
package agent

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/memory"
	"icooclaw/pkg/providers"
	"icooclaw/pkg/skill"
	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"
)

// AgentConfig holds configuration for an agent instance.
type AgentConfig struct {
	Name             string
	SystemPrompt     string
	Model            string
	MaxTokens        int
	Temperature      float64
	MaxToolIterations int
}

// AgentInstance represents a configured agent instance.
// It holds agent-specific configuration and state, while delegating
// message processing to the Loop.
type AgentInstance struct {
	config AgentConfig

	// Core dependencies
	provider providers.Provider
	tools    *tools.Registry
	memory   memory.Loader
	skills   skill.Loader
	storage  *storage.Storage
	bus      *bus.MessageBus

	// State
	sessionCache sync.Map // session key -> last active time

	logger *slog.Logger
}

// NewAgentInstance creates a new AgentInstance.
func NewAgentInstance(config AgentConfig, opts ...AgentOption) *AgentInstance {
	a := &AgentInstance{
		config: config,
		tools:  tools.NewRegistry(),
		logger: slog.Default(),
	}

	for _, opt := range opts {
		opt(a)
	}

	return a
}

// AgentOption is a functional option for AgentInstance.
type AgentOption func(*AgentInstance)

// WithAgentProvider sets the provider.
func WithAgentProvider(p providers.Provider) AgentOption {
	return func(a *AgentInstance) { a.provider = p }
}

// WithAgentTools sets the tools registry.
func WithAgentTools(t *tools.Registry) AgentOption {
	return func(a *AgentInstance) { a.tools = t }
}

// WithAgentMemory sets the memory loader.
func WithAgentMemory(m memory.Loader) AgentOption {
	return func(a *AgentInstance) { a.memory = m }
}

// WithAgentSkills sets the skill loader.
func WithAgentSkills(s skill.Loader) AgentOption {
	return func(a *AgentInstance) { a.skills = s }
}

// WithAgentStorage sets the storage.
func WithAgentStorage(s *storage.Storage) AgentOption {
	return func(a *AgentInstance) { a.storage = s }
}

// WithAgentBus sets the message bus.
func WithAgentBus(b *bus.MessageBus) AgentOption {
	return func(a *AgentInstance) { a.bus = b }
}

// WithAgentLogger sets the logger.
func WithAgentLogger(l *slog.Logger) AgentOption {
	return func(a *AgentInstance) { a.logger = l }
}

// Name returns the agent name.
func (a *AgentInstance) Name() string {
	return a.config.Name
}

// Config returns the agent configuration.
func (a *AgentInstance) Config() AgentConfig {
	return a.config
}

// Tools returns the tools registry.
func (a *AgentInstance) Tools() *tools.Registry {
	return a.tools
}

// Provider returns the provider.
func (a *AgentInstance) Provider() providers.Provider {
	return a.provider
}

// Memory returns the memory loader.
func (a *AgentInstance) Memory() memory.Loader {
	return a.memory
}

// RegisterTool registers a tool with this agent.
func (a *AgentInstance) RegisterTool(tool tools.Tool) {
	a.tools.Register(tool)
}

// GetSessionKey returns a session key for the given channel and chat ID.
func (a *AgentInstance) GetSessionKey(channel, chatID string) string {
	return fmt.Sprintf("%s:%s:%s", a.config.Name, channel, chatID)
}

// AgentRegistry manages multiple agent instances.
type AgentRegistry struct {
	agents   map[string]*AgentInstance
	defaults *AgentConfig
	mu       sync.RWMutex
	logger   *slog.Logger
}

// NewAgentRegistry creates a new agent registry.
func NewAgentRegistry(logger *slog.Logger) *AgentRegistry {
	if logger == nil {
		logger = slog.Default()
	}
	return &AgentRegistry{
		agents: make(map[string]*AgentInstance),
		logger: logger,
	}
}

// Register registers an agent instance.
func (r *AgentRegistry) Register(agent *AgentInstance) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.agents[agent.Name()] = agent
	r.logger.Debug("agent registered", "name", agent.Name())
}

// Get gets an agent by name.
func (r *AgentRegistry) Get(name string) (*AgentInstance, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agent, ok := r.agents[name]
	return agent, ok
}

// GetDefault returns the default agent.
func (r *AgentRegistry) GetDefault() *AgentInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// First try "default" name
	if agent, ok := r.agents["default"]; ok {
		return agent
	}

	// Fall back to first registered agent
	for _, agent := range r.agents {
		return agent
	}

	return nil
}

// List returns all agent names.
func (r *AgentRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.agents))
	for name := range r.agents {
		names = append(names, name)
	}
	return names
}

// SetDefaults sets the default agent configuration.
func (r *AgentRegistry) SetDefaults(config AgentConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.defaults = &config
}

// GetDefaults returns the default agent configuration.
func (r *AgentRegistry) GetDefaults() *AgentConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.defaults
}

// --- Legacy Agent type for backward compatibility ---

// Agent represents an AI agent (legacy type for backward compatibility).
// Deprecated: Use AgentInstance and Loop instead.
type Agent = AgentInstance

// Option is a functional option for Agent (legacy).
// Deprecated: Use AgentOption instead.
type Option = AgentOption

// WithProvider sets the provider (legacy).
// Deprecated: Use WithAgentProvider instead.
var WithProvider = WithAgentProvider

// WithTools sets the tools registry (legacy).
// Deprecated: Use WithAgentTools instead.
var WithTools = WithAgentTools

// WithMemory sets the memory loader (legacy).
// Deprecated: Use WithAgentMemory instead.
var WithMemory = WithAgentMemory

// WithSkills sets the skill loader (legacy).
// Deprecated: Use WithAgentSkills instead.
var WithSkills = WithAgentSkills

// WithStorage sets the storage (legacy).
// Deprecated: Use WithAgentStorage instead.
var WithStorage = WithAgentStorage

// WithBus sets the message bus (legacy).
// Deprecated: Use WithAgentBus instead.
var WithBus = WithAgentBus

// WithLogger sets the logger (legacy).
// Deprecated: Use WithAgentLogger instead.
var WithLogger = WithAgentLogger

// New creates a new Agent (legacy).
// Deprecated: Use NewAgentInstance instead.
func New(name string, opts ...Option) *Agent {
	return NewAgentInstance(AgentConfig{Name: name}, opts...)
}

// --- Helper functions ---

// BuildSystemPrompt builds a system prompt from various sources.
func BuildSystemPrompt(basePrompt string, skills []string, skillLoader skill.Loader) string {
	if skillLoader == nil || len(skills) == 0 {
		return basePrompt
	}

	ctx := context.Background()
	var skillPrompts string

	for _, skillName := range skills {
		skill, err := skillLoader.Load(ctx, skillName)
		if err != nil {
			continue
		}
		if skill.Prompt != "" {
			skillPrompts += "\n\n## " + skillName + "\n" + skill.Prompt
		}
	}

	if skillPrompts != "" {
		return basePrompt + "\n\n# Skills" + skillPrompts
	}

	return basePrompt
}