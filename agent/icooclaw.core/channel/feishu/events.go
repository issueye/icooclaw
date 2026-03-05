package feishu

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

// ============ 事件类型定义 ============

// EventType 事件类型
type EventType string

const (
	// 消息事件
	EventIMMessageReceive  EventType = "im.message.receive_v1"
	EventIMMessageRead     EventType = "im.message.read_v1"
	EventIMMessageReaction EventType = "im.message.reaction_v1"

	// 群组事件
	EventChatDisbanded EventType = "im.chat.disbanded_v1"
	EventChatUpdated   EventType = "im.chat.updated_v1"

	// 群成员事件
	EventChatMemberAdded      EventType = "im.chat.member.added_v1"
	EventChatMemberWithdraw   EventType = "im.chat.member.withdrawn_v1"
	EventChatMemberBotAdded   EventType = "im.chat.member.bot_added_v1"
	EventChatMemberBotRemoved EventType = "im.chat.member.bot_removed_v1"

	// 用户事件
	EventUserCreated EventType = "contact.user.created_v1"
	EventUserUpdated EventType = "contact.user.updated_v1"
	EventUserDeleted EventType = "contact.user.deleted_v1"
)

// EventHandler 事件处理器
type EventHandler func(ctx context.Context, event *Event) error

// ============ 事件结构定义 ============

// Event 通用事件结构
type Event struct {
	Schema string       `json:"schema"`
	Header *EventHeader `json:"header"`
	Event  interface{}  `json:"event,omitempty"` // 原始事件数据
}

// EventHeader 事件头
type EventHeader struct {
	EventID    string `json:"event_id"`
	EventType  string `json:"event_type"`
	AppID      string `json:"app_id"`
	TenantKey  string `json:"tenant_key"`
	CreateTime string `json:"create_time"`
}

// IMMessageEvent IM 消息事件
type IMMessageEvent struct {
	Sender  *MessageSender `json:"sender,omitempty"`
	Message *Message       `json:"message,omitempty"`
}

// Message 飞书消息体
type Message struct {
	MessageID   string     `json:"message_id"`
	ChatID      string     `json:"chat_id"`
	ChatType    string     `json:"chat_type"`
	MessageType string     `json:"message_type"`
	Content     string     `json:"content"`
	CreateTime  string     `json:"create_time,omitempty"`
	ParentID    string     `json:"parent_id,omitempty"`
	RootID      string     `json:"root_id,omitempty"`
	Mentions    []*Mention `json:"mentions,omitempty"`
}

// Mention @信息
type Mention struct {
	Key  string    `json:"key"`
	ID   *SenderID `json:"id,omitempty"`
	Name string    `json:"name,omitempty"`
}

// ChatEvent 群组事件
type ChatEvent struct {
	ChatID    string `json:"chat_id"`
	Name      string `json:"name,omitempty"`
	OwnerID   string `json:"owner_id,omitempty"`
	TenantKey string `json:"tenant_key"`
}

// ChatMemberEvent 群成员事件
type ChatMemberEvent struct {
	ChatID    string   `json:"chat_id"`
	TenantKey string   `json:"tenant_key"`
	Members   []Member `json:"members,omitempty"`
}

// Member 群成员
type Member struct {
	MemberIDType string `json:"member_id_type"`
	MemberID     string `json:"member_id"`
	Name         string `json:"name,omitempty"`
	TenantKey    string `json:"tenant_key"`
}

// UserEvent 用户事件
type UserEvent struct {
	User *User `json:"user"`
}

// User 用户信息
type User struct {
	OpenID        string      `json:"open_id"`
	UserID        string      `json:"user_id"`
	UnionID       string      `json:"union_id"`
	Name          string      `json:"name"`
	EnName        string      `json:"en_name,omitempty"`
	Nickname      string      `json:"nickname,omitempty"`
	Avatar        *Avatar     `json:"avatar,omitempty"`
	Mobile        string      `json:"mobile,omitempty"`
	Gender        int         `json:"gender,omitempty"`
	Email         string      `json:"email,omitempty"`
	Status        *UserStatus `json:"status,omitempty"`
	DepartmentIDs []string    `json:"department_ids,omitempty"`
	LeaderUserID  string      `json:"leader_user_id,omitempty"`
	City          string      `json:"city,omitempty"`
	Country       string      `json:"country,omitempty"`
	WorkStation   string      `json:"work_station,omitempty"`
	JoinTime      int64       `json:"join_time,omitempty"`
	EmployeeNo    string      `json:"employee_no,omitempty"`
	TenantKey     string      `json:"tenant_key,omitempty"`
}

// Avatar 头像信息
type Avatar struct {
	Avatar72     string `json:"avatar_72"`
	Avatar240    string `json:"avatar_240"`
	Avatar640    string `json:"avatar_640"`
	AvatarOrigin string `json:"avatar_origin"`
}

