package consts

type RoleType string

const (
	RoleUser      RoleType = "user"
	RoleAgent     RoleType = "agent"
	RoleSystem    RoleType = "system"
	RoleTool      RoleType = "tool"
	RoleAssistant RoleType = "assistant"
)

func (r RoleType) ToString() string {
	return string(r)
}
