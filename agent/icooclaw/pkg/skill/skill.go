// Package skill provides skill management for icooclaw.
package skill

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"
)

// Skill represents a skill that can be activated.
type Skill struct {
	Name        string
	Description string
	Prompt      string
	Tools       []string
	Config      map[string]any
}

// Loader loads skills from storage.
type Loader interface {
	Load(ctx context.Context, name string) (*Skill, error)
	List(ctx context.Context) ([]*Skill, error)
}

// DefaultLoader is the default skill loader.
type DefaultLoader struct {
	storage *storage.Storage
	logger  *slog.Logger
	mu      sync.RWMutex
	cache   map[string]*Skill
}

// NewLoader creates a new skill loader.
func NewLoader(s *storage.Storage, logger *slog.Logger) *DefaultLoader {
	if logger == nil {
		logger = slog.Default()
	}
	return &DefaultLoader{
		storage: s,
		logger:  logger,
		cache:   make(map[string]*Skill),
	}
}

// Load loads a skill by name.
func (l *DefaultLoader) Load(ctx context.Context, name string) (*Skill, error) {
	// Check cache first
	l.mu.RLock()
	if sk, ok := l.cache[name]; ok {
		l.mu.RUnlock()
		return sk, nil
	}
	l.mu.RUnlock()

	// Load from storage
	sk, err := l.storage.GetSkill(name)
	if err != nil {
		return nil, fmt.Errorf("skill %s not found: %w", name, err)
	}

	// Parse skill
	skill := &Skill{
		Name:        sk.Name,
		Description: sk.Description,
		Prompt:      sk.Prompt,
		Config:      make(map[string]any),
	}

	// Parse tools
	if sk.Tools != "" {
		json.Unmarshal([]byte(sk.Tools), &skill.Tools)
	}

	// Parse config
	if sk.Config != "" {
		json.Unmarshal([]byte(sk.Config), &skill.Config)
	}

	// Cache it
	l.mu.Lock()
	l.cache[name] = skill
	l.mu.Unlock()

	return skill, nil
}

// List lists all skills.
func (l *DefaultLoader) List(ctx context.Context) ([]*Skill, error) {
	skills, err := l.storage.ListEnabledSkills()
	if err != nil {
		return nil, err
	}

	result := make([]*Skill, 0, len(skills))
	for _, sk := range skills {
		skill := &Skill{
			Name:        sk.Name,
			Description: sk.Description,
			Prompt:      sk.Prompt,
			Config:      make(map[string]any),
		}

		if sk.Tools != "" {
			json.Unmarshal([]byte(sk.Tools), &skill.Tools)
		}

		if sk.Config != "" {
			json.Unmarshal([]byte(sk.Config), &skill.Config)
		}

		result = append(result, skill)
	}

	return result, nil
}

// Refresh refreshes the skill cache.
func (l *DefaultLoader) Refresh() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.cache = make(map[string]*Skill)
	return nil
}

// Executor executes skills.
type Executor struct {
	loader Loader
	tools  *tools.Registry
	logger *slog.Logger
}

// NewExecutor creates a new skill executor.
func NewExecutor(loader Loader, registry *tools.Registry, logger *slog.Logger) *Executor {
	if logger == nil {
		logger = slog.Default()
	}
	return &Executor{
		loader: loader,
		tools:  registry,
		logger: logger,
	}
}

// GetPrompt returns the skill prompt.
func (e *Executor) GetPrompt(ctx context.Context, name string) (string, error) {
	skill, err := e.loader.Load(ctx, name)
	if err != nil {
		return "", err
	}
	return skill.Prompt, nil
}

// GetTools returns the tools for a skill.
func (e *Executor) GetTools(ctx context.Context, name string) ([]tools.Tool, error) {
	skill, err := e.loader.Load(ctx, name)
	if err != nil {
		return nil, err
	}

	result := make([]tools.Tool, 0, len(skill.Tools))
	for _, toolName := range skill.Tools {
		tool, err := e.tools.Get(toolName)
		if err != nil {
			e.logger.Warn("tool not found for skill", "skill", name, "tool", toolName)
			continue
		}
		result = append(result, tool)
	}

	return result, nil
}

// Manager manages skill registration and execution.
type Manager struct {
	loader   Loader
	executor *Executor
	storage  *storage.Storage
	logger   *slog.Logger
	mu       sync.RWMutex
}

// NewManager creates a new skill manager.
func NewManager(s *storage.Storage, registry *tools.Registry, logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	loader := NewLoader(s, logger)
	return &Manager{
		loader:   loader,
		executor: NewExecutor(loader, registry, logger),
		storage:  s,
		logger:   logger,
	}
}

// GetSkill gets a skill by name.
func (m *Manager) GetSkill(ctx context.Context, name string) (*Skill, error) {
	return m.loader.Load(ctx, name)
}

// ListSkills lists all skills.
func (m *Manager) ListSkills(ctx context.Context) ([]*Skill, error) {
	return m.loader.List(ctx)
}

// CreateSkill creates a new skill.
func (m *Manager) CreateSkill(skill *Skill) error {
	toolsJSON, _ := json.Marshal(skill.Tools)
	configJSON, _ := json.Marshal(skill.Config)

	return m.storage.SaveSkill(&storage.Skill{
		Name:        skill.Name,
		Description: skill.Description,
		Prompt:      skill.Prompt,
		Tools:       string(toolsJSON),
		Config:      string(configJSON),
		Enabled:     true,
	})
}

// UpdateSkill updates a skill.
func (m *Manager) UpdateSkill(skill *Skill) error {
	return m.CreateSkill(skill)
}

// DeleteSkill deletes a skill.
func (m *Manager) DeleteSkill(name string) error {
	return m.storage.DeleteSkill(name)
}

// EnableSkill enables a skill.
func (m *Manager) EnableSkill(name string) error {
	skill, err := m.storage.GetSkill(name)
	if err != nil {
		return err
	}
	skill.Enabled = true
	return m.storage.SaveSkill(skill)
}

// DisableSkill disables a skill.
func (m *Manager) DisableSkill(name string) error {
	skill, err := m.storage.GetSkill(name)
	if err != nil {
		return err
	}
	skill.Enabled = false
	return m.storage.SaveSkill(skill)
}

// GetPrompt gets the prompt for a skill.
func (m *Manager) GetPrompt(ctx context.Context, name string) (string, error) {
	return m.executor.GetPrompt(ctx, name)
}

// GetTools gets the tools for a skill.
func (m *Manager) GetTools(ctx context.Context, name string) ([]tools.Tool, error) {
	return m.executor.GetTools(ctx, name)
}
