// Package providers provides LLM provider abstraction for icooclaw.
package providers

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"errors"
	icooclawErrors "icooclaw/pkg/errors"
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
		return nil, fmt.Errorf("provider %s not found: %w", name, err)
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

// GetByModel gets a provider by model name.
func (f *Factory) GetByModel(model string) (Provider, error) {
	// Check if model has provider prefix (e.g., "openrouter/xxx")
	if idx := strings.Index(model, "/"); idx > 0 {
		providerName := model[:idx]
		return f.Get(providerName)
	}

	// Search providers for the model
	f.mu.RLock()
	for _, p := range f.providers {
		if p.GetDefaultModel() == model {
			f.mu.RUnlock()
			return p, nil
		}
	}
	f.mu.RUnlock()

	// Search in database
	providers, err := f.storage.Provider().List()
	if err != nil {
		return nil, err
	}

	for _, cfg := range providers {
		if cfg.DefaultModel == model {
			return f.Get(cfg.Name)
		}
		for _, m := range cfg.Models {
			if m == model {
				return f.Get(cfg.Name)
			}
		}
	}

	return nil, fmt.Errorf("no provider found for model %s", model)
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

// CreateChain creates a fallback chain from provider names.
func (f *Factory) CreateChain(names ...string) (*FallbackChain, error) {
	providers := make([]Provider, 0, len(names))
	for _, name := range names {
		p, err := f.Get(name)
		if err != nil {
			return nil, fmt.Errorf("failed to get provider %s: %w", name, err)
		}
		providers = append(providers, p)
	}
	return NewFallbackChain(providers...), nil
}

// createFromConfig creates a provider from configuration.
func (f *Factory) createFromConfig(cfg *storage.Provider) (Provider, error) {
	switch cfg.Type {
	case "openai":
		return NewOpenAIProvider(cfg), nil
	case "anthropic":
		return NewAnthropicProvider(cfg), nil
	case "deepseek":
		return NewDeepSeekProvider(cfg), nil
	case "openrouter":
		return NewOpenRouterProvider(cfg), nil
	default:
		return nil, fmt.Errorf("unknown provider type: %s", cfg.Type)
	}
}

// FallbackChain provides automatic failover between providers.
type FallbackChain struct {
	providers []Provider
	cooldowns map[string]time.Time
	mu        sync.RWMutex
}

// NewFallbackChain creates a new FallbackChain.
func NewFallbackChain(providers ...Provider) *FallbackChain {
	return &FallbackChain{
		providers: providers,
		cooldowns: make(map[string]time.Time),
	}
}

// Chat sends a chat request with automatic failover.
func (fc *FallbackChain) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	var lastErr error

	for _, p := range fc.providers {
		// Check cooldown
		if !fc.isAvailable(p.GetName()) {
			continue
		}

		// Set model if not specified
		if req.Model == "" {
			req.Model = p.GetDefaultModel()
		}

		resp, err := p.Chat(ctx, req)
		if err == nil {
			return resp, nil
		}

		// Handle error
		lastErr = err
		if failoverErr := fc.classifyError(err, p); failoverErr != nil {
			if !failoverErr.IsRetriable() {
				return nil, err
			}
			fc.setCooldown(p.GetName(), failoverErr.Reason)
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, icooclawErrors.ErrProviderUnavailable
}

// ChatStream sends a streaming chat request with automatic failover.
func (fc *FallbackChain) ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error {
	var lastErr error

	for _, p := range fc.providers {
		if !fc.isAvailable(p.GetName()) {
			continue
		}

		if req.Model == "" {
			req.Model = p.GetDefaultModel()
		}

		err := p.ChatStream(ctx, req, callback)
		if err == nil {
			return nil
		}

		lastErr = err
		if failoverErr := fc.classifyError(err, p); failoverErr != nil {
			if !failoverErr.IsRetriable() {
				return err
			}
			fc.setCooldown(p.GetName(), failoverErr.Reason)
		}
	}

	if lastErr != nil {
		return lastErr
	}
	return icooclawErrors.ErrProviderUnavailable
}

// isAvailable checks if a provider is available (not in cooldown).
func (fc *FallbackChain) isAvailable(name string) bool {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	cooldownEnd, exists := fc.cooldowns[name]
	if !exists {
		return true
	}
	return time.Now().After(cooldownEnd)
}

// setCooldown sets a cooldown for a provider.
func (fc *FallbackChain) setCooldown(name string, reason icooclawErrors.FailoverReason) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	var duration time.Duration
	switch reason {
	case icooclawErrors.FailoverRateLimit:
		duration = 60 * time.Second
	case icooclawErrors.FailoverTimeout:
		duration = 30 * time.Second
	case icooclawErrors.FailoverAuth:
		duration = 5 * time.Minute
	default:
		duration = 10 * time.Second
	}

	fc.cooldowns[name] = time.Now().Add(duration)
}

// classifyError classifies an error for failover decisions.
func (fc *FallbackChain) classifyError(err error, p Provider) *icooclawErrors.FailoverError {
	// Check for specific error types
	if errors.Is(err, icooclawErrors.ErrRateLimited) {
		return icooclawErrors.NewFailoverError(icooclawErrors.FailoverRateLimit, p.GetName(), "", 429, err)
	}
	if errors.Is(err, icooclawErrors.ErrTimeout) {
		return icooclawErrors.NewFailoverError(icooclawErrors.FailoverTimeout, p.GetName(), "", 0, err)
	}
	if errors.Is(err, icooclawErrors.ErrAuthFailed) {
		return icooclawErrors.NewFailoverError(icooclawErrors.FailoverAuth, p.GetName(), "", 401, err)
	}

	// Default to unknown
	return icooclawErrors.NewFailoverError(icooclawErrors.FailoverUnknown, p.GetName(), "", 0, err)
}

// GetDefaultModel returns the default model of the first provider.
func (fc *FallbackChain) GetDefaultModel() string {
	if len(fc.providers) > 0 {
		return fc.providers[0].GetDefaultModel()
	}
	return ""
}

// GetName returns the name of the first provider.
func (fc *FallbackChain) GetName() string {
	if len(fc.providers) > 0 {
		return fc.providers[0].GetName()
	}
	return "fallback-chain"
}
