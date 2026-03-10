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
	Agent    AgentConfig    `mapstructure:"agent"`
	Database DatabaseConfig `mapstructure:"database"`
	Gateway  GatewayConfig  `mapstructure:"gateway"`
	Logging  LoggingConfig  `mapstructure:"logging"`
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
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal into config struct
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// setDefaults sets default values in viper.
func setDefaults(v *viper.Viper, cfg *Config) {
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
		return fmt.Errorf("agent.workspace is required")
	}
	if c.Database.Path == "" {
		return fmt.Errorf("database.path is required")
	}
	if c.Gateway.Enabled && (c.Gateway.Port <= 0 || c.Gateway.Port > 65535) {
		return fmt.Errorf("gateway.port must be between 1 and 65535")
	}
	return nil
}

// EnsureWorkspace ensures the workspace directory exists.
func (c *Config) EnsureWorkspace() error {
	if err := os.MkdirAll(c.Agent.Workspace, 0755); err != nil {
		return fmt.Errorf("failed to create workspace directory: %w", err)
	}
	return nil
}

// EnsureDatabasePath ensures the database directory exists.
func (c *Config) EnsureDatabasePath() error {
	dir := filepath.Dir(c.Database.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
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
