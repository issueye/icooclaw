// Package skill provides skill management for icooclaw.
package skill

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"icooclaw/pkg/storage"
)

// Skill represents a skill that can be activated.
type Skill struct {
	Name        string
	Description string
	Path        string // 技能路径 默认 workspace/.skills/<name>-<version>/
}

// cacheEntry represents a cached skill with expiration time.
type cacheEntry struct {
	skill     *Skill
	expiresAt time.Time
}

// Default cache TTL (5 minutes)
const defaultCacheTTL = 5 * time.Minute

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
	cache   map[string]*cacheEntry
	cacheTTL time.Duration
}

// NewLoader creates a new skill loader.
func NewLoader(s *storage.Storage, logger *slog.Logger) *DefaultLoader {
	if logger == nil {
		logger = slog.Default()
	}
	return &DefaultLoader{
		storage:  s,
		logger:   logger,
		cache:    make(map[string]*cacheEntry),
		cacheTTL: defaultCacheTTL,
	}
}

// NewLoaderWithTTL creates a new skill loader with custom cache TTL.
func NewLoaderWithTTL(s *storage.Storage, logger *slog.Logger, ttl time.Duration) *DefaultLoader {
	l := NewLoader(s, logger)
	l.cacheTTL = ttl
	return l
}

// Load loads a skill by name.
func (l *DefaultLoader) Load(ctx context.Context, name string) (*Skill, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Check cache first
	l.mu.RLock()
	if entry, ok := l.cache[name]; ok {
		if time.Now().Before(entry.expiresAt) {
			l.mu.RUnlock()
			return entry.skill, nil
		}
		// Cache entry expired, will be refreshed below
	}
	l.mu.RUnlock()

	// Check context again before storage operation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Load from storage
	sk, err := l.storage.Skill().GetSkill(name)
	if err != nil {
		return nil, fmt.Errorf("skill %s not found: %w", name, err)
	}

	// Parse skill
	skill := &Skill{
		Name:        sk.Name,
		Description: sk.Description,
		Path:        sk.Path,
	}

	// Cache it with expiration
	l.mu.Lock()
	l.cache[name] = &cacheEntry{
		skill:     skill,
		expiresAt: time.Now().Add(l.cacheTTL),
	}
	l.mu.Unlock()

	return skill, nil
}

// List lists all skills.
func (l *DefaultLoader) List(ctx context.Context) ([]*Skill, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	skills, err := l.storage.Skill().ListEnabledSkills()
	if err != nil {
		return nil, err
	}

	result := make([]*Skill, 0, len(skills))
	for _, sk := range skills {
		skill := &Skill{
			Name:        sk.Name,
			Description: sk.Description,
			Path:        sk.Path,
		}

		result = append(result, skill)
	}

	return result, nil
}

// Refresh refreshes the skill cache.
func (l *DefaultLoader) Refresh() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.cache = make(map[string]*cacheEntry)
	return nil
}

// SetCacheTTL updates the cache TTL.
func (l *DefaultLoader) SetCacheTTL(ttl time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cacheTTL = ttl
}

// Manager manages skill registration and execution.
type Manager struct {
	loader  Loader
	storage *storage.Storage
	logger  *slog.Logger
	mu      sync.RWMutex
}

// NewManager creates a new skill manager.
func NewManager(s *storage.Storage, logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	return &Manager{
		loader:  NewLoader(s, logger),
		storage: s,
		logger:  logger,
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
	return m.storage.Skill().SaveSkill(&storage.Skill{
		Name:        skill.Name,
		Description: skill.Description,
		Path:        skill.Path,
		Enabled:     true,
	})
}

// UpdateSkill updates a skill.
func (m *Manager) UpdateSkill(skill *Skill) error {
	return m.CreateSkill(skill)
}

// DeleteSkill deletes a skill.
func (m *Manager) DeleteSkill(name string) error {
	return m.storage.Skill().DeleteSkill(name)
}

// EnableSkill enables a skill.
func (m *Manager) EnableSkill(name string) error {
	skill, err := m.storage.Skill().GetSkill(name)
	if err != nil {
		return err
	}
	skill.Enabled = true
	return m.storage.Skill().SaveSkill(skill)
}

// DisableSkill disables a skill.
func (m *Manager) DisableSkill(name string) error {
	skill, err := m.storage.Skill().GetSkill(name)
	if err != nil {
		return err
	}
	skill.Enabled = false
	return m.storage.Skill().SaveSkill(skill)
}
