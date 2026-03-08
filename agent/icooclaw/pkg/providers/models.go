// Package providers provides LLM provider abstraction for icooclaw.
package providers

import "fmt"

// ModelInfo contains information about a model.
type ModelInfo struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Provider        string   `json:"provider"`
	ContextWindow   int      `json:"context_window"`
	MaxOutputTokens int      `json:"max_output_tokens"`
	SupportsVision  bool     `json:"supports_vision"`
	SupportsTools   bool     `json:"supports_tools"`
	SupportsJSON    bool     `json:"supports_json"`
	SupportsStream  bool     `json:"supports_stream"`
	InputPrice      float64  `json:"input_price"`  // per 1M tokens
	OutputPrice     float64  `json:"output_price"` // per 1M tokens
	Aliases         []string `json:"aliases,omitempty"`
}

// Built-in model definitions
var modelRegistry = map[string]*ModelInfo{
	// OpenAI models
	"gpt-4o": {
		ID:              "gpt-4o",
		Name:            "GPT-4o",
		Provider:        "openai",
		ContextWindow:   128000,
		MaxOutputTokens: 4096,
		SupportsVision:  true,
		SupportsTools:   true,
		SupportsJSON:    true,
		SupportsStream:  true,
		InputPrice:      2.50,
		OutputPrice:     10.00,
	},
	"gpt-4o-mini": {
		ID:              "gpt-4o-mini",
		Name:            "GPT-4o Mini",
		Provider:        "openai",
		ContextWindow:   128000,
		MaxOutputTokens: 16384,
		SupportsVision:  true,
		SupportsTools:   true,
		SupportsJSON:    true,
		SupportsStream:  true,
		InputPrice:      0.15,
		OutputPrice:     0.60,
	},
	"gpt-4-turbo": {
		ID:              "gpt-4-turbo",
		Name:            "GPT-4 Turbo",
		Provider:        "openai",
		ContextWindow:   128000,
		MaxOutputTokens: 4096,
		SupportsVision:  true,
		SupportsTools:   true,
		SupportsJSON:    true,
		SupportsStream:  true,
		InputPrice:      10.00,
		OutputPrice:     30.00,
	},
	"o1": {
		ID:              "o1",
		Name:            "o1",
		Provider:        "openai",
		ContextWindow:   200000,
		MaxOutputTokens: 100000,
		SupportsVision:  true,
		SupportsTools:   false,
		SupportsJSON:    true,
		SupportsStream:  false,
		InputPrice:      15.00,
		OutputPrice:     60.00,
	},
	"o1-mini": {
		ID:              "o1-mini",
		Name:            "o1 Mini",
		Provider:        "openai",
		ContextWindow:   128000,
		MaxOutputTokens: 65536,
		SupportsVision:  false,
		SupportsTools:   false,
		SupportsJSON:    true,
		SupportsStream:  false,
		InputPrice:      1.75,
		OutputPrice:     7.00,
	},

	// Anthropic models
	"claude-3-5-sonnet-20241022": {
		ID:              "claude-3-5-sonnet-20241022",
		Name:            "Claude 3.5 Sonnet",
		Provider:        "anthropic",
		ContextWindow:   200000,
		MaxOutputTokens: 8192,
		SupportsVision:  true,
		SupportsTools:   true,
		SupportsJSON:    true,
		SupportsStream:  true,
		InputPrice:      3.00,
		OutputPrice:     15.00,
		Aliases:         []string{"claude-3.5-sonnet", "claude-3-5-sonnet"},
	},
	"claude-3-opus-20240229": {
		ID:              "claude-3-opus-20240229",
		Name:            "Claude 3 Opus",
		Provider:        "anthropic",
		ContextWindow:   200000,
		MaxOutputTokens: 4096,
		SupportsVision:  true,
		SupportsTools:   true,
		SupportsJSON:    true,
		SupportsStream:  true,
		InputPrice:      15.00,
		OutputPrice:     75.00,
		Aliases:         []string{"claude-3-opus"},
	},
	"claude-3-haiku-20240307": {
		ID:              "claude-3-haiku-20240307",
		Name:            "Claude 3 Haiku",
		Provider:        "anthropic",
		ContextWindow:   200000,
		MaxOutputTokens: 4096,
		SupportsVision:  true,
		SupportsTools:   true,
		SupportsJSON:    true,
		SupportsStream:  true,
		InputPrice:      0.25,
		OutputPrice:     1.25,
		Aliases:         []string{"claude-3-haiku"},
	},

	// DeepSeek models
	"deepseek-chat": {
		ID:              "deepseek-chat",
		Name:            "DeepSeek Chat",
		Provider:        "deepseek",
		ContextWindow:   64000,
		MaxOutputTokens: 4096,
		SupportsVision:  false,
		SupportsTools:   true,
		SupportsJSON:    true,
		SupportsStream:  true,
		InputPrice:      0.14,
		OutputPrice:     0.28,
	},
	"deepseek-reasoner": {
		ID:              "deepseek-reasoner",
		Name:            "DeepSeek Reasoner",
		Provider:        "deepseek",
		ContextWindow:   64000,
		MaxOutputTokens: 4096,
		SupportsVision:  false,
		SupportsTools:   true,
		SupportsJSON:    true,
		SupportsStream:  true,
		InputPrice:      0.55,
		OutputPrice:     2.19,
		Aliases:         []string{"deepseek-r1"},
	},

	// Google Gemini models
	"gemini-2.0-flash": {
		ID:              "gemini-2.0-flash",
		Name:            "Gemini 2.0 Flash",
		Provider:        "gemini",
		ContextWindow:   1048576,
		MaxOutputTokens: 8192,
		SupportsVision:  true,
		SupportsTools:   true,
		SupportsJSON:    true,
		SupportsStream:  true,
		InputPrice:      0.10,
		OutputPrice:     0.40,
	},
	"gemini-1.5-pro": {
		ID:              "gemini-1.5-pro",
		Name:            "Gemini 1.5 Pro",
		Provider:        "gemini",
		ContextWindow:   2097152,
		MaxOutputTokens: 8192,
		SupportsVision:  true,
		SupportsTools:   true,
		SupportsJSON:    true,
		SupportsStream:  true,
		InputPrice:      1.25,
		OutputPrice:     5.00,
	},

	// Mistral models
	"mistral-large-latest": {
		ID:              "mistral-large-latest",
		Name:            "Mistral Large",
		Provider:        "mistral",
		ContextWindow:   128000,
		MaxOutputTokens: 4096,
		SupportsVision:  false,
		SupportsTools:   true,
		SupportsJSON:    true,
		SupportsStream:  true,
		InputPrice:      2.00,
		OutputPrice:     6.00,
	},
	"codestral-latest": {
		ID:              "codestral-latest",
		Name:            "Codestral",
		Provider:        "mistral",
		ContextWindow:   32768,
		MaxOutputTokens: 4096,
		SupportsVision:  false,
		SupportsTools:   false,
		SupportsJSON:    true,
		SupportsStream:  true,
		InputPrice:      0.30,
		OutputPrice:     0.90,
	},

	// Groq models
	"llama-3.3-70b-versatile": {
		ID:              "llama-3.3-70b-versatile",
		Name:            "Llama 3.3 70B",
		Provider:        "groq",
		ContextWindow:   128000,
		MaxOutputTokens: 8192,
		SupportsVision:  false,
		SupportsTools:   true,
		SupportsJSON:    true,
		SupportsStream:  true,
		InputPrice:      0.59,
		OutputPrice:     0.79,
	},
	"llama-3.1-8b-instant": {
		ID:              "llama-3.1-8b-instant",
		Name:            "Llama 3.1 8B",
		Provider:        "groq",
		ContextWindow:   128000,
		MaxOutputTokens: 8192,
		SupportsVision:  false,
		SupportsTools:   true,
		SupportsJSON:    true,
		SupportsStream:  true,
		InputPrice:      0.05,
		OutputPrice:     0.08,
	},

	// Qwen models
	"qwen-max": {
		ID:              "qwen-max",
		Name:            "Qwen Max",
		Provider:        "qwen",
		ContextWindow:   32768,
		MaxOutputTokens: 8192,
		SupportsVision:  false,
		SupportsTools:   true,
		SupportsJSON:    true,
		SupportsStream:  true,
		InputPrice:      2.00,
		OutputPrice:     6.00,
	},
	"qwen-plus": {
		ID:              "qwen-plus",
		Name:            "Qwen Plus",
		Provider:        "qwen",
		ContextWindow:   128000,
		MaxOutputTokens: 6144,
		SupportsVision:  false,
		SupportsTools:   true,
		SupportsJSON:    true,
		SupportsStream:  true,
		InputPrice:      0.40,
		OutputPrice:     1.20,
	},
	"qwen-turbo": {
		ID:              "qwen-turbo",
		Name:            "Qwen Turbo",
		Provider:        "qwen",
		ContextWindow:   128000,
		MaxOutputTokens: 6144,
		SupportsVision:  false,
		SupportsTools:   true,
		SupportsJSON:    true,
		SupportsStream:  true,
		InputPrice:      0.05,
		OutputPrice:     0.20,
	},

	// Zhipu models
	"glm-4": {
		ID:              "glm-4",
		Name:            "GLM-4",
		Provider:        "zhipu",
		ContextWindow:   128000,
		MaxOutputTokens: 4096,
		SupportsVision:  false,
		SupportsTools:   true,
		SupportsJSON:    true,
		SupportsStream:  true,
		InputPrice:      0.10,
		OutputPrice:     0.10,
	},
	"glm-4-flash": {
		ID:              "glm-4-flash",
		Name:            "GLM-4 Flash",
		Provider:        "zhipu",
		ContextWindow:   128000,
		MaxOutputTokens: 4096,
		SupportsVision:  false,
		SupportsTools:   true,
		SupportsJSON:    true,
		SupportsStream:  true,
		InputPrice:      0.001,
		OutputPrice:     0.001,
	},

	// Moonshot models
	"moonshot-v1-8k": {
		ID:              "moonshot-v1-8k",
		Name:            "Moonshot V1 8K",
		Provider:        "moonshot",
		ContextWindow:   8192,
		MaxOutputTokens: 4096,
		SupportsVision:  false,
		SupportsTools:   false,
		SupportsJSON:    true,
		SupportsStream:  true,
		InputPrice:      0.50,
		OutputPrice:     0.50,
	},
	"moonshot-v1-32k": {
		ID:              "moonshot-v1-32k",
		Name:            "Moonshot V1 32K",
		Provider:        "moonshot",
		ContextWindow:   32768,
		MaxOutputTokens: 4096,
		SupportsVision:  false,
		SupportsTools:   false,
		SupportsJSON:    true,
		SupportsStream:  true,
		InputPrice:      1.00,
		OutputPrice:     1.00,
	},
	"moonshot-v1-128k": {
		ID:              "moonshot-v1-128k",
		Name:            "Moonshot V1 128K",
		Provider:        "moonshot",
		ContextWindow:   131072,
		MaxOutputTokens: 4096,
		SupportsVision:  false,
		SupportsTools:   false,
		SupportsJSON:    true,
		SupportsStream:  true,
		InputPrice:      2.00,
		OutputPrice:     2.00,
	},

	// xAI Grok models
	"grok-2-latest": {
		ID:              "grok-2-latest",
		Name:            "Grok 2",
		Provider:        "grok",
		ContextWindow:   131072,
		MaxOutputTokens: 4096,
		SupportsVision:  true,
		SupportsTools:   true,
		SupportsJSON:    true,
		SupportsStream:  true,
		InputPrice:      2.00,
		OutputPrice:     10.00,
	},

	// Ollama models (local, no pricing)
	"llama3.2": {
		ID:              "llama3.2",
		Name:            "Llama 3.2",
		Provider:        "ollama",
		ContextWindow:   131072,
		MaxOutputTokens: 4096,
		SupportsVision:  true,
		SupportsTools:   true,
		SupportsJSON:    true,
		SupportsStream:  true,
		InputPrice:      0,
		OutputPrice:     0,
	},
	"qwen2.5": {
		ID:              "qwen2.5",
		Name:            "Qwen 2.5",
		Provider:        "ollama",
		ContextWindow:   131072,
		MaxOutputTokens: 4096,
		SupportsVision:  false,
		SupportsTools:   true,
		SupportsJSON:    true,
		SupportsStream:  true,
		InputPrice:      0,
		OutputPrice:     0,
	},
}

