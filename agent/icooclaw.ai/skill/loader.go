package skill

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"icooclaw.core/storage"
)

// Skill 技能定义
// 统一后的技能结构体，整合了各模块的需求
type Skill struct {
	ID          string                 `json:"id"`          // 技能ID
	Name        string                 `json:"name"`        // 技能名称
	Description string                 `json:"description"` // 技能描述
	Content     string                 `json:"content"`     // 技能内容(Markdown格式)
	Prompt      string                 `json:"prompt"`      // 提示词(用于AI)
	Tools       []string               `json:"tools"`       // 引用的工具列表
	Scripts     []string               `json:"scripts"`     // 脚本列表
	References  []string               `json:"references"`  // 引用的其他技能
	Config      map[string]interface{} `json:"config"`      // 配置项
	Tags        []string               `json:"tags"`        // 标签
	Source      string                 `json:"source"`      // 来源: builtin, workspace, remote
	AlwaysLoad  bool                   `json:"always_load"` // 是否始终加载
	Enabled     bool                   `json:"enabled"`     // 是否启用
	Version     string                 `json:"version"`     // 版本号
	CreatedAt   time.Time              `json:"created_at"`  // 创建时间
	UpdatedAt   time.Time              `json:"updated_at"`  // 更新时间
}

// SkillLoader 技能加载器实现
// 实现了 Loader 接口
type SkillLoader struct {
	storage    *storage.Storage    // 存储
	logger     *slog.Logger        // 日志记录器
	mu         sync.RWMutex        // 读写锁
	loaded     map[string]*Skill   // 已加载的技能(按名称索引)
	loadedByID map[string]*Skill   // 已加载的技能(按ID索引)
	alwaysLoad []*Skill            // 总是加载的技能
	tagIndex   map[string][]*Skill // 标签索引
	lastLoad   time.Time           // 最后加载时间
}

// NewLoader 创建技能加载器
func NewLoader(storage *storage.Storage, logger *slog.Logger) *SkillLoader {
	if logger == nil {
		logger = slog.Default()
	}
	return &SkillLoader{
		storage:    storage,
		logger:     logger,
		loaded:     make(map[string]*Skill),
		loadedByID: make(map[string]*Skill),
		alwaysLoad: make([]*Skill, 0),
		tagIndex:   make(map[string][]*Skill),
	}
}

// Load 加载所有技能
// 从数据库和内置定义加载技能，构建索引
func (l *SkillLoader) Load(ctx context.Context) ([]*Skill, error) {
	l.logger.Info("开始加载技能...")
	start := time.Now()

	// 清空现有数据
	l.mu.Lock()
	l.loaded = make(map[string]*Skill)
	l.loadedByID = make(map[string]*Skill)
	l.alwaysLoad = make([]*Skill, 0)
	l.tagIndex = make(map[string][]*Skill)
	l.mu.Unlock()

	skills := make([]*Skill, 0)

	// 从存储加载技能
	storageSkills, err := l.loadFromStorage(ctx)
	if err != nil {
		l.logger.Error("从存储加载技能失败", "error", err)
		return nil, fmt.Errorf("加载技能失败: %w", err)
	}
	skills = append(skills, storageSkills...)

	// 加载内置技能
	builtInSkills := l.loadBuiltInSkills()
	skills = append(skills, builtInSkills...)

	// 构建索引
	l.buildIndex(skills)

	l.lastLoad = time.Now()
	l.logger.Info("技能加载完成",
		"count", len(skills),
		"alwaysLoad", len(l.alwaysLoad),
		"duration", time.Since(start))

	return skills, nil
}

// loadFromStorage 从存储加载技能
func (l *SkillLoader) loadFromStorage(ctx context.Context) ([]*Skill, error) {
	sks, err := l.storage.Skill().GetAll()
	if err != nil {
		return nil, err
	}

	skills := make([]*Skill, 0, len(sks))
	for _, sk := range sks {
		skill := l.convertFromStorage(&sk)
		skills = append(skills, skill)
	}

	return skills, nil
}

