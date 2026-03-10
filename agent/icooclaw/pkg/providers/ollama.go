// Package providers provides LLM provider abstraction for icooclaw.
package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"icooclaw/pkg/consts"
	"icooclaw/pkg/storage"
)

// OllamaProvider implements Provider for Ollama (local).
type OllamaProvider struct {
	*BaseProvider
}

// NewOllamaProvider creates a new Ollama provider.
func NewOllamaProvider(cfg *storage.Provider) Provider {
	providerName := consts.ProviderOllama
	apiBase := cfg.APIBase
	if apiBase == "" {
		apiBase = "http://localhost:11434"
	}
	defaultModel := cfg.DefaultModel
	if defaultModel == "" {
		defaultModel = "llama3.2"
	}

	return &OllamaProvider{
		BaseProvider: NewBaseProvider(providerName.ToString(), cfg.APIKey, apiBase, defaultModel),
	}
}

// Chat sends a chat request to Ollama.
func (p *OllamaProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// Convert messages to Ollama format
	messages := make([]map[string]string, 0, len(req.Messages))
	for _, msg := range req.Messages {
		messages = append(messages, map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	ollamaReq := map[string]any{
		"model":    req.Model,
		"messages": messages,
		"stream":   false,
	}

	if len(req.Tools) > 0 {
		tools := make([]map[string]any, 0, len(req.Tools))
		for _, t := range req.Tools {
			tools = append(tools, map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        t.Function.Name,
					"description": t.Function.Description,
					"parameters":  t.Function.Parameters,
				},
			})
		}
		ollamaReq["tools"] = tools
	}

	var reqBody io.Reader
	data, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	reqBody = bytes.NewReader(data)

	url := p.apiBase + "/api/chat"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Model     string `json:"model"`
		CreatedAt string `json:"created_at"`
		Message   struct {
			Role      string `json:"role"`
			Content   string `json:"content"`
			ToolCalls []struct {
				Function struct {
					Name      string         `json:"name"`
					Arguments map[string]any `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
		Done bool `json:"done"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var toolCalls []ToolCall
	for _, tc := range result.Message.ToolCalls {
		argsJSON, _ := json.Marshal(tc.Function.Arguments)
		toolCalls = append(toolCalls, ToolCall{
			ID:   fmt.Sprintf("call_%s", tc.Function.Name),
			Type: "function",
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      tc.Function.Name,
				Arguments: string(argsJSON),
			},
		})
	}

	return &ChatResponse{
		Model:     result.Model,
		Content:   result.Message.Content,
		ToolCalls: toolCalls,
	}, nil
}

// ChatStream sends a streaming chat request to Ollama.
func (p *OllamaProvider) ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error {
	// Convert messages to Ollama format
	messages := make([]map[string]string, 0, len(req.Messages))
	for _, msg := range req.Messages {
		messages = append(messages, map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	ollamaReq := map[string]any{
		"model":    req.Model,
		"messages": messages,
		"stream":   true,
	}

	data, err := json.Marshal(ollamaReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := p.apiBase + "/api/chat"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var chunk struct {
			Model   string `json:"model"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			Done bool `json:"done"`
		}

		if err := json.Unmarshal(scanner.Bytes(), &chunk); err != nil {
			continue
		}

		if err := callback(chunk.Message.Content, "", nil, chunk.Done); err != nil {
			return err
		}

		if chunk.Done {
			return nil
		}
	}

	return scanner.Err()
}
