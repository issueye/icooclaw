package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// ============ 用户 API ============

// GetUserByID 根据 ID 获取用户信息
// idType: open_id, user_id, union_id
func (c *Channel) GetUserByID(ctx context.Context, userID, idType string) (*User, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取 token 失败: %w", err)
	}

	url := fmt.Sprintf("https://open.feishu.cn/open-apis/contact/v3/users/%s?user_id_type=%s", userID, idType)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data *struct {
			User *User `json:"user"`
		} `json:"data"`
	}

	err = doWithRetry(ctx, func() error {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return err
		}

		if result.Code != 0 {
			return NewFeishuError(result.Code, result.Msg)
		}
		return nil
	}, DefaultRetryConfig, func(attempt int, err error) {
		c.logger.Warn("获取用户信息失败，准备重试", "attempt", attempt, "error", err)
	})

	if err != nil {
		return nil, err
	}

	if result.Data == nil || result.Data.User == nil {
		return nil, fmt.Errorf("用户不存在: %s", userID)
	}

	return result.Data.User, nil
}

// GetUserInfo 批量获取用户信息
func (c *Channel) GetUserInfo(ctx context.Context, userIDs []string, idType string) ([]*User, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取 token 失败: %w", err)
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"user_ids":     userIDs,
		"user_id_type": idType,
	})

	url := "https://open.feishu.cn/open-apis/contact/v3/users/batch/get_id"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data *struct {
			Users []*User `json:"users"`
		} `json:"data"`
	}

	err = doWithRetry(ctx, func() error {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return err
		}

		if result.Code != 0 {
			return NewFeishuError(result.Code, result.Msg)
		}
		return nil
	}, DefaultRetryConfig, nil)

	if err != nil {
		return nil, err
	}

	if result.Data == nil {
		return nil, nil
	}

	return result.Data.Users, nil
}

// ============ 用户缓存 ============

// UserCache 用户缓存
type UserCache struct {
	users map[string]*cachedUser
	ttl   time.Duration
	mu    sync.RWMutex
}

type cachedUser struct {
	user      *User
	expiresAt time.Time
}

// NewUserCache 创建用户缓存
func NewUserCache(ttl time.Duration) *UserCache {
	return &UserCache{
		users: make(map[string]*cachedUser),
		ttl:   ttl,
	}
}

// Get 获取缓存的用户
func (c *UserCache) Get(openID string) *User {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, ok := c.users[openID]
	if !ok || time.Now().After(cached.expiresAt) {
		return nil
	}
	return cached.user
}

// Set 设置用户缓存
func (c *UserCache) Set(openID string, user *User) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.users[openID] = &cachedUser{
		user:      user,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Delete 删除用户缓存
func (c *UserCache) Delete(openID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.users, openID)
}

// Clear 清空缓存
func (c *UserCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.users = make(map[string]*cachedUser)
}

// GetUserWithCache 带缓存的获取用户信息
func (c *Channel) GetUserWithCache(ctx context.Context, openID string) (*User, error) {
	// 先查缓存
	if c.userCache != nil {
		if user := c.userCache.Get(openID); user != nil {
			return user, nil
		}
	}

	// 调用 API
	user, err := c.GetUserByID(ctx, openID, "open_id")
	if err != nil {
		return nil, err
	}

	// 写入缓存
	if c.userCache != nil {
		c.userCache.Set(openID, user)
	}

	return user, nil
}

// ============ 群聊 API ============

// ChatInfo 群聊信息
type ChatInfo struct {
	ChatID          string   `json:"chat_id"`
	Name            string   `json:"name"`
	Description     string   `json:"description,omitempty"`
	OwnerID         string   `json:"owner_id,omitempty"`
	OwnerIDType     string   `json:"owner_id_type,omitempty"`
	MemberCount     int      `json:"member_count,omitempty"`
	ChatMode        string   `json:"chat_mode,omitempty"`     // group, p2p
	ChatType        string   `json:"chat_type,omitempty"`     // public, private
	JoinMessage     string   `json:"join_message,omitempty"`
	LeaveMessage    string   `json:"leave_message,omitempty"`
	TenantKey       string   `json:"tenant_key,omitempty"`
	UserIDList      []string `json:"user_id_list,omitempty"`
	GroupMessage    string   `json:"group_message,omitempty"`
	GroupMessageID  string   `json:"group_message_id,omitempty"`
}

// GetChatInfo 获取群聊信息
func (c *Channel) GetChatInfo(ctx context.Context, chatID string) (*ChatInfo, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取 token 失败: %w", err)
	}

	url := fmt.Sprintf("https://open.feishu.cn/open-apis/im/v1/chats/%s?user_id_type=open_id", chatID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data *struct {
			ChatInfo *ChatInfo `json:"chat_info,omitempty"`
		} `json:"data"`
	}

	err = doWithRetry(ctx, func() error {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return err
		}

		if result.Code != 0 {
			return NewFeishuError(result.Code, result.Msg)
		}
		return nil
	}, DefaultRetryConfig, nil)

	if err != nil {
		return nil, err
	}

	if result.Data == nil || result.Data.ChatInfo == nil {
		return nil, fmt.Errorf("群聊不存在: %s", chatID)
	}

	return result.Data.ChatInfo, nil
}

// GetChatMembers 获取群成员列表
func (c *Channel) GetChatMembers(ctx context.Context, chatID string, pageSize int, pageToken string) ([]*Member, string, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("获取 token 失败: %w", err)
	}

	url := fmt.Sprintf("https://open.feishu.cn/open-apis/im/v1/chats/%s/members?user_id_type=open_id&member_id_type=open_id", chatID)
	if pageSize > 0 {
		url += fmt.Sprintf("&page_size=%d", pageSize)
	}
	if pageToken != "" {
		url += fmt.Sprintf("&page_token=%s", pageToken)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data *struct {
			Items     []*Member `json:"items,omitempty"`
			PageToken string    `json:"page_token,omitempty"`
			HasMore   bool      `json:"has_more"`
		} `json:"data"`
	}

	err = doWithRetry(ctx, func() error {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return err
		}

		if result.Code != 0 {
			return NewFeishuError(result.Code, result.Msg)
		}
		return nil
	}, DefaultRetryConfig, nil)

	if err != nil {
		return nil, "", err
	}

	if result.Data == nil {
		return nil, "", nil
	}

	return result.Data.Items, result.Data.PageToken, nil
}