package web

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
	return "向外部 API 和网站发送 HTTP 请求。"
}

// Parameters returns the tool parameters.
func (t *HTTPTool) Parameters() map[string]any {
	return map[string]any{
		"url": map[string]any{
			"type":        "string",
			"description": "请求的 URL",
			"required":    true,
		},
		"method": map[string]any{
			"type":        "string",
			"description": "HTTP 方法 (GET, POST, PUT, DELETE)",
			"required":    true,
		},
		"headers": map[string]any{
			"type":        "object",
			"description": "HTTP 请求头，键值对形式",
		},
		"body": map[string]any{
			"type":        "string",
			"description": "POST/PUT 请求的请求体",
		},
	}
}

// Execute executes the HTTP request.
func (t *HTTPTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	reqURL, ok := args["url"].(string)
	if !ok {
		return &tools.Result{Success: false, Error: fmt.Errorf("需要提供 url 参数")}
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

func flattenHeaders(headers http.Header) map[string]string {
	result := make(map[string]string)
	for key, values := range headers {
		if len(values) > 0 {
			result[key] = values[0]
		}
	}
	return result
}
