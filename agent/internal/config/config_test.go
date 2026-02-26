package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfig_Structure tests for Config struct
func TestConfig_Structure(t *testing.T) {
	cfg := Config{
		Providers: ProvidersConfig{
			OpenRouter: ProviderSettings{
				Enabled: true,
				APIKey:  "test_key",
				APIBase: "https://api.openrouter.ai",
				Model:   "gpt-4",
			},
		},
		Channels:  ChannelsConfig{},
		Agents:    AgentsConfig{},
		Tools:     ToolsConfig{Enabled: true},
		MCP:       MCPConfig{Enabled: false},
		Memory:    MemoryConfig{ConsolidationThreshold: 50},
		Workspace: "/home/user/workspace",
		Database:  DatabaseConfig{Path: "./data/test.db"},
		Log:       LogConfig{Level: "debug", Format: "json"},
		Security:  SecurityConfig{RestrictToWorkspace: true},
		Scheduler: SchedulerConfig{Enabled: true, HeartbeatInterval: 30},
	}

	assert.True(t, cfg.Tools.Enabled)
	assert.Equal(t, 50, cfg.Memory.ConsolidationThreshold)
	assert.Equal(t, "/home/user/workspace", cfg.Workspace)
	assert.Equal(t, "debug", cfg.Log.Level)
}

// TestProvidersConfig tests for ProvidersConfig struct
func TestProvidersConfig_Structure(t *testing.T) {
	cfg := ProvidersConfig{
		OpenRouter: ProviderSettings{
			Enabled: true,
			APIKey:  "key1",
			APIBase: "https://api.openrouter.ai",
			Model:   "model1",
		},
		OpenAI: ProviderSettings{
			Enabled: false,
			APIKey:  "key2",
			Model:   "model2",
		},
		Anthropic: ProviderSettings{
			Enabled: true,
			APIKey:  "key3",
			Model:   "claude-3",
		},
	}

	assert.True(t, cfg.OpenRouter.Enabled)
	assert.False(t, cfg.OpenAI.Enabled)
	assert.True(t, cfg.Anthropic.Enabled)
}

// TestProviderSettings tests for ProviderSettings struct
func TestProviderSettings_Structure(t *testing.T) {
	settings := ProviderSettings{
		Enabled: true,
		APIKey:  "secret_key",
		APIBase: "https://api.example.com",
		Model:   "gpt-4-turbo",
	}

	assert.True(t, settings.Enabled)
	assert.Equal(t, "secret_key", settings.APIKey)
	assert.Equal(t, "https://api.example.com", settings.APIBase)
	assert.Equal(t, "gpt-4-turbo", settings.Model)
}

// TestChannelsConfig tests for ChannelsConfig struct
func TestChannelsConfig_Structure(t *testing.T) {
	cfg := ChannelsConfig{
		Telegram: ChannelSettings{
			Enabled: true,
			Token:   "test_token",
		},
		Discord: ChannelSettings{
			Enabled: false,
		},
		WebSocket: ChannelSettings{
			Enabled: true,
			Host:    "localhost",
			Port:    8080,
		},
	}

	assert.True(t, cfg.Telegram.Enabled)
	assert.False(t, cfg.Discord.Enabled)
	assert.True(t, cfg.WebSocket.Enabled)
	assert.Equal(t, "localhost", cfg.WebSocket.Host)
	assert.Equal(t, 8080, cfg.WebSocket.Port)
}

// TestChannelSettings tests for ChannelSettings struct
func TestChannelSettings_Structure(t *testing.T) {
	settings := ChannelSettings{
		Enabled:      true,
		Token:        "bot_token",
		AppID:        "app_id",
		AppSecret:    "app_secret",
		ClientID:     "client_id",
		ClientSecret: "client_secret",
		BotToken:     "bot_token",
		IMAPHost:     "imap.example.com",
		IMAPPort:     993,
		SMTPHost:     "smtp.example.com",
		SMTPPort:     465,
		Address:      "user@example.com",
		Host:         "localhost",
		Port:         3000,
		Extra:        map[string]string{"key": "value"},
	}

	assert.True(t, settings.Enabled)
	assert.Equal(t, "bot_token", settings.Token)
	assert.Equal(t, "app_id", settings.AppID)
	assert.Equal(t, 993, settings.IMAPPort)
	assert.Equal(t, "localhost", settings.Host)
	assert.Equal(t, 3000, settings.Port)
}

