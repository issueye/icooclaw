package storage

import (
	"fmt"

	"gorm.io/gorm"

	icooclawErrors "icooclaw/pkg/errors"
)

// Skill represents a skill configuration.
type Skill struct {
	Model
	Name        string `gorm:"column:name;type:varchar(100);uniqueIndex;not null;comment:技能名称" json:"name"` // 技能名称
	Description string `gorm:"column:description;type:text;comment:技能描述" json:"description"`                // 技能描述
	Enabled     bool   `gorm:"column:enabled;type:tinyint(1);default:true;comment:是否启用" json:"enabled"`     // 是否启用
	Version     string `gorm:"column:version;type:varchar(10);default:1.0.0;comment:版本号" json:"version"`    // 版本号
	Path        string `gorm:"column:path;type:text;comment:技能路径" json:"path"`                              // 技能路径 默认 workspace/.skills/<name>-<version>/
}

// TableName returns the table name for Skill.
func (Skill) TableName() string {
	return tableNamePrefix + "skills"
}

type SkillStorage struct {
	db *gorm.DB
}

func NewSkillStorage(db *gorm.DB) *SkillStorage {
	return &SkillStorage{db: db}
}

// SaveSkill saves a skill configuration.
func (s *SkillStorage) SaveSkill(sk *Skill) error {
	result := s.db.Create(sk)
	return result.Error
}

// GetSkill gets a skill by name.
func (s *SkillStorage) GetSkill(name string) (*Skill, error) {
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
func (s *SkillStorage) ListSkills() ([]*Skill, error) {
	var skills []*Skill
	result := s.db.Order("name").Find(&skills)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list skills: %w", result.Error)
	}
	return skills, nil
}

// ListEnabledSkills lists all enabled skills.
func (s *SkillStorage) ListEnabledSkills() ([]*Skill, error) {
	var skills []*Skill
	result := s.db.Where("enabled = ?", true).Order("name").Find(&skills)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list enabled skills: %w", result.Error)
	}
	return skills, nil
}

// DeleteSkill deletes a skill by name.
func (s *SkillStorage) DeleteSkill(name string) error {
	result := s.db.Where("name = ?", name).Delete(&Skill{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete skill: %w", result.Error)
	}
	return nil
}

type QuerySkill struct {
	Page    Page   `json:"page"`
	KeyWord string `json:"key_word"`
	Enabled *bool  `json:"enabled"`
}

type ResQuerySkill struct {
	Page    Page    `json:"page"`
	Records []Skill `json:"records"`
}

// Page gets skills with pagination.
func (s *SkillStorage) Page(query *QuerySkill) (*ResQuerySkill, error) {
	var res ResQuerySkill

	qry := s.db.Model(&Skill{})

	if query.KeyWord != "" {
		qry = qry.Where("name LIKE ? OR description LIKE ?", "%"+query.KeyWord+"%", "%"+query.KeyWord+"%")
	}

	if query.Enabled != nil {
		qry = qry.Where("enabled = ?", *query.Enabled)
	}

	qry = qry.Order("name")

	result := qry.Count(&res.Page.Total)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to count skills: %w", result.Error)
	}

	if query.Page.Page == 0 || query.Page.Size == 0 {
		result = qry.Find(&res.Records)
	} else {
		result = qry.Limit(query.Page.Size).
			Offset((query.Page.Page - 1) * query.Page.Size).
			Find(&res.Records)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get skills: %w", result.Error)
	}

	return &res, nil
}
