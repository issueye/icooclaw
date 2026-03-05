package feishu

import (
	"encoding/json"
	"fmt"
)

// ============ 卡片消息构建器 ============

// CardMessage 卡片消息
type CardMessage struct {
	Config   *CardConfig   `json:"config,omitempty"`
	Header   *CardHeader   `json:"header,omitempty"`
	Elements []CardElement `json:"elements"`
}

// CardConfig 卡片配置
type CardConfig struct {
	WideScreenMode bool `json:"wide_screen_mode"`
	EnableForward  bool `json:"enable_forward"`
}

// CardHeader 卡片头部
type CardHeader struct {
	Title    *CardText `json:"title"`
	Template string    `json:"template,omitempty"` // blue, wathet, turquoise, green, yellow, orange, red, carmine, violet, purple, indigo, grey
}

// CardElement 卡片元素接口
type CardElement interface {
	ElementType() string
}

// CardText 文本元素
type CardText struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

// ElementType 实现 CardElement 接口
func (t *CardText) ElementType() string { return "div" }

// CardDiv 分块元素
type CardDiv struct {
	Tag    string      `json:"tag"`
	Text   *CardText   `json:"text,omitempty"`
	Fields []*CardText `json:"fields,omitempty"`
	Extra  CardElement `json:"extra,omitempty"`
}

// ElementType 实现 CardElement 接口
func (d *CardDiv) ElementType() string { return "div" }

// CardAction 操作按钮容器
type CardAction struct {
	Tag     string        `json:"tag"`
	Actions []*CardButton `json:"actions"`
	Layout  string        `json:"layout,omitempty"` // bisected, trisection, flow
}

// ElementType 实现 CardElement 接口
func (a *CardAction) ElementType() string { return "action" }

// CardButton 按钮元素
type CardButton struct {
	Tag   string                 `json:"tag"`
	Text  *CardText              `json:"text"`
	URL   string                 `json:"url,omitempty"`
	Type  string                 `json:"type,omitempty"` // primary, default
	Value map[string]interface{} `json:"value,omitempty"`
}

// CardImage 图片元素
type CardImage struct {
	Tag    string    `json:"tag"`
	ImgKey string    `json:"img_key"`
	Alt    *CardText `json:"alt"`
}

// ElementType 实现 CardElement 接口
func (i *CardImage) ElementType() string { return "img" }

// CardNote 备注元素
type CardNote struct {
	Tag      string        `json:"tag"`
	Elements []CardElement `json:"elements"`
}

// ElementType 实现 CardElement 接口
func (n *CardNote) ElementType() string { return "note" }

// CardDivider 分割线
type CardDivider struct {
	Tag string `json:"tag"`
}

// ElementType 实现 CardElement 接口
func (d *CardDivider) ElementType() string { return "hr" }

// CardMarkdown Markdown 元素
type CardMarkdown struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

// ElementType 实现 CardElement 接口
func (m *CardMarkdown) ElementType() string { return "div" }

// ============ 卡片构建器 ============

// CardBuilder 卡片构建器
type CardBuilder struct {
	card *CardMessage
}

// NewCardBuilder 创建卡片构建器
func NewCardBuilder() *CardBuilder {
	return &CardBuilder{
		card: &CardMessage{
			Config: &CardConfig{
				WideScreenMode: true,
				EnableForward:  true,
			},
			Elements: make([]CardElement, 0),
		},
	}
}

// SetHeader 设置卡片头部
func (b *CardBuilder) SetHeader(title, template string) *CardBuilder {
	b.card.Header = &CardHeader{
		Title: &CardText{
			Tag:     "plain_text",
			Content: title,
		},
		Template: template,
	}
	return b
}

// AddText 添加文本块
func (b *CardBuilder) AddText(content string) *CardBuilder {
	b.card.Elements = append(b.card.Elements, &CardDiv{
		Tag: "div",
		Text: &CardText{
			Tag:     "plain_text",
			Content: content,
		},
	})
	return b
}

// AddMarkdown 添加 Markdown 文本
func (b *CardBuilder) AddMarkdown(content string) *CardBuilder {
	b.card.Elements = append(b.card.Elements, &CardDiv{
		Tag: "div",
		Text: &CardText{
			Tag:     "lark_md",
			Content: content,
		},
	})
	return b
}