// TestAgentsConfig tests for AgentsConfig struct
func TestAgentsConfig_Structure(t *testing.T) {
	cfg := AgentsConfig{
		Defaults: AgentSettings{
			Name:         "default_agent",
			Model:        "gpt-4",
			Temperature:  0.7,
			MaxTokens:    2000,
			MemoryWindow: 10,
			SystemPrompt: "You are a helpful assistant.",
		},
		DefaultProvider: "openrouter",
	}

	assert.Equal(t, "default_agent", cfg.Defaults.Name)
	assert.Equal(t, 0.7, cfg.Defaults.Temperature)
	assert.Equal(t, "openrouter", cfg.DefaultProvider)
}

// TestAgentSettings tests for AgentSettings struct
func TestAgentSettings_Structure(t *testing.T) {
	settings := AgentSettings{
		Name:         "my_agent",
		Model:        "claude-3",
		Temperature:  1.0,
		MaxTokens:    4000,
		MemoryWindow: 20,
		SystemPrompt: "You are a coding assistant.",
	}

	assert.Equal(t, "my_agent", settings.Name)
	assert.Equal(t, "claude-3", settings.Model)
	assert.Equal(t, 1.0, settings.Temperature)
	assert.Equal(t, 4000, settings.MaxTokens)
	assert.Equal(t, 20, settings.MemoryWindow)
}

// TestToolsConfig tests for ToolsConfig struct
func TestToolsConfig_Structure(t *testing.T) {
	cfg := ToolsConfig{
		MCPServers: map[string]MCPServerConfig{
			"server1": {
				Command:   "npx",
				Args:      []string{"-y", "server-package"},
				Transport: "stdio",
			},
		},
		Enabled:         true,
		AllowFileRead:   true,
		AllowFileWrite:  true,
		AllowFileEdit:   false,
		AllowFileDelete: false,
		AllowExec:       false,
		Workspace:       "/home/user/workspace",
		ExecTimeout:     30,
	}

	assert.True(t, cfg.Enabled)
	assert.True(t, cfg.AllowFileRead)
	assert.False(t, cfg.AllowFileEdit)
	assert.Equal(t, 30, cfg.ExecTimeout)
}

// TestMCPServerConfig tests for MCPServerConfig struct
func TestMCPServerConfig_Structure(t *testing.T) {
	cfg := MCPServerConfig{
		Command:     "node",
		Args:        []string{"server.js"},
		Env:         map[string]string{"KEY": "value"},
		Transport:   "http",
		URL:         "http://localhost:3000",
		AuthHeaders: map[string]string{"Authorization": "Bearer token"},
		Timeout:     30,
	}

	assert.Equal(t, "node", cfg.Command)
	assert.Equal(t, []string{"server.js"}, cfg.Args)
	assert.Equal(t, "http", cfg.Transport)
	assert.Equal(t, "http://localhost:3000", cfg.URL)
}

// TestMCPConfig tests for MCPConfig struct
func TestMCPConfig_Structure(t *testing.T) {
	cfg := MCPConfig{
		Enabled: true,
		Servers: map[string]MCPServerConfig{
			"test": {
				Command:   "test_cmd",
				Transport: "stdio",
			},
		},
	}

	assert.True(t, cfg.Enabled)
	assert.Len(t, cfg.Servers, 1)
}

// TestMemoryConfig tests for MemoryConfig struct
func TestMemoryConfig_Structure(t *testing.T) {
	cfg := MemoryConfig{
		ConsolidationThreshold: 100,
		SummaryEnabled:         true,
		AutoSave:               true,
		MaxMemoryAge:           60,
	}

	assert.Equal(t, 100, cfg.ConsolidationThreshold)
	assert.True(t, cfg.SummaryEnabled)
	assert.True(t, cfg.AutoSave)
	assert.Equal(t, 60, cfg.MaxMemoryAge)
}

// TestDatabaseConfig tests for DatabaseConfig struct
func TestDatabaseConfig_Structure(t *testing.T) {
	cfg := DatabaseConfig{
		Path: "./data/icooclaw.db",
	}

	assert.Equal(t, "./data/icooclaw.db", cfg.Path)
}

// TestLogConfig tests for LogConfig struct
func TestLogConfig_Structure(t *testing.T) {
	cfg := LogConfig{
		Level:  "debug",
		Format: "json",
	}

	assert.Equal(t, "debug", cfg.Level)
	assert.Equal(t, "json", cfg.Format)
}

// TestSecurityConfig tests for SecurityConfig struct
func TestSecurityConfig_Structure(t *testing.T) {
	cfg := SecurityConfig{
		RestrictToWorkspace: true,
	}

	assert.True(t, cfg.RestrictToWorkspace)
}

// TestSchedulerConfig tests for SchedulerConfig struct
func TestSchedulerConfig_Structure(t *testing.T) {
	cfg := SchedulerConfig{
		Enabled:           true,
		HeartbeatInterval: 15,
	}

	assert.True(t, cfg.Enabled)
	assert.Equal(t, 15, cfg.HeartbeatInterval)
}

// TestInitLogger tests for InitLogger
func TestInitLogger(t *testing.T) {
	tests := []struct {
		name   string
		level  string
		format string
	}{
		{"Debug JSON", "debug", "json"},
		{"Info Text", "info", "text"},
		{"Warn JSON", "warn", "json"},
		{"Error Text", "error", "text"},
		{"Default level", "invalid", "text"},
		{"Default format", "info", "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := InitLogger(tt.level, tt.format, "")
			require.NotNil(t, logger)
		})
	}
}

// TestExtractExtra tests for extractExtra function
func TestExtractExtra(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]string
	}{
		{
			name: "Basic",
			input: map[string]interface{}{
				"enabled": true,
				"token":   "secret",
				"extra1":  "value1",
				"extra2":  "value2",
			},
			expected: map[string]string{
				"extra1": "value1",
				"extra2": "value2",
			},
		},
		{
			name: "No extras",
			input: map[string]interface{}{
				"enabled": true,
				"token":   "secret",
			},
			expected: map[string]string{},
		},
		{
			name: "Host only",
			input: map[string]interface{}{
				"host": "localhost",
			},
			expected: map[string]string{
				"host": "localhost",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractExtra(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestLoad tests for Load function (with no config file)
func TestLoad_NoConfigFile(t *testing.T) {
	// Set environment variables that would affect config loading
	os.Unsetenv("ICOOCLAW_CONFIG")

	// This should return default config or error about missing config
	// We just verify it doesn't panic
	cfg, err := Load()
	if err == nil {
		require.NotNil(t, cfg)
		// Check some defaults
		assert.NotEmpty(t, cfg.Database.Path)
		assert.NotEmpty(t, cfg.Workspace)
	}
}

// TestExpandPath tests for expandPath function
func TestExpandPath(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldError bool
		setup       func()
		teardown    func()
	}{
		{
			name:        "Empty path",
			input:       "",
			shouldError: true,
		},
		{
			name:        "Relative path",
			input:       "./test/path",
			shouldError: false,
		},
		{
			name:        "Absolute path",
			input:       "/tmp/test/path",
			shouldError: false,
		},
		{
			name:  "Home path",
			input: "~/test/path",
			setup: func() {
				home := os.Getenv("HOME")
				if home == "" {
					os.Setenv("HOME", "/tmp/testhome")
				}
			},
			teardown: func() {
				os.Unsetenv("HOME")
			},
			shouldError: false,
		},
		{
			name:        "Environment variable path",
			input:       "$HOME/test",
			shouldError: false,
			setup: func() {
				os.Setenv("HOME", "/tmp/testhome")
			},
			teardown: func() {
				os.Unsetenv("HOME")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			defer func() {
				if tt.teardown != nil {
					tt.teardown()
				}
			}()

			result, err := expandPath(tt.input)
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, result)
				assert.IsType(t, "", result)
			}
		})
	}
}

// TestInitWorkspace tests for InitWorkspace function
func TestInitWorkspace(t *testing.T) {
	t.Run("Create new workspace", func(t *testing.T) {
		tmpDir := t.TempDir()
		workspacePath := filepath.Join(tmpDir, "workspace")

		err := InitWorkspace(workspacePath)
		require.NoError(t, err)

		_, err = os.Stat(workspacePath)
		assert.NoError(t, err)
		assert.True(t, isDir(workspacePath))
	})

	t.Run("Existing workspace", func(t *testing.T) {
		tmpDir := t.TempDir()
		workspacePath := filepath.Join(tmpDir, "existing_workspace")

		err := os.MkdirAll(workspacePath, 0755)
		require.NoError(t, err)

		err = InitWorkspace(workspacePath)
		require.NoError(t, err)
	})

	t.Run("With templates", func(t *testing.T) {
		tmpDir := t.TempDir()
		templateDir := filepath.Join(tmpDir, "templates")
		workspacePath := filepath.Join(tmpDir, "workspace")

		err := os.MkdirAll(filepath.Join(templateDir, "subdir"), 0755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(templateDir, "test.md"), []byte("test content"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(templateDir, "subdir", "nested.md"), []byte("nested content"), 0644)
		require.NoError(t, err)

		TemplatesDir = templateDir
		defer func() { TemplatesDir = "" }()

		err = InitWorkspace(workspacePath)
		require.NoError(t, err)

		_, err = os.Stat(filepath.Join(workspacePath, "test.md"))
		assert.NoError(t, err)
		_, err = os.Stat(filepath.Join(workspacePath, "subdir", "nested.md"))
		assert.NoError(t, err)
	})

	t.Run("Existing files not overwritten", func(t *testing.T) {
		tmpDir := t.TempDir()
		templateDir := filepath.Join(tmpDir, "templates")
		workspacePath := filepath.Join(tmpDir, "workspace")

		err := os.MkdirAll(templateDir, 0755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(templateDir, "test.md"), []byte("template content"), 0644)
		require.NoError(t, err)

		err = os.MkdirAll(workspacePath, 0755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(workspacePath, "test.md"), []byte("existing content"), 0644)
		require.NoError(t, err)

		TemplatesDir = templateDir
		defer func() { TemplatesDir = "" }()

		err = InitWorkspace(workspacePath)
		require.NoError(t, err)

		content, err := os.ReadFile(filepath.Join(workspacePath, "test.md"))
		require.NoError(t, err)
		assert.Equal(t, "existing content", string(content))
	})

	t.Run("Empty path", func(t *testing.T) {
		err := InitWorkspace("")
		assert.Error(t, err)
	})
}

// TestCopyTemplatesToWorkspace tests for copyTemplatesToWorkspace function
func TestCopyTemplatesToWorkspace(t *testing.T) {
	t.Run("No templates directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		TemplatesDir = ""
		defer func() { TemplatesDir = "" }()

		err := copyTemplatesToWorkspace(tmpDir)
		assert.NoError(t, err)
	})

	t.Run("Copy files successfully", func(t *testing.T) {
		tmpDir := t.TempDir()
		templateDir := filepath.Join(tmpDir, "templates")
		workspacePath := filepath.Join(tmpDir, "workspace")

		err := os.MkdirAll(templateDir, 0755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(templateDir, "file1.md"), []byte("content1"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(templateDir, "file2.md"), []byte("content2"), 0644)
		require.NoError(t, err)

		TemplatesDir = templateDir
		defer func() { TemplatesDir = "" }()

		err = copyTemplatesToWorkspace(workspacePath)
		require.NoError(t, err)

		content1, err := os.ReadFile(filepath.Join(workspacePath, "file1.md"))
		require.NoError(t, err)
		assert.Equal(t, "content1", string(content1))

		content2, err := os.ReadFile(filepath.Join(workspacePath, "file2.md"))
		require.NoError(t, err)
		assert.Equal(t, "content2", string(content2))
	})

	t.Run("Copy nested directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		templateDir := filepath.Join(tmpDir, "templates")
		workspacePath := filepath.Join(tmpDir, "workspace")

		err := os.MkdirAll(filepath.Join(templateDir, "level1", "level2"), 0755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(templateDir, "level1", "level2", "deep.md"), []byte("deep content"), 0644)
		require.NoError(t, err)

		TemplatesDir = templateDir
		defer func() { TemplatesDir = "" }()

		err = copyTemplatesToWorkspace(workspacePath)
		require.NoError(t, err)

		content, err := os.ReadFile(filepath.Join(workspacePath, "level1", "level2", "deep.md"))
		require.NoError(t, err)
		assert.Equal(t, "deep content", string(content))
	})
}

// TestCopyFile tests for copyFile function
func TestCopyFile(t *testing.T) {
	t.Run("Copy file successfully", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcPath := filepath.Join(tmpDir, "source.txt")
		dstPath := filepath.Join(tmpDir, "dest.txt")

		content := []byte("test file content")
		err := os.WriteFile(srcPath, content, 0644)
		require.NoError(t, err)

		err = copyFile(srcPath, dstPath)
		require.NoError(t, err)

		copiedContent, err := os.ReadFile(dstPath)
		require.NoError(t, err)
		assert.Equal(t, content, copiedContent)
	})

	t.Run("Source file not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcPath := filepath.Join(tmpDir, "nonexistent.txt")
		dstPath := filepath.Join(tmpDir, "dest.txt")

		err := copyFile(srcPath, dstPath)
		assert.Error(t, err)
	})

	t.Run("Overwrite destination", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcPath := filepath.Join(tmpDir, "source.txt")
		dstPath := filepath.Join(tmpDir, "dest.txt")

		err := os.WriteFile(srcPath, []byte("new content"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(dstPath, []byte("old content"), 0644)
		require.NoError(t, err)

		err = copyFile(srcPath, dstPath)
		require.NoError(t, err)

		content, err := os.ReadFile(dstPath)
		require.NoError(t, err)
		assert.Equal(t, "new content", string(content))
	})
}

// isDir checks if path is a directory
func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
