package dingtalk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

const (
	dingtalkAPIBase = "https://oapi.dingtalk.com"
)

// APIClient provides DingTalk API access.
type APIClient struct {
	clientID     string
	clientSecret string
	logger       *slog.Logger

	accessToken     string
	tokenExpireTime time.Time
	tokenMu         sync.RWMutex
	httpClient      *http.Client
}

// NewAPIClient creates a new DingTalk API client.
func NewAPIClient(clientID, clientSecret string, logger *slog.Logger) *APIClient {
	return &APIClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		logger:       logger,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

// GetAccessToken gets the access token for API calls.
func (c *APIClient) GetAccessToken(ctx context.Context) (string, error) {
	c.tokenMu.RLock()
	if c.accessToken != "" && time.Now().Before(c.tokenExpireTime) {
		token := c.accessToken
		c.tokenMu.RUnlock()
		return token, nil
	}
	c.tokenMu.RUnlock()

	return c.refreshAccessToken(ctx)
}

// refreshAccessToken refreshes the access token.
func (c *APIClient) refreshAccessToken(ctx context.Context) (string, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	// Double check after acquiring write lock
	if c.accessToken != "" && time.Now().Before(c.tokenExpireTime) {
		return c.accessToken, nil
	}

	// Build request URL
	apiURL := fmt.Sprintf("%s/gettoken?appkey=%s&appsecret=%s",
		dingtalkAPIBase,
		url.QueryEscape(c.clientID),
		url.QueryEscape(c.clientSecret))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if result.ErrCode != 0 {
		return "", fmt.Errorf("API error (code=%d msg=%s)", result.ErrCode, result.ErrMsg)
	}

	c.accessToken = result.AccessToken
	// Set expire time with 5 minute buffer
	c.tokenExpireTime = time.Now().Add(time.Duration(result.ExpiresIn-300) * time.Second)

	return c.accessToken, nil
}

// SendMessage sends a message to a user or chat.
func (c *APIClient) SendMessage(ctx context.Context, agentID int64, userID, msgType, msgContent string) error {
	token, err := c.GetAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("get access token: %w", err)
	}

	// Build request body
	body := map[string]any{
		"agent_id": agentID,
		"userid":   userID,
		"msgtype":  msgType,
		msgType:    json.RawMessage(msgContent),
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	apiURL := fmt.Sprintf("%s/topapi/message/corpconversation/asyncsend_v2?access_token=%s",
		dingtalkAPIBase, token)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
		TaskID  int64  `json:"task_id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if result.ErrCode != 0 {
		return fmt.Errorf("API error (code=%d msg=%s)", result.ErrCode, result.ErrMsg)
	}

	return nil
}

// UploadMedia uploads a media file to DingTalk.
func (c *APIClient) UploadMedia(ctx context.Context, filePath, mediaType string) (string, error) {
	token, err := c.GetAccessToken(ctx)
	if err != nil {
		return "", fmt.Errorf("get access token: %w", err)
	}

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add type field
	if err := writer.WriteField("type", mediaType); err != nil {
		return "", fmt.Errorf("write type field: %w", err)
	}

	// Add media file
	part, err := writer.CreateFormFile("media", filePath)
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return "", fmt.Errorf("copy file: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close writer: %w", err)
	}

	// Build request
	apiURL := fmt.Sprintf("%s/media/upload?access_token=%s", dingtalkAPIBase, token)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, &buf)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		ErrCode  int    `json:"errcode"`
		ErrMsg   string `json:"errmsg"`
		MediaID  string `json:"media_id"`
		Type     string `json:"type"`
		CreateAt int64  `json:"created_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if result.ErrCode != 0 {
		return "", fmt.Errorf("API error (code=%d msg=%s)", result.ErrCode, result.ErrMsg)
	}

	return result.MediaID, nil
}

// GetUserInfo gets user information by user ID.
func (c *APIClient) GetUserInfo(ctx context.Context, userID string) (*UserInfo, error) {
	token, err := c.GetAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	apiURL := fmt.Sprintf("%s/topapi/v2/user/get?access_token=%s", dingtalkAPIBase, token)

	body := map[string]string{"userid": userID}
	bodyBytes, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
		Result  struct {
			UserID   string `json:"userid"`
			Name     string `json:"name"`
			Avatar   string `json:"avatar"`
			Mobile   string `json:"mobile"`
			Email    string `json:"email"`
			DeptID   int64  `json:"dept_id"`
			Title    string `json:"title"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if result.ErrCode != 0 {
		return nil, fmt.Errorf("API error (code=%d msg=%s)", result.ErrCode, result.ErrMsg)
	}

	return &UserInfo{
		UserID: result.Result.UserID,
		Name:   result.Result.Name,
		Avatar: result.Result.Avatar,
		Mobile: result.Result.Mobile,
		Email:  result.Result.Email,
		Title:  result.Result.Title,
	}, nil
}

// UserInfo contains user information.
type UserInfo struct {
	UserID string
	Name   string
	Avatar string
	Mobile string
	Email  string
	Title  string
}