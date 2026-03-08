// Package storage provides data storage for icooclaw using GORM.
package storage

import (
	"fmt"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	icooclawErrors "icooclaw/pkg/errors"
)

// Storage provides SQLite-based storage using GORM.
type Storage struct {
	db   *gorm.DB
	path string
}

// New creates a new Storage instance.
func New(path string) (*Storage, error) {
	db, err := gorm.Open(sqlite.Open(path+"?_journal_mode=WAL&_busy_timeout=5000"), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Get underlying sql.DB for connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying db: %w", err)
	}
	sqlDB.SetMaxOpenConns(1) // SQLite recommends single connection
	sqlDB.SetMaxIdleConns(1)

	s := &Storage{
		db:   db,
		path: path,
	}

	if err := s.autoMigrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate: %w", err)
	}

	return s, nil
}

// autoMigrate runs auto migration for all models.
func (s *Storage) autoMigrate() error {
	return s.db.AutoMigrate(
		&Provider{},
		&Channel{},
		&Session{},
		&Binding{},
		&Memory{},
		&Tool{},
		&Skill{},
	)
}

// Close closes the database connection.
func (s *Storage) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// DB returns the underlying GORM database.
func (s *Storage) DB() *gorm.DB {
	return s.db
}

// Provider represents a provider configuration.
type Provider struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Name         string         `gorm:"uniqueIndex;not null" json:"name"`
	Type         string         `gorm:"not null" json:"type"`
	APIKey       string         `json:"api_key"`
	APIBase      string         `json:"api_base"`
	DefaultModel string         `json:"default_model"`
	Models       []string       `json:"models"`   // JSON array
	Config       string         `json:"config"`   // JSON object
	Metadata     map[string]any `json:"metadata"` // JSON object
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// TableName returns the table name for Provider.
func (Provider) TableName() string {
	return "providers"
}

// SaveProvider saves a provider configuration.
func (s *Storage) SaveProvider(p *Provider) error {
	result := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{"type", "api_key", "api_base", "default_model", "models", "config", "updated_at"}),
	}).Create(p)
	return result.Error
}

// GetProvider gets a provider by name.
func (s *Storage) GetProvider(name string) (*Provider, error) {
	var p Provider
	result := s.db.Where("name = ?", name).First(&p)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, icooclawErrors.ErrRecordNotFound
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get provider: %w", result.Error)
	}
	return &p, nil
}

// ListProviders lists all providers.
func (s *Storage) ListProviders() ([]*Provider, error) {
	var providers []*Provider
	result := s.db.Order("name").Find(&providers)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list providers: %w", result.Error)
	}
	return providers, nil
}

// DeleteProvider deletes a provider by name.
func (s *Storage) DeleteProvider(name string) error {
	result := s.db.Where("name = ?", name).Delete(&Provider{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete provider: %w", result.Error)
	}
	return nil
}

// Channel represents a channel configuration.
type Channel struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"uniqueIndex;not null" json:"name"`
	Type        string    `gorm:"not null" json:"type"`
	Enabled     bool      `gorm:"default:true" json:"enabled"`
	Config      string    `json:"config"`      // JSON object
	Permissions string    `json:"permissions"` // JSON array
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName returns the table name for Channel.
func (Channel) TableName() string {
	return "channels"
}

// SaveChannel saves a channel configuration.
func (s *Storage) SaveChannel(c *Channel) error {
	result := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{"type", "enabled", "config", "permissions", "updated_at"}),
	}).Create(c)
	return result.Error
}

// GetChannel gets a channel by name.
func (s *Storage) GetChannel(name string) (*Channel, error) {
	var c Channel
	result := s.db.Where("name = ?", name).First(&c)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, icooclawErrors.ErrRecordNotFound
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get channel: %w", result.Error)
	}
	return &c, nil
}

// ListChannels lists all channels.
func (s *Storage) ListChannels() ([]*Channel, error) {
	var channels []*Channel
	result := s.db.Order("name").Find(&channels)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list channels: %w", result.Error)
	}
	return channels, nil
}

// ListEnabledChannels lists all enabled channels.
func (s *Storage) ListEnabledChannels() ([]*Channel, error) {
	var channels []*Channel
	result := s.db.Where("enabled = ?", true).Order("name").Find(&channels)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list enabled channels: %w", result.Error)
	}
	return channels, nil
}

// DeleteChannel deletes a channel by name.
func (s *Storage) DeleteChannel(name string) error {
	result := s.db.Where("name = ?", name).Delete(&Channel{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete channel: %w", result.Error)
	}
	return nil
}

// Binding represents an agent binding.
type Binding struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Channel   string    `gorm:"not null;uniqueIndex:idx_binding" json:"channel"`
	ChatID    string    `gorm:"not null;uniqueIndex:idx_binding" json:"chat_id"`
	AgentName string    `gorm:"not null" json:"agent_name"`
	Enabled   bool      `gorm:"default:true" json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName returns the table name for Binding.
func (Binding) TableName() string {
	return "bindings"
}

// SaveBinding saves an agent binding.
func (s *Storage) SaveBinding(b *Binding) error {
	result := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "channel"}, {Name: "chat_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"agent_name", "enabled"}),
	}).Create(b)
	return result.Error
}

