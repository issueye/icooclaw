package skill

import (
	"context"
	"log/slog"
	"strings"

	"icooclaw.core/storage"
)

// Skill 技能
type Skill struct {
	Name        string   // 技能名称
	Description string   // 技能描述
	Content     string   // 技能内容
	References  []string // 引用的技能
	Scripts     []string // 脚本
	AlwaysLoad  bool     // 是否始终加载该技能
	Enabled     bool     // 是否启用该技能
}

// SkillLoader 技能加载器
type SkillLoader struct {
	storage    *storage.Storage // 存储
	logger     *slog.Logger     // 日志记录器
	loaded     []*Skill         // 已加载的技能
	alwaysLoad []*Skill         // 总是加载的技能
}

// NewSkillLoader 创建技能加载器
func NewLoader(storage *storage.Storage, logger *slog.Logger) *SkillLoader {
	return &SkillLoader{
		storage:    storage,           // 存储
		logger:     logger,            // 日志记录器
		loaded:     make([]*Skill, 0), // 已加载的技能
		alwaysLoad: make([]*Skill, 0), // 总是加载的技能
	}
}

// Load 加载技能
func (l *SkillLoader) Load(ctx context.Context) ([]*Skill, error) {
	l.logger.Info("加载技能中...")

	// 从存储中获取所有技能
	sks, err := l.storage.Skill().GetAll()
	if err != nil {
		return nil, err
	}

	skills := make([]*Skill, 0)
	for _, sk := range sks {
		// 过滤未启用的技能
		if sk.Enabled {
			skill := &Skill{
				Name:        sk.Name,
				Description: sk.Description,
				Content:     sk.Content,
				AlwaysLoad:  sk.AlwaysLoad,
				Enabled:     sk.Enabled,
				References:  []string{},
				Scripts:     []string{},
			}

			skills = append(skills, skill)
		}
	}

	// 加载内置技能
	builtInSkills := l.loadBuiltInSkills()
	skills = append(skills, builtInSkills...)

	return skills, nil
}

func (l *SkillLoader) loadBuiltInSkills() []*Skill {
	builtInSkills := GetBuiltInSkills()

	for _, skill := range builtInSkills {
		l.loaded = append(l.loaded, skill)
		if skill.AlwaysLoad {
			l.alwaysLoad = append(l.alwaysLoad, skill)
		}
	}

	return builtInSkills
}

func (l *SkillLoader) GetAlwaysLoad() []*Skill {
	return l.alwaysLoad
}

func (l *SkillLoader) GetByName(name string) *Skill {
	for _, skill := range l.loaded {
		if strings.EqualFold(skill.Name, name) {
			return skill
		}
	}
	return nil
}

func (l *SkillLoader) Reload(ctx context.Context) ([]*Skill, error) {
	l.loaded = make([]*Skill, 0)
	l.alwaysLoad = make([]*Skill, 0)
	skills, err := l.Load(ctx)
	return skills, err
}
