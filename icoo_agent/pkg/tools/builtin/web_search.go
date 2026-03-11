package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"icooclaw/pkg/tools"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

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
func (t *WebSearchTool) Parameters() map[string]any {
	return map[string]any{
		"query": map[string]any{
			"type":        "string",
			"description": "The search query",
		},
		"max_results": map[string]any{
			"type":        "integer",
			"description": "Maximum number of results (default: 5)",
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
