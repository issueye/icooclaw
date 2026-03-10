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
	"strings"

	"icooclaw/pkg/storage"
)

// GeminiProvider implements Provider for Google Gemini.
type GeminiProvider struct {
	*BaseProvider
	projectID string
	location  string
}

// NewGeminiProvider creates a new Gemini provider.
func NewGeminiProvider(cfg *storage.Provider) Provider {
	apiBase := cfg.APIBase
	if apiBase == "" {
		apiBase = "https://generativelanguage.googleapis.com/v1beta"
	}
	defaultModel := cfg.DefaultModel
	if defaultModel == "" {
		defaultModel = "gemini-2.0-flash"
	}

	return &GeminiProvider{
		BaseProvider: NewBaseProvider("gemini", cfg.APIKey, apiBase, defaultModel),
	}
}

// Chat sends a chat request to Gemini.
func (p *GeminiProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// Convert messages to Gemini format
	contents := make([]map[string]any, 0, len(req.Messages))
	var systemInstruction string

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			systemInstruction = msg.Content
			continue
		}

		role := "user"
		if msg.Role == "assistant" {
			role = "model"
		}

		contents = append(contents, map[string]any{
			"role": role,
			"parts": []map[string]string{
				{"text": msg.Content},
			},
		})
	}

	geminiReq := map[string]any{
		"contents": contents,
	}

	if systemInstruction != "" {
		geminiReq["systemInstruction"] = map[string]any{
			"parts": []map[string]string{
				{"text": systemInstruction},
			},
		}
	}

	// Convert tools
	if len(req.Tools) > 0 {
		declarations := make([]map[string]any, 0, len(req.Tools))
		for _, t := range req.Tools {
			declarations = append(declarations, map[string]any{
				"name":        t.Function.Name,
				"description": t.Function.Description,
				"parameters":  t.Function.Parameters,
			})
		}
		geminiReq["tools"] = []map[string]any{
			{"functionDeclarations": declarations},
		}
	}

	// Build URL with API key
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.apiBase, req.Model, p.apiKey)

	var reqBody io.Reader
	if geminiReq != nil {
		data, err := json.Marshal(geminiReq)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

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
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text         string `json:"text"`
					FunctionCall struct {
						Name string         `json:"name"`
						Args map[string]any `json:"args"`
					} `json:"functionCall"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}

	var content string
	var toolCalls []ToolCall

	for _, part := range result.Candidates[0].Content.Parts {
		if part.Text != "" {
			content += part.Text
		}
		if part.FunctionCall.Name != "" {
			argsJSON, _ := json.Marshal(part.FunctionCall.Args)
			toolCalls = append(toolCalls, ToolCall{
				ID:   fmt.Sprintf("call_%s", part.FunctionCall.Name),
				Type: "function",
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      part.FunctionCall.Name,
					Arguments: string(argsJSON),
				},
			})
		}
	}

	return &ChatResponse{
		Content:   content,
		ToolCalls: toolCalls,
		Usage: Usage{
			PromptTokens:     result.UsageMetadata.PromptTokenCount,
			CompletionTokens: result.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      result.UsageMetadata.TotalTokenCount,
		},
	}, nil
}

// ChatStream sends a streaming chat request to Gemini.
func (p *GeminiProvider) ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error {
	// Convert messages to Gemini format
	contents := make([]map[string]any, 0, len(req.Messages))
	var systemInstruction string

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			systemInstruction = msg.Content
			continue
		}

		role := "user"
		if msg.Role == "assistant" {
			role = "model"
		}

		contents = append(contents, map[string]any{
			"role": role,
			"parts": []map[string]string{
				{"text": msg.Content},
			},
		})
	}

	geminiReq := map[string]any{
		"contents": contents,
	}

	if systemInstruction != "" {
		geminiReq["systemInstruction"] = map[string]any{
			"parts": []map[string]string{
				{"text": systemInstruction},
			},
		}
	}

	// Build URL with API key
	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?key=%s&alt=sse", p.apiBase, req.Model, p.apiKey)

	var reqBody io.Reader
	if geminiReq != nil {
		data, err := json.Marshal(geminiReq)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, reqBody)
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
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		var result struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
				FinishReason string `json:"finishReason"`
			} `json:"candidates"`
		}

		if err := json.Unmarshal([]byte(data), &result); err != nil {
			continue
		}

		if len(result.Candidates) > 0 {
			done := result.Candidates[0].FinishReason != ""
			var content string
			for _, part := range result.Candidates[0].Content.Parts {
				content += part.Text
			}
			if err := callback(content, "", nil, done); err != nil {
				return err
			}
		}
	}

	return scanner.Err()
}

// MistralProvider implements Provider for Mistral AI.
type MistralProvider struct {
	*BaseProvider
}

// NewMistralProvider creates a new Mistral provider.
func NewMistralProvider(cfg *storage.Provider) Provider {
	apiBase := cfg.APIBase
	if apiBase == "" {
		apiBase = "https://api.mistral.ai/v1"
	}
	defaultModel := cfg.DefaultModel
	if defaultModel == "" {
		defaultModel = "mistral-large-latest"
	}

	return &MistralProvider{
		BaseProvider: NewBaseProvider("mistral", cfg.APIKey, apiBase, defaultModel),
	}
}

// Chat sends a chat request to Mistral.
func (p *MistralProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
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

// ChatStream sends a streaming chat request to Mistral.
func (p *MistralProvider) ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error {
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
