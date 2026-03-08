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

	"icooclaw/pkg/storage"
)

// OllamaProvider implements Provider for Ollama (local).
type OllamaProvider struct {
	*BaseProvider
}

// NewOllamaProvider creates a new Ollama provider.
func NewOllamaProvider(cfg *storage.Provider) *OllamaProvider {
	apiBase := cfg.APIBase
	if apiBase == "" {
		apiBase = "http://localhost:11434"
	}
	defaultModel := cfg.DefaultModel
	if defaultModel == "" {
		defaultModel = "llama3.2"
	}

	return &OllamaProvider{
		BaseProvider: NewBaseProvider("ollama", "", apiBase, defaultModel),
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

// MoonshotProvider implements Provider for Moonshot (月之暗面).
type MoonshotProvider struct {
	*BaseProvider
}

// NewMoonshotProvider creates a new Moonshot provider.
func NewMoonshotProvider(cfg *storage.Provider) *MoonshotProvider {
	apiBase := cfg.APIBase
	if apiBase == "" {
		apiBase = "https://api.moonshot.cn/v1"
	}
	defaultModel := cfg.DefaultModel
	if defaultModel == "" {
		defaultModel = "moonshot-v1-8k"
	}

	return &MoonshotProvider{
		BaseProvider: NewBaseProvider("moonshot", cfg.APIKey, apiBase, defaultModel),
	}
}

// Chat sends a chat request to Moonshot.
func (p *MoonshotProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	req.Stream = false
	resp, err := p.doRequest(ctx, "POST", "/chat/completions", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.handleError(resp)
	}

	var result struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage Usage `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &ChatResponse{
		ID:      result.ID,
		Model:   result.Model,
		Content: result.Choices[0].Message.Content,
		Usage:   result.Usage,
	}, nil
}

// ChatStream sends a streaming chat request to Moonshot.
func (p *MoonshotProvider) ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error {
	req.Stream = true
	resp, err := p.doRequest(ctx, "POST", "/chat/completions", req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return p.handleError(resp)
	}

	return p.streamResponse(resp, callback)
}

// ZhipuProvider implements Provider for Zhipu AI (智谱).
type ZhipuProvider struct {
	*BaseProvider
}

// NewZhipuProvider creates a new Zhipu provider.
func NewZhipuProvider(cfg *storage.Provider) *ZhipuProvider {
	apiBase := cfg.APIBase
	if apiBase == "" {
		apiBase = "https://open.bigmodel.cn/api/paas/v4"
	}
	defaultModel := cfg.DefaultModel
	if defaultModel == "" {
		defaultModel = "glm-4"
	}

	return &ZhipuProvider{
		BaseProvider: NewBaseProvider("zhipu", cfg.APIKey, apiBase, defaultModel),
	}
}

// Chat sends a chat request to Zhipu.
func (p *ZhipuProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	req.Stream = false
	resp, err := p.doRequest(ctx, "POST", "/chat/completions", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.handleError(resp)
	}

	var result struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Role      string     `json:"role"`
				Content   string     `json:"content"`
				ToolCalls []ToolCall `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage Usage `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &ChatResponse{
		ID:        result.ID,
		Model:     result.Model,
		Content:   result.Choices[0].Message.Content,
		ToolCalls: result.Choices[0].Message.ToolCalls,
		Usage:     result.Usage,
	}, nil
}

// ChatStream sends a streaming chat request to Zhipu.
func (p *ZhipuProvider) ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error {
	req.Stream = true
	resp, err := p.doRequest(ctx, "POST", "/chat/completions", req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return p.handleError(resp)
	}

	return p.streamResponse(resp, callback)
}

// QwenProvider implements Provider for Alibaba Qwen (通义千问).
type QwenProvider struct {
	*BaseProvider
}

// NewQwenProvider creates a new Qwen provider.
func NewQwenProvider(cfg *storage.Provider) *QwenProvider {
	apiBase := cfg.APIBase
	if apiBase == "" {
		apiBase = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}
	defaultModel := cfg.DefaultModel
	if defaultModel == "" {
		defaultModel = "qwen-plus"
	}

	return &QwenProvider{
		BaseProvider: NewBaseProvider("qwen", cfg.APIKey, apiBase, defaultModel),
	}
}

// Chat sends a chat request to Qwen.
func (p *QwenProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	req.Stream = false
	resp, err := p.doRequest(ctx, "POST", "/chat/completions", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.handleError(resp)
	}

	var result struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Role      string     `json:"role"`
				Content   string     `json:"content"`
				Reasoning string     `json:"reasoning_content"`
				ToolCalls []ToolCall `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage Usage `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &ChatResponse{
		ID:        result.ID,
		Model:     result.Model,
		Content:   result.Choices[0].Message.Content,
		Reasoning: result.Choices[0].Message.Reasoning,
		ToolCalls: result.Choices[0].Message.ToolCalls,
		Usage:     result.Usage,
	}, nil
}

// ChatStream sends a streaming chat request to Qwen.
func (p *QwenProvider) ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error {
	req.Stream = true
	resp, err := p.doRequest(ctx, "POST", "/chat/completions", req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return p.handleError(resp)
	}

	return p.streamResponse(resp, callback)
}

// SiliconFlowProvider implements Provider for SiliconFlow.
type SiliconFlowProvider struct {
	*BaseProvider
}

// NewSiliconFlowProvider creates a new SiliconFlow provider.
func NewSiliconFlowProvider(cfg *storage.Provider) *SiliconFlowProvider {
	apiBase := cfg.APIBase
	if apiBase == "" {
		apiBase = "https://api.siliconflow.cn/v1"
	}
	defaultModel := cfg.DefaultModel
	if defaultModel == "" {
		defaultModel = "Qwen/Qwen2.5-7B-Instruct"
	}

	return &SiliconFlowProvider{
		BaseProvider: NewBaseProvider("siliconflow", cfg.APIKey, apiBase, defaultModel),
	}
}

// Chat sends a chat request to SiliconFlow.
func (p *SiliconFlowProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	req.Stream = false
	resp, err := p.doRequest(ctx, "POST", "/chat/completions", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.handleError(resp)
	}

	var result struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Role      string     `json:"role"`
				Content   string     `json:"content"`
				ToolCalls []ToolCall `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage Usage `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &ChatResponse{
		ID:        result.ID,
		Model:     result.Model,
		Content:   result.Choices[0].Message.Content,
		ToolCalls: result.Choices[0].Message.ToolCalls,
		Usage:     result.Usage,
	}, nil
}

// ChatStream sends a streaming chat request to SiliconFlow.
func (p *SiliconFlowProvider) ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error {
	req.Stream = true
	resp, err := p.doRequest(ctx, "POST", "/chat/completions", req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return p.handleError(resp)
	}

	return p.streamResponse(resp, callback)
}

// GrokProvider implements Provider for xAI Grok.
type GrokProvider struct {
	*BaseProvider
}

// NewGrokProvider creates a new Grok provider.
func NewGrokProvider(cfg *storage.Provider) *GrokProvider {
	apiBase := cfg.APIBase
	if apiBase == "" {
		apiBase = "https://api.x.ai/v1"
	}
	defaultModel := cfg.DefaultModel
	if defaultModel == "" {
		defaultModel = "grok-2-latest"
	}

	return &GrokProvider{
		BaseProvider: NewBaseProvider("grok", cfg.APIKey, apiBase, defaultModel),
	}
}

// Chat sends a chat request to Grok.
func (p *GrokProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	req.Stream = false
	resp, err := p.doRequest(ctx, "POST", "/chat/completions", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.handleError(resp)
	}

	var result struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Role      string     `json:"role"`
				Content   string     `json:"content"`
				ToolCalls []ToolCall `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage Usage `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &ChatResponse{
		ID:        result.ID,
		Model:     result.Model,
		Content:   result.Choices[0].Message.Content,
		ToolCalls: result.Choices[0].Message.ToolCalls,
		Usage:     result.Usage,
	}, nil
}

// ChatStream sends a streaming chat request to Grok.
func (p *GrokProvider) ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error {
	req.Stream = true
	resp, err := p.doRequest(ctx, "POST", "/chat/completions", req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return p.handleError(resp)
	}

	return p.streamResponse(resp, callback)
}
