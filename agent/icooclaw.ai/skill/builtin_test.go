package skill

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetBuiltInSkills(t *testing.T) {
	skills := GetBuiltInSkills()

	assert.NotEmpty(t, skills, "Built-in skills should not be empty")
	assert.Len(t, skills, 4, "Should have 4 built-in skills")

	skillNames := make(map[string]bool)
	for _, s := range skills {
		skillNames[s.Name] = true
	}

	assert.True(t, skillNames["identity"], "Should have identity skill")
	assert.True(t, skillNames["file"], "Should have file skill")
	assert.True(t, skillNames["shell"], "Should have shell skill")
	assert.True(t, skillNames["web"], "Should have web skill")
}

func TestBuiltInSkills_AlwaysLoad(t *testing.T) {
	skills := GetBuiltInSkills()

	alwaysLoadCount := 0
	for _, s := range skills {
		if s.AlwaysLoad {
			alwaysLoadCount++
		}
	}

	// identity, file, shell 都是 AlwaysLoad
	assert.Equal(t, 3, alwaysLoadCount, "3 skills should have AlwaysLoad=true")
}

func TestBuiltInSkills_Enabled(t *testing.T) {
	skills := GetBuiltInSkills()

	for _, s := range skills {
		assert.True(t, s.Enabled, "Skill %s should be enabled by default", s.Name)
	}
}

func TestBuiltInSkills_HaveContent(t *testing.T) {
	skills := GetBuiltInSkills()

	for _, s := range skills {
		assert.NotEmpty(t, s.Content, "Skill %s should have content", s.Name)
		assert.NotEmpty(t, s.Description, "Skill %s should have description", s.Name)
	}
}

func TestNewIdentitySkill(t *testing.T) {
	skill := newIdentitySkill()

	assert.Equal(t, "identity", skill.Name)
	assert.Equal(t, "身份设定技能", skill.Description)
	assert.True(t, skill.AlwaysLoad)
	assert.True(t, skill.Enabled)
	assert.Contains(t, skill.Content, "SOUL.md")
	assert.Contains(t, skill.Content, "USER.md")
}

func TestNewFileSkill(t *testing.T) {
	skill := newFileSkill()

	assert.Equal(t, "file", skill.Name)
	assert.Equal(t, "文件操作技能", skill.Description)
	assert.True(t, skill.AlwaysLoad)
	assert.True(t, skill.Enabled)
	assert.Contains(t, skill.Content, "file_read")
	assert.Contains(t, skill.Content, "file_write")
}

func TestNewShellSkill(t *testing.T) {
	skill := newShellSkill()

	assert.Equal(t, "shell", skill.Name)
	assert.Equal(t, "Shell命令执行技能", skill.Description)
	assert.True(t, skill.AlwaysLoad)
	assert.True(t, skill.Enabled)
	assert.Contains(t, skill.Content, "shell_exec")
}

func TestNewWebSkill(t *testing.T) {
	skill := newWebSkill()

	assert.Equal(t, "web", skill.Name)
	assert.Equal(t, "网页搜索和抓取技能", skill.Description)
	assert.False(t, skill.AlwaysLoad)
	assert.True(t, skill.Enabled)
	assert.Contains(t, skill.Content, "web_search")
	assert.Contains(t, skill.Content, "web_fetch")
}

func TestSkill_Structure(t *testing.T) {
	s := Skill{
		Name:        "test_skill",
		Description: "Test description",
		Content:     "Test content",
		AlwaysLoad:  true,
		Enabled:     true,
	}

	assert.Equal(t, "test_skill", s.Name)
	assert.Equal(t, "Test description", s.Description)
	assert.Equal(t, "Test content", s.Content)
	assert.True(t, s.AlwaysLoad)
	assert.True(t, s.Enabled)
}