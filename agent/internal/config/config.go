package config

import (
	"log/slog"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config 全局配置
type Config struct {
	Providers ProvidersConfig `mapstructure:"providers"`
	Channels  ChannelsConfig  `mapstructure:"channels"`
	Agents    AgentsConfig    `mapstructure:"agents"`
	Tools     ToolsConfig     `mapstructure:"tools"`
	MCP       MCPConfig       `mapstructure:"mcp"`
	Memory    MemoryConfig    `mapstructure:"memory"`
	Workspace string          `mapstructure:"workspace"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Log       LogConfig       `mapstructure:"log"`
	Security  SecurityConfig  `mapstructure:"security"`
	Scheduler SchedulerConfig `mapstructure:"scheduler"`
}

// ProvidersConfig Provider配置
type ProvidersConfig struct {
	OpenRouter  ProviderSettings     `mapstructure:"openrouter"`
	OpenAI      ProviderSettings     `mapstructure:"openai"`
	Anthropic   ProviderSettings     `mapstructure:"anthropic"`
	DeepSeek    ProviderSettings     `mapstructure:"deepseek"`
	Custom      ProviderSettings     `mapstructure:"custom"`
	Ollama      OllamaSettings       `mapstructure:"ollama"`
	AzureOpenAI AzureSettings        `mapstructure:"azure_openai"`
	LocalAI     ProviderSettings     `mapstructure:"localai"`
	OneAPI      ProviderSettings     `mapstructure:"oneapi"`
	Compatible  []CompatibleSettings `mapstructure:"compatible"` // 支持多个 OpenAI 兼容的 LLM
}

// ProviderSettings 单个Provider设置
type ProviderSettings struct {
	Enabled bool   `mapstructure:"enabled"`
	APIKey  string `mapstructure:"api_key"`
	APIBase string `mapstructure:"api_base"`
	Model   string `mapstructure:"model"`
}

// OllamaSettings Ollama 设置
type OllamaSettings struct {
	Enabled bool   `mapstructure:"enabled"`
	APIBase string `mapstructure:"api_base"` // 如 http://localhost:11434
	Model   string `mapstructure:"model"`    // 如 llama2, codellama 等
}

// AzureSettings Azure OpenAI 设置
type AzureSettings struct {
	Enabled    bool   `mapstructure:"enabled"`
	APIKey     string `mapstructure:"api_key"`
	Endpoint   string `mapstructure:"endpoint"`    // 如 https://xxx.openai.azure.com
	Deployment string `mapstructure:"deployment"`  // 部署名称
	APIVersion string `mapstructure:"api_version"` // 如 2024-02-15-preview
}

// CompatibleSettings OpenAI 兼容 LLM 设置（支持额外请求头）
type CompatibleSettings struct {
	Enabled bool              `mapstructure:"enabled"`
	Name    string            `mapstructure:"name"` // Provider 名称
	APIKey  string            `mapstructure:"api_key"`
	APIBase string            `mapstructure:"api_base"` // 如 http://localhost:8000/v1
	Model   string            `mapstructure:"model"`    // 默认模型
	Headers map[string]string `mapstructure:"headers"`  // 额外请求头
}

// ChannelsConfig 通道配置
type ChannelsConfig struct {
	Telegram  ChannelSettings `mapstructure:"telegram"`
	Discord   ChannelSettings `mapstructure:"discord"`
	Feishu    ChannelSettings `mapstructure:"feishu"`
	DingTalk  ChannelSettings `mapstructure:"dingtalk"`
	Slack     ChannelSettings `mapstructure:"slack"`
	Email     ChannelSettings `mapstructure:"email"`
	WebSocket ChannelSettings `mapstructure:"websocket"`
	Webhook   ChannelSettings `mapstructure:"webhook"`
}

// ChannelSettings 通道设置
type ChannelSettings struct {
	Enabled      bool              `mapstructure:"enabled"`
	Token        string            `mapstructure:"token"`
	AppID        string            `mapstructure:"app_id"`
	AppSecret    string            `mapstructure:"app_secret"`
	ClientID     string            `mapstructure:"client_id"`
	ClientSecret string            `mapstructure:"client_secret"`
	BotToken     string            `mapstructure:"bot_token"`
	AppToken     string            `mapstructure:"app_token"`
	IMAPHost     string            `mapstructure:"imap_host"`
	IMAPPort     int               `mapstructure:"imap_port"`
	SMTPHost     string            `mapstructure:"smtp_host"`
	SMTPPort     int               `mapstructure:"smtp_port"`
	Address      string            `mapstructure:"address"`
	Host         string            `mapstructure:"host"`
	Port         int               `mapstructure:"port"`
	Extra        map[string]string `mapstructure:"-"`
}

// AgentsConfig Agent配置
type AgentsConfig struct {
	Defaults        AgentSettings `mapstructure:"defaults"`
	DefaultProvider string        `mapstructure:"default_provider"`
}

// AgentSettings Agent设置
type AgentSettings struct {
	Name         string  `mapstructure:"name"`
	Model        string  `mapstructure:"model"`
	Temperature  float64 `mapstructure:"temperature"`
	MaxTokens    int     `mapstructure:"max_tokens"`
	MemoryWindow int     `mapstructure:"memory_window"`
	SystemPrompt string  `mapstructure:"system_prompt"`
}

// ToolsConfig 工具配置
type ToolsConfig struct {
	MCPServers      map[string]MCPServerConfig `mapstructure:"mcp_servers"`
	Enabled         bool                       `mapstructure:"enabled"`
	AllowFileRead   bool                       `mapstructure:"allow_file_read"`
	AllowFileWrite  bool                       `mapstructure:"allow_file_write"`
	AllowFileEdit   bool                       `mapstructure:"allow_file_edit"`
	AllowFileDelete bool                       `mapstructure:"allow_file_delete"`
	AllowExec       bool                       `mapstructure:"allow_exec"`
	Workspace       string                     `mapstructure:"workspace"`
	ExecTimeout     int                        `mapstructure:"exec_timeout"`
	JS              JSToolsConfig              `mapstructure:"js"`
}

// JSToolsConfig JavaScript 工具配置
type JSToolsConfig struct {
	Enabled     bool                `mapstructure:"enabled"`
	ToolsDir    string              `mapstructure:"tools_dir"`
	MaxMemory   int64               `mapstructure:"max_memory"`
	Timeout     int                 `mapstructure:"timeout"`
	Permissions JSPermissionsConfig `mapstructure:"permissions"`
}

// JSPermissionsConfig JS 工具权限配置
type JSPermissionsConfig struct {
	FileRead       bool     `mapstructure:"file_read"`
	FileWrite      bool     `mapstructure:"file_write"`
	FileDelete     bool     `mapstructure:"file_delete"`
	Network        bool     `mapstructure:"network"`
	Exec           bool     `mapstructure:"exec"`
	HTTPTimeout    int      `mapstructure:"http_timeout"`
	ExecTimeout    int      `mapstructure:"exec_timeout"`
	AllowedDomains []string `mapstructure:"allowed_domains"`
}

// MCPServerConfig MCP服务器配置
type MCPServerConfig struct {
	Command     string            `mapstructure:"command"`
	Args        []string          `mapstructure:"args"`
	Env         map[string]string `mapstructure:"env"`
	Transport   string            `mapstructure:"transport"` // stdio or http
	URL         string            `mapstructure:"url"`
	AuthHeaders map[string]string `mapstructure:"auth_headers"`
	Timeout     time.Duration     `mapstructure:"timeout"`
}

// MCPConfig MCP 配置
type MCPConfig struct {
	Enabled bool                       `mapstructure:"enabled"`
	Servers map[string]MCPServerConfig `mapstructure:"servers"`
}

// MemoryConfig 内存配置
type MemoryConfig struct {
	ConsolidationThreshold int  `mapstructure:"consolidation_threshold"` // 消息数达到此值时整合
	SummaryEnabled         bool `mapstructure:"summary_enabled"`
	AutoSave               bool `mapstructure:"auto_save"`
	MaxMemoryAge           int  `mapstructure:"max_memory_age"` // 天数
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	RestrictToWorkspace bool `mapstructure:"restrict_to_workspace"`
}

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	Enabled           bool `mapstructure:"enabled"`
	HeartbeatInterval int  `mapstructure:"heartbeat_interval"` // 分钟
}

// Load 加载配置文件
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("$HOME/.icooclaw")
	viper.SetDefault("database.path", "./data/icooclaw.db")
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "text")
	viper.SetDefault("workspace", "$HOME/.icooclaw/workspace")
	viper.SetDefault("agents.default_provider", "openrouter")
	viper.SetDefault("scheduler.enabled", false)
	viper.SetDefault("scheduler.heartbeat_interval", 30)
	viper.SetDefault("mcp.enabled", false)
	viper.SetDefault("memory.consolidation_threshold", 50)
	viper.SetDefault("memory.summary_enabled", true)
	viper.SetDefault("memory.auto_save", false)
	viper.SetDefault("memory.max_memory_age", 30)
	viper.SetDefault("tools.allow_exec", false)
	viper.SetDefault("tools.exec_timeout", 30)
	viper.SetDefault("tools.js.enabled", true)
	viper.SetDefault("tools.js.tools_dir", "tools")
	viper.SetDefault("tools.js.max_memory", 10485760)
	viper.SetDefault("tools.js.timeout", 30)
	viper.SetDefault("tools.js.permissions.file_read", false)
	viper.SetDefault("tools.js.permissions.file_write", false)
	viper.SetDefault("tools.js.permissions.file_delete", false)
	viper.SetDefault("tools.js.permissions.network", false)
	viper.SetDefault("tools.js.permissions.exec", false)
	viper.SetDefault("tools.js.permissions.http_timeout", 30)
	viper.SetDefault("tools.js.permissions.exec_timeout", 30)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// 配置文件不存在，使用默认配置
			slog.Info("Config file not found, using defaults")
		} else {
			return nil, err
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// 处理环境变量
	viper.AutomaticEnv()

	// 解析 extra 字段
	if telegramCfg, ok := viper.Get("channels.telegram").(map[string]interface{}); ok {
		cfg.Channels.Telegram.Extra = extractExtra(telegramCfg)
	}

	return &cfg, nil
}

// extractExtra 从配置中提取额外字段
func extractExtra(m map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		if k != "enabled" && k != "token" {
			if s, ok := v.(string); ok {
				result[k] = s
			}
		}
	}
	return result
}

// InitLogger 初始化日志系统
func InitLogger(level string, format string) *slog.Logger {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		AddSource: true,
	}

	switch level {
	case "debug":
		opts.Level = slog.LevelDebug
	case "info":
		opts.Level = slog.LevelInfo
	case "warn":
		opts.Level = slog.LevelWarn
	case "error":
		opts.Level = slog.LevelError
	default:
		opts.Level = slog.LevelInfo
	}

	if format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
