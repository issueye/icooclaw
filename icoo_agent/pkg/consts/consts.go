package consts

import "fmt"

type RoleType string

const (
	RoleUser       RoleType = "user"
	RoleAgent      RoleType = "agent"
	RoleSystem     RoleType = "system"
	RoleAssistant  RoleType = "assistant"
	RoleTool       RoleType = "tool"
	RoleToolCall   RoleType = "tool_call"
	RoleToolResult RoleType = "tool_result"
)

func (r RoleType) ToString() string {
	return string(r)
}

func ToRole(role string) RoleType {
	return RoleType(role)
}

// DEF_GATEWAY_PORT 默认网关端口
const DEF_GATEWAY_PORT = 16777

// DEF_GATEWAY_HOST 默认网关主机
const DEF_GATEWAY_HOST = "0.0.0.0"

// DEFAULT_MODEL_KEY 默认模型键名
const DEFAULT_MODEL_KEY = "agent.default_model"

const SKILL_DIR = "skills"

const DEFAULT_TOOL_ITERATIONS = 30

// ProviderType represents a provider type.
type ProviderType string

const (
	ProviderOpenAI         ProviderType = "openai"
	ProviderAnthropic      ProviderType = "anthropic"
	ProviderDeepSeek       ProviderType = "deepseek"
	ProviderOpenRouter     ProviderType = "openrouter"
	ProviderGemini         ProviderType = "gemini"
	ProviderMistral        ProviderType = "mistral"
	ProviderGroq           ProviderType = "groq"
	ProviderAzure          ProviderType = "azure"
	ProviderOllama         ProviderType = "ollama"
	ProviderMoonshot       ProviderType = "moonshot"
	ProviderZhipu          ProviderType = "zhipu"
	ProviderQwen           ProviderType = "qwen"
	ProviderQwenCodingPlan ProviderType = "qwen_coding_plan"
	ProviderSiliconFlow    ProviderType = "siliconflow"
	ProviderGrok           ProviderType = "grok"
)

func (p ProviderType) ToString() string {
	return string(p)
}

func ToProviderType(providerType string) ProviderType {
	return ProviderType(providerType)
}

func (p ProviderType) String() string {
	return string(p)
}

const DEFAULT_AGENT_NAME = "default"

// GetSessionKey 生成会话键，格式: channel:sessionID
func GetSessionKey(channel, sessionID string) string {
	return fmt.Sprintf("%s:%s", channel, sessionID)
}

// GetDefSessionKey 使用默认代理名生成会话键（已废弃，保留兼容）
// Deprecated: 使用 GetSessionKey 代替
func GetDefSessionKey(channel, sessionID string) string {
	return GetSessionKey(channel, sessionID)
}
