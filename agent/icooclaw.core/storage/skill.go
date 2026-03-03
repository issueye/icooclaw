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
func (s *SkillStorage) GetByID(id uint) (*Skill, error) {
	var skill Skill
	err := s.db.First(&skill, id).Error
	return &skill, err
}

// GetByName 通过名称获取技能
func (s *SkillStorage) GetByName(name string) (*Skill, error) {
	var skill Skill
	err := s.db.Where("name = ?", name).First(&skill).Error
	return &skill, err
}

// Delete 删除技能
func (s *SkillStorage) Delete(id uint) error {
	return s.db.Delete(&Skill{}, id).Error
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
	err := s.db.Where("always_load = ?", true).Find(&skills).Error
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

// Page 分页获取技能
func (s *SkillStorage) Page(q *QuerySkill) (*ResQuerySkill, error) {
	var total int64
	query := s.db.Model(&Skill{})
	if q.KeyWord != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+q.KeyWord+"%", "%"+q.KeyWord+"%")
	}
	if q.Enabled != nil {
		query = query.Where("enabled = ?", *q.Enabled)
	}
	if q.Source != "" {
		query = query.Where("source = ?", q.Source)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

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

// QuerySkill 技能查询参数
type QuerySkill struct {
	Page    Page   `json:"page"`
	KeyWord string `json:"key_word"`
	Enabled *bool  `json:"enabled"`
	Source  string `json:"source"`
}

// ResQuerySkill 技能查询结果
type ResQuerySkill struct {
	Page    Page    `json:"page"`
	Records []Skill `json:"records"`
}
