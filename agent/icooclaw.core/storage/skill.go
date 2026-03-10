package storage

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Skill 技能模型
type Skill struct {
	Model              // 嵌入 Model 结构体
	Name        string `gorm:"size:100;uniqueIndex" json:"name"`
	Source      string `gorm:"size:20" json:"source"` // builtin, workspace, remote
	Description string `gorm:"size:500" json:"description"`
	Content     string `gorm:"type:text" json:"content"` // SKILL.md内容
	Enabled     bool   `gorm:"default:true" json:"enabled"`
	AlwaysLoad  bool   `gorm:"default:false" json:"always_load"` // 是否总是加载
	Metadata    string `gorm:"type:text" json:"metadata"`        // JSON元数据
}

// TableName 表名
func (Skill) TableName() string {
	return tableNamePrefix + "skills"
}

// BeforeCreate 创建前回调
func (c *Skill) BeforeCreate(tx *gorm.DB) error {
	c.ID = uuid.New().String()
	return nil
}

// SkillStorage 技能存储
type SkillStorage struct {
	db *gorm.DB
}

// NewSkillStorage 创建技能存储
func NewSkillStorage(db *gorm.DB) *SkillStorage {
	return &SkillStorage{db: db}
}

// CreateOrUpdate 创建或更新技能
func (s *SkillStorage) CreateOrUpdate(skill *Skill) error {
	return s.db.Save(skill).Error
}

// Create 创建技能
func (s *SkillStorage) Create(skill *Skill) error {
	return s.db.Create(skill).Error
}

// Update 更新技能
func (s *SkillStorage) Update(skill *Skill) error {
	return s.db.Save(skill).Error
}

// GetByID 通过ID获取技能
func (s *SkillStorage) GetByID(id string) (*Skill, error) {
	var skill Skill
	err := s.db.First(&skill, "id = ?", id).Error
	return &skill, err
}

// GetByName 通过名称获取技能
func (s *SkillStorage) GetByName(name string) (*Skill, error) {
	var skill Skill
	err := s.db.Where("name = ?", name).First(&skill).Error
	return &skill, err
}

// Delete 删除技能
func (s *SkillStorage) Delete(id string) error {
	return s.db.Delete(&Skill{}, "id = ?", id).Error
}

// DeleteByIDs 批量删除技能
// 根据ID列表批量删除技能，返回删除的数量和错误
func (s *SkillStorage) DeleteByIDs(ids []string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result := s.db.Delete(&Skill{}, "id IN ?", ids)
	return result.RowsAffected, result.Error
}

// DeleteByNames 根据名称批量删除技能
// 删除指定名称列表的技能，返回删除的数量和错误
func (s *SkillStorage) DeleteByNames(names []string) (int64, error) {
	if len(names) == 0 {
		return 0, nil
	}
	result := s.db.Delete(&Skill{}, "name IN ? AND source != ?", names, "builtin")
	return result.RowsAffected, result.Error
}

// GetAll 获取所有技能
func (s *SkillStorage) GetAll() ([]Skill, error) {
	var skills []Skill
	err := s.db.Find(&skills).Error
	return skills, err
}

// GetEnabled 获取启用的技能
func (s *SkillStorage) GetEnabled() ([]Skill, error) {
	var skills []Skill
	err := s.db.Where("enabled = ?", true).Find(&skills).Error
	return skills, err
}

// GetAlwaysLoad 获取总是加载的技能
func (s *SkillStorage) GetAlwaysLoad() ([]Skill, error) {
	var skills []Skill
	err := s.db.Where("always_load = ? AND enabled = ?", true, true).Find(&skills).Error
	return skills, err
}

// Upsert 创建或更新技能
func (s *SkillStorage) Upsert(skill *Skill) error {
	var existing Skill
	err := s.db.Where("name = ?", skill.Name).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return s.Create(skill)
	}
	skill.ID = existing.ID
	return s.Update(skill)
}

// BatchUpsert 批量创建或更新技能
// 批量处理技能列表，返回成功处理的数量和错误
func (s *SkillStorage) BatchUpsert(skills []*Skill) (int, error) {
	if len(skills) == 0 {
		return 0, nil
	}

	successCount := 0
	var errs []string

	for _, skill := range skills {
		if err := s.Upsert(skill); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", skill.Name, err))
		} else {
			successCount++
		}
	}

	if len(errs) > 0 {
		return successCount, fmt.Errorf("批量操作部分失败: %s", strings.Join(errs, "; "))
	}
	return successCount, nil
}

