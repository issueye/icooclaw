// Package builtin provides built-in tools for icooclaw.
package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"icooclaw/pkg/tools"
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

// WebSearchTool provides web search functionality.
type WebSearchTool struct {
	client *http.Client
}

// NewWebSearchTool creates a new web search tool.
func NewWebSearchTool() *WebSearchTool {
	return &WebSearchTool{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the tool name.
func (t *WebSearchTool) Name() string {
	return "web_search"
}

// Description returns the tool description.
func (t *WebSearchTool) Description() string {
	return "Search the web for information using DuckDuckGo."
}

// Parameters returns the tool parameters.
func (t *WebSearchTool) Parameters() map[string]tools.Parameter {
	return map[string]tools.Parameter{
		"query": {
			Type:        "string",
			Description: "The search query",
		},
		"max_results": {
			Type:        "integer",
			Description: "Maximum number of results (default: 5)",
		},
	}
}

// Execute executes the web search.
func (t *WebSearchTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	query, ok := args["query"].(string)
	if !ok {
		return &tools.Result{Success: false, Error: fmt.Errorf("query is required")}
	}

	maxResults := 5
	if m, ok := args["max_results"].(int); ok && m > 0 {
		maxResults = m
	}

	// Use DuckDuckGo Instant Answer API
	searchURL := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_html=1", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return &tools.Result{Success: false, Error: err}
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return &tools.Result{Success: false, Error: err}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &tools.Result{Success: false, Error: err}
	}

	var result struct {
		AbstractText   string `json:"AbstractText"`
		AbstractURL    string `json:"AbstractURL"`
		AbstractSource string `json:"AbstractSource"`
		Heading        string `json:"Heading"`
		RelatedTopics  []struct {
			Text string `json:"Text"`
			URL  string `json:"FirstURL"`
		} `json:"RelatedTopics"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return &tools.Result{Success: false, Error: err}
	}

	// Build response
	var response strings.Builder
	if result.AbstractText != "" {
		response.WriteString(fmt.Sprintf("**%s**\n%s\nSource: %s\n\n", result.Heading, result.AbstractText, result.AbstractURL))
	}

	for i, topic := range result.RelatedTopics {
		if i >= maxResults {
			break
		}
		if topic.Text != "" {
			response.WriteString(fmt.Sprintf("- %s\n  URL: %s\n\n", topic.Text, topic.URL))
		}
	}

	if response.Len() == 0 {
		response.WriteString("No results found.")
	}

	return &tools.Result{Success: true, Content: response.String()}
}

// CalculatorTool provides calculation functionality.
type CalculatorTool struct{}

// NewCalculatorTool creates a new calculator tool.
func NewCalculatorTool() *CalculatorTool {
	return &CalculatorTool{}
}

// Name returns the tool name.
func (t *CalculatorTool) Name() string {
	return "calculator"
}

// Description returns the tool description.
func (t *CalculatorTool) Description() string {
	return "Perform mathematical calculations."
}

// Parameters returns the tool parameters.
func (t *CalculatorTool) Parameters() map[string]tools.Parameter {
	return map[string]tools.Parameter{
		"expression": {
			Type:        "string",
			Description: "Mathematical expression to evaluate (e.g., '2 + 2', '10 * 5')",
		},
	}
}

// Execute executes the calculation.
func (t *CalculatorTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	expr, ok := args["expression"].(string)
	if !ok {
		return &tools.Result{Success: false, Error: fmt.Errorf("expression is required")}
	}

	// Simple expression parser (supports +, -, *, /)
	result, err := evaluateExpression(expr)
	if err != nil {
		return &tools.Result{Success: false, Error: err}
	}

	return &tools.Result{Success: true, Content: fmt.Sprintf("%s = %v", expr, result)}
}

// DateTimeTool provides date/time functionality.
type DateTimeTool struct{}

// NewDateTimeTool creates a new datetime tool.
func NewDateTimeTool() *DateTimeTool {
	return &DateTimeTool{}
}

// Name returns the tool name.
func (t *DateTimeTool) Name() string {
	return "datetime"
}

// Description returns the tool description.
func (t *DateTimeTool) Description() string {
	return "Get current date and time information."
}

// Parameters returns the tool parameters.
func (t *DateTimeTool) Parameters() map[string]tools.Parameter {
	return map[string]tools.Parameter{
		"timezone": {
			Type:        "string",
			Description: "Timezone (e.g., 'UTC', 'America/New_York')",
		},
		"format": {
			Type:        "string",
			Description: "Time format (e.g., '2006-01-02 15:04:05')",
		},
	}
}

// Execute executes the datetime tool.
func (t *DateTimeTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	now := time.Now()

	// Handle timezone
	if tz, ok := args["timezone"].(string); ok {
		loc, err := time.LoadLocation(tz)
		if err != nil {
			return &tools.Result{Success: false, Error: fmt.Errorf("invalid timezone: %v", err)}
		}
		now = now.In(loc)
	}

	// Handle format
	format := time.RFC3339
	if f, ok := args["format"].(string); ok {
		format = f
	}

	result := map[string]any{
		"formatted": now.Format(format),
		"timestamp": now.Unix(),
		"date":      now.Format("2006-01-02"),
		"time":      now.Format("15:04:05"),
		"weekday":   now.Weekday().String(),
		"unix_nano": now.UnixNano(),
		"year":      now.Year(),
		"month":     int(now.Month()),
		"day":       now.Day(),
		"hour":      now.Hour(),
		"minute":    now.Minute(),
		"second":    now.Second(),
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return &tools.Result{Success: true, Content: string(resultJSON)}
}

// RegisterBuiltinTools registers all built-in tools.
func RegisterBuiltinTools(registry *tools.Registry) {
	registry.Register(NewHTTPTool())
	registry.Register(NewWebSearchTool())
	registry.Register(NewCalculatorTool())
	registry.Register(NewDateTimeTool())
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

// Simple expression evaluator
func evaluateExpression(expr string) (float64, error) {
	// Remove spaces
	expr = strings.ReplaceAll(expr, " ", "")

	var result float64
	var op byte
	var numStr string

	for i, ch := range expr {
		if ch >= '0' && ch <= '9' || ch == '.' {
			numStr += string(ch)
		} else if ch == '+' || ch == '-' || ch == '*' || ch == '/' {
			if numStr == "" {
				return 0, fmt.Errorf("invalid expression")
			}

			var num float64
			fmt.Sscanf(numStr, "%f", &num)

			if i == 0 || op == 0 {
				result = num
			} else {
				switch op {
				case '+':
					result += num
				case '-':
					result -= num
				case '*':
					result *= num
				case '/':
					if num == 0 {
						return 0, fmt.Errorf("division by zero")
					}
					result /= num
				}
			}

			op = byte(ch)
			numStr = ""
		}
	}

	// Handle last number
	if numStr != "" {
		var num float64
		fmt.Sscanf(numStr, "%f", &num)

		if op == 0 {
			result = num
		} else {
			switch op {
			case '+':
				result += num
			case '-':
				result -= num
			case '*':
				result *= num
			case '/':
				if num == 0 {
					return 0, fmt.Errorf("division by zero")
				}
				result /= num
			}
		}
	}

	return result, nil
}
