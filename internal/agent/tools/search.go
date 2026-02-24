package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// WebSearchTool 网络搜索工具
type WebSearchTool struct {
	baseTool *BaseTool
}

// NewWebSearchTool 创建网络搜索工具
func NewWebSearchTool() *WebSearchTool {
	tool := NewBaseTool(
		"web_search",
		"在网络上搜索信息。使用搜索引擎查找相关内容。",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "搜索关键词",
				},
				"num_results": map[string]interface{}{
					"type":        "number",
					"description": "返回结果数量 (1-10)",
					"default":     5,
				},
			},
			"required": []string{"query"},
		},
		nil,
	)

	return &WebSearchTool{
		baseTool: tool,
	}
}

// Name 获取名称
func (t *WebSearchTool) Name() string {
	return t.baseTool.Name()
}

// Description 获取描述
func (t *WebSearchTool) Description() string {
	return t.baseTool.Description()
}

// Parameters 获取参数定义
func (t *WebSearchTool) Parameters() map[string]interface{} {
	return t.baseTool.Parameters()
}

// Execute 执行搜索
func (t *WebSearchTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return "", fmt.Errorf("invalid or missing query")
	}

	numResults := 5
	if n, ok := params["num_results"].(float64); ok {
		numResults = int(n)
		if numResults < 1 {
			numResults = 1
		}
		if numResults > 10 {
			numResults = 10
		}
	}

	// 使用内置搜索（这里可以接入真实的搜索 API）
	// 为了演示，返回模拟结果
	results := []map[string]interface{}{
		{
			"title":     "搜索结果: " + query,
			"url":       "https://example.com/search?q=" + strings.ReplaceAll(query, " ", "+"),
			"snippet":   "这是关于 '" + query + "' 的搜索结果摘要。",
			"source":    "Web Search",
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}

	// 构建结果
	result := map[string]interface{}{
		"query":       query,
		"num_results": len(results),
		"results":     results,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// ToDefinition 转换为工具定义
func (t *WebSearchTool) ToDefinition() ToolDefinition {
	return t.baseTool.ToDefinition()
}

// WebFetchTool 网页获取工具
type WebFetchTool struct {
	baseTool *BaseTool
}

// NewWebFetchTool 创建网页获取工具
func NewWebFetchTool() *WebFetchTool {
	tool := NewBaseTool(
		"web_fetch",
		"从 URL 获取网页内容。可以用于获取网页的 HTML 或文本内容。",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "要获取的 URL",
				},
				"extract_text": map[string]interface{}{
					"type":        "boolean",
					"description": "是否提取纯文本（去除 HTML 标签）",
					"default":     true,
				},
			},
			"required": []string{"url"},
		},
		nil,
	)

	return &WebFetchTool{
		baseTool: tool,
	}
}

// Name 获取名称
func (t *WebFetchTool) Name() string {
	return t.baseTool.Name()
}

// Description 获取描述
func (t *WebFetchTool) Description() string {
	return t.baseTool.Description()
}

// Parameters 获取参数定义
func (t *WebFetchTool) Parameters() map[string]interface{} {
	return t.baseTool.Parameters()
}

// Execute 获取网页
func (t *WebFetchTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	url, ok := params["url"].(string)
	if !ok || url == "" {
		return "", fmt.Errorf("invalid or missing url")
	}

	// 验证 URL
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	extractText := true
	if e, ok := params["extract_text"].(bool); ok {
		extractText = e
	}

	// 使用 HTTP 客户端获取内容
	client := &HTTPRequestTool{}
	result, err := client.Execute(ctx, map[string]interface{}{
		"url":    url,
		"method": "GET",
		"headers": map[string]string{
			"User-Agent": "icooclaw/1.0",
		},
		"timeout": 30,
	})

	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}

	// 解析结果
	var respData map[string]interface{}
	if err := json.Unmarshal([]byte(result), &respData); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// 提取内容
	content := ""
	if body, ok := respData["body"].(string); ok {
		if extractText {
			content = extractTextFromHTML(body)
		} else {
			content = body
		}
	}

	// 构建结果
	output := map[string]interface{}{
		"url":         url,
		"status_code": respData["status_code"],
		"content":     content,
		"length":      len(content),
	}

	if !extractText {
		output["html"] = content
	}

	outputJSON, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(outputJSON), nil
}

// extractTextFromHTML 从 HTML 提取纯文本
func extractTextFromHTML(html string) string {
	// 简单的 HTML 标签移除
	text := html

	// 移除 script 和 style 标签及其内容
	text = removeTag(text, "script")
	text = removeTag(text, "style")

	// 替换常见的 HTML 实体
	replacements := map[string]string{
		"&nbsp;": " ",
		"&amp;":  "&",
		"&lt;":   "<",
		"&gt;":   ">",
		"&quot;": `"`,
		"&#39;":  "'",
		"<br>":   "\n",
		"<br/>":  "\n",
		"<br />": "\n",
		"</p>":   "\n\n",
		"</div>": "\n",
		"</li>":  "\n",
		"</tr>":  "\n",
		"</td>":  "\t",
		"</th>":  "\t",
	}

	for k, v := range replacements {
		text = strings.ReplaceAll(text, k, v)
	}

	// 移除所有 HTML 标签
	var result strings.Builder
	inTag := false
	for _, c := range text {
		if c == '<' {
			inTag = true
		} else if c == '>' {
			inTag = false
		} else if !inTag {
			result.WriteRune(c)
		}
	}

	// 清理空白
	lines := strings.Split(result.String(), "\n")
	var cleaned []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}

	return strings.Join(cleaned, "\n")
}

// removeTag 移除指定标签及其内容
func removeTag(html, tag string) string {
	start := 0
	for {
		i := strings.Index(html[start:], "<"+tag)
		if i == -1 {
			break
		}
		i += start

		// 找到结束标签
		j := strings.Index(html[i:], "</"+tag+">")
		if j == -1 {
			break
		}
		j += i + len("</"+tag+">")

		html = html[:i] + html[j:]
		start = i
	}
	return html
}

// ToDefinition 转换为工具定义
func (t *WebFetchTool) ToDefinition() ToolDefinition {
	return t.baseTool.ToDefinition()
}