// BatchUpdateEnabled 批量更新启用状态
// 根据ID列表批量更新技能的启用状态，返回更新的数量和错误
func (s *SkillStorage) BatchUpdateEnabled(ids []string, enabled bool) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result := s.db.Model(&Skill{}).Where("id IN ?", ids).Update("enabled", enabled)
	return result.RowsAffected, result.Error
}

// BatchUpdateAlwaysLoad 批量更新始终加载状态
// 根据ID列表批量更新技能的始终加载状态，返回更新的数量和错误
func (s *SkillStorage) BatchUpdateAlwaysLoad(ids []string, alwaysLoad bool) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result := s.db.Model(&Skill{}).Where("id IN ?", ids).Update("always_load", alwaysLoad)
	return result.RowsAffected, result.Error
}

// Page 分页获取技能
func (s *SkillStorage) Page(q *QuerySkill) (*ResQuerySkill, error) {
	var total int64
	query := s.db.Model(&Skill{})
	
	// 应用过滤条件
	if q.KeyWord != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+q.KeyWord+"%", "%"+q.KeyWord+"%")
	}
	if q.Enabled != nil {
		query = query.Where("enabled = ?", *q.Enabled)
	}
	if q.Source != "" {
		query = query.Where("source = ?", q.Source)
	}
	// 标签过滤（从metadata中查询）
	if q.Tag != "" {
		query = query.Where("metadata LIKE ?", "%"+q.Tag+"%")
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	var skills []Skill
	err := query.Order("created_at DESC").
		Offset((q.Page.Page - 1) * q.Page.Size).
		Limit(q.Page.Size).
		Find(&skills).Error

	q.Page.Total = int(total)
	return &ResQuerySkill{
		Page:    q.Page,
		Records: skills,
	}, err
}

// GetByTag 根据标签获取技能
// 从metadata中解析tags字段进行匹配
func (s *SkillStorage) GetByTag(tag string) ([]Skill, error) {
	var skills []Skill
	err := s.db.Where("metadata LIKE ?", "%\"tags\":%"+tag+"%").Find(&skills).Error
	return skills, err
}

// ExportSkills 导出所有技能
// 将技能数据导出为JSON格式，用于备份或迁移
func (s *SkillStorage) ExportSkills() ([]byte, error) {
	skills, err := s.GetAll()
	if err != nil {
		return nil, fmt.Errorf("获取技能列表失败: %w", err)
	}

	exportData := struct {
		Version   string    `json:"version"`
		ExportAt  time.Time `json:"export_at"`
		Count     int       `json:"count"`
		Skills    []Skill   `json:"skills"`
	}{
		Version:  "1.0",
		ExportAt: time.Now(),
		Count:    len(skills),
		Skills:   skills,
	}

	return json.MarshalIndent(exportData, "", "  ")
}

// ImportSkills 导入技能
// 从JSON数据导入技能，支持覆盖或跳过已存在的技能
func (s *SkillStorage) ImportSkills(data []byte, overwrite bool) (int, int, error) {
	var importData struct {
		Version  string  `json:"version"`
		Skills   []Skill `json:"skills"`
	}

	if err := json.Unmarshal(data, &importData); err != nil {
		return 0, 0, fmt.Errorf("解析导入数据失败: %w", err)
	}

	successCount := 0
	skipCount := 0
	var errs []string

	for _, skill := range importData.Skills {
		// 跳过内置技能
		if skill.Source == "builtin" {
			skipCount++
			continue
		}

		// 检查是否已存在
		existing, err := s.GetByName(skill.Name)
		if err == nil && existing != nil {
			if !overwrite {
				skipCount++
				continue
			}
			skill.ID = existing.ID
		}

		// 重置时间戳
		skill.CreatedAt = time.Now()
		skill.UpdatedAt = time.Now()

		if err := s.Upsert(&skill); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", skill.Name, err))
		} else {
			successCount++
		}
	}

	if len(errs) > 0 {
		return successCount, skipCount, fmt.Errorf("部分导入失败: %s", strings.Join(errs, "; "))
	}
	return successCount, skipCount, nil
}

// QuerySkill 技能查询参数
type QuerySkill struct {
	Page    Page    `json:"page"`
	KeyWord string  `json:"key_word"`
	Enabled *bool   `json:"enabled"`
	Source  string  `json:"source"`
	Tag     string  `json:"tag"`
}

// ResQuerySkill 技能查询结果
type ResQuerySkill struct {
	Page    Page    `json:"page"`
	Records []Skill `json:"records"`
}
