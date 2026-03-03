package consts

type RoleType string

const (
	RoleUser      RoleType = "user"
	RoleAgent     RoleType = "agent"
	RoleSystem    RoleType = "system"
	RoleTool      RoleType = "tool"
	RoleToolCall  RoleType = "tool_call"
	RoleAssistant RoleType = "assistant"
)

func (r RoleType) ToString() string {
	return string(r)
}

// DEF_GATEWAY_PORT 默认网关端口
const DEF_GATEWAY_PORT = 16777

// DEF_GATEWAY_HOST 默认网关主机
const DEF_GATEWAY_HOST = "0.0.0.0"
