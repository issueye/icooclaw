package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"icooclaw/pkg/tools"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPTool provides HTTP request functionality.
type HTTPTool struct {
	client *http.Client
}

// NewHTTPTool creates a new HTTP tool.
func NewHTTPTool() *HTTPTool {
	return &HTTPTool{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the tool name.
func (t *HTTPTool) Name() string {
	return "http_request"
}

// Description returns the tool description.
func (t *HTTPTool) Description() string {
	return "Make HTTP requests to external APIs and websites."
}

// Parameters returns the tool parameters.
func (t *HTTPTool) Parameters() map[string]tools.Parameter {
	return map[string]tools.Parameter{
		"url": {
			Type:        "string",
			Description: "The URL to request",
		},
		"method": {
			Type:        "string",
			Description: "HTTP method (GET, POST, PUT, DELETE)",
		},
		"headers": {
			Type:        "object",
			Description: "HTTP headers as key-value pairs",
		},
		"body": {
			Type:        "string",
			Description: "Request body for POST/PUT requests",
		},
	}
}

// Execute executes the HTTP request.
func (t *HTTPTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	reqURL, ok := args["url"].(string)
	if !ok {
		return &tools.Result{Success: false, Error: fmt.Errorf("url is required")}
	}

	method := "GET"
	if m, ok := args["method"].(string); ok {
		method = strings.ToUpper(m)
	}

	// Create request
	var req *http.Request
	var err error

	if body, ok := args["body"].(string); ok && body != "" {
		req, err = http.NewRequestWithContext(ctx, method, reqURL, strings.NewReader(body))
	} else {
		req, err = http.NewRequestWithContext(ctx, method, reqURL, nil)
	}

	if err != nil {
		return &tools.Result{Success: false, Error: err}
	}

	// Set headers
	if headers, ok := args["headers"].(map[string]any); ok {
		for key, value := range headers {
			req.Header.Set(key, fmt.Sprint(value))
		}
	}

	// Execute
	resp, err := t.client.Do(req)
	if err != nil {
		return &tools.Result{Success: false, Error: err}
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &tools.Result{Success: false, Error: err}
	}

	result := map[string]any{
		"status":     resp.StatusCode,
		"statusText": resp.Status,
		"headers":    flattenHeaders(resp.Header),
		"body":       string(respBody),
	}

	// Try to parse JSON
	if strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
		var jsonBody any
		if err := json.Unmarshal(respBody, &jsonBody); err == nil {
			result["json"] = jsonBody
		}
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return &tools.Result{Success: true, Content: string(resultJSON)}
}
