//go:build ignore

package main

import (
	"os"
	"strings"
)

func main() {
	content, err := os.ReadFile("E:\\code\\issueye\\icooclaw\\agent\\icooclaw.core\\bus\\bus.go")
	if err != nil {
		panic(err)
	}

	oldStruct := `type InboundMessage struct {
	ID        string         ` + "`" + `json:"id,omitempty"` + "`" + `        // 消息 ID
	Channel   string         ` + "`" + `json:"channel,omitempty"` + "`" + `   // 通道
	ChatID    string         ` + "`" + `json:"chat_id,omitempty"` + "`" + `   // 会话 ID
	UserID    string         ` + "`" + `json:"user_id,omitempty"` + "`" + `   // 用户 ID
	Content   string         ` + "`" + `json:"content,omitempty"` + "`" + `   // 内容
	Timestamp time.Time      ` + "`" + `json:"timestamp,omitempty"` + "`" + ` // 时间戳
	Metadata  map[string]any ` + "`" + `json:"metadata,omitempty"` + "`" + `  // 元数据
}`

	newStruct := `type InboundMessage struct {
	ID        string         ` + "`" + `json:"id,omitempty"` + "`" + `          // 消息 ID
	SessionID uint           ` + "`" + `json:"session_id,omitempty"` + "`" + `  // 会话 ID
	Channel   string         ` + "`" + `json:"channel,omitempty"` + "`" + `     // 通道
	ChatID    string         ` + "`" + `json:"chat_id,omitempty"` + "`" + `     // 会话 ID
	UserID    string         ` + "`" + `json:"user_id,omitempty"` + "`" + `     // 用户 ID
	Content   string         ` + "`" + `json:"content,omitempty"` + "`" + `     // 内容
	Timestamp time.Time      ` + "`" + `json:"timestamp,omitempty"` + "`" + `   // 时间戳
	Metadata  map[string]any ` + "`" + `json:"metadata,omitempty"` + "`" + `    // 元数据
}`

	newContent := strings.Replace(string(content), oldStruct, newStruct, 1)
	if newContent == string(content) {
		panic("未找到匹配的内容")
	}

	err = os.WriteFile("E:\\code\\issueye\\icooclaw\\agent\\icooclaw.core\\bus\\bus.go", []byte(newContent), 0644)
	if err != nil {
		panic(err)
	}
}
