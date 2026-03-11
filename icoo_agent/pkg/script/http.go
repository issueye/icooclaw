// Package script provides JavaScript scripting engine for icooclaw.
package script

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTPClient provides HTTP client operations.
type HTTPClient struct {
	cfg    *Config
	logger *slog.Logger
	client *http.Client
}

// NewHTTPClient creates a new HTTPClient builtin.
func NewHTTPClient(cfg *Config, logger *slog.Logger) *HTTPClient {
	if logger == nil {
		logger = slog.Default()
	}

	timeout := time.Duration(cfg.HTTPTimeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	return &HTTPClient{
		cfg:    cfg,
		logger: logger,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Name returns the builtin name.
func (h *HTTPClient) Name() string {
	return "http"
}

// Object returns the http object.
func (h *HTTPClient) Object() map[string]any {
	return map[string]any{
		"get":     h.Get,
		"post":    h.Post,
		"put":     h.Put,
		"delete":  h.Delete,
		"request": h.Request,
	}
}

// Get performs a GET request.
func (h *HTTPClient) Get(url string, headers map[string]string) (map[string]any, error) {
	return h.Request("GET", url, nil, headers)
}

// Post performs a POST request.
func (h *HTTPClient) Post(url string, body any, headers map[string]string) (map[string]any, error) {
	return h.Request("POST", url, body, headers)
}

// Put performs a PUT request.
func (h *HTTPClient) Put(url string, body any, headers map[string]string) (map[string]any, error) {
	return h.Request("PUT", url, body, headers)
}

// Delete performs a DELETE request.
func (h *HTTPClient) Delete(url string, headers map[string]string) (map[string]any, error) {
	return h.Request("DELETE", url, nil, headers)
}

// Request performs an HTTP request.
func (h *HTTPClient) Request(method, reqURL string, body any, headers map[string]string) (map[string]any, error) {
	if !h.cfg.AllowNetwork {
		return nil, fmt.Errorf("network access is not allowed")
	}

	// Check domain whitelist
	if len(h.cfg.AllowedDomains) > 0 {
		parsedURL, err := url.Parse(reqURL)
		if err != nil {
			return nil, fmt.Errorf("invalid URL: %w", err)
		}
		allowed := false
		for _, domain := range h.cfg.AllowedDomains {
			if strings.HasSuffix(parsedURL.Host, domain) {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, fmt.Errorf("domain not allowed: %s", parsedURL.Host)
		}
	}

	// Prepare body
	var reqBody io.Reader
	if body != nil {
		switch v := body.(type) {
		case string:
			reqBody = strings.NewReader(v)
		case map[string]any:
			data, _ := json.Marshal(v)
			reqBody = bytes.NewReader(data)
		default:
			data, _ := json.Marshal(v)
			reqBody = bytes.NewReader(data)
		}
	}

	// Create request
	req, err := http.NewRequest(method, reqURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Set default content type for JSON body
	if body != nil && req.Header.Get("Content-Type") == "" {
		if _, ok := body.(map[string]any); ok {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	// Execute request
	startTime := time.Now()
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Build result
	result := map[string]any{
		"status":     resp.StatusCode,
		"statusText": resp.Status,
		"headers":    flattenHeaders(resp.Header),
		"body":       string(respBody),
		"duration":   time.Since(startTime).String(),
		"ok":         resp.StatusCode >= 200 && resp.StatusCode < 300,
	}

	// Try to parse JSON
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		var jsonBody any
		if err := json.Unmarshal(respBody, &jsonBody); err == nil {
			result["json"] = jsonBody
		}
	}

	return result, nil
}

func flattenHeaders(headers http.Header) map[string]string {
	result := make(map[string]string)
	for key, values := range headers {
		if len(values) > 0 {
			result[key] = values[0]
		}
	}
	return result
}