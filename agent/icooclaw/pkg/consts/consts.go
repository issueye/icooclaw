package consts

type RoleType string

const (
	RoleUser       RoleType = "user"
	RoleAgent      RoleType = "agent"
	RoleSystem     RoleType = "system"
	RoleAssistant  RoleType = "assistant"
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
