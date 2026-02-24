package bus

import (
	"sync"
	"time"
)

// EventType 事件类型
type EventType string

const (
	// EventAgentStart Agent启动
	EventAgentStart EventType = "agent_start"
	// EventAgentStop Agent停止
	EventAgentStop EventType = "agent_stop"
	// EventAgentResponse Agent响应
	EventAgentResponse EventType = "agent_response"
	// EventToolCall 工具调用
	EventToolCall EventType = "tool_call"
	// EventToolResult 工具结果
	EventToolResult EventType = "tool_result"
	// EventError 错误事件
	EventError EventType = "error"
	// EventMessageReceived 消息接收
	EventMessageReceived EventType = "message_received"
	// EventMessageSent 消息发送
	EventMessageSent EventType = "message_sent"
	// EventTaskStart 任务开始
	EventTaskStart EventType = "task_start"
	// EventTaskComplete 任务完成
	EventTaskComplete EventType = "task_complete"
	// EventHeartbeat 心跳
	EventHeartbeat EventType = "heartbeat"
)

// Event 事件
type Event struct {
	Type      EventType
	Channel   string
	ChatID    string
	SessionID uint
	Data      interface{}
	Timestamp time.Time
}

// NewEvent 创建事件
func NewEvent(eventType EventType, channel, chatID string, sessionID uint, data interface{}) Event {
	return Event{
		Type:      eventType,
		Channel:   channel,
		ChatID:    chatID,
		SessionID: sessionID,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// EventBus 事件总线
type EventBus struct {
	listeners map[EventType][]chan Event
	mu        sync.RWMutex
}

// NewEventBus 创建事件总线
func NewEventBus() *EventBus {
	return &EventBus{
		listeners: make(map[EventType][]chan Event),
	}
}

// Subscribe 订阅事件
func (eb *EventBus) Subscribe(eventType EventType) chan Event {
	ch := make(chan Event, 10)

	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.listeners[eventType] = append(eb.listeners[eventType], ch)
	return ch
}

// Unsubscribe 取消订阅
func (eb *EventBus) Unsubscribe(eventType EventType, ch chan Event) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	listeners := eb.listeners[eventType]
	for i, listener := range listeners {
		if listener == ch {
			listeners = append(listeners[:i], listeners[i+1:]...)
			break
		}
	}
	eb.listeners[eventType] = listeners
	close(ch)
}

// Publish 发布事件
func (eb *EventBus) Publish(event Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	listeners := eb.listeners[event.Type]
	for _, listener := range listeners {
		select {
		case listener <- event:
		default:
			// 监听器满了，跳过
		}
	}
}

// Close 关闭事件总线
func (eb *EventBus) Close() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	for _, listeners := range eb.listeners {
		for _, ch := range listeners {
			close(ch)
		}
	}
	eb.listeners = make(map[EventType][]chan Event)
}
