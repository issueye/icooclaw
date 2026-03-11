package storage

import (
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	icooclawErrors "icooclaw/pkg/errors"
)

// Binding represents an agent binding.
type Binding struct {
	Model
	Channel    string `gorm:"column:channel;type:varchar(50);not null;uniqueIndex:idx_binding;comment:渠道" json:"channel"`
	SessionID  string `gorm:"column:session_id;type:varchar(100);not null;uniqueIndex:idx_binding;comment:会话ID" json:"session_id"`
	AgentName  string `gorm:"column:agent_name;type:varchar(100);not null;comment:代理名称" json:"agent_name"`
	Enabled    bool   `gorm:"column:enabled;type:tinyint(1);default:true;comment:是否启用" json:"enabled"`
}

// TableName returns the table name for Binding.
func (Binding) TableName() string {
	return tableNamePrefix + "bindings"
}

type BindingStorage struct {
	db *gorm.DB
}

func NewBindingStorage(db *gorm.DB) *BindingStorage {
	return &BindingStorage{db: db}
}

// SaveBinding saves an agent binding.
func (s *BindingStorage) SaveBinding(b *Binding) error {
	result := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "channel"}, {Name: "session_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"agent_name", "enabled"}),
	}).Create(b)
	return result.Error
}

// GetBinding gets a binding by channel and session ID.
func (s *BindingStorage) GetBinding(channel, sessionID string) (*Binding, error) {
	var b Binding
	result := s.db.Where("channel = ? AND session_id = ?", channel, sessionID).First(&b)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, icooclawErrors.ErrRecordNotFound
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get binding: %w", result.Error)
	}
	return &b, nil
}

// ListBindings lists all bindings.
func (s *BindingStorage) ListBindings() ([]*Binding, error) {
	var bindings []*Binding
	result := s.db.Order("channel, session_id").Find(&bindings)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list bindings: %w", result.Error)
	}
	return bindings, nil
}

// DeleteBinding deletes a binding.
func (s *BindingStorage) DeleteBinding(channel, sessionID string) error {
	result := s.db.Where("channel = ? AND session_id = ?", channel, sessionID).Delete(&Binding{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete binding: %w", result.Error)
	}
	return nil
}

type QueryBinding struct {
	Page       Page   `json:"page"`
	KeyWord    string `json:"key_word"`
	Channel    string `json:"channel"`
	AgentName  string `json:"agent_name"`
	Enabled    *bool  `json:"enabled"`
}

type ResQueryBinding struct {
	Page    Page     `json:"page"`
	Records []Binding `json:"records"`
}

// Page gets bindings with pagination.
func (s *BindingStorage) Page(query *QueryBinding) (*ResQueryBinding, error) {
	var res ResQueryBinding

	qry := s.db.Model(&Binding{})

	if query.KeyWord != "" {
		qry = qry.Where("agent_name LIKE ? OR session_id LIKE ?", "%"+query.KeyWord+"%", "%"+query.KeyWord+"%")
	}

	if query.Channel != "" {
		qry = qry.Where("channel = ?", query.Channel)
	}

	if query.AgentName != "" {
		qry = qry.Where("agent_name = ?", query.AgentName)
	}

	if query.Enabled != nil {
		qry = qry.Where("enabled = ?", *query.Enabled)
	}

	qry = qry.Order("channel, session_id")

	result := qry.Count(&res.Page.Total)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to count bindings: %w", result.Error)
	}

	if query.Page.Page == 0 || query.Page.Size == 0 {
		result = qry.Find(&res.Records)
	} else {
		result = qry.Limit(query.Page.Size).
			Offset((query.Page.Page - 1) * query.Page.Size).
			Find(&res.Records)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get bindings: %w", result.Error)
	}

	return &res, nil
}
