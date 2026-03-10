package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

// SendText sends a text message to a chat.
func (c *Channel) SendText(ctx context.Context, chatID, text string) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	content, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return fmt.Errorf("marshal text content: %w", err)
	}

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(chatID).
			MsgType(larkim.MsgTypeText).
			Content(string(content)).
			Build()).
		Build()

	resp, err := c.client.Im.V1.Message.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("feishu send text: %w", channels.ErrTemporary)
	}

	if !resp.Success() {
		return fmt.Errorf("feishu api error (code=%d msg=%s): %w", resp.Code, resp.Msg, channels.ErrTemporary)
	}

	return nil
}

// SendImage sends an image message to a chat.
func (c *Channel) SendImage(ctx context.Context, chatID string, imagePath string) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	file, err := os.Open(imagePath)
	if err != nil {
		return fmt.Errorf("open image file: %w", err)
	}
	defer file.Close()

	// Upload image to get image_key
	uploadReq := larkim.NewCreateImageReqBuilder().
		Body(larkim.NewCreateImageReqBodyBuilder().
			ImageType("message").
			Image(file).
			Build()).
		Build()

	uploadResp, err := c.client.Im.V1.Image.Create(ctx, uploadReq)
	if err != nil {
		return fmt.Errorf("feishu image upload: %w", err)
	}
	if !uploadResp.Success() {
		return fmt.Errorf("feishu image upload api error (code=%d msg=%s)", uploadResp.Code, uploadResp.Msg)
	}
	if uploadResp.Data == nil || uploadResp.Data.ImageKey == nil {
		return fmt.Errorf("feishu image upload: no image_key returned")
	}

	imageKey := *uploadResp.Data.ImageKey

	// Send image message
	content, _ := json.Marshal(map[string]string{"image_key": imageKey})
	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(chatID).
			MsgType(larkim.MsgTypeImage).
			Content(string(content)).
			Build()).
		Build()

	resp, err := c.client.Im.V1.Message.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("feishu image send: %w", err)
	}
	if !resp.Success() {
		return fmt.Errorf("feishu image send api error (code=%d msg=%s)", resp.Code, resp.Msg)
	}
	return nil
}

// SendFile sends a file message to a chat.
func (c *Channel) SendFile(ctx context.Context, chatID string, filePath string, fileType string) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	// Map file type to Feishu file type
	feishuFileType := "stream"
	switch fileType {
	case "audio":
		feishuFileType = "opus"
	case "video":
		feishuFileType = "mp4"
	}

	filename := filePath
	if len(filename) > 100 {
		filename = filename[len(filename)-100:]
	}

	// Upload file to get file_key
	uploadReq := larkim.NewCreateFileReqBuilder().
		Body(larkim.NewCreateFileReqBodyBuilder().
			FileType(feishuFileType).
			FileName(filename).
			File(file).
			Build()).
		Build()

	uploadResp, err := c.client.Im.V1.File.Create(ctx, uploadReq)
	if err != nil {
		return fmt.Errorf("feishu file upload: %w", err)
	}
	if !uploadResp.Success() {
		return fmt.Errorf("feishu file upload api error (code=%d msg=%s)", uploadResp.Code, uploadResp.Msg)
	}
	if uploadResp.Data == nil || uploadResp.Data.FileKey == nil {
		return fmt.Errorf("feishu file upload: no file_key returned")
	}

	fileKey := *uploadResp.Data.FileKey

	// Send file message
	content, _ := json.Marshal(map[string]string{"file_key": fileKey})
	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(chatID).
			MsgType(larkim.MsgTypeFile).
			Content(string(content)).
			Build()).
		Build()

	resp, err := c.client.Im.V1.Message.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("feishu file send: %w", err)
	}
	if !resp.Success() {
		return fmt.Errorf("feishu file send api error (code=%d msg=%s)", resp.Code, resp.Msg)
	}
	return nil
}

// GetUserInfo gets user information by user ID.
func (c *Channel) GetUserInfo(ctx context.Context, userID string) (*UserInfo, error) {
	// Use open_id to get user info
	req := larkim.NewGetUserReqBuilder().
		UserIdType("open_id").
		UserId(userID).
		Build()

	resp, err := c.client.Im.V1.User.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("get user info: %w", err)
	}
	if !resp.Success() {
		return nil, fmt.Errorf("get user api error (code=%d msg=%s)", resp.Code, resp.Msg)
	}

	info := &UserInfo{}
	if resp.Data != nil && resp.Data.User != nil {
		if resp.Data.User.Name != nil {
			info.Name = *resp.Data.User.Name
		}
		if resp.Data.User.OpenId != nil {
			info.OpenID = *resp.Data.User.OpenId
		}
		if resp.Data.User.UnionId != nil {
			info.UnionID = *resp.Data.User.UnionId
		}
	}

	return info, nil
}

// UserInfo contains user information.
type UserInfo struct {
	Name    string
	OpenID  string
	UnionID string
}