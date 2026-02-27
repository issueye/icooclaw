package builtons

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/icooclaw/icooclaw/internal/script/config"
)

// === http 客户端对象 ===

type httpClient struct {
	cfg    *config.Config
	logger *slog.Logger
}

func NewHTTPClient(cfg *config.Config, logger *slog.Logger) *httpClient {
	return &httpClient{
		cfg:    cfg,
		logger: logger,
	}
}

// Get 发送 GET 请求
func (h *httpClient) Get(url string) (map[string]interface{}, error) {
	return h.Request("GET", url, nil, nil)
}

// Post 发送 POST 请求
func (h *httpClient) Post(url string, body interface{}) (map[string]interface{}, error) {
	return h.Request("POST", url, body, nil)
}

// PostJSON 发送 JSON POST 请求
func (h *httpClient) PostJSON(url string, body interface{}) (map[string]interface{}, error) {
	headers := map[string]string{"Content-Type": "application/json"}
	return h.Request("POST", url, body, headers)
}

// Request 发送 HTTP 请求
func (h *httpClient) Request(method, url string, body interface{}, headers map[string]string) (map[string]interface{}, error) {
	if !h.cfg.AllowNetwork {
		return nil, fmt.Errorf("network access is not allowed")
	}

	// 检查域名白名单
	if len(h.cfg.AllowedDomains) > 0 {
		allowed := false
		for _, domain := range h.cfg.AllowedDomains {
			if strings.Contains(url, domain) {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, fmt.Errorf("domain not in whitelist")
		}
	}

	timeout := time.Duration(h.cfg.HTTPTimeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	client := &http.Client{Timeout: timeout}

	var reqBody io.Reader
	if body != nil {
		switch v := body.(type) {
		case string:
			reqBody = strings.NewReader(v)
		case []byte:
			reqBody = bytes.NewReader(v)
		default:
			jsonBody, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal body: %w", err)
			}
			reqBody = bytes.NewReader(jsonBody)
		}
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	if headers == nil {
		headers = make(map[string]string)
	}
	if _, ok := headers["Content-Type"]; !ok && body != nil {
		headers["Content-Type"] = "application/json"
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	result := map[string]interface{}{
		"status_code": resp.StatusCode,
		"status":      resp.Status,
		"headers":     resp.Header,
		"body":        string(respBody),
		"ok":          resp.StatusCode >= 200 && resp.StatusCode < 300,
	}

	// 尝试解析 JSON
	var jsonBody interface{}
	if err := json.Unmarshal(respBody, &jsonBody); err == nil {
		result["json"] = jsonBody
	}

	return result, nil
}

// Download 下载文件
func (h *httpClient) Download(url string, savePath string) error {
	if !h.cfg.AllowNetwork || !h.cfg.AllowFileWrite {
		return fmt.Errorf("operation not allowed")
	}

	timeout := time.Duration(h.cfg.HTTPTimeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	client := &http.Client{Timeout: timeout}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// 确保目录存在
	dir := filepath.Dir(savePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	file, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}