// GetBinding gets a binding by channel and chat ID.
func (s *Storage) GetBinding(channel, chatID string) (*Binding, error) {
	var b Binding
	result := s.db.Where("channel = ? AND chat_id = ?", channel, chatID).First(&b)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, icooclawErrors.ErrRecordNotFound
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get binding: %w", result.Error)
	}
	return &b, nil
}

// ListBindings lists all bindings.
func (s *Storage) ListBindings() ([]*Binding, error) {
	var bindings []*Binding
	result := s.db.Order("channel, chat_id").Find(&bindings)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list bindings: %w", result.Error)
	}
	return bindings, nil
}

// DeleteBinding deletes a binding.
func (s *Storage) DeleteBinding(channel, chatID string) error {
	result := s.db.Where("channel = ? AND chat_id = ?", channel, chatID).Delete(&Binding{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete binding: %w", result.Error)
	}
	return nil
}

// Session represents a chat session.
type Session struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	SessionID  string    `gorm:"uniqueIndex;not null" json:"session_id"`
	Channel    string    `gorm:"not null;index:idx_session_channel_chat" json:"channel"`
	ChatID     string    `gorm:"not null;index:idx_session_channel_chat" json:"chat_id"`
	AgentName  string    `json:"agent_name"`
	Context    string    `json:"context"`
	Summary    string    `json:"summary"`
	LastActive time.Time `json:"last_active"`
	CreatedAt  time.Time `json:"created_at"`
}

// TableName returns the table name for Session.
func (Session) TableName() string {
	return "sessions"
}

// SaveSession saves a session.
func (s *Storage) SaveSession(sess *Session) error {
	sess.LastActive = time.Now()
	result := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "session_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"channel", "chat_id", "agent_name", "context", "summary", "last_active"}),
	}).Create(sess)
	return result.Error
}

// GetSession gets a session by session ID.
func (s *Storage) GetSession(sessionID string) (*Session, error) {
	var sess Session
	result := s.db.Where("session_id = ?", sessionID).First(&sess)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, icooclawErrors.ErrRecordNotFound
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get session: %w", result.Error)
	}
	return &sess, nil
}

// GetSessionByChat gets a session by channel and chat ID.
func (s *Storage) GetSessionByChat(channel, chatID string) (*Session, error) {
	var sess Session
	result := s.db.Where("channel = ? AND chat_id = ?", channel, chatID).First(&sess)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, icooclawErrors.ErrRecordNotFound
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get session: %w", result.Error)
	}
	return &sess, nil
}

// DeleteSession deletes a session.
func (s *Storage) DeleteSession(sessionID string) error {
	result := s.db.Where("session_id = ?", sessionID).Delete(&Session{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete session: %w", result.Error)
	}
	return nil
}

// Memory represents a memory entry.
type Memory struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	SessionID string    `gorm:"not null;index" json:"session_id"`
	Role      string    `gorm:"not null" json:"role"`
	Content   string    `gorm:"not null" json:"content"`
	Metadata  string    `json:"metadata"` // JSON object
	CreatedAt time.Time `json:"created_at"`
}

// TableName returns the table name for Memory.
func (Memory) TableName() string {
	return "memory"
}

// SaveMemory saves a memory entry.
func (s *Storage) SaveMemory(m *Memory) error {
	return s.db.Create(m).Error
}