// AddDivider 添加分割线
func (b *CardBuilder) AddDivider() *CardBuilder {
	b.card.Elements = append(b.card.Elements, &CardDivider{Tag: "hr"})
	return b
}

// AddImage 添加图片
func (b *CardBuilder) AddImage(imgKey, alt string) *CardBuilder {
	b.card.Elements = append(b.card.Elements, &CardImage{
		Tag:    "img",
		ImgKey: imgKey,
		Alt: &CardText{
			Tag:     "plain_text",
			Content: alt,
		},
	})
	return b
}

// AddNote 添加备注
func (b *CardBuilder) AddNote(contents ...string) *CardBuilder {
	elements := make([]CardElement, len(contents))
	for i, c := range contents {
		elements[i] = &CardText{
			Tag:     "plain_text",
			Content: c,
		}
	}
	b.card.Elements = append(b.card.Elements, &CardNote{
		Tag:      "note",
		Elements: elements,
	})
	return b
}

// AddActions 添加操作按钮
func (b *CardBuilder) AddActions(buttons ...*CardButton) *CardBuilder {
	action := &CardAction{
		Tag:     "action",
		Actions: buttons,
	}
	b.card.Elements = append(b.card.Elements, action)
	return b
}

// AddFields 添加字段列表
func (b *CardBuilder) AddFields(fields ...string) *CardBuilder {
	cardFields := make([]*CardText, len(fields))
	for i, f := range fields {
		cardFields[i] = &CardText{
			Tag:     "lark_md",
			Content: f,
		}
	}
	b.card.Elements = append(b.card.Elements, &CardDiv{
		Tag:    "div",
		Fields: cardFields,
	})
	return b
}

// Build 构建卡片消息
func (b *CardBuilder) Build() *CardMessage {
	return b.card
}

// BuildJSON 构建 JSON 字符串
func (b *CardBuilder) BuildJSON() (string, error) {
	data, err := json.Marshal(b.card)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ============ 按钮构建器 ============

// NewPrimaryButton 创建主要按钮
func NewPrimaryButton(text, url string) *CardButton {
	return &CardButton{
		Tag: "button",
		Text: &CardText{
			Tag:     "plain_text",
			Content: text,
		},
		URL:  url,
		Type: "primary",
	}
}

// NewDefaultButton 创建默认按钮
func NewDefaultButton(text, url string) *CardButton {
	return &CardButton{
		Tag: "button",
		Text: &CardText{
			Tag:     "plain_text",
			Content: text,
		},
		URL:  url,
		Type: "default",
	}
}

// NewCallbackButton 创建回调按钮
func NewCallbackButton(text string, value map[string]interface{}) *CardButton {
	return &CardButton{
		Tag: "button",
		Text: &CardText{
			Tag:     "plain_text",
			Content: text,
		},
		Type:  "primary",
		Value: value,
	}
}

// ============ 预设卡片模板 ============

// NewWelcomeCard 创建欢迎卡片
func NewWelcomeCard(userName string) *CardMessage {
	return NewCardBuilder().
		SetHeader("欢迎使用", "blue").
		AddMarkdown(fmt.Sprintf("👋 你好，**%s**！\n\n我是 AI 助手，有什么可以帮助你的吗？", userName)).
		AddDivider().
		AddNote("由 icooclaw 提供支持").
		Build()
}

// NewErrorCard 创建错误卡片
func NewErrorCard(title, message string) *CardMessage {
	return NewCardBuilder().
		SetHeader("❌ "+title, "red").
		AddText(message).
		Build()
}

// NewSuccessCard 创建成功卡片
func NewSuccessCard(title, message string) *CardMessage {
	return NewCardBuilder().
		SetHeader("✅ "+title, "green").
		AddText(message).
		Build()
}

// NewInfoCard 创建信息卡片
func NewInfoCard(title, message string) *CardMessage {
	return NewCardBuilder().
		SetHeader("ℹ️ "+title, "blue").
		AddMarkdown(message).
		Build()
}
