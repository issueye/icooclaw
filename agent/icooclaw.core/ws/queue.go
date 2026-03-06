package ws

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const (
	// DefaultMaxConcurrent 默认最大并发对话数
	DefaultMaxConcurrent = 4
)

// QueueItem 队列项
type QueueItem struct {
	SessionID    string
	ConnectionID string
	Content      string
	Timestamp    time.Time
	Ctx          context.Context
	Cancel       context.CancelFunc
}

// QueueStatus 队列状态
type QueueStatus struct {
	ActiveCount   int         `json:"active_count"`
	WaitingCount  int         `json:"waiting_count"`
	MaxConcurrent int         `json:"max_concurrent"`
	ActiveItems   []QueueInfo `json:"active_items"`
	WaitingItems  []QueueInfo `json:"waiting_items"`
}

// QueueInfo 队列项信息
type QueueInfo struct {
	SessionID    string    `json:"session_id"`
	ConnectionID string    `json:"connection_id"`
	Timestamp    time.Time `json:"timestamp"`
	Position     int       `json:"position,omitempty"` // 队列位置
}

// QueueHandler 队列处理器接口
type QueueHandler interface {
	ProcessItem(ctx context.Context, item *QueueItem) error
}

// QueueHandlerFunc 队列处理函数
type QueueHandlerFunc func(ctx context.Context, item *QueueItem) error

// ProcessItem 实现 QueueHandler 接口
func (hf QueueHandlerFunc) ProcessItem(ctx context.Context, item *QueueItem) error {
	return hf(ctx, item)
}

// ConversationQueue 对话队列管理器
type ConversationQueue struct {
	maxConcurrent int
	active        map[string]*QueueItem // 正在处理的对话
	waiting       []*QueueItem          // 等待队列
	mu            sync.RWMutex
	handler       QueueHandler
	logger        *slog.Logger
	notifier      QueueNotifier
}

// QueueNotifier 队列状态通知器
type QueueNotifier interface {
	NotifyQueueStatus(sessionID string, status *QueueStatus)
	NotifyQueuePosition(sessionID string, position int)
}

// QueueNotifierFunc 队列状态通知函数
type QueueNotifierFunc func(sessionID string, status *QueueStatus)

// NotifyQueueStatus 实现 QueueNotifier 接口
func (nf QueueNotifierFunc) NotifyQueueStatus(sessionID string, status *QueueStatus) {
	nf(sessionID, status)
}

// NewConversationQueue 创建对话队列
func NewConversationQueue(maxConcurrent int, logger *slog.Logger) *ConversationQueue {
	if maxConcurrent <= 0 {
		maxConcurrent = DefaultMaxConcurrent
	}
	return &ConversationQueue{
		maxConcurrent: maxConcurrent,
		active:        make(map[string]*QueueItem),
		waiting:       make([]*QueueItem, 0),
		logger:        logger,
	}
}

// SetHandler 设置处理器
func (q *ConversationQueue) SetHandler(handler QueueHandler) {
	q.handler = handler
}

// SetNotifier 设置通知器
func (q *ConversationQueue) SetNotifier(notifier QueueNotifier) {
	q.notifier = notifier
}

// Enqueue 将对话加入队列
func (q *ConversationQueue) Enqueue(ctx context.Context, sessionID string, connectionID, content string) (*QueueItem, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// 检查是否已经在处理中
	if _, exists := q.active[sessionID]; exists {
		q.logger.Warn("会话已在处理中", "session_id", sessionID)
		return nil, false
	}

	// 检查是否已在等待队列中
	for _, item := range q.waiting {
		if item.SessionID == sessionID {
			q.logger.Warn("会话已在等待队列中", "session_id", sessionID)
			return nil, false
		}
	}

	itemCtx, cancel := context.WithCancel(ctx)
	item := &QueueItem{
		SessionID:    sessionID,
		ConnectionID: connectionID,
		Content:      content,
		Timestamp:    time.Now(),
		Ctx:          itemCtx,
		Cancel:       cancel,
	}

	// 如果有空闲槽位，直接处理
	if len(q.active) < q.maxConcurrent {
		q.active[sessionID] = item
		q.logger.Info("对话开始处理", "session_id", sessionID, "active_count", len(q.active))
		go q.processItem(item)
		return item, true
	}

	// 否则加入等待队列
	q.waiting = append(q.waiting, item)
	position := len(q.waiting)
	q.logger.Info("对话加入等待队列", "session_id", sessionID, "position", position)

	// 通知队列位置
	if q.notifier != nil {
		q.notifier.NotifyQueuePosition(sessionID, position)
	}

	return item, true
}

