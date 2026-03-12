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
	"time"

	"icooclaw/pkg/errors"
)

// BaseProvider provides common functionality for providers.
type BaseProvider struct {
	name         string
	apiKey       string
	apiBase      string
	defaultModel string
	httpClient   *http.Client
}

// NewBaseProvider creates a new BaseProvider.
func NewBaseProvider(name, apiKey, apiBase, defaultModel string) *BaseProvider {
	return &BaseProvider{
		name:         name,
		apiKey:       apiKey,
		apiBase:      apiBase,
		defaultModel: defaultModel,
		httpClient: &http.Client{
			Timeout: 300 * time.Second, // 5 minutes for long LLM responses
		},
	}
}

// GetName returns the provider name.
func (p *BaseProvider) GetName() string {
	return p.name
}

// GetDefaultModel returns the default model.
func (p *BaseProvider) GetDefaultModel() string {
	return p.defaultModel
}

// doRequest performs an HTTP request.
func (p *BaseProvider) doRequest(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	url := p.apiBase + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// doRequestWithHeaders performs an HTTP request with custom headers.
func (p *BaseProvider) doRequestWithHeaders(ctx context.Context, method, path string, body any, headers map[string]string) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	url := p.apiBase + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// handleError handles HTTP error responses.
func (p *BaseProvider) handleError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case 401:
		return errors.NewFailoverError(errors.FailoverAuth, p.name, "", resp.StatusCode, fmt.Errorf("auth failed: %s", string(body)))
	case 429:
		return errors.NewFailoverError(errors.FailoverRateLimit, p.name, "", resp.StatusCode, fmt.Errorf("rate limited: %s", string(body)))
	case 500, 502, 503, 504:
		return errors.NewFailoverError(errors.FailoverTimeout, p.name, "", resp.StatusCode, fmt.Errorf("server error: %s", string(body)))
	default:
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}
}

// parseStreamChunk parses a streaming chunk.
func parseStreamChunk(data string) (content string, reasoning string, toolCalls []ToolCall, done bool, err error) {
	if data == "[DONE]" {
		return "", "", nil, true, nil
	}

	var chunk struct {
		Choices []struct {
			Delta struct {
				Content   string `json:"content"`
				Reasoning string `json:"reasoning_content"`
				// ToolCalls in streaming format uses index instead of id
				ToolCalls []struct {
					Index int    `json:"index"`
					ID    string `json:"id"`
					Type  string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"delta"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}

	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return "", "", nil, false, err
	}

	if len(chunk.Choices) > 0 {
		done = chunk.Choices[0].FinishReason != ""

		// Convert streaming tool calls to ToolCall format
		// We use ID field to store index temporarily for merging
		var calls []ToolCall
		for _, tc := range chunk.Choices[0].Delta.ToolCalls {
			// Create a unique key for merging: use index as part of ID
			// Format: "stream_index:N" where N is the index
			streamID := fmt.Sprintf("stream_index:%d", tc.Index)
			if tc.ID != "" {
				// If we have a real ID, use it but remember the index
				streamID = tc.ID
			}

			calls = append(calls, ToolCall{
				ID:   streamID,
				Type: tc.Type,
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}

		return chunk.Choices[0].Delta.Content, chunk.Choices[0].Delta.Reasoning, calls, done, nil
	}

	return "", "", nil, false, nil
}

// streamResponse handles streaming response parsing.
func (p *BaseProvider) streamResponse(resp *http.Response, callback StreamCallback) error {
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		content, reasoning, toolCalls, done, err := parseStreamChunk(data)
		if err != nil {
			continue
		}

		if err := callback(content, reasoning, toolCalls, done); err != nil {
			return err
		}

		if done {
			return nil
		}
	}

	return scanner.Err()
}