// UserStatus 用户状态
type UserStatus struct {
	IsFrozen    bool `json:"is_frozen"`
	IsResigned  bool `json:"is_resigned"`
	IsUnjoin    bool `json:"is_unjoin"`
	IsActivated bool `json:"is_activated"`
}

// ============ 事件分发器 ============

// EventDispatcher 事件分发器
type EventDispatcher struct {
	handlers map[EventType]EventHandler
	logger   *slog.Logger
}

// NewEventDispatcher 创建事件分发器
func NewEventDispatcher(logger *slog.Logger) *EventDispatcher {
	return &EventDispatcher{
		handlers: make(map[EventType]EventHandler),
		logger:   logger,
	}
}

// Register 注册事件处理器
func (d *EventDispatcher) Register(eventType EventType, handler EventHandler) {
	d.handlers[eventType] = handler
	d.logger.Debug("注册事件处理器", "event_type", eventType)
}

// Dispatch 分发事件
func (d *EventDispatcher) Dispatch(ctx context.Context, event *Event) error {
	if event.Header == nil {
		return errors.New("事件缺少 header")
	}

	eventType := EventType(event.Header.EventType)
	handler, ok := d.handlers[eventType]
	if !ok {
		d.logger.Debug("未注册的事件类型", "event_type", eventType)
		return nil
	}

	return handler(ctx, event)
}

// HasHandler 检查是否注册了处理器
func (d *EventDispatcher) HasHandler(eventType EventType) bool {
	_, ok := d.handlers[eventType]
	return ok
}

// ============ 事件适配器 ============

// ParseIMMessageEvent 解析 IM 消息事件
func ParseIMMessageEvent(event *Event) (*IMMessageEvent, error) {
	if event.Event == nil {
		return nil, errors.New("事件数据为空")
	}

	// 尝试解析为 IMMessageEvent
	var imEvent IMMessageEvent
	data, ok := event.Event.(map[string]interface{})
	if !ok {
		return nil, errors.New("事件数据格式错误")
	}

	// 解析 sender
	if sender, ok := data["sender"].(map[string]interface{}); ok {
		imEvent.Sender = &MessageSender{}
		if senderID, ok := sender["sender_id"].(map[string]interface{}); ok {
			imEvent.Sender.SenderID = &SenderID{}
			if openID, ok := senderID["open_id"].(string); ok {
				imEvent.Sender.SenderID.OpenID = openID
			}
			if userID, ok := senderID["user_id"].(string); ok {
				imEvent.Sender.SenderID.UserID = userID
			}
		}
		if tenantKey, ok := sender["tenant_key"].(string); ok {
			imEvent.Sender.TenantKey = tenantKey
		}
	}

	// 解析 message
	if message, ok := data["message"].(map[string]interface{}); ok {
		imEvent.Message = &Message{}
		if messageID, ok := message["message_id"].(string); ok {
			imEvent.Message.MessageID = messageID
		}
		if chatID, ok := message["chat_id"].(string); ok {
			imEvent.Message.ChatID = chatID
		}
		if chatType, ok := message["chat_type"].(string); ok {
			imEvent.Message.ChatType = chatType
		}
		if msgType, ok := message["message_type"].(string); ok {
			imEvent.Message.MessageType = msgType
		}
		if content, ok := message["content"].(string); ok {
			imEvent.Message.Content = content
		}
		if createTime, ok := message["create_time"].(string); ok {
			imEvent.Message.CreateTime = createTime
		}
	}

	return &imEvent, nil
}

// ParseChatMemberEvent 解析群成员事件
func ParseChatMemberEvent(event *Event) (*ChatMemberEvent, error) {
	if event.Event == nil {
		return nil, errors.New("事件数据为空")
	}

	data, ok := event.Event.(map[string]interface{})
	if !ok {
		return nil, errors.New("事件数据格式错误")
	}

	var memberEvent ChatMemberEvent
	if chatID, ok := data["chat_id"].(string); ok {
		memberEvent.ChatID = chatID
	}
	if tenantKey, ok := data["tenant_key"].(string); ok {
		memberEvent.TenantKey = tenantKey
	}

	return &memberEvent, nil
}

// ============ 事件发布 ============

// ChannelEvent 渠道事件（用于 MessageBus）
type ChannelEvent struct {
	Type      string      `json:"type"`
	Channel   string      `json:"channel"`
	ChatID    string      `json:"chat_id,omitempty"`
	UserID    string      `json:"user_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// NewChannelEvent 创建渠道事件
func NewChannelEvent(eventType, chatID, userID string, data interface{}) *ChannelEvent {
	return &ChannelEvent{
		Type:      eventType,
		Channel:   "feishu",
		ChatID:    chatID,
		UserID:    userID,
		Data:      data,
		Timestamp: time.Now(),
	}
}
