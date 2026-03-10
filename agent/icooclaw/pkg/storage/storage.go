// Package storage provides data storage for icooclaw using GORM.
package storage

import (
	"fmt"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Storage provides SQLite-based storage using GORM.
type Storage struct {
	db       *gorm.DB
	path     string
	skill    *SkillStorage
	binding  *BindingStorage
	session  *SessionStorage
	message  *MessageStorage
	memory   *MemoryStorage
	tool     *ToolStorage
	provider *ProviderStorage
	mcp      *MCPStorage
	channel  *ChannelStorage
	param    *ParamStorage
	task     *TaskStorage
}

func (s *Storage) Skill() *SkillStorage {
	return s.skill
}

func (s *Storage) Binding() *BindingStorage {
	return s.binding
}

func (s *Storage) Session() *SessionStorage {
	return s.session
}

func (s *Storage) Memory() *MemoryStorage {
	return s.memory
}

func (s *Storage) Tool() *ToolStorage {
	return s.tool
}

func (s *Storage) Provider() *ProviderStorage {
	return s.provider
}

func (s *Storage) MCP() *MCPStorage {
	return s.mcp
}

func (s *Storage) Channel() *ChannelStorage {
	return s.channel
}

func (s *Storage) Message() *MessageStorage {
	return s.message
}

func (s *Storage) Param() *ParamStorage {
	return s.param
}

func (s *Storage) Task() *TaskStorage {
	return s.task
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
		db:       db,
		path:     path,
		skill:    NewSkillStorage(db),
		binding:  NewBindingStorage(db),
		session:  NewSessionStorage(db),
		message:  NewMessageStorage(db),
		memory:   NewMemoryStorage(db),
		tool:     NewToolStorage(db),
		provider: NewProviderStorage(db),
		mcp:      NewMCPStorage(db),
		channel:  NewChannelStorage(db),
		param:    NewParamStorage(db),
		task:     NewTaskStorage(db),
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
		&Message{},
		&Tool{},
		&Skill{},
		&MCPConfig{},
		&ParamConfig{},
		&Task{},
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
