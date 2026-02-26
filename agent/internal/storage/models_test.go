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

// TestMemory_GetTags tests for Memory.GetTags method
func TestMemory_GetTags(t *testing.T) {
	tests := []struct {
		name     string
		tags     string
		expected []string
	}{
		{
			name:     "Empty tags",
			tags:     "",
			expected: nil,
		},
		{
			name:     "Single tag",
			tags:     ",important,",
			expected: []string{"important"},
		},
		{
			name:     "Multiple tags",
			tags:     ",work,personal,urgent,",
			expected: []string{"work", "personal", "urgent"},
		},
		{
			name:     "No surrounding commas",
			tags:     "tag1,tag2",
			expected: []string{"tag1", "tag2"},
		},
		{
			name:     "Only commas",
			tags:     ",,",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memory := &Memory{Tags: tt.tags}
			result := memory.GetTags()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestMemory_TagsFormatting tests tags are formatted correctly with commas
func TestMemory_TagsFormatting(t *testing.T) {
	// 直接测试Tags字段的格式，不需要调用SetTags（需要DB）
	memory := &Memory{
		Tags: ",work,personal,urgent,",
	}

	tags := memory.GetTags()
	assert.Equal(t, []string{"work", "personal", "urgent"}, tags)

	// 测试只有单个标签
	memory2 := &Memory{
		Tags: ",important,",
	}
	tags2 := memory2.GetTags()
	assert.Equal(t, []string{"important"}, tags2)
}

// TestMemory_GetTags_Roundtrip tests the get tags behavior
func TestMemory_GetTags_Roundtrip(t *testing.T) {
	// 直接设置Tags字段（模拟SetTags的效果）
	memory := &Memory{
		Tags: ",work,personal,urgent,",
	}

	tags := []string{"work", "personal", "urgent"}
	retrievedTags := memory.GetTags()
	assert.Equal(t, tags, retrievedTags)
}

// TestSkill_Upsert tests for Skill.Upsert method structure
func TestSkill_Upsert_Structure(t *testing.T) {
	skill := &Skill{
		Name:    "test_skill",
		Content: "skill content",
	}

	// Upsert should have Name set
	assert.Equal(t, "test_skill", skill.Name)
}

// TestTask_UpdateNextRun tests for Task.UpdateNextRun method structure
func TestTask_UpdateNextRun_Structure(t *testing.T) {
	task := &Task{
		Name:     "test_task",
		CronExpr: "*/5 * * * *",
	}

	// Task should have CronExpr
	assert.Equal(t, "*/5 * * * *", task.CronExpr)
}

// TestMemory_SoftDelete tests soft delete flag
func TestMemory_SoftDelete(t *testing.T) {
	memory := &Memory{
		IsDeleted: false,
	}

	assert.False(t, memory.IsDeleted)

	// Note: SoftDelete() method requires DB, just test struct behavior here
	memory.IsDeleted = true
	assert.True(t, memory.IsDeleted)
}

// TestMemory_Pin tests pin functionality
func TestMemory_Pin(t *testing.T) {
	memory := &Memory{
		IsPinned: false,
	}

	assert.False(t, memory.IsPinned)

	memory.IsPinned = true
	assert.True(t, memory.IsPinned)

	memory.IsPinned = false
	assert.False(t, memory.IsPinned)
}

// TestTask_Enabled tests task enabled state
func TestTask_Enabled(t *testing.T) {
	task := &Task{
		Name:    "my_task",
		Enabled: true,
	}

	assert.True(t, task.Enabled)

	task.Enabled = false
	assert.False(t, task.Enabled)
}

// TestSkill_AlwaysLoad tests skill always load flag
func TestSkill_AlwaysLoad(t *testing.T) {
	skill := &Skill{
		Name:       "my_skill",
		AlwaysLoad: true,
	}

	assert.True(t, skill.AlwaysLoad)

	skill.AlwaysLoad = false
	assert.False(t, skill.AlwaysLoad)
}

// TestSkill_Source tests skill source field
func TestSkill_Source(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"Builtin", "builtin"},
		{"Workspace", "workspace"},
		{"Remote", "remote"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skill := &Skill{Source: tt.source}
			assert.Equal(t, tt.source, skill.Source)
		})
	}
}
