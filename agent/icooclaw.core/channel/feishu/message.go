package feishu

import "encoding/json"

// ============ 消息类型定义 ============

// MessageType 消息类型
type MessageType string

const (
	MsgTypeText        MessageType = "text"
	MsgTypePost        MessageType = "post"         // 富文本
	MsgTypeImage       MessageType = "image"
	MsgTypeFile        MessageType = "file"
	MsgTypeAudio       MessageType = "audio"
	MsgTypeMedia       MessageType = "media"
	MsgTypeSticker     MessageType = "sticker"
	MsgTypeInteractive MessageType = "interactive" // 卡片
	MsgTypeShareChat   MessageType = "share_chat"
	MsgTypeShareUser   MessageType = "share_user"
)

// ReceiveIDType 接收者 ID 类型
type ReceiveIDType string

const (
	ReceiveIDTypeOpenID  ReceiveIDType = "open_id"
	ReceiveIDTypeUserID  ReceiveIDType = "user_id"
	ReceiveIDTypeUnionID ReceiveIDType = "union_id"
	ReceiveIDTypeChatID  ReceiveIDType = "chat_id"
)

// ============ 出站消息结构 ============

// OutboundMessage 出站消息
type OutboundMessage struct {
	ReceiveID     string        `json:"receive_id"`
	ReceiveIDType ReceiveIDType `json:"receive_id_type,omitempty"`
	MsgType       MessageType   `json:"msg_type"`
	Content       string        `json:"content"` // JSON 字符串
	UUID          string        `json:"uuid,omitempty"` // 消息唯一标识，用于幂等
}

// ============ 文本消息 ============

// TextContent 文本消息内容
type TextContent struct {
	Text string `json:"text"`
}

// NewTextContent 创建文本内容
func NewTextContent(text string) string {
	content, _ := json.Marshal(&TextContent{Text: text})
	return string(content)
}

// ============ 富文本消息 ============

// PostContent 富文本消息内容
type PostContent struct {
	ZhCN *PostBody `json:"zh_cn,omitempty"`
	EnUS *PostBody `json:"en_us,omitempty"`
}

// PostBody 富文本内容体
type PostBody struct {
	Title   string         `json:"title,omitempty"`
	Content [][]PostElement `json:"content"`
}

// PostElement 富文本元素接口
type PostElement interface {
	PostElementType() string
}

// PostTextElement 文本元素
type PostTextElement struct {
	Tag      string `json:"tag"`
	Text     string `json:"text"`
	UnEscape bool   `json:"un_escape,omitempty"`
}

// PostElementType 实现 PostElement 接口
func (e *PostTextElement) PostElementType() string { return "text" }

// PostLinkElement 链接元素
type PostLinkElement struct {
	Tag  string `json:"tag"`
	Text string `json:"text"`
	Href string `json:"href"`
}

// PostElementType 实现 PostElement 接口
func (e *PostLinkElement) PostElementType() string { return "a" }

// PostAtElement @元素
type PostAtElement struct {
	Tag    string `json:"tag"`
	UserID string `json:"user_id"`
}

// PostElementType 实现 PostElement 接口
func (e *PostAtElement) PostElementType() string { return "at" }

// PostImageElement 图片元素
type PostImageElement struct {
	Tag    string `json:"tag"`
	ImageKey string `json:"image_key"`
}

// PostElementType 实现 PostElement 接口
func (e *PostImageElement) PostElementType() string { return "img" }

// NewPostContent 创建富文本内容
func NewPostContent(title string, paragraphs [][]PostElement) string {
	content := &PostContent{
		ZhCN: &PostBody{
			Title:   title,
			Content: paragraphs,
		},
	}
	jsonContent, _ := json.Marshal(content)
	return string(jsonContent)
}

// ============ 图片消息 ============

// ImageContent 图片消息内容
type ImageContent struct {
	ImageKey string `json:"image_key"`
}

// NewImageContent 创建图片内容
func NewImageContent(imageKey string) string {
	content, _ := json.Marshal(&ImageContent{ImageKey: imageKey})
	return string(content)
}

// ============ 文件消息 ============

// FileContent 文件消息内容
type FileContent struct {
	FileKey string `json:"file_key"`
}

// NewFileContent 创建文件内容
func NewFileContent(fileKey string) string {
	content, _ := json.Marshal(&FileContent{FileKey: fileKey})
	return string(content)
}

// ============ 音频消息 ============

// AudioContent 音频消息内容
type AudioContent struct {
	FileKey string `json:"file_key"`
}

// NewAudioContent 创建音频内容
func NewAudioContent(fileKey string) string {
	content, _ := json.Marshal(&AudioContent{FileKey: fileKey})
	return string(content)
}

// ============ 媒体消息 ============

// MediaContent 媒体消息内容
type MediaContent struct {
	FileKey  string `json:"file_key"`
	ImageKey string `json:"image_key,omitempty"`
}

// NewMediaContent 创建媒体内容
func NewMediaContent(fileKey, imageKey string) string {
	content, _ := json.Marshal(&MediaContent{FileKey: fileKey, ImageKey: imageKey})
	return string(content)
}

// ============ 表情消息 ============

// StickerContent 表情消息内容
type StickerContent struct {
	StickerKey string `json:"sticker_key"`
}

// NewStickerContent 创建表情内容
func NewStickerContent(stickerKey string) string {
	content, _ := json.Marshal(&StickerContent{StickerKey: stickerKey})
	return string(content)
}

// ============ 入站消息结构 ============

// InboundMessage 入站消息
type InboundMessage struct {
	MessageID   string                 `json:"message_id"`
	ChatID      string                 `json:"chat_id"`
	ChatType    string                 `json:"chat_type"` // p2p, group
	MessageType string                 `json:"message_type"`
	Content     string                 `json:"content"`
	Sender      *MessageSender         `json:"sender,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// MessageSender 消息发送者
type MessageSender struct {
	SenderID *SenderID `json:"sender_id,omitempty"`
	TenantKey string   `json:"tenant_key,omitempty"`
}

// SenderID 发送者 ID
type SenderID struct {
	OpenID  string `json:"open_id,omitempty"`
	UserID  string `json:"user_id,omitempty"`
	UnionID string `json:"union_id,omitempty"`
}

// ============ 消息响应结构 ============

// MessageResponse 消息发送响应
type MessageResponse struct {
	Code int `json:"code"`
	Msg  string `json:"msg"`
	Data *MessageData `json:"data,omitempty"`
}

// MessageData 消息数据
type MessageData struct {
	MessageID string `json:"message_id"`
}

// ============ 消息内容解析 ============

// ParseTextContent 解析文本消息内容
func ParseTextContent(content string) (string, error) {
	var tc TextContent
	if err := json.Unmarshal([]byte(content), &tc); err != nil {
		return "", err
	}
	return tc.Text, nil
}

// ParsePostContent 解析富文本消息内容
func ParsePostContent(content string) (*PostContent, error) {
	var pc PostContent
	if err := json.Unmarshal([]byte(content), &pc); err != nil {
		return nil, err
	}
	return &pc, nil
}

// ParseImageContent 解析图片消息内容
func ParseImageContent(content string) (string, error) {
	var ic ImageContent
	if err := json.Unmarshal([]byte(content), &ic); err != nil {
		return "", err
	}
	return ic.ImageKey, nil
}

// ParseFileContent 解析文件消息内容
func ParseFileContent(content string) (string, error) {
	var fc FileContent
	if err := json.Unmarshal([]byte(content), &fc); err != nil {
		return "", err
	}
	return fc.FileKey, nil
}