package config

import (
	"os"
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
			logger := InitLogger(tt.level, tt.format)
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
