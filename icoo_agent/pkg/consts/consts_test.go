package consts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoleType_String(t *testing.T) {
	tests := []struct {
		role     RoleType
		expected string
	}{
		{RoleUser, "user"},
		{RoleAgent, "agent"},
		{RoleSystem, "system"},
		{RoleToolCall, "tool_call"},
		{RoleAssistant, "assistant"},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.role))
		})
	}
}

func TestRoleType_Constants(t *testing.T) {
	// 确保所有常量都定义正确
	assert.Equal(t, RoleType("user"), RoleUser)
	assert.Equal(t, RoleType("agent"), RoleAgent)
	assert.Equal(t, RoleType("system"), RoleSystem)
	assert.Equal(t, RoleType("tool_call"), RoleToolCall)
	assert.Equal(t, RoleType("assistant"), RoleAssistant)
}

func TestRoleType_Type(t *testing.T) {
	// 确保 RoleType 是 string 的别名
	var r RoleType = "test"
	var s string = string(r)
	assert.Equal(t, "test", s)
}

func TestRoleType_Usage(t *testing.T) {
	// 模拟实际使用场景
	roles := map[RoleType]string{
		RoleUser:       "用户消息",
		RoleAgent:      "Agent 消息",
		RoleSystem:     "系统消息",
		RoleToolCall:   "工具调用",
		RoleToolResult: "工具返回",
		RoleAssistant:  "助手消息",
	}

	assert.Equal(t, "用户消息", roles[RoleUser])
	assert.Equal(t, "Agent 消息", roles[RoleAgent])
	assert.Equal(t, "系统消息", roles[RoleSystem])
	assert.Equal(t, "工具返回", roles[RoleToolResult])
	assert.Equal(t, "工具调用", roles[RoleToolCall])
	assert.Equal(t, "助手消息", roles[RoleAssistant])
}

func TestRoleType_Comparison(t *testing.T) {
	// 测试角色比较
	userRole := RoleUser
	assert.True(t, userRole == RoleUser)
	assert.False(t, userRole == RoleAgent)
	// RoleType 可以与 string 比较（隐式转换）
	assert.True(t, string(userRole) == "user")
}

func TestRoleType_SwitchCase(t *testing.T) {
	// 测试在 switch 中使用
	getRoleName := func(r RoleType) string {
		switch r {
		case RoleUser:
			return "user"
		case RoleAgent:
			return "agent"
		case RoleSystem:
			return "system"
		case RoleToolResult:
			return "tool_result"
		case RoleToolCall:
			return "tool_call"
		case RoleAssistant:
			return "assistant"
		default:
			return "unknown"
		}
	}

	assert.Equal(t, "user", getRoleName(RoleUser))
	assert.Equal(t, "agent", getRoleName(RoleAgent))
	assert.Equal(t, "system", getRoleName(RoleSystem))
	assert.Equal(t, "tool_result", getRoleName(RoleToolResult))
	assert.Equal(t, "tool_call", getRoleName(RoleToolCall))
	assert.Equal(t, "assistant", getRoleName(RoleAssistant))
	assert.Equal(t, "unknown", getRoleName(RoleType("custom")))
}