// GetModelInfo returns model information by ID.
func GetModelInfo(modelID string) *ModelInfo {
	// Direct lookup
	if info, ok := modelRegistry[modelID]; ok {
		return info
	}

	// Check aliases
	for _, info := range modelRegistry {
		for _, alias := range info.Aliases {
			if alias == modelID {
				return info
			}
		}
	}

	return nil
}

// ListModels lists all available models.
func ListModels() []*ModelInfo {
	models := make([]*ModelInfo, 0, len(modelRegistry))
	for _, info := range modelRegistry {
		models = append(models, info)
	}
	return models
}

// ListModelsByProvider lists models for a specific provider.
func ListModelsByProvider(provider string) []*ModelInfo {
	models := make([]*ModelInfo, 0)
	for _, info := range modelRegistry {
		if info.Provider == provider {
			models = append(models, info)
		}
	}
	return models
}

// CalculateCost calculates the cost for a request.
func CalculateCost(modelID string, inputTokens, outputTokens int) (float64, error) {
	info := GetModelInfo(modelID)
	if info == nil {
		return 0, fmt.Errorf("unknown model: %s", modelID)
	}

	inputCost := float64(inputTokens) * info.InputPrice / 1000000
	outputCost := float64(outputTokens) * info.OutputPrice / 1000000

	return inputCost + outputCost, nil
}