// processItem 处理队列项
func (q *ConversationQueue) processItem(item *QueueItem) {
	defer q.complete(item.SessionID)

	if q.handler != nil {
		if err := q.handler.ProcessItem(item.Ctx, item); err != nil {
			q.logger.Error("处理对话失败", "session_id", item.SessionID, "error", err)
		}
	}
}

// complete 完成对话处理
func (q *ConversationQueue) complete(sessionID string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// 从活跃列表中移除
	delete(q.active, sessionID)
	q.logger.Info("对话处理完成", "session_id", sessionID, "active_count", len(q.active))

	// 检查等待队列
	if len(q.waiting) > 0 && len(q.active) < q.maxConcurrent {
		// 取出第一个等待项
		next := q.waiting[0]
		q.waiting = q.waiting[1:]

		// 加入活跃列表
		q.active[next.SessionID] = next
		q.logger.Info("开始处理等待中的对话", "session_id", next.SessionID, "active_count", len(q.active))

		// 通知队列状态变化
		if q.notifier != nil {
			q.notifier.NotifyQueuePosition(next.SessionID, 0)
		}

		// 更新其他等待项的位置
		for i, item := range q.waiting {
			if q.notifier != nil {
				q.notifier.NotifyQueuePosition(item.SessionID, i+1)
			}
		}

		go q.processItem(next)
	}
}

// Cancel 取消对话
func (q *ConversationQueue) Cancel(sessionID string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	// 检查是否在活跃列表中
	if item, exists := q.active[sessionID]; exists {
		item.Cancel()
		delete(q.active, sessionID)
		q.logger.Info("取消活跃对话", "session_id", sessionID)
		return true
	}

	// 检查是否在等待队列中
	for i, item := range q.waiting {
		if item.SessionID == sessionID {
			q.waiting = append(q.waiting[:i], q.waiting[i+1:]...)
			q.logger.Info("取消等待中的对话", "session_id", sessionID)
			return true
		}
	}

	return false
}

// GetStatus 获取队列状态
func (q *ConversationQueue) GetStatus() *QueueStatus {
	q.mu.RLock()
	defer q.mu.RUnlock()

	status := &QueueStatus{
		ActiveCount:   len(q.active),
		WaitingCount:  len(q.waiting),
		MaxConcurrent: q.maxConcurrent,
		ActiveItems:   make([]QueueInfo, 0, len(q.active)),
		WaitingItems:  make([]QueueInfo, 0, len(q.waiting)),
	}

	for _, item := range q.active {
		status.ActiveItems = append(status.ActiveItems, QueueInfo{
			SessionID:    item.SessionID,
			ConnectionID: item.ConnectionID,
			Timestamp:    item.Timestamp,
		})
	}

	for i, item := range q.waiting {
		status.WaitingItems = append(status.WaitingItems, QueueInfo{
			SessionID:    item.SessionID,
			ConnectionID: item.ConnectionID,
			Timestamp:    item.Timestamp,
			Position:     i + 1,
		})
	}

	return status
}

// GetPosition 获取会话在队列中的位置
func (q *ConversationQueue) GetPosition(sessionID string) int {
	q.mu.RLock()
	defer q.mu.RUnlock()

	// 如果在活跃列表中，返回0
	if _, exists := q.active[sessionID]; exists {
		return 0
	}

	// 检查等待队列
	for i, item := range q.waiting {
		if item.SessionID == sessionID {
			return i + 1
		}
	}

	return -1 // 不在队列中
}

// IsActive 检查会话是否正在处理
func (q *ConversationQueue) IsActive(sessionID string) bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	_, exists := q.active[sessionID]
	return exists
}

// IsWaiting 检查会话是否在等待队列中
func (q *ConversationQueue) IsWaiting(sessionID string) bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	for _, item := range q.waiting {
		if item.SessionID == sessionID {
			return true
		}
	}
	return false
}

// IsQueued 检查会话是否在队列中（活跃或等待）
func (q *ConversationQueue) IsQueued(sessionID string) bool {
	return q.IsActive(sessionID) || q.IsWaiting(sessionID)
}

// SetMaxConcurrent 设置最大并发数
func (q *ConversationQueue) SetMaxConcurrent(max int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.maxConcurrent = max
	q.logger.Info("更新最大并发数", "max_concurrent", max)
}

// Close 关闭队列
func (q *ConversationQueue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()

	// 取消所有活跃项
	for _, item := range q.active {
		item.Cancel()
	}

	// 取消所有等待项
	for _, item := range q.waiting {
		item.Cancel()
	}

	q.active = make(map[string]*QueueItem)
	q.waiting = make([]*QueueItem, 0)
}
