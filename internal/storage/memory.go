package storage

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// Memory 长期记忆模型
type Memory struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Type        string    `gorm:"size:20;index" json:"type"`              // memory, history, session, user
	Key         string    `gorm:"size:255;index" json:"key"`              // 记忆键
	Content     string    `gorm:"type:text" json:"content"`               // 记忆内容
	SessionID   *uint     `gorm:"index" json:"session_id"`                // 关联会话ID
	UserID      string    `gorm:"size:100;index" json:"user_id"`          // 用户ID
	Tags        string    `gorm:"size:500" json:"tags"`                  // 标签，逗号分隔
	Importance  int       `gorm:"default:0" json:"importance"`            // 重要性级别 0-10
	IsPinned    bool      `gorm:"default:false" json:"is_pinned"`         // 是否置顶
	IsDeleted   bool      `gorm:"default:false;index" json:"is_deleted"`  // 软删除标记
	ExpiresAt   *time.Time `gorm:"index" json:"expires_at"`                // 过期时间
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 表名
func (Memory) TableName() string {
	return "memories"
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

// Create 创建记忆
func (m *Memory) Create() error {
	return DB.Create(m).Error
}

// Update 更新记忆
func (m *Memory) Update() error {
	return DB.Save(m).Error
}

// Delete 删除记忆
func (m *Memory) Delete() error {
	return DB.Delete(m).Error
}

// GetByID 通过ID获取记忆
func GetMemoryByID(id uint) (*Memory, error) {
	var memory Memory
	err := DB.First(&memory, id).Error
	return &memory, err
}

// GetByKey 通过Key获取记忆
func GetMemoryByKey(key string) (*Memory, error) {
	var memory Memory
	err := DB.Where("key = ?", key).First(&memory).Error
	return &memory, err
}

// GetByType 通过类型获取记忆
func GetMemoriesByType(memType string) ([]Memory, error) {
	var memories []Memory
	err := DB.Where("type = ?", memType).Order("updated_at DESC").Find(&memories).Error
	return memories, err
}

// GetAll 获取所有记忆
func GetAllMemories() ([]Memory, error) {
	var memories []Memory
	err := DB.Order("updated_at DESC").Find(&memories).Error
	return memories, err
}

// Upsert 创建或更新记忆
func (m *Memory) Upsert() error {
	var existing Memory
	err := DB.Where("key = ? AND type = ?", m.Key, m.Type).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return m.Create()
	}
	m.ID = existing.ID
	return m.Update()
}

// SearchMemories 搜索记忆
func SearchMemories(query string) ([]Memory, error) {
	var memories []Memory
	err := DB.Where("is_deleted = ? AND (content LIKE ? OR tags LIKE ?)", false, "%"+query+"%", "%"+query+"%").Order("importance DESC, updated_at DESC").Find(&memories).Error
	return memories, err
}

// GetByTypeAndSession 按类型和会话获取记忆
func GetMemoriesByTypeAndSession(memType string, sessionID uint) ([]Memory, error) {
	var memories []Memory
	err := DB.Where("type = ? AND session_id = ? AND is_deleted = ?", memType, sessionID, false).Order("is_pinned DESC, importance DESC, updated_at DESC").Find(&memories).Error
	return memories, err
}

// GetByUserID 按用户ID获取记忆
func GetMemoriesByUserID(userID string) ([]Memory, error) {
	var memories []Memory
	err := DB.Where("user_id = ? AND is_deleted = ?", userID, false).Order("is_pinned DESC, importance DESC, updated_at DESC").Find(&memories).Error
	return memories, err
}

// GetBySessionID 按会话ID获取记忆
func GetMemoriesBySessionID(sessionID uint) ([]Memory, error) {
	var memories []Memory
	err := DB.Where("session_id = ? AND is_deleted = ?", sessionID, false).Order("is_pinned DESC, importance DESC, updated_at DESC").Find(&memories).Error
	return memories, err
}

// GetPinnedMemories 获取置顶记忆
func GetPinnedMemories() ([]Memory, error) {
	var memories []Memory
	err := DB.Where("is_pinned = ? AND is_deleted = ?", true, false).Order("updated_at DESC").Find(&memories).Error
	return memories, err
}

