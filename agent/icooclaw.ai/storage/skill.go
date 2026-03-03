package storage

import (
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

// Create 创建技能
func (s *Skill) Create() error {
	return DB.Create(s).Error
}

// Update 更新技能
func (s *Skill) Update() error {
	return DB.Save(s).Error
}

// Delete 删除技能
func (s *Skill) Delete() error {
	return DB.Delete(s).Error
}

// GetByID 通过ID获取技能
func GetSkillByID(id uint) (*Skill, error) {
	var skill Skill
	err := DB.
		Table(tableNamePrefix+"skills").
		First(&skill, id).Error
	return &skill, err
}

// GetByName 通过名称获取技能
func GetSkillByName(name string) (*Skill, error) {
	var skill Skill
	err := DB.
		Table(tableNamePrefix+"skills").
		Where("name = ?", name).
		First(&skill).Error
	return &skill, err
}

// GetEnabledSkills 获取所有启用的技能
func GetEnabledSkills() ([]Skill, error) {
	var skills []Skill
	err := DB.
		Table(tableNamePrefix+"skills").
		Where("enabled = ?", true).
		Find(&skills).Error
	return skills, err
}

// GetAlwaysLoadSkills 获取总是加载的技能
func GetAlwaysLoadSkills() ([]Skill, error) {
	var skills []Skill
	err := DB.
		Table(tableNamePrefix+"skills").
		Where("always_load = ?", true).
		Find(&skills).Error
	return skills, err
}

// Upsert 创建或更新技能
func (s *Skill) Upsert() error {
	var existing Skill
	err := DB.
		Table(tableNamePrefix+"skills").
		Where("name = ?", s.Name).
		First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return s.Create()
	}
	s.ID = existing.ID
	return s.Update()
}
