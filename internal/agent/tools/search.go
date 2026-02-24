package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// WebSearchToolConfig 网络搜索工具配置
type WebSearchToolConfig struct {
	// API 配置
	BraveAPIKey    string // Brave Search API Key (可选)
	GoogleAPIKey   string // Google Custom Search API Key (可选)
	GoogleEngineID string // Google Custom Search Engine ID (可选)

	// 默认配置
	DefaultNumResults int    // 默认结果数量
	Timeout           int    // 超时时间(秒)
	UserAgent         string // User-Agent
}

// DefaultWebSearchToolConfig 默认配置
var DefaultWebSearchToolConfig = WebSearchToolConfig{
	DefaultNumResults: 5,
	Timeout:           30,
	UserAgent:         "icooclaw/1.0 (Web Search Tool)",
}

// WebSearchTool 网络搜索工具
type WebSearchTool struct {
	baseTool   *BaseTool
	config     *WebSearchToolConfig
	httpClient *http.Client
}

// NewWebSearchTool 创建网络搜索工具
func NewWebSearchToolWithConfig(config WebSearchToolConfig) *WebSearchTool {
	if config.DefaultNumResults == 0 {
		config.DefaultNumResults = 5
	}
	if config.Timeout == 0 {
		config.Timeout = 30
	}
	if config.UserAgent == "" {
		config.UserAgent = "icooclaw/1.0 (Web Search Tool)"
	}

	tool := NewBaseTool(
		"web_search",
		"在网络上搜索信息。支持 DuckDuckGo（免费）、Brave Search 和 Google Custom Search。",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "搜索关键词",
				},
				"num_results": map[string]interface{}{
					"type":        "number",
					"description": "返回结果数量 (1-20)",
					"default":     5,
				},
				"engine": map[string]interface{}{
					"type":        "string",
					"description": "搜索引擎: duckduckgo, brave, google (默认: duckduckgo)",
					"default":     "duckduckgo",
				},
				"safe_search": map[string]interface{}{
					"type":        "boolean",
					"description": "安全搜索 (仅 Brave/Google 支持)",
					"default":     true,
				},
			},
			"required": []string{"query"},
		},
		nil,
	)

	return &WebSearchTool{
		baseTool:   tool,
		config:     &config,
		httpClient: &http.Client{Timeout: time.Duration(config.Timeout) * time.Second},
	}
}

// NewWebSearchTool 创建网络搜索工具（默认配置）
func NewWebSearchTool() *WebSearchTool {
	return NewWebSearchToolWithConfig(DefaultWebSearchToolConfig)
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

	numResults := t.config.DefaultNumResults
	if n, ok := params["num_results"].(float64); ok {
		numResults = int(n)
		if numResults < 1 {
			numResults = 1
		}
		if numResults > 20 {
			numResults = 20
		}
	}

	engine := "duckduckgo"
	if e, ok := params["engine"].(string); ok && e != "" {
		engine = e
	}

	var results []map[string]interface{}
	var err error

	switch engine {
	case "brave":
		results, err = t.searchBrave(ctx, query, numResults)
	case "google":
		results, err = t.searchGoogle(ctx, query, numResults)
	case "duckduckgo":
		fallthrough
	default:
		results, err = t.searchDuckDuckGo(ctx, query, numResults)
	}

	if err != nil {
		return "", err
	}

	// 构建结果
	result := map[string]interface{}{
		"query":       query,
		"engine":      engine,
		"num_results": len(results),
		"results":     results,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// searchDuckDuckGo 使用 DuckDuckGo 搜索
func (t *WebSearchTool) searchDuckDuckGo(ctx context.Context, query string, numResults int) ([]map[string]interface{}, error) {
	// 使用 DuckDuckGo HTML API
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s&b=%d", url.QueryEscape(query), numResults)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", t.config.UserAgent)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search failed with status: %d", resp.StatusCode)
	}

	// 解析 HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var results []map[string]interface{}

	doc.Find(".result").EachWithBreak(func(i int, s *goquery.Selection) bool {
		if i >= numResults {
			return false
		}

		title := s.Find(".result__title").Text()
		title = strings.TrimSpace(title)

		linkNode := s.Find(".result__url")
		link := linkNode.Text()
		link = strings.TrimSpace(link)

		// 获取完整 URL
		if href, ok := s.Find(".result__a").Attr("href"); ok {
			// DuckDuckGo 使用重定向 URL
			if strings.Contains(href, "uddg=") {
				if decoded, err := url.QueryUnescape(strings.Split(href, "uddg=")[1]); err == nil {
					link = decoded
				}
			} else {
				link = href
			}
		}

		snippet := s.Find(".result__snippet").Text()
		snippet = strings.TrimSpace(snippet)

		if title != "" && link != "" {
			results = append(results, map[string]interface{}{
				"title":   title,
				"url":     link,
				"snippet": snippet,
				"source":  "DuckDuckGo",
			})
		}

		return true
	})

	return results, nil
}

