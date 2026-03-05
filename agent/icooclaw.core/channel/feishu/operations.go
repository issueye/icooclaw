package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ============ 消息操作 API ============

// RecallMessage 撤回消息
func (c *Channel) RecallMessage(ctx context.Context, messageID string) error {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://open.feishu.cn/open-apis/im/v1/messages/%s/recall", messageID)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	return doWithRetry(ctx, func() error {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		var result struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return err
		}
		if result.Code != 0 {
			return NewFeishuError(result.Code, result.Msg)
		}
		return nil
	}, DefaultRetryConfig, nil)
}

// UpdateMessage 更新消息内容（仅支持卡片消息）
func (c *Channel) UpdateMessage(ctx context.Context, messageID string, card *CardMessage) error {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return err
	}

	cardJSON, _ := json.Marshal(card)
	payload, _ := json.Marshal(map[string]interface{}{
		"content": string(cardJSON),
	})

	url := fmt.Sprintf("https://open.feishu.cn/open-apis/im/v1/messages/%s?msg_type=interactive", messageID)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	return doWithRetry(ctx, func() error {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		var result struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return err
		}
		if result.Code != 0 {
			return NewFeishuError(result.Code, result.Msg)
		}
		return nil
	}, DefaultRetryConfig, nil)
}

// ReplyMessage 回复消息（引用原消息）
func (c *Channel) ReplyMessage(ctx context.Context, messageID, content string) error {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return err
	}

	contentJSON, _ := json.Marshal(map[string]string{"text": content})
	payload, _ := json.Marshal(map[string]interface{}{
		"content":       string(contentJSON),
		"msg_type":      "text",
		"reply_in_chat": true,
	})

	url := fmt.Sprintf("https://open.feishu.cn/open-apis/im/v1/messages/%s/reply", messageID)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	return doWithRetry(ctx, func() error {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		var result struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return err
		}
		if result.Code != 0 {
			return NewFeishuError(result.Code, result.Msg)
		}
		return nil
	}, DefaultRetryConfig, nil)
}

// ReplyCard 回复卡片消息
func (c *Channel) ReplyCard(ctx context.Context, messageID string, card *CardMessage) error {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return err
	}

	cardJSON, _ := json.Marshal(card)
	payload, _ := json.Marshal(map[string]interface{}{
		"content":       string(cardJSON),
		"msg_type":      "interactive",
		"reply_in_chat": true,
	})

	url := fmt.Sprintf("https://open.feishu.cn/open-apis/im/v1/messages/%s/reply", messageID)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	return doWithRetry(ctx, func() error {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		var result struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return err
		}
		if result.Code != 0 {
			return NewFeishuError(result.Code, result.Msg)
		}
		return nil
	}, DefaultRetryConfig, nil)
}

// ============ 表情回复 ============

// ReactionContent 表情回复内容
type ReactionContent struct {
	ReactionType ReactionType `json:"reaction_type"`
}

// ReactionType 表情类型
type ReactionType struct {
	EmojiType string `json:"emoji_type"`
}

// AddReaction 添加表情回复
func (c *Channel) AddReaction(ctx context.Context, messageID, emojiType string) error {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return err
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"reaction_type": map[string]string{
			"emoji_type": emojiType,
		},
	})

	url := fmt.Sprintf("https://open.feishu.cn/open-apis/im/v1/messages/%s/reactions", messageID)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	return doWithRetry(ctx, func() error {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		var result struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return err
		}
		if result.Code != 0 {
			return NewFeishuError(result.Code, result.Msg)
		}
		return nil
	}, DefaultRetryConfig, nil)
}

// RemoveReaction 移除表情回复
func (c *Channel) RemoveReaction(ctx context.Context, messageID, reactionID string) error {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://open.feishu.cn/open-apis/im/v1/messages/%s/reactions/%s", messageID, reactionID)

	req, _ := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	return doWithRetry(ctx, func() error {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		var result struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return err
		}
		if result.Code != 0 {
			return NewFeishuError(result.Code, result.Msg)
		}
		return nil
	}, DefaultRetryConfig, nil)
}

// ============ 消息置顶 ============

// PinMessage 置顶消息
func (c *Channel) PinMessage(ctx context.Context, messageID string) error {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://open.feishu.cn/open-apis/im/v1/pins?message_id=%s", messageID)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	return doWithRetry(ctx, func() error {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		var result struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return err
		}
		if result.Code != 0 {
			return NewFeishuError(result.Code, result.Msg)
		}
		return nil
	}, DefaultRetryConfig, nil)
}

// UnpinMessage 取消置顶消息
func (c *Channel) UnpinMessage(ctx context.Context, messageID string) error {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://open.feishu.cn/open-apis/im/v1/pins?message_id=%s", messageID)

	req, _ := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	return doWithRetry(ctx, func() error {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		var result struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return err
		}
		if result.Code != 0 {
			return NewFeishuError(result.Code, result.Msg)
		}
		return nil
	}, DefaultRetryConfig, nil)
}

// ============ 消息读取 ============

// GetMessage 获取消息详情
func (c *Channel) GetMessage(ctx context.Context, messageID string) (*Message, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://open.feishu.cn/open-apis/im/v1/messages/%s", messageID)

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data *struct {
			Items []*Message `json:"items"`
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

	if result.Data == nil || len(result.Data.Items) == 0 {
		return nil, fmt.Errorf("消息不存在: %s", messageID)
	}

	return result.Data.Items[0], nil
}