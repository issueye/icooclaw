// Package agent provides the core agent implementation for icooclaw.
package agent

import (
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
	Name              string
	SystemPrompt      string
	Model             string
	MaxTokens         int
	Temperature       float64
	MaxToolIterations int
}

// Instance represents a configured agent instance.
// It holds agent-specific configuration and state, while delegating
// message processing to the Loop.
type Instance struct {
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

// NewInstance creates a new Instance.
func NewInstance(config AgentConfig, opts ...AgentOption) *Instance {
	a := &Instance{
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
type AgentOption func(*Instance)

// WithProvider sets the provider.
func WithProvider(p providers.Provider) AgentOption {
	return func(a *Instance) { a.provider = p }
}

// WithTools sets the tools registry.
func WithTools(t *tools.Registry) AgentOption {
	return func(a *Instance) { a.tools = t }
}

// WithMemory sets the memory loader.
func WithMemory(m memory.Loader) AgentOption {
	return func(a *Instance) { a.memory = m }
}

// WithSkills sets the skill loader.
func WithSkills(s skill.Loader) AgentOption {
	return func(a *Instance) { a.skills = s }
}

// WithStorage sets the storage.
func WithStorage(s *storage.Storage) AgentOption {
	return func(a *Instance) { a.storage = s }
}

// WithBus sets the message bus.
func WithBus(b *bus.MessageBus) AgentOption {
	return func(a *Instance) { a.bus = b }
}

// WithLogger sets the logger.
func WithLogger(l *slog.Logger) AgentOption {
	return func(a *Instance) { a.logger = l }
}

// Name returns the agent name.
func (a *Instance) Name() string {
	return a.config.Name
}

// Config returns the agent configuration.
func (a *Instance) Config() AgentConfig {
	return a.config
}

// Tools returns the tools registry.
func (a *Instance) Tools() *tools.Registry {
	return a.tools
}

// Provider returns the provider.
func (a *Instance) Provider() providers.Provider {
	return a.provider
}

// Memory returns the memory loader.
func (a *Instance) Memory() memory.Loader {
	return a.memory
}

// RegisterTool registers a tool with this agent.
func (a *Instance) RegisterTool(tool tools.Tool) {
	a.tools.Register(tool)
}

// GetSessionKey returns a session key for the given channel and session ID.
// Format: channel:sessionID
func (a *Instance) GetSessionKey(channel, sessionID string) string {
	return fmt.Sprintf("%s:%s", channel, sessionID)
}

// AgentRegistry manages multiple agent instances.
type AgentRegistry struct {
	agents   map[string]*Instance
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
		agents: make(map[string]*Instance),
		logger: logger,
	}
}

// Register registers an agent instance.
func (r *AgentRegistry) Register(agent *Instance) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.agents[agent.Name()] = agent
	r.logger.Debug("agent registered", "name", agent.Name())
}

// Get gets an agent by name.
func (r *AgentRegistry) Get(name string) (*Instance, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agent, ok := r.agents[name]
	return agent, ok
}

// GetDefault returns the default agent.
func (r *AgentRegistry) GetDefault() *Instance {
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
