// Package providers provides LLM provider abstraction for icooclaw.
package providers

import (
	"fmt"
	"log/slog"
	"sync"

	"icooclaw/pkg/consts"
	"icooclaw/pkg/storage"
)

// ProviderFactory creates a provider from configuration.
type ProviderFactory func(cfg *storage.Provider) Provider

// Registry manages provider factories and instances.
type Registry struct {
	factories map[consts.ProviderType]ProviderFactory
	providers map[string]Provider
	logger    *slog.Logger
	mu        sync.RWMutex
}

// NewRegistry creates a new provider registry.
func NewRegistry(logger *slog.Logger) *Registry {
	if logger == nil {
		logger = slog.Default()
	}
	r := &Registry{
		factories: make(map[consts.ProviderType]ProviderFactory),
		providers: make(map[string]Provider),
		logger:    logger,
	}

	// Register built-in providers
	r.RegisterBuiltins()

	return r
}

// RegisterBuiltins registers all built-in provider factories.
func (r *Registry) RegisterBuiltins() {
	r.RegisterFactory(consts.ProviderOpenAI, NewOpenAIProvider)
	r.RegisterFactory(consts.ProviderAnthropic, NewAnthropicProvider)
	r.RegisterFactory(consts.ProviderDeepSeek, NewDeepSeekProvider)
	r.RegisterFactory(consts.ProviderOpenRouter, NewOpenRouterProvider)
	r.RegisterFactory(consts.ProviderGemini, NewGeminiProvider)
	r.RegisterFactory(consts.ProviderMistral, NewMistralProvider)
	r.RegisterFactory(consts.ProviderGroq, NewGroqProvider)
	r.RegisterFactory(consts.ProviderAzure, NewAzureOpenAIProvider)
	r.RegisterFactory(consts.ProviderOllama, NewOllamaProvider)
	r.RegisterFactory(consts.ProviderMoonshot, NewMoonshotProvider)
	r.RegisterFactory(consts.ProviderZhipu, NewZhipuProvider)
	r.RegisterFactory(consts.ProviderQwen, NewQwenProvider)
	r.RegisterFactory(consts.ProviderSiliconFlow, NewSiliconFlowProvider)
	r.RegisterFactory(consts.ProviderGrok, NewGrokProvider)
}

// RegisterFactory registers a provider factory.
func (r *Registry) RegisterFactory(providerType consts.ProviderType, factory ProviderFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.factories[providerType] = factory
	r.logger.Debug("provider factory registered", "type", providerType)
}

// CreateProvider creates a provider from configuration.
func (r *Registry) CreateProvider(cfg *storage.Provider) (Provider, error) {
	r.mu.RLock()
	factory, ok := r.factories[consts.ProviderType(cfg.Type)]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown provider type: %s", cfg.Type)
	}

	return factory(cfg), nil
}

// Register registers a provider instance.
func (r *Registry) Register(name string, provider Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.providers[name] = provider
	r.logger.Debug("provider registered", "name", name, "type", provider.GetName())
}

// Get gets a provider by name.
func (r *Registry) Get(name string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", name)
	}

	return provider, nil
}

// List lists all registered providers.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// LoadFromConfig loads providers from configuration.
func (r *Registry) LoadFromConfig(configs []*storage.Provider) error {
	for _, cfg := range configs {
		provider, err := r.CreateProvider(cfg)
		if err != nil {
			r.logger.Warn("failed to create provider", "name", cfg.Name, "error", err)
			continue
		}

		r.Register(cfg.Name, provider)
	}

	return nil
}

// ProviderInfo contains information about a provider.
type ProviderInfo struct {
	Name   string              `json:"name"`
	Type   consts.ProviderType `json:"type"`
	Model  string              `json:"model"`
	Models []string            `json:"models,omitempty"`
}

// GetInfo returns information about a provider.
func (r *Registry) GetInfo(name string) (*ProviderInfo, error) {
	provider, err := r.Get(name)
	if err != nil {
		return nil, err
	}

	return &ProviderInfo{
		Name:  name,
		Type:  consts.ProviderType(provider.GetName()),
		Model: provider.GetModel(),
	}, nil
}

// ListInfo returns information about all providers.
func (r *Registry) ListInfo() []*ProviderInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]*ProviderInfo, 0, len(r.providers))
	for name, provider := range r.providers {
		infos = append(infos, &ProviderInfo{
			Name:  name,
			Type:  consts.ProviderType(provider.GetName()),
			Model: provider.GetModel(),
		})
	}

	return infos
}

// Global registry instance
var globalRegistry *Registry
var registryOnce sync.Once

// GetRegistry returns the global registry instance.
func GetRegistry(logger *slog.Logger) *Registry {
	registryOnce.Do(func() {
		globalRegistry = NewRegistry(logger)
	})
	return globalRegistry
}