// searchBrave 使用 Brave Search API
func (t *WebSearchTool) searchBrave(ctx context.Context, query string, numResults int) ([]map[string]interface{}, error) {
	if t.config.BraveAPIKey == "" {
		return nil, fmt.Errorf("Brave Search API key not configured")
	}

	searchURL := fmt.Sprintf("https://api.search.brave.com/res/v1/web/search?q=%s&count=%d",
		url.QueryEscape(query), numResults)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", t.config.UserAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", t.config.BraveAPIKey)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Brave Search API error: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		Web struct {
			Results []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Description string `json:"description"`
			} `json:"results"`
		} `json:"web"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var results []map[string]interface{}
	for _, r := range result.Web.Results {
		results = append(results, map[string]interface{}{
			"title":   r.Title,
			"url":     r.URL,
			"snippet": r.Description,
			"source":  "Brave Search",
		})
	}

	return results, nil
}

// searchGoogle 使用 Google Custom Search API
func (t *WebSearchTool) searchGoogle(ctx context.Context, query string, numResults int) ([]map[string]interface{}, error) {
	if t.config.GoogleAPIKey == "" || t.config.GoogleEngineID == "" {
		return nil, fmt.Errorf("Google Custom Search API key or Engine ID not configured")
	}

	searchURL := fmt.Sprintf("https://www.googleapis.com/customsearch/v1?key=%s&cx=%s&q=%s&num=%d",
		url.QueryEscape(t.config.GoogleAPIKey),
		url.QueryEscape(t.config.GoogleEngineID),
		url.QueryEscape(query),
		numResults)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", t.config.UserAgent)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Google Search API error: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		Items []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var results []map[string]interface{}
	for _, r := range result.Items {
		results = append(results, map[string]interface{}{
			"title":   r.Title,
			"url":     r.Link,
			"snippet": r.Snippet,
			"source":  "Google Search",
		})
	}

	return results, nil
}

// ToDefinition 转换为工具定义
func (t *WebSearchTool) ToDefinition() ToolDefinition {
	return t.baseTool.ToDefinition()
}

// WebFetchToolConfig 网页获取工具配置
type WebFetchToolConfig struct {
	Timeout        int               // 超时时间(秒)
	UserAgent      string            // User-Agent
	MaxContentSize int64             // 最大内容大小(字节)
	AllowedDomains []string          // 允许的域名 (为空则不限制)
	BlockedDomains []string          // 禁止的域名
	Headers        map[string]string // 默认请求头
	ProxyURL       string            // 代理 URL
}

// DefaultWebFetchToolConfig 默认配置
var DefaultWebFetchToolConfig = WebFetchToolConfig{
	Timeout:        30,
	UserAgent:      "icooclaw/1.0 (Web Fetch Tool)",
	MaxContentSize: 2 * 1024 * 1024, // 2MB
}

// WebFetchTool 网页获取工具
type WebFetchTool struct {
	baseTool   *BaseTool
	config     *WebFetchToolConfig
	httpClient *http.Client
}

// NewWebFetchToolWithConfig 创建网页获取工具（带配置）
func NewWebFetchToolWithConfig(config WebFetchToolConfig) *WebFetchTool {
	if config.Timeout == 0 {
		config.Timeout = 30
	}
	if config.UserAgent == "" {
		config.UserAgent = "icooclaw/1.0 (Web Fetch Tool)"
	}
	if config.MaxContentSize == 0 {
		config.MaxContentSize = 2 * 1024 * 1024
	}

	tool := NewBaseTool(
		"web_fetch",
		"从 URL 获取网页内容。支持提取纯文本、JSON 解析、CSS 选择器提取、自动内容摘要等功能。",
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
				"max_length": map[string]interface{}{
					"type":        "number",
					"description": "返回内容最大长度（字符数），0 表示不限制",
					"default":     10000,
				},
				"selector": map[string]interface{}{
					"type":        "string",
					"description": "CSS 选择器，用于提取特定元素（如: '.article-content', '#main-text'）",
				},
				"query": map[string]interface{}{
					"type":        "string",
					"description": "JSONPath 查询（当响应为 JSON 时使用）",
				},
				"headers": map[string]interface{}{
					"type":        "object",
					"description": "自定义请求头",
				},
			},
			"required": []string{"url"},
		},
		nil,
	)

	return &WebFetchTool{
		baseTool:   tool,
		config:     &config,
		httpClient: &http.Client{Timeout: time.Duration(config.Timeout) * time.Second},
	}
}

// NewWebFetchTool 创建网页获取工具（默认配置）
func NewWebFetchTool() *WebFetchTool {
	return NewWebFetchToolWithConfig(DefaultWebFetchToolConfig)
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
	urlStr, ok := params["url"].(string)
	if !ok || urlStr == "" {
		return "", fmt.Errorf("invalid or missing url")
	}

	// 验证 URL
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		urlStr = "https://" + urlStr
	}

	// 检查域名是否允许
	if err := t.checkDomain(urlStr); err != nil {
		return "", err
	}

	extractText := true
	if e, ok := params["extract_text"].(bool); ok {
		extractText = e
	}

	maxLength := 10000
	if ml, ok := params["max_length"].(float64); ok {
		maxLength = int(ml)
	}

	selector := ""
	if s, ok := params["selector"].(string); ok {
		selector = s
	}

	jsonQuery := ""
	if q, ok := params["query"].(string); ok {
		jsonQuery = q
	}

	// 构建请求头
	headers := make(map[string]string)
	if t.config.Headers != nil {
		for k, v := range t.config.Headers {
			headers[k] = v
		}
	}
	headers["User-Agent"] = t.config.UserAgent

	// 添加自定义请求头
	if h, ok := params["headers"].(map[string]interface{}); ok {
		for k, v := range h {
			if vv, ok := v.(string); ok {
				headers[k] = vv
			}
		}
	}

	// 发送请求
	body, statusCode, contentType, err := t.fetchURL(ctx, urlStr, headers)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}

	// 检查内容大小
	if int64(len(body)) > t.config.MaxContentSize {
		return "", fmt.Errorf("content too large: %d bytes (max: %d)", len(body), t.config.MaxContentSize)
	}

	var content string

	// 根据内容类型处理
	if strings.Contains(contentType, "application/json") || jsonQuery != "" {
		// JSON 处理
		content, err = t.extractJSON(string(body), jsonQuery)
		if err != nil {
			return "", fmt.Errorf("failed to extract JSON: %w", err)
		}
	} else if strings.Contains(contentType, "text/html") || selector != "" {
		// HTML 处理
		content, err = t.extractHTML(string(body), selector, extractText)
		if err != nil {
			return "", fmt.Errorf("failed to extract HTML: %w", err)
		}
	} else {
		// 纯文本
		content = string(body)
	}

	// 截断内容
	if maxLength > 0 && len(content) > maxLength {
		content = t.truncateContent(content, maxLength)
	}

	// 构建结果
	result := map[string]interface{}{
		"url":          urlStr,
		"status_code":  statusCode,
		"content_type": contentType,
		"content":      content,
		"length":       len(content),
		"truncated":    maxLength > 0 && len(content) > maxLength,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// fetchURL 获取 URL 内容
func (t *WebFetchTool) fetchURL(ctx context.Context, urlStr string, headers map[string]string) ([]byte, int, string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, 0, "", err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, 0, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, "", err
	}

	return body, resp.StatusCode, resp.Header.Get("Content-Type"), nil
}

// checkDomain 检查域名是否允许
func (t *WebFetchTool) checkDomain(urlStr string) error {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	host := parsedURL.Host

	// 检查禁止列表
	for _, blocked := range t.config.BlockedDomains {
		if strings.Contains(host, blocked) {
			return fmt.Errorf("domain is blocked: %s", host)
		}
	}

	// 检查允许列表
	if len(t.config.AllowedDomains) > 0 {
		allowed := false
		for _, allowedDomain := range t.config.AllowedDomains {
			if strings.Contains(host, allowedDomain) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("domain not in allowed list: %s", host)
		}
	}

	return nil
}

// extractHTML 从 HTML 提取内容
func (t *WebFetchTool) extractHTML(html, selector string, extractText bool) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", err
	}

	var content string

	if selector != "" {
		// 使用 CSS 选择器
		selection := doc.Find(selector)
		if selection.Length() == 0 {
			return "", fmt.Errorf("no elements found with selector: %s", selector)
		}

		if extractText {
			content = selection.Text()
		} else {
			content, _ = selection.Html()
		}
	} else {
		// 默认提取文本
		if extractText {
			content = extractTextFromHTML(html)
		} else {
			content = html
		}
	}

	return content, nil
}

// extractJSON 从 JSON 提取内容
func (t *WebFetchTool) extractJSON(body, query string) (string, error) {
	var data interface{}
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return "", err
	}

	// 简单的 JSON 路径查询
	if query != "" {
		result := t.queryJSON(data, strings.Split(query, "."))
		if result == nil {
			return "", fmt.Errorf("JSON path not found: %s", query)
		}
		return t.jsonToString(result), nil
	}

	// 默认返回格式化 JSON
	formatted, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(formatted), nil
}

// queryJSON 简单的 JSON 路径查询
func (t *WebFetchTool) queryJSON(data interface{}, path []string) interface{} {
	if len(path) == 0 {
		return data
	}

	current := data
	for _, key := range path {
		if key == "" {
			continue
		}

		switch v := current.(type) {
		case map[string]interface{}:
			if val, ok := v[key]; ok {
				current = val
			} else {
				return nil
			}
		case []interface{}:
			// 支持数组索引
			if idx := regexp.MustCompile(`(\w+)\[(\d+)\]`).FindStringSubmatch(key); len(idx) == 3 {
				key = idx[1]
				if key != "" {
					if val, ok := v[0].(map[string]interface{})[key]; ok {
						current = val
					} else {
						return nil
					}
				}
				current = v
			}
			// 直接数字索引
			if i := strings.Index(key, "["); i != -1 {
				arrKey := key[:i]
				idxStr := key[i+1 : len(key)-1]
				idx, _ := strconv.Atoi(idxStr)
				if arrKey == "" && idx < len(v) {
					current = v[idx]
				} else if val, ok := v[0].(map[string]interface{})[arrKey]; ok {
					current = val
				} else {
					return nil
				}
			}
		default:
			return nil
		}
	}

	return current
}

// jsonToString 将 JSON 值转换为字符串
func (t *WebFetchTool) jsonToString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case bool, float64, int:
		return fmt.Sprintf("%v", val)
	case []interface{}:
		parts := make([]string, len(val))
		for i, item := range val {
			parts[i] = t.jsonToString(item)
		}
		return strings.Join(parts, ", ")
	case map[string]interface{}:
		parts := make([]string, 0)
		for k, v := range val {
			parts = append(parts, fmt.Sprintf("%s: %s", k, t.jsonToString(v)))
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", v)
	}
}

// truncateContent 截断内容
func (t *WebFetchTool) truncateContent(content string, maxLength int) string {
	if len(content) <= maxLength {
		return content
	}

	// 在单词边界截断
	truncated := content[:maxLength]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > maxLength/2 {
		truncated = truncated[:lastSpace]
	}

	return truncated + "... (truncated)"
}

// extractTextFromHTML 从 HTML 提取纯文本
func extractTextFromHTML(html string) string {
	// 移除 script 和 style 标签及其内容
	text := removeTag(html, "script")
	text = removeTag(text, "style")

	// 替换常见的 HTML 实体
	replacements := map[string]string{
		"&nbsp;": " ", "&amp;": "&", "&lt;": "<", "&gt;": ">",
		"&quot;": `"`, "&#39;": "'", "<br>": "\n", "<br/>": "\n",
		"<br />": "\n", "</p>": "\n\n", "</div>": "\n", "</li>": "\n",
		"</tr>": "\n", "</td>": "\t", "</th>": "\t",
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
