// Package providers provides LLM provider abstraction for icooclaw.
package providers

import (
	"testing"

	"icooclaw/pkg/consts"
	"icooclaw/pkg/storage"
)

func TestGetModelInfo(t *testing.T) {
	tests := []struct {
		modelID   string
		wantName  string
		wantFound bool
	}{
		{"gpt-4o", "GPT-4o", true},
		{"claude-3-5-sonnet-20241022", "Claude 3.5 Sonnet", true},
		{"claude-3.5-sonnet", "Claude 3.5 Sonnet", true}, // alias
		{"deepseek-chat", "DeepSeek Chat", true},
		{"unknown-model", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			info := GetModelInfo(tt.modelID)
			if tt.wantFound {
				if info == nil {
					t.Errorf("Expected to find model %s", tt.modelID)
					return
				}
				if info.Name != tt.wantName {
					t.Errorf("Expected name %s, got %s", tt.wantName, info.Name)
				}
			} else {
				if info != nil {
					t.Errorf("Expected not to find model %s", tt.modelID)
				}
			}
		})
	}
}

func TestListModels(t *testing.T) {
	models := ListModels()
	if len(models) == 0 {
		t.Error("Expected at least one model")
	}
}

func TestListModelsByProvider(t *testing.T) {
	openaiModels := ListModelsByProvider("openai")
	if len(openaiModels) == 0 {
		t.Error("Expected at least one OpenAI model")
	}

	anthropicModels := ListModelsByProvider("anthropic")
	if len(anthropicModels) == 0 {
		t.Error("Expected at least one Anthropic model")
	}
}

func TestCalculateCost(t *testing.T) {
	tests := []struct {
		modelID      string
		inputTokens  int
		outputTokens int
		wantError    bool
	}{
		{"gpt-4o", 1000, 500, false},
		{"claude-3-5-sonnet-20241022", 1000, 500, false},
		{"unknown-model", 1000, 500, true},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			cost, err := CalculateCost(tt.modelID, tt.inputTokens, tt.outputTokens)
			if tt.wantError {
				if err == nil {
					t.Error("Expected error")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if cost <= 0 {
					t.Errorf("Expected positive cost, got %f", cost)
				}
			}
		})
	}
}

func TestRegistry_RegisterFactory(t *testing.T) {
	registry := NewRegistry(nil)

	// Check built-in factories
	factories := []consts.ProviderType{
		consts.ProviderOpenAI,
		consts.ProviderAnthropic,
		consts.ProviderDeepSeek,
		consts.ProviderOpenRouter,
		consts.ProviderGemini,
		consts.ProviderMistral,
		consts.ProviderGroq,
		consts.ProviderOllama,
		consts.ProviderMoonshot,
		consts.ProviderQwen,
		consts.ProviderSiliconFlow,
	}

	for _, ft := range factories {
		if _, ok := registry.factories[ft]; !ok {
			t.Errorf("Expected factory for %s", ft)
		}
	}
}

func TestRegistry_CreateProvider(t *testing.T) {
	registry := NewRegistry(nil)

	tests := []struct {
		providerType string
		wantError    bool
	}{
		{"openai", false},
		{"anthropic", false},
		{"deepseek", false},
		{"unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.providerType, func(t *testing.T) {
			cfg := &storage.Provider{
				Name:         "test",
				Type:         consts.ToProviderType(tt.providerType),
				APIKey:       "test-key",
				DefaultModel: "test-model",
			}

			provider, err := registry.CreateProvider(cfg)
			if tt.wantError {
				if err == nil {
					t.Error("Expected error")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if provider == nil {
					t.Error("Expected provider")
				}
			}
		})
	}
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	registry := NewRegistry(nil)

	cfg := &storage.Provider{
		Name:         "test-openai",
		Type:         "openai",
		APIKey:       "test-key",
		DefaultModel: "gpt-4o",
	}

	provider, err := registry.CreateProvider(cfg)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	registry.Register("my-openai", provider)

	// Get provider
	got, err := registry.Get("my-openai")
	if err != nil {
		t.Fatalf("Failed to get provider: %v", err)
	}

	if got.GetName() != "openai" {
		t.Errorf("Expected name 'openai', got %s", got.GetName())
	}

	// List providers
	names := registry.List()
	if len(names) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(names))
	}
}

func TestModelInfo_SupportsVision(t *testing.T) {
	tests := []struct {
		modelID    string
		wantVision bool
	}{
		{"gpt-4o", true},
		{"gpt-4o-mini", true},
		{"deepseek-chat", false},
		{"gemini-2.0-flash", true},
		{"claude-3-5-sonnet-20241022", true},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			info := GetModelInfo(tt.modelID)
			if info == nil {
				t.Fatalf("Model not found: %s", tt.modelID)
			}
			if info.SupportsVision != tt.wantVision {
				t.Errorf("Expected SupportsVision=%v, got %v", tt.wantVision, info.SupportsVision)
			}
		})
	}
}

func TestModelInfo_SupportsTools(t *testing.T) {
	tests := []struct {
		modelID   string
		wantTools bool
	}{
		{"gpt-4o", true},
		{"o1", false},
		{"deepseek-chat", true},
		{"claude-3-5-sonnet-20241022", true},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			info := GetModelInfo(tt.modelID)
			if info == nil {
				t.Fatalf("Model not found: %s", tt.modelID)
			}
			if info.SupportsTools != tt.wantTools {
				t.Errorf("Expected SupportsTools=%v, got %v", tt.wantTools, info.SupportsTools)
			}
		})
	}
}

func TestModelInfo_ContextWindow(t *testing.T) {
	tests := []struct {
		modelID     string
		wantContext int
	}{
		{"gpt-4o", 128000},
		{"claude-3-5-sonnet-20241022", 200000},
		{"gemini-1.5-pro", 2097152},
		{"deepseek-chat", 64000},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			info := GetModelInfo(tt.modelID)
			if info == nil {
				t.Fatalf("Model not found: %s", tt.modelID)
			}
			if info.ContextWindow != tt.wantContext {
				t.Errorf("Expected ContextWindow=%d, got %d", tt.wantContext, info.ContextWindow)
			}
		})
	}
}
