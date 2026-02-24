package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestSession tests for Session struct
func TestSession_Structure(t *testing.T) {
	session := Session{
		ID:        1,
		Key:       "telegram:chat_123",
		Channel:   "telegram",
		ChatID:    "chat_123",
		UserID:    "user_456",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	assert.Equal(t, uint(1), session.ID)
	assert.Equal(t, "telegram:chat_123", session.Key)
	assert.Equal(t, "telegram", session.Channel)
	assert.Equal(t, "chat_123", session.ChatID)
	assert.Equal(t, "user_456", session.UserID)
}

func TestSession_TableName(t *testing.T) {
	assert.Equal(t, "sessions", (&Session{}).TableName())
}

// TestMessage tests for Message struct
func TestMessage_Structure(t *testing.T) {
	msg := Message{
		ID:               1,
		SessionID:        1,
		Role:             "user",
		Content:          "Hello",
		ToolCalls:        "",
		ToolCallID:       "",
		ToolName:         "",
		ReasoningContent: "",
		Timestamp:        time.Now(),
	}

	assert.Equal(t, uint(1), msg.ID)
	assert.Equal(t, uint(1), msg.SessionID)
	assert.Equal(t, "user", msg.Role)
	assert.Equal(t, "Hello", msg.Content)
}

// TestMessage_TableName tests that Message has correct table name
func TestMessage_TableName(t *testing.T) {
	assert.Equal(t, "messages", (&Message{}).TableName())
}

// TestTask tests for Task struct
func TestTask_Structure(t *testing.T) {
	task := Task{
		ID:          1,
		Name:        "test_task",
		Description: "A test task",
		CronExpr:    "*/5 * * * *",
		Interval:    300,
		Message:     "Hello",
		Channel:     "telegram",
		ChatID:      "chat_123",
		Enabled:     true,
		NextRunAt:   time.Now(),
		LastRunAt:   time.Now(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	assert.Equal(t, uint(1), task.ID)
	assert.Equal(t, "test_task", task.Name)
	assert.True(t, task.Enabled)
	assert.Equal(t, "*/5 * * * *", task.CronExpr)
}

func TestTask_TableName(t *testing.T) {
	assert.Equal(t, "tasks", (&Task{}).TableName())
}

// TestSkill tests for Skill struct
func TestSkill_Structure(t *testing.T) {
	skill := Skill{
		ID:          1,
		Name:        "test_skill",
		Description: "A test skill",
		Content:     "skill content",
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	assert.Equal(t, uint(1), skill.ID)
	assert.Equal(t, "test_skill", skill.Name)
	assert.True(t, skill.Enabled)
	assert.Equal(t, "skill content", skill.Content)
}

func TestSkill_TableName(t *testing.T) {
	assert.Equal(t, "skills", (&Skill{}).TableName())
}

// TestMemory tests for Memory struct
func TestMemory_Structure(t *testing.T) {
	memory := Memory{
		ID:        1,
		Type:      "memory",
		Key:       "important_key",
		Content:   "Important information",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	assert.Equal(t, uint(1), memory.ID)
	assert.Equal(t, "memory", memory.Type)
	assert.Equal(t, "important_key", memory.Key)
	assert.Equal(t, "Important information", memory.Content)
}

func TestMemory_TableName(t *testing.T) {
	assert.Equal(t, "memories", (&Memory{}).TableName())
}

// TestChannelConfig tests for ChannelConfig struct
func TestChannelConfig_Structure(t *testing.T) {
	config := ChannelConfig{
		ID:        1,
		Name:      "telegram",
		Enabled:   true,
		Config:    `{"token":"test"}`,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	assert.Equal(t, uint(1), config.ID)
	assert.Equal(t, "telegram", config.Name)
	assert.True(t, config.Enabled)
	assert.Contains(t, config.Config, "token")
}

func TestChannelConfig_TableName(t *testing.T) {
	assert.Equal(t, "channel_configs", (&ChannelConfig{}).TableName())
}

// TestProviderConfig tests for ProviderConfig struct
func TestProviderConfig_Structure(t *testing.T) {
	config := ProviderConfig{
		ID:        1,
		Name:      "openai",
		Enabled:   true,
		Config:    `{"api_key":"test"}`,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	assert.Equal(t, uint(1), config.ID)
	assert.Equal(t, "openai", config.Name)
	assert.True(t, config.Enabled)
}

func TestProviderConfig_TableName(t *testing.T) {
	assert.Equal(t, "provider_configs", (&ProviderConfig{}).TableName())
}
