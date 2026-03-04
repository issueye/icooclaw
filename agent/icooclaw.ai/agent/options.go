package agent

import (
	"icooclaw.ai/memory"
	"icooclaw.ai/skill"
	"icooclaw.ai/tools"
	icooclawbus "icooclaw.core/bus"
)

// AgentOption Agent 选项函数
type AgentOption func(*Agent)

// WithTools 设置工具注册表
func WithTools(registry *tools.Registry) AgentOption {
	return func(a *Agent) {
		a.tools = registry
	}
}

// WithMemoryStore 设置记忆存储
func WithMemoryStore(store memory.Loader) AgentOption {
	return func(a *Agent) {
		a.memory = store
	}
}

// WithSkillLoader 设置技能加载器
func WithSkillLoader(loader skill.Loader) AgentOption {
	return func(a *Agent) {
		a.skills = loader
	}
}

// WithMessageBus 设置消息总线
func WithMessageBus(bus *icooclawbus.MessageBus) AgentOption {
	return func(a *Agent) {
		a.bus = bus
	}
}
