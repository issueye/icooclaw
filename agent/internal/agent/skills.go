package agent

import (
	"context"
	"log/slog"
	"strings"

	"github.com/icooclaw/icooclaw/internal/storage"
)

// Skill 技能
type Skill struct {
	Name        string
	Description string
	Content     string
	AlwaysLoad  bool
	Enabled     bool
}

// SkillsLoader 技能加载器
type SkillsLoader struct {
	storage    *storage.Storage
	logger     *slog.Logger
	loaded     []Skill
	alwaysLoad []Skill
}

// NewSkillsLoader 创建技能加载器
func NewSkillsLoader(storage *storage.Storage, logger *slog.Logger) *SkillsLoader {
	return &SkillsLoader{
		storage:    storage,
		logger:     logger,
		loaded:     make([]Skill, 0),
		alwaysLoad: make([]Skill, 0),
	}
}

// Load 加载技能
func (s *SkillsLoader) Load(ctx context.Context) error {
	s.logger.Info("Loading skills")

	// 从数据库加载技能
	skills, err := s.storage.GetEnabledSkills()
	if err != nil {
		s.logger.Warn("Failed to load skills from database", "error", err)
	}

	// 添加内置技能
	s.loadBuiltInSkills()

	// 处理加载的技能
	for _, skill := range skills {
		s.loaded = append(s.loaded, Skill{
			Name:        skill.Name,
			Description: skill.Description,
			Content:     skill.Content,
			AlwaysLoad:  skill.AlwaysLoad,
			Enabled:     skill.Enabled,
		})
		if skill.AlwaysLoad {
			s.alwaysLoad = append(s.alwaysLoad, Skill{
				Name:        skill.Name,
				Description: skill.Description,
				Content:     skill.Content,
				AlwaysLoad:  skill.AlwaysLoad,
				Enabled:     skill.Enabled,
			})
		}
	}

	s.logger.Info("Skills loaded", "count", len(s.loaded))
	return nil
}

// loadBuiltInSkills 加载内置技能
func (s *SkillsLoader) loadBuiltInSkills() {
	// 添加内置技能
	builtInSkills := []Skill{
		{
			Name:        "file",
			Description: "文件操作技能",
			Content: `## 文件操作技能
你可以帮助用户读取、写入和管理文件。
- 读取文件：使用file_read工具
- 写入文件：使用file_write工具
- 列出目录：使用file_list工具`,
			AlwaysLoad: true,
			Enabled:    true,
		},
		{
			Name:        "shell",
			Description: "Shell命令执行技能",
			Content: `## Shell命令执行技能
你可以帮助用户执行Shell命令。
- 执行命令：使用shell_exec工具`,
			AlwaysLoad: true,
			Enabled:    true,
		},
		{
			Name:        "web",
			Description: "网页搜索和抓取技能",
			Content: `## 网页搜索和抓取技能
你可以帮助用户搜索互联网和抓取网页内容。
- 搜索网页：使用web_search工具
- 抓取网页：使用web_fetch工具`,
			AlwaysLoad: false,
			Enabled:    true,
		},
	}

	for _, skill := range builtInSkills {
		s.loaded = append(s.loaded, skill)
		if skill.AlwaysLoad {
			s.alwaysLoad = append(s.alwaysLoad, skill)
		}
	}
}

// GetLoaded 获取已加载的技能
func (s *SkillsLoader) GetLoaded() []Skill {
	return s.loaded
}

// GetAlwaysLoad 获取总是加载的技能
func (s *SkillsLoader) GetAlwaysLoad() []Skill {
	return s.alwaysLoad
}

// GetByName 根据名称获取技能
func (s *SkillsLoader) GetByName(name string) *Skill {
	for _, skill := range s.loaded {
		if strings.EqualFold(skill.Name, name) {
			return &skill
		}
	}
	return nil
}

// Reload 重新加载技能
func (s *SkillsLoader) Reload(ctx context.Context) error {
	s.loaded = make([]Skill, 0)
	s.alwaysLoad = make([]Skill, 0)
	return s.Load(ctx)
}
