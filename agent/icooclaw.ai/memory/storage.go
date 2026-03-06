package memory

import (
	"fmt"
	"log/slog"
	"strings"

	"icooclaw.core/consts"
	"icooclaw.core/storage"
)

// Storage 存储结构体
type Storage struct {
	storage *storage.Storage
	logger  *slog.Logger
}

// NewStorage 创建新的存储实例
func NewStorage(storage *storage.Storage, logger *slog.Logger) *Storage {
	return &Storage{
		storage: storage,
		logger:  logger,
	}
}

// Load 加载记忆
func (s *Storage) Load(id string, maxCount int) ([]*Message, error) {
	// 从数据库加载消息
	msgs, err := s.storage.Message().GetBySessionID(id, maxCount, 0)
	if err != nil {
		return nil, err
	}

	// 转换为 Message 结构体
	rtnMsgs := make([]*Message, 0, len(msgs))
	for _, msg := range msgs {
		rtnMsg := &Message{
			Role:             msg.Role.ToString(),
			ToolCallID:       msg.ToolCallID,
			ToolCallName:     msg.ToolName,
			ToolArguments:    msg.ToolArguments,
			ToolCallResult:   msg.ToolResult,
			Content:          msg.Content,
			Thinking:         msg.Thinking,
			ReasoningContent: msg.ReasoningContent,
		}
		rtnMsgs = append(rtnMsgs, rtnMsg)
	}

	return rtnMsgs, nil
}

// Save 保存记忆
func (s *Storage) Save(id string, msg *Message) error {
	saveData := &storage.Message{}
	saveData.Role = consts.ToRole(msg.Role)
	saveData.Content = msg.Content
	saveData.ToolCallID = msg.ToolCallID
	saveData.ToolName = msg.ToolCallName
	saveData.ToolArguments = msg.ToolArguments
	saveData.ToolResult = msg.ToolCallResult
	saveData.Thinking = msg.Thinking
	saveData.ReasoningContent = msg.ReasoningContent
	err := s.storage.Message().CreateOrUpdate(saveData)
	if err != nil {
		s.logger.Error("保存消息失败", slog.Any("错误信息", err.Error()))
		return err
	}

	return nil
}

// BatchSave 批量保存记忆
func (s *Storage) BatchSave(id string, messages []*Message) error {
	// 保存消息到数据库
	for _, msg := range messages {
		err := s.Save(id, msg)
		if err != nil {
			continue
		}
	}

	return nil
}

// Delete 删除记忆
func (s *Storage) Delete(id string) error {
	err := s.storage.Message().Delete(id)
	if err != nil {
		s.logger.Error("删除记忆失败", slog.Any("错误信息", err.Error()))
		return err
	}
	return nil
}

// DeleteBySessionID 删除会话的所有记忆
func (s *Storage) DeleteBySessionID(id string) error {
	err := s.storage.Message().DeleteBySessionID(id)
	if err != nil {
		s.logger.Error("删除会话的所有记忆失败", slog.Any("错误信息", err.Error()))
		return err
	}
	return nil
}

// Update 更新记忆
func (s *Storage) Update(id string, msg *Message) error {
	err := s.Save(id, msg)
	if err != nil {
		return err
	}

	return nil
}

// SummarizeMessages 对消息进行摘要
func (s *Storage) SummarizeMessages(messages []*Message) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	// 简单摘要策略：提取关键信息
	// 1. 用户问题和助手回答的模式
	// 2. 工具调用记录
	// 3. 关键决策

	var summaryParts []string
	summaryParts = append(summaryParts, fmt.Sprintf("会话包含 %d 条消息", len(messages)))

	// 统计工具调用
	toolCallCount := 0
	userMsgCount := 0
	assistantMsgCount := 0

	for _, msg := range messages {
		if msg.Role == consts.RoleUser.ToString() {
			userMsgCount++
		} else if msg.Role == consts.RoleAssistant.ToString() {
			assistantMsgCount++
			if msg.ToolCallID != "" {
				toolCallCount++
			}
		}
	}

	summaryParts = append(summaryParts, fmt.Sprintf("用户消息: %d, 助手消息: %d, 工具调用: %d",
		userMsgCount, assistantMsgCount, toolCallCount))

	// 提取最近的5对对话作为样例
	if len(messages) >= 2 {
		summaryParts = append(summaryParts, "\n最近对话样例:")
		sampleCount := 0
		for i := len(messages) - 1; i >= 0 && sampleCount < 5; i-- {
			if messages[i].Role == consts.RoleUser.ToString() && i+1 < len(messages) && messages[i+1].Role == consts.RoleAssistant.ToString() {
				userContent := truncate(messages[i].Content, 100)
				assistantContent := truncate(messages[i+1].Content, 100)
				summaryParts = append(summaryParts, fmt.Sprintf("Q: %s\nA: %s", userContent, assistantContent))
				sampleCount++
			}
		}
	}

	return strings.Join(summaryParts, "\n"), nil
}

// truncate 截断字符串，使用 rune 保证中文字符不乱码
func truncate(s string, maxLen int) string {
	r := []rune(s)
	if len(r) <= maxLen {
		return s
	}
	return string(r[:maxLen]) + "..."
}

// Search 搜索记忆
func (s *Storage) Search(id string, query string) ([]*Message, error) {
	return nil, nil
}
