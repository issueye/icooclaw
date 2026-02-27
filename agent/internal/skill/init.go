// Package skill 提供技能加载和管理功能
//
// 主要功能:
//   - 技能加载器: 从数据库和内置定义加载技能
//   - 内置技能: 预定义的 identity、file、shell、web 技能
//   - 技能管理: 查询、获取技能
//
// 使用示例:
//
//	loader := skill.NewLoader(storage, logger)
//	if err := loader.Load(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	skills := loader.GetLoaded()
package skill
