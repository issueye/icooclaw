// Package config provides configuration management for icooclaw.
package config

import (
	"fmt"
	"icooclaw/pkg/consts"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config represents the main configuration structure.
// Only basic configuration is stored in config file.
// Dynamic configuration is stored in SQLite database.
type Config struct {
	Mode     string         `mapstructure:"mode"`     // 模式 debug 或 release
	Agent    AgentConfig    `mapstructure:"agent"`    // 基本智能体配置
	Database DatabaseConfig `mapstructure:"database"` // 数据库配置
	Gateway  GatewayConfig  `mapstructure:"gateway"`  // 网关配置
	Logging  LoggingConfig  `mapstructure:"logging"`  // 日志配置
	Channels ChannelsConfig `mapstructure:"channels"` // 渠道配置
}

// AgentConfig contains basic agent configuration.
type AgentConfig struct {
	Workspace       string              `mapstructure:"workspace"`
	DefaultModel    string              `mapstructure:"default_model"`
	DefaultProvider consts.ProviderType `mapstructure:"default_provider"`
}

// DatabaseConfig contains database configuration.
type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

// GatewayConfig contains HTTP gateway configuration.
type GatewayConfig struct {
	Enabled bool `mapstructure:"enabled"`
	Port    int  `mapstructure:"port"`
}

// LoggingConfig contains logging configuration.
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// ChannelsConfig contains channel-specific configurations.
type ChannelsConfig struct {
	Feishu   FeishuConfig   `mapstructure:"feishu"`
	DingTalk DingTalkConfig `mapstructure:"dingtalk"`
}

// FeishuConfig contains Feishu/Lark channel configuration.
type FeishuConfig struct {
	Enabled           bool     `mapstructure:"enabled"`
	AppID             string   `mapstructure:"app_id"`
	AppSecret         string   `mapstructure:"app_secret"`
	EncryptKey        string   `mapstructure:"encrypt_key"`
	VerificationToken string   `mapstructure:"verification_token"`
	AllowFrom         []string `mapstructure:"allow_from"`
	ReasoningChatID   string   `mapstructure:"reasoning_chat_id"`
}

// DingTalkConfig contains DingTalk channel configuration.
type DingTalkConfig struct {
	Enabled         bool     `mapstructure:"enabled"`
	ClientID        string   `mapstructure:"client_id"`
	ClientSecret    string   `mapstructure:"client_secret"`
	AgentID         int64    `mapstructure:"agent_id"`
	AllowFrom       []string `mapstructure:"allow_from"`
	ReasoningChatID string   `mapstructure:"reasoning_chat_id"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Agent: AgentConfig{
			Workspace:       "./workspace",
			DefaultModel:    "gpt-4",
			DefaultProvider: consts.ProviderQwen,
		},
		Database: DatabaseConfig{
			Path: "./data/icooclaw.db",
		},
		Gateway: GatewayConfig{
			Enabled: true,
			Port:    8080,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}
}

// Load loads configuration from file and environment variables.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path == "" {
		path = "config.toml"
	}

	// Check if config file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Use default config if file doesn't exist
		return cfg, nil
	}

	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("toml")

	// Set default values
	setDefaults(v, cfg)

	// Enable environment variable override
	v.SetEnvPrefix("ICOOCLAW")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// Unmarshal into config struct
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return cfg, nil
}

// setDefaults sets default values in viper.
func setDefaults(v *viper.Viper, cfg *Config) {
	v.SetDefault("mode", cfg.Mode)
	v.SetDefault("agent.workspace", cfg.Agent.Workspace)
	v.SetDefault("agent.default_model", cfg.Agent.DefaultModel)
	v.SetDefault("agent.default_provider", cfg.Agent.DefaultProvider)
	v.SetDefault("database.path", cfg.Database.Path)
	v.SetDefault("gateway.enabled", cfg.Gateway.Enabled)
	v.SetDefault("gateway.port", cfg.Gateway.Port)
	v.SetDefault("logging.level", cfg.Logging.Level)
	v.SetDefault("logging.format", cfg.Logging.Format)
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.Agent.Workspace == "" {
		return fmt.Errorf("agent.workspace 是必需的")
	}
	if c.Database.Path == "" {
		return fmt.Errorf("database.path 是必需的")
	}
	if c.Gateway.Enabled && (c.Gateway.Port <= 0 || c.Gateway.Port > 65535) {
		return fmt.Errorf("gateway.port 必须在 1 到 65535 之间")
	}
	return nil
}

// EnsureWorkspace ensures the workspace directory exists.
func (c *Config) EnsureWorkspace() error {
	if err := os.MkdirAll(c.Agent.Workspace, 0755); err != nil {
		return fmt.Errorf("创建工作目录失败: %w", err)
	}
	return nil
}

// EnsureDatabasePath ensures the database directory exists.
func (c *Config) EnsureDatabasePath() error {
	dir := filepath.Dir(c.Database.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建数据库目录失败:%w", err)
	}
	return nil
}

// GetWorkspacePath returns the absolute path to the workspace.
func (c *Config) GetWorkspacePath() (string, error) {
	return filepath.Abs(c.Agent.Workspace)
}

// GetDatabasePath returns the absolute path to the database.
func (c *Config) GetDatabasePath() (string, error) {
	return filepath.Abs(c.Database.Path)
}
