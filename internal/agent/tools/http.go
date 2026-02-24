package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPRequestTool HTTP 请求工具
type HTTPRequestTool struct {
	baseTool *BaseTool
	client   *http.Client
}

// NewHTTPRequestTool 创建 HTTP 请求工具
func NewHTTPRequestTool() *HTTPRequestTool {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	tool := NewBaseTool(
		"http_request",
		"发送 HTTP 请求。支持 GET、POST、PUT、DELETE 等方法。可以用于调用外部 API。",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "请求的 URL 地址",
				},
				"method": map[string]interface{}{
					"type":        "string",
					"description": "HTTP 方法: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS",
					"enum":        []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"},
					"default":     "GET",
				},
				"headers": map[string]interface{}{
					"type":        "object",
					"description": "请求头，键值对形式",
				},
				"body": map[string]interface{}{
					"type":        "string",
					"description": "请求体内容",
				},
				"timeout": map[string]interface{}{
					"type":        "number",
					"description": "请求超时时间（秒）",
					"default":     30,
				},
			},
			"required": []string{"url"},
		},
		nil,
	)

	return &HTTPRequestTool{
		baseTool: tool,
		client:   client,
	}
}

// Name 获取名称
func (t *HTTPRequestTool) Name() string {
	return t.baseTool.Name()
}

// Description 获取描述
func (t *HTTPRequestTool) Description() string {
	return t.baseTool.Description()
}

// Parameters 获取参数定义
func (t *HTTPRequestTool) Parameters() map[string]interface{} {
	return t.baseTool.Parameters()
}

// Execute 执行 HTTP 请求
func (t *HTTPRequestTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	// 提取参数
	url, ok := params["url"].(string)
	if !ok || url == "" {
		return "", fmt.Errorf("invalid or missing URL")
	}

	method := "GET"
	if m, ok := params["method"].(string); ok && m != "" {
		method = strings.ToUpper(m)
	}

	headers := make(map[string]string)
	if h, ok := params["headers"].(map[string]interface{}); ok {
		for k, v := range h {
			if sv, ok := v.(string); ok {
				headers[k] = sv
			}
		}
	}

	var body io.Reader
	if b, ok := params["body"].(string); ok && b != "" {
		body = strings.NewReader(b)
	}

	// 设置超时
	timeout := 30
	if to, ok := params["timeout"].(float64); ok {
		timeout = int(to)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// 如果没有设置 Content-Type 但有 body，默认设置为 JSON
	if body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// 设置超时
	t.client.Timeout = time.Duration(timeout) * time.Second

	// 发送请求
	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// 构建结果
	result := map[string]interface{}{
		"status_code": resp.StatusCode,
		"status":      resp.Status,
		"headers":     resp.Header,
		"body":        string(respBody),
	}

	// 尝试解析 JSON
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, respBody, "", "  "); err == nil {
		result["body_json"] = json.RawMessage(respBody)
		result["body_formatted"] = prettyJSON.String()
	}

	// 返回格式化结果
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// ToDefinition 转换为工具定义
func (t *HTTPRequestTool) ToDefinition() ToolDefinition {
	return t.baseTool.ToDefinition()
}
