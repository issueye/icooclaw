// Package providers provides LLM provider abstraction for icooclaw.
package providers

import (
	"fmt"
	"sync"

	"icooclaw/pkg/consts"
	"icooclaw/pkg/storage"
)

// Factory creates Provider instances.
type Factory struct {
	storage   *storage.Storage
	providers map[string]Provider
	mu        sync.RWMutex
}

// NewFactory creates a new Factory.
func NewFactory(s *storage.Storage) *Factory {
	return &Factory{
		storage:   s,
		providers: make(map[string]Provider),
	}
}

// Register registers a provider instance.
func (f *Factory) Register(name string, p Provider) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.providers[name] = p
}

// Get gets a provider by name.
func (f *Factory) Get(name string) (Provider, error) {
	f.mu.RLock()
	if p, ok := f.providers[name]; ok {
		f.mu.RUnlock()
		return p, nil
	}
	f.mu.RUnlock()

	// Try to load from database
	cfg, err := f.storage.Provider().GetByName(name)
	if err != nil {
		return nil, fmt.Errorf("供应商 %s 未找到: %w", name, err)
	}

	p, err := f.createFromConfig(cfg)
	if err != nil {
		return nil, err
	}

	f.mu.Lock()
	f.providers[name] = p
	f.mu.Unlock()
	return p, nil
}

// List lists all provider names.
func (f *Factory) List() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	names := make([]string, 0, len(f.providers))
	for name := range f.providers {
		names = append(names, name)
	}
	return names
}

// createFromConfig creates a provider from configuration.
func (f *Factory) createFromConfig(cfg *storage.Provider) (Provider, error) {
	switch cfg.Type {
	case consts.ProviderOpenAI:
		return NewOpenAIProvider(cfg), nil
	case consts.ProviderAnthropic:
		return NewAnthropicProvider(cfg), nil
	case consts.ProviderDeepSeek:
		return NewDeepSeekProvider(cfg), nil
	case consts.ProviderOpenRouter:
		return NewOpenRouterProvider(cfg), nil
	case consts.ProviderQwen, consts.ProviderQwenCodingPlan:
		return NewQwenProvider(cfg), nil
	default:
		return nil, fmt.Errorf("未支持的供应商类型: %s", cfg.Type)
	}
}
