package provider

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/icooclaw/icooclaw/internal/config"
)

// ProviderSpec Provider规格定义
type ProviderSpec struct {
	Name            string
	Keywords        []string
	EnvKey          string
	DisplayName     string
	LiteLLMPrefix   string
	SkipPrefixes    []string
	APIBaseTemplate string
	DefaultModel    string
}

// Registry Provider注册表
type Registry struct {
	providers map[string]Provider
	logger    *slog.Logger
}

// NewRegistry 创建注册表
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
		logger:    slog.Default(),
	}
}

// Register 注册Provider
func (r *Registry) Register(name string, provider Provider) {
	r.providers[name] = provider
	r.logger.Info("Registered provider", "name", name)
}

// Get 获取Provider
func (r *Registry) Get(name string) (Provider, error) {
	provider, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", name)
	}
	return provider, nil
}

// List 列出所有Provider
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// Count 获取Provider数量
func (r *Registry) Count() int {
	return len(r.providers)
}

// RegisterFromConfig 从配置注册Provider
func (r *Registry) RegisterFromConfig(cfg config.ProvidersConfig) error {
	// 注册OpenRouter
	if cfg.OpenRouter.Enabled {
		r.Register("openrouter", NewOpenRouterProvider(cfg.OpenRouter.APIKey, cfg.OpenRouter.Model))
	}

	// 注册OpenAI
	if cfg.OpenAI.Enabled {
		r.Register("openai", NewOpenAIProvider(cfg.OpenAI.APIKey, cfg.OpenAI.Model))
	}

	// 注册Anthropic
	if cfg.Anthropic.Enabled {
		r.Register("anthropic", NewAnthropicProvider(cfg.Anthropic.APIKey, cfg.Anthropic.Model))
	}

	// 注册DeepSeek
	if cfg.DeepSeek.Enabled {
		r.Register("deepseek", NewDeepSeekProvider(cfg.DeepSeek.APIKey, cfg.DeepSeek.Model))
	}

	// 注册Custom
	if cfg.Custom.Enabled {
		r.Register("custom", NewCustomProvider(cfg.Custom.APIKey, cfg.Custom.APIBase, cfg.Custom.Model))
	}

	// 如果没有注册任何provider，至少注册一个默认的
	if len(r.providers) == 0 {
		return fmt.Errorf("no provider enabled")
	}

	return nil
}

// DetectProvider 检测Provider类型
func DetectProvider(apiBase string) string {
	apiBase = strings.ToLower(apiBase)
	if strings.Contains(apiBase, "openrouter") {
		return "openrouter"
	}
	if strings.Contains(apiBase, "openai") {
		return "openai"
	}
	if strings.Contains(apiBase, "anthropic") {
		return "anthropic"
	}
	if strings.Contains(apiBase, "deepseek") {
		return "deepseek"
	}
	return "custom"
}
