package skill

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLoader(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	loader := NewLoader(nil, logger)

	assert.NotNil(t, loader)
	assert.NotNil(t, loader.loaded)
	assert.NotNil(t, loader.alwaysLoad)
}

func TestLoader_GetLoaded_Empty(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	loader := NewLoader(nil, logger)

	// 初始状态应该为空
	loaded := loader.GetLoaded()
	assert.Empty(t, loaded)
}

func TestLoader_GetAlwaysLoad_Empty(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	loader := NewLoader(nil, logger)

	alwaysLoad := loader.GetAlwaysLoad()
	assert.Empty(t, alwaysLoad)
}

func TestLoader_GetByName_NotFound(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	loader := NewLoader(nil, logger)

	skill := loader.GetByName("nonexistent")
	assert.Nil(t, skill)
}

func TestLoader_Load_WithoutStorage(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	loader := NewLoader(nil, logger)

	// Load 在 storage 为 nil 时会调用 GetEnabledSkills 导致 panic
	// 我们只测试 loadBuiltInSkills 的效果
	loader.loadBuiltInSkills()

	loaded := loader.GetLoaded()
	assert.NotEmpty(t, loaded, "Should have loaded built-in skills")
	assert.Len(t, loaded, 4, "Should have 4 built-in skills")
}

func TestLoader_GetByName_AfterLoad(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	loader := NewLoader(nil, logger)

	// 直接加载内置技能，避免调用 Load 方法
	loader.loadBuiltInSkills()

	// 测试查找内置技能
	identitySkill := loader.GetByName("identity")
	assert.NotNil(t, identitySkill)
	assert.Equal(t, "identity", identitySkill.Name)

	fileSkill := loader.GetByName("file")
	assert.NotNil(t, fileSkill)
	assert.Equal(t, "file", fileSkill.Name)

	// 测试大小写不敏感
	shellSkill := loader.GetByName("SHELL")
	assert.NotNil(t, shellSkill)
	assert.Equal(t, "shell", shellSkill.Name)
}

func TestLoader_GetAlwaysLoad_AfterLoad(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	loader := NewLoader(nil, logger)

	// 直接加载内置技能
	loader.loadBuiltInSkills()

	alwaysLoad := loader.GetAlwaysLoad()
	assert.NotEmpty(t, alwaysLoad, "Should have always-load skills")

	// 检查 AlwaysLoad 的技能
	alwaysLoadNames := make(map[string]bool)
	for _, s := range alwaysLoad {
		alwaysLoadNames[s.Name] = true
	}

	assert.True(t, alwaysLoadNames["identity"], "identity should be in alwaysLoad")
	assert.True(t, alwaysLoadNames["file"], "file should be in alwaysLoad")
	assert.True(t, alwaysLoadNames["shell"], "shell should be in alwaysLoad")
	assert.False(t, alwaysLoadNames["web"], "web should not be in alwaysLoad")
}

func TestLoader_Reload(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	loader := NewLoader(nil, logger)

	// 第一次加载内置技能
	loader.loadBuiltInSkills()
	assert.Len(t, loader.GetLoaded(), 4)

	// 清空并重新加载
	loader.loaded = make([]Skill, 0)
	loader.alwaysLoad = make([]Skill, 0)
	loader.loadBuiltInSkills()
	assert.Len(t, loader.GetLoaded(), 4)
}