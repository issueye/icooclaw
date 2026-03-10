package skill

import (
	"context"
	"strings"
)

// Loader 技能加载器接口
// 定义了技能加载和管理的基本操作
type Loader interface {
	// Load 加载所有技能
	// 从存储中加载所有技能，包括数据库和内置技能
	Load(ctx context.Context) ([]*Skill, error)

	// Reload 重新加载技能
	// 清空缓存并重新加载所有技能
	Reload(ctx context.Context) ([]*Skill, error)

	// GetByName 根据名称获取技能
	// 名称匹配不区分大小写
	GetByName(name string) *Skill

	// GetByID 根据ID获取技能
	GetByID(id string) *Skill

	// GetAlwaysLoad 获取始终加载的技能列表
	// 返回所有标记为 always_load 且启用的技能
	GetAlwaysLoad() []*Skill

	// GetByTag 根据标签获取技能
	// 返回包含指定标签的所有技能
	GetByTag(tag string) []*Skill

	// Search 搜索技能
	// 根据关键字搜索技能名称、描述和标签
	Search(keyword string) []*Skill
}

// SkillFilter 技能过滤器
// 用于筛选技能列表
type SkillFilter struct {
	Source  string   // 来源过滤: builtin, workspace, remote
	Enabled *bool    // 启用状态过滤
	Tags    []string // 标签过滤
	Keyword string   // 关键字搜索
}

// FilterSkills 过滤技能列表
// 根据过滤器条件筛选技能
func FilterSkills(skills []*Skill, filter SkillFilter) []*Skill {
	if filter.Source == "" && filter.Enabled == nil && len(filter.Tags) == 0 && filter.Keyword == "" {
		return skills
	}

	result := make([]*Skill, 0)
	for _, skill := range skills {
		if matchesFilter(skill, filter) {
			result = append(result, skill)
		}
	}
	return result
}

// matchesFilter 检查技能是否匹配过滤器
func matchesFilter(skill *Skill, filter SkillFilter) bool {
	// 来源过滤
	if filter.Source != "" && skill.Source != filter.Source {
		return false
	}

	// 启用状态过滤
	if filter.Enabled != nil && skill.Enabled != *filter.Enabled {
		return false
	}

	// 标签过滤
	if len(filter.Tags) > 0 {
		hasTag := false
		for _, tag := range filter.Tags {
			for _, skillTag := range skill.Tags {
				if tag == skillTag {
					hasTag = true
					break
				}
			}
			if hasTag {
				break
			}
		}
		if !hasTag {
			return false
		}
	}

	// 关键字搜索
	if filter.Keyword != "" {
		keyword := strings.ToLower(filter.Keyword)
		nameMatch := strings.Contains(strings.ToLower(skill.Name), keyword)
		descMatch := strings.Contains(strings.ToLower(skill.Description), keyword)
		tagMatch := false
		for _, tag := range skill.Tags {
			if strings.Contains(strings.ToLower(tag), keyword) {
				tagMatch = true
				break
			}
		}
		if !nameMatch && !descMatch && !tagMatch {
			return false
		}
	}

	return true
}
