package storage

import (
	"time"

	"gorm.io/gorm"
)

type MemoryType string

const (
	MemoryTypeMemory  MemoryType = "memory"  // 长期记忆
	MemoryTypeHistory MemoryType = "history" // 会话历史
	MemoryTypeSession MemoryType = "session" // 会话记忆
	MemoryTypeUser    MemoryType = "user"    // 用户记忆
)

func (memType MemoryType) String() string {
	return string(memType)
}

// Memory 长期记忆模型
type Memory struct {
	Model                  // 嵌入 Model 结构体
	Type       MemoryType  `gorm:"size:20;index" json:"type"`             // memory, history, session, user
	Key        string      `gorm:"size:255;index" json:"key"`             // 记忆键
	Content    string      `gorm:"type:text" json:"content"`              // 记忆内容
	SessionID  *uint       `gorm:"index" json:"session_id"`               // 关联会话ID
	UserID     string      `gorm:"size:100;index" json:"user_id"`         // 用户ID
	Tags       StringArray `gorm:"size:500" json:"tags"`                  // 标签，逗号分隔
	Importance int         `gorm:"default:0" json:"importance"`           // 重要性级别 0-10
	IsPinned   bool        `gorm:"default:false" json:"is_pinned"`        // 是否置顶
	IsDeleted  bool        `gorm:"default:false;index" json:"is_deleted"` // 软删除标记
	ExpiresAt  *time.Time  `gorm:"index" json:"expires_at"`               // 过期时间
}

// TableName 表名
func (Memory) TableName() string {
	return tableNamePrefix + "memories"
}

// BeforeCreate 创建前回调
func (m *Memory) BeforeCreate(tx *gorm.DB) error {
	m.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate 更新前回调
func (m *Memory) BeforeUpdate(tx *gorm.DB) error {
	m.UpdatedAt = time.Now()
	return nil
}

// GetTags 获取标签
func (m *Memory) GetTags() []string {
	return m.Tags
}

// MemoryStorage 记忆存储
type MemoryStorage struct {
	db *gorm.DB
}

// NewMemoryStorage 创建记忆存储
func NewMemoryStorage(db *gorm.DB) *MemoryStorage {
	return &MemoryStorage{db: db}
}

// Create 创建记忆
func (s *MemoryStorage) Create(memory *Memory) error {
	return s.db.Create(memory).Error
}

// Update 更新记忆
func (s *MemoryStorage) Update(memory *Memory) error {
	return s.db.Save(memory).Error
}

// GetByID 通过ID获取记忆
func (s *MemoryStorage) GetByID(id uint) (*Memory, error) {
	var memory Memory
	err := s.db.First(&memory, id).Error
	return &memory, err
}

// GetByKey 通过Key获取记忆
func (s *MemoryStorage) GetByKey(key string) (*Memory, error) {
	var memory Memory
	err := s.db.Where("key = ?", key).First(&memory).Error
	return &memory, err
}

// Delete 删除记忆
func (s *MemoryStorage) Delete(id uint) error {
	return s.db.Delete(&Memory{}, id).Error
}

// GetAll 获取所有记忆
func (s *MemoryStorage) GetAll() ([]Memory, error) {
	var memories []Memory
	err := s.db.Order("updated_at DESC").Find(&memories).Error
	return memories, err
}

// GetByType 通过类型获取记忆
func (s *MemoryStorage) GetByType(memType MemoryType) ([]Memory, error) {
	var memories []Memory
	err := s.db.Where("type = ? AND is_deleted = ?", memType, false).Order("updated_at DESC").Find(&memories).Error
	return memories, err
}

// Search 搜索记忆
func (s *MemoryStorage) Search(query string) ([]Memory, error) {
	var memories []Memory
	err := s.db.
		Where("is_deleted = ? AND (content LIKE ? OR tags LIKE ?)", false, "%"+query+"%", "%"+query+"%").
		Order("importance DESC, updated_at DESC").
		Find(&memories).Error
	return memories, err
}

// GetByUserID 按用户ID获取记忆
func (s *MemoryStorage) GetByUserID(userID string) ([]Memory, error) {
	var memories []Memory
	err := s.db.
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Order("is_pinned DESC, importance DESC, updated_at DESC").
		Find(&memories).Error
	return memories, err
}

// GetBySessionID 按会话ID获取记忆
func (s *MemoryStorage) GetBySessionID(sessionID uint) ([]Memory, error) {
	var memories []Memory
	err := s.db.
		Where("session_id = ? AND is_deleted = ?", sessionID, false).
		Order("is_pinned DESC, importance DESC, updated_at DESC").
		Find(&memories).Error
	return memories, err
}

// GetPinned 获取置顶记忆
func (s *MemoryStorage) GetPinned() ([]Memory, error) {
	var memories []Memory
	err := s.db.
		Where("is_pinned = ? AND is_deleted = ?", true, false).
		Order("updated_at DESC").
		Find(&memories).Error
	return memories, err
}

// SoftDelete 软删除记忆
func (s *MemoryStorage) SoftDelete(id uint) error {
	return s.db.Model(&Memory{}).Where("id = ?", id).Update("is_deleted", true).Error
}

// Restore 恢复记忆
func (s *MemoryStorage) Restore(id uint) error {
	return s.db.Model(&Memory{}).Where("id = ?", id).Update("is_deleted", false).Error
}

// Pin 置顶记忆
func (s *MemoryStorage) Pin(id uint) error {
	return s.db.Model(&Memory{}).Where("id = ?", id).Update("is_pinned", true).Error
}

// Unpin 取消置顶
func (s *MemoryStorage) Unpin(id uint) error {
	return s.db.Model(&Memory{}).Where("id = ?", id).Update("is_pinned", false).Error
}

// Page 分页获取记忆
func (s *MemoryStorage) Page(q *QueryMemory) (*ResQueryMemory, error) {
	var total int64
	query := s.db.Model(&Memory{}).Where("is_deleted = ?", false)
	if q.Type != "" {
		query = query.Where("type = ?", q.Type)
	}
	if q.KeyWord != "" {
		query = query.Where("content LIKE ? OR tags LIKE ?", "%"+q.KeyWord+"%", "%"+q.KeyWord+"%")
	}
	if q.UserID != "" {
		query = query.Where("user_id = ?", q.UserID)
	}
	if q.SessionID != nil {
		query = query.Where("session_id = ?", q.SessionID)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	var memories []Memory
	err := query.Order("is_pinned DESC, importance DESC, updated_at DESC").
		Offset((q.Page.Page - 1) * q.Page.Size).
		Limit(q.Page.Size).
		Find(&memories).Error

	q.Page.Total = int(total)
	return &ResQueryMemory{
		Page:    q.Page,
		Records: memories,
	}, err
}

// QueryMemory 记忆查询参数
type QueryMemory struct {
	Page      Page       `json:"page"`
	Type      MemoryType `json:"type"`
	KeyWord   string     `json:"key_word"`
	UserID    string     `json:"user_id"`
	SessionID *uint      `json:"session_id"`
}

// ResQueryMemory 记忆查询结果
type ResQueryMemory struct {
	Page    Page     `json:"page"`
	Records []Memory `json:"records"`
}