// GetMemory gets memory entries for a session.
func (s *Storage) GetMemory(sessionID string, limit int) ([]*Memory, error) {
	if limit <= 0 {
		limit = 100
	}
	var memories []*Memory
	result := s.db.Where("session_id = ?", sessionID).
		Order("created_at DESC").
		Limit(limit).
		Find(&memories)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get memory: %w", result.Error)
	}
	return memories, nil
}

// DeleteMemory deletes memory entries for a session.
func (s *Storage) DeleteMemory(sessionID string) error {
	result := s.db.Where("session_id = ?", sessionID).Delete(&Memory{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete memory: %w", result.Error)
	}
	return nil
}

// Tool represents a tool configuration.
type Tool struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Name       string    `gorm:"uniqueIndex;not null" json:"name"`
	Type       string    `gorm:"not null" json:"type"` // builtin, mcp, custom
	Definition string    `json:"definition"`           // JSON tool definition
	Config     string    `json:"config"`               // JSON config
	Enabled    bool      `gorm:"default:true" json:"enabled"`
	CreatedAt  time.Time `json:"created_at"`
}

// TableName returns the table name for Tool.
func (Tool) TableName() string {
	return "tools"
}

// SaveTool saves a tool configuration.
func (s *Storage) SaveTool(t *Tool) error {
	result := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{"type", "definition", "config", "enabled"}),
	}).Create(t)
	return result.Error
}

// GetTool gets a tool by name.
func (s *Storage) GetTool(name string) (*Tool, error) {
	var t Tool
	result := s.db.Where("name = ?", name).First(&t)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, icooclawErrors.ErrRecordNotFound
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get tool: %w", result.Error)
	}
	return &t, nil
}

// ListTools lists all tools.
func (s *Storage) ListTools() ([]*Tool, error) {
	var tools []*Tool
	result := s.db.Order("name").Find(&tools)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list tools: %w", result.Error)
	}
	return tools, nil
}

// ListEnabledTools lists all enabled tools.
func (s *Storage) ListEnabledTools() ([]*Tool, error) {
	var tools []*Tool
	result := s.db.Where("enabled = ?", true).Order("name").Find(&tools)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list enabled tools: %w", result.Error)
	}
	return tools, nil
}

// DeleteTool deletes a tool by name.
func (s *Storage) DeleteTool(name string) error {
	result := s.db.Where("name = ?", name).Delete(&Tool{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete tool: %w", result.Error)
	}
	return nil
}

// Skill represents a skill configuration.
type Skill struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"uniqueIndex;not null" json:"name"`
	Description string    `json:"description"`
	Prompt      string    `json:"prompt"`
	Tools       string    `json:"tools"`  // JSON array of tool names
	Config      string    `json:"config"` // JSON config
	Enabled     bool      `gorm:"default:true" json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
}

// TableName returns the table name for Skill.
func (Skill) TableName() string {
	return "skills"
}

// SaveSkill saves a skill configuration.
func (s *Storage) SaveSkill(sk *Skill) error {
	result := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{"description", "prompt", "tools", "config", "enabled"}),
	}).Create(sk)
	return result.Error
}

// GetSkill gets a skill by name.
func (s *Storage) GetSkill(name string) (*Skill, error) {
	var sk Skill
	result := s.db.Where("name = ?", name).First(&sk)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, icooclawErrors.ErrRecordNotFound
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get skill: %w", result.Error)
	}
	return &sk, nil
}

// ListSkills lists all skills.
func (s *Storage) ListSkills() ([]*Skill, error) {
	var skills []*Skill
	result := s.db.Order("name").Find(&skills)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list skills: %w", result.Error)
	}
	return skills, nil
}

// ListEnabledSkills lists all enabled skills.
func (s *Storage) ListEnabledSkills() ([]*Skill, error) {
	var skills []*Skill
	result := s.db.Where("enabled = ?", true).Order("name").Find(&skills)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list enabled skills: %w", result.Error)
	}
	return skills, nil
}

// DeleteSkill deletes a skill by name.
func (s *Storage) DeleteSkill(name string) error {
	result := s.db.Where("name = ?", name).Delete(&Skill{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete skill: %w", result.Error)
	}
	return nil
}