// convertFromStorage 将存储模型转换为技能对象
func (l *SkillLoader) convertFromStorage(sk *storage.Skill) *Skill {
	skill := &Skill{
		ID:          sk.ID,
		Name:        sk.Name,
		Description: sk.Description,
		Content:     sk.Content,
		Source:      sk.Source,
		AlwaysLoad:  sk.AlwaysLoad,
		Enabled:     sk.Enabled,
		CreatedAt:   sk.CreatedAt,
		UpdatedAt:   sk.UpdatedAt,
		Tools:       []string{},
		Scripts:     []string{},
		References:  []string{},
		Tags:        []string{},
		Config:      make(map[string]interface{}),
	}

	// 解析元数据
	if sk.Metadata != "" {
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(sk.Metadata), &metadata); err != nil {
			l.logger.Warn("解析技能元数据失败", "skill", sk.Name, "error", err)
		} else {
			// 提取Prompt
			if prompt, ok := metadata["prompt"].(string); ok {
				skill.Prompt = prompt
			}
			// 提取Tools
			if tools, ok := metadata["tools"].([]interface{}); ok {
				for _, t := range tools {
					if toolStr, ok := t.(string); ok {
						skill.Tools = append(skill.Tools, toolStr)
					}
				}
			}
			// 提取Tags
			if tags, ok := metadata["tags"].([]interface{}); ok {
				for _, t := range tags {
					if tagStr, ok := t.(string); ok {
						skill.Tags = append(skill.Tags, tagStr)
					}
				}
			}
			// 提取Version
			if version, ok := metadata["version"].(string); ok {
				skill.Version = version
			}
			// 保存原始配置
			skill.Config = metadata
		}
	}

	return skill
}

// loadBuiltInSkills 加载内置技能
func (l *SkillLoader) loadBuiltInSkills() []*Skill {
	builtInSkills := GetBuiltInSkills()

	for _, skill := range builtInSkills {
		// 确保内置技能有ID
		if skill.ID == "" {
			skill.ID = "builtin_" + skill.Name
		}
		if skill.Source == "" {
			skill.Source = "builtin"
		}
	}

	return builtInSkills
}

// buildIndex 构建技能索引
func (l *SkillLoader) buildIndex(skills []*Skill) {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, skill := range skills {
		// 按名称索引
		l.loaded[strings.ToLower(skill.Name)] = skill
		// 按ID索引
		if skill.ID != "" {
			l.loadedByID[skill.ID] = skill
		}
		// 收集始终加载的技能
		if skill.AlwaysLoad && skill.Enabled {
			l.alwaysLoad = append(l.alwaysLoad, skill)
		}
		// 构建标签索引
		for _, tag := range skill.Tags {
			tagLower := strings.ToLower(tag)
			l.tagIndex[tagLower] = append(l.tagIndex[tagLower], skill)
		}
	}
}

// Reload 重新加载技能
// 清空缓存并重新加载所有技能
func (l *SkillLoader) Reload(ctx context.Context) ([]*Skill, error) {
	l.logger.Info("重新加载技能...")
	return l.Load(ctx)
}

// GetByName 根据名称获取技能
// 名称匹配不区分大小写
func (l *SkillLoader) GetByName(name string) *Skill {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.loaded[strings.ToLower(name)]
}

// GetByID 根据ID获取技能
func (l *SkillLoader) GetByID(id string) *Skill {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.loadedByID[id]
}

// GetAlwaysLoad 获取始终加载的技能列表
func (l *SkillLoader) GetAlwaysLoad() []*Skill {
	l.mu.RLock()
	defer l.mu.RUnlock()
	result := make([]*Skill, len(l.alwaysLoad))
	copy(result, l.alwaysLoad)
	return result
}

// GetByTag 根据标签获取技能
func (l *SkillLoader) GetByTag(tag string) []*Skill {
	l.mu.RLock()
	defer l.mu.RUnlock()
	skills, ok := l.tagIndex[strings.ToLower(tag)]
	if !ok {
		return []*Skill{}
	}
	result := make([]*Skill, len(skills))
	copy(result, skills)
	return result
}

// Search 搜索技能
// 根据关键字搜索技能名称和描述
func (l *SkillLoader) Search(keyword string) []*Skill {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if keyword == "" {
		return l.getAllSkills()
	}

	keywordLower := strings.ToLower(keyword)
	result := make([]*Skill, 0)

	for _, skill := range l.loaded {
		if strings.Contains(strings.ToLower(skill.Name), keywordLower) ||
			strings.Contains(strings.ToLower(skill.Description), keywordLower) ||
			l.containsTag(skill, keywordLower) {
			result = append(result, skill)
		}
	}

	return result
}

// containsTag 检查技能是否包含指定标签
func (l *SkillLoader) containsTag(skill *Skill, tag string) bool {
	for _, t := range skill.Tags {
		if strings.ToLower(t) == tag {
			return true
		}
	}
	return false
}

// getAllSkills 获取所有技能
func (l *SkillLoader) getAllSkills() []*Skill {
	result := make([]*Skill, 0, len(l.loaded))
	for _, skill := range l.loaded {
		result = append(result, skill)
	}
	return result
}

// GetLoaded 获取已加载的所有技能
func (l *SkillLoader) GetLoaded() []*Skill {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.getAllSkills()
}

// GetEnabled 获取启用的技能
func (l *SkillLoader) GetEnabled() []*Skill {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make([]*Skill, 0)
	for _, skill := range l.loaded {
		if skill.Enabled {
			result = append(result, skill)
		}
	}
	return result
}

// GetLastLoadTime 获取最后加载时间
func (l *SkillLoader) GetLastLoadTime() time.Time {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.lastLoad
}