// GetExpiredMemories 获取过期记忆
func GetExpiredMemories() ([]Memory, error) {
	var memories []Memory
	err := DB.Where("expires_at IS NOT NULL AND expires_at < ? AND is_deleted = ?", time.Now(), false).Find(&memories).Error
	return memories, err
}

// SoftDelete 软删除记忆
func (m *Memory) SoftDelete() error {
	m.IsDeleted = true
	return m.Update()
}

// Restore 恢复记忆
func (m *Memory) Restore() error {
	m.IsDeleted = false
	return m.Update()
}

// Pin 置顶记忆
func (m *Memory) Pin() error {
	m.IsPinned = true
	return m.Update()
}

// Unpin 取消置顶
func (m *Memory) Unpin() error {
	m.IsPinned = false
	return m.Update()
}

// SetTags 设置标签
func (m *Memory) SetTags(tags []string) error {
	m.Tags = "," + strings.Join(tags, ",") + ","
	return m.Update()
}

// GetTags 获取标签列表
func (m *Memory) GetTags() []string {
	if m.Tags == "" {
		return nil
	}
	// 去掉首尾逗号后分割
	tagsStr := strings.Trim(m.Tags, ",")
	if tagsStr == "" {
		return nil
	}
	return strings.Split(tagsStr, ",")
}

// BatchCreate 批量创建记忆
func BatchCreateMemories(memories []*Memory) error {
	if len(memories) == 0 {
		return nil
	}
	return DB.Create(memories).Error
}

// BatchDelete 批量删除记忆
func BatchDeleteMemories(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return DB.Model(&Memory{}).Where("id IN ?", ids).Update("is_deleted", true).Error
}

// CountByType 按类型统计记忆数量
func CountMemoriesByType(memType string) (int64, error) {
	var count int64
	err := DB.Model(&Memory{}).Where("type = ? AND is_deleted = ?", memType, false).Count(&count).Error
	return count, err
}

// GetMemoriesByTags 按标签获取记忆
func GetMemoriesByTags(tag string) ([]Memory, error) {
	var memories []Memory
	err := DB.Where("tags LIKE ? AND is_deleted = ?", "%,"+tag+",%", false).Order("importance DESC, updated_at DESC").Find(&memories).Error
	return memories, err
}

// ClearSessionMemories 清除会话记忆
func ClearSessionMemories(sessionID uint) error {
	return DB.Where("session_id = ? AND type = ?", sessionID, "session").Delete(&Memory{}).Error
}

// ClearUserMemories 清除用户记忆
func ClearUserMemories(userID string) error {
	return DB.Where("user_id = ? AND type = ?", userID, "user").Delete(&Memory{}).Error
}

// ChannelConfig 通道配置模型
type ChannelConfig struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:50;uniqueIndex" json:"name"` // telegram, discord...
	Enabled   bool      `gorm:"default:false" json:"enabled"`
	Config    string    `gorm:"type:text" json:"config"` // JSON配置
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 表名
func (ChannelConfig) TableName() string {
	return "channel_configs"
}

// Create 创建通道配置
func (c *ChannelConfig) Create() error {
	return DB.Create(c).Error
}

// Update 更新通道配置
func (c *ChannelConfig) Update() error {
	return DB.Save(c).Error
}

// GetByName 通过名称获取通道配置
func GetChannelConfigByName(name string) (*ChannelConfig, error) {
	var config ChannelConfig
	err := DB.Where("name = ?", name).First(&config).Error
	return &config, err
}

// ProviderConfig Provider配置模型
type ProviderConfig struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:50;uniqueIndex" json:"name"` // openai, anthropic...
	Enabled   bool      `gorm:"default:false" json:"enabled"`
	Config    string    `gorm:"type:text" json:"config"` // JSON配置
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 表名
func (ProviderConfig) TableName() string {
	return "provider_configs"
}

// Create 创建Provider配置
func (p *ProviderConfig) Create() error {
	return DB.Create(p).Error
}

// Update 更新Provider配置
func (p *ProviderConfig) Update() error {
	return DB.Save(p).Error
}

// GetByName 通过名称获取Provider配置
func GetProviderConfigByName(name string) (*ProviderConfig, error) {
	var config ProviderConfig
	err := DB.Where("name = ?", name).First(&config).Error
	return &config, err
}
