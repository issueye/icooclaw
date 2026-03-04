package skill

import "context"

type Loader interface {
	Load(ctx context.Context) ([]*Skill, error)   // 加载技能
	Reload(ctx context.Context) ([]*Skill, error) // 重新加载技能
}
