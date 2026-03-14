package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"icooclaw/pkg/agent"
	"icooclaw/pkg/agent/react"
	"icooclaw/pkg/bus"
	"icooclaw/pkg/channels/consts"
	"icooclaw/pkg/gateway/models"
	"icooclaw/pkg/gateway/websocket"
	"icooclaw/pkg/storage"
)

// ChatHandler handles chat-related HTTP and WebSocket requests.
type ChatHandler struct {
	logger       *slog.Logger
	storage      *storage.Storage
	wsManager    *websocket.Manager
	bus          *bus.MessageBus
	agentManager *agent.AgentManager
}

// NewChatHandler creates a new ChatHandler.
func NewChatHandler(
	logger *slog.Logger,
	storage *storage.Storage,
) *ChatHandler {
	return &ChatHandler{
		logger: logger,
	}
}

func (h *ChatHandler) WithAgentManager(m *agent.AgentManager) *ChatHandler {
	h.agentManager = m
	if h.wsManager != nil {
		h.wsManager.WithAgentManager(m)
	}
	return h
}

// WithWebSocketManager sets the WebSocket manager.
func (h *ChatHandler) WithWebSocketManager(m *websocket.Manager) *ChatHandler {
	h.wsManager = m
	return h
}

// WithBus sets the message bus.
func (h *ChatHandler) WithBus(b *bus.MessageBus) *ChatHandler {
	h.bus = b
	return h
}

func (h *ChatHandler) WithStorage(s *storage.Storage) *ChatHandler {
	h.storage = s
	return h
}

// HandleWebSocket handles WebSocket connection upgrade.
func (h *ChatHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	if h.wsManager == nil {
		h.logger.With("name", "【网关服务】").Error("WebSocket管理器未配置")
		http.Error(w, "【网关服务】WebSocket管理器未配置", http.StatusInternalServerError)
		return
	}
	h.wsManager.HandleWebSocket(w, r)
}

// HandleWebSocketWithSessionID handles WebSocket connection with a specific session ID.
func (h *ChatHandler) HandleWebSocketWithSessionID(w http.ResponseWriter, r *http.Request) {
	if h.wsManager == nil {
		h.logger.With("name", "【网关服务】").Error("WebSocket管理器未配置")
		http.Error(w, "【网关服务】WebSocket管理器未配置", http.StatusInternalServerError)
		return
	}

	// Get session ID from URL path
	sessionID := r.PathValue("session_id")
	if sessionID == "" {
		// Try query parameter
		sessionID = r.URL.Query().Get("session_id")
	}

	h.wsManager.HandleWebSocketWithSessionID(w, r, sessionID)
}

// ChatRequest represents a chat request.
type ChatRequest struct {
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
	Stream    bool   `json:"stream,omitempty"`
	AgentName string `json:"agent_name,omitempty"`
}

// ChatResponse represents a chat response.
type ChatResponse struct {
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
	AgentName string `json:"agent_name,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// HandleChat handles HTTP chat requests.
func (h *ChatHandler) HandleChat(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*ChatRequest](r)
	if err != nil {
		h.logger.Error("failed to bind chat request", "error", err)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		h.logger.With("name", "【网关服务】").Error("内容不能为空")
		http.Error(w, "【网关服务】内容不能为空", http.StatusBadRequest)
		return
	}

	if req.SessionID == "" {
		h.logger.With("name", "【网关服务】").Error("会话ID不能为空")
		http.Error(w, "【网关服务】会话ID不能为空", http.StatusBadRequest)
		return
	}

	// Process with agent loop
	if h.agentManager != nil {
		inbound := bus.InboundMessage{
			Channel:   consts.WEBSOCKET,
			SessionID: req.SessionID,
			Sender:    bus.SenderInfo{ID: "http", Name: "HTTP Client"},
			Text:      req.Content,
			Timestamp: time.Now(),
		}

		finalResponse, err := h.agentManager.RunAgent(inbound)

		if err != nil {
			h.logger.With("name", "【网关服务】").Error("处理聊天失败", "error", err)
			http.Error(w, "【网关服务】处理聊天失败", http.StatusInternalServerError)
			return
		}

		models.WriteData(w, models.BaseResponse[*ChatResponse]{
			Code:    http.StatusOK,
			Message: "success",
			Data: &ChatResponse{
				SessionID: req.SessionID,
				Content:   finalResponse,
				Timestamp: time.Now().Unix(),
			},
		})
		return
	}

	h.logger.With("name", "【网关服务】").Error("未配置智能体或消息总线")
	http.Error(w, "【网关服务】未配置智能体或消息总线", http.StatusBadRequest)
}

// HandleChatStream handles HTTP chat requests with SSE streaming.
func (h *ChatHandler) HandleChatStream(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*ChatRequest](r)
	if err != nil {
		h.logger.With("name", "【网关服务】").Error("failed to bind chat request", "error", err)
		http.Error(w, "【网关服务】无效请求参数", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		h.logger.With("name", "【网关服务】").Error("内容不能为空")
		http.Error(w, "【网关服务】内容不能为空", http.StatusBadRequest)
		return
	}

	if req.SessionID == "" {
		h.logger.With("name", "【网关服务】").Error("会话ID不能为空")
		http.Error(w, "【网关服务】会话ID不能为空", http.StatusBadRequest)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		h.logger.With("name", "【网关服务】").Error("不支持流式传输")
		http.Error(w, "【网关服务】不支持流式传输", http.StatusInternalServerError)
		return
	}

	// Send start event
	h.logger.With("name", "【网关服务】").Info("开始处理聊天请求", "session_id", req.SessionID)
	h.writeSSE(w, "start", map[string]string{"session_id": req.SessionID})
	flusher.Flush()

	// Process with agent loop
	if h.agentManager != nil {
		inbound := bus.InboundMessage{
			Channel:   consts.WEBSOCKET,
			SessionID: req.SessionID,
			Sender:    bus.SenderInfo{ID: "http", Name: "HTTP Client"},
			Text:      req.Content,
			Timestamp: time.Now(),
		}

		err := h.agentManager.RunAgentStream(inbound, func(chunk react.StreamChunk) error {
			// 发送流式内容事件
			h.writeSSE(w, "content", map[string]string{
				"session_id": req.SessionID,
				"content":    chunk.Content,
			})

			flusher.Flush()
			return nil
		})

		if err != nil {
			h.writeSSE(w, "error", map[string]string{"error": "处理消息失败: " + err.Error()})
			flusher.Flush()
			return
		}

		h.writeSSE(w, "content", map[string]string{
			"session_id": req.SessionID,
			"type":       "end",
		})

		flusher.Flush()
	} else {
		// 没有配置智能体循环
		h.writeSSE(w, "error", map[string]string{"error": "服务未配置：缺少智能体"})
		flusher.Flush()
	}

	// 发送结束事件事件
	h.writeSSE(w, "end", map[string]string{"session_id": req.SessionID})
	flusher.Flush()
}

// writeSSE writes a Server-Sent Event.
func (h *ChatHandler) writeSSE(w http.ResponseWriter, event string, data interface{}) {
	dataBytes, _ := json.Marshal(data)
	w.Write([]byte("event: " + event + "\n"))
	w.Write([]byte("data: " + string(dataBytes) + "\n\n"))
}

// GetQueueStatus returns the current queue status.
func (h *ChatHandler) GetQueueStatus(w http.ResponseWriter, r *http.Request) {
	if h.wsManager == nil {
		models.WriteData(w, models.BaseResponse[any]{
			Code:    http.StatusOK,
			Message: "【网关服务】WebSocket管理器未配置",
			Data:    nil,
		})
		return
	}

	status := h.wsManager.GetQueueStatus()

	models.WriteData(w, models.BaseResponse[*websocket.QueueStatus]{
		Code:    http.StatusOK,
		Message: "【网关服务】队列状态获取成功",
		Data:    status,
	})
}

// SetMaxConcurrentRequest represents a request to set max concurrent connections.
type SetMaxConcurrentRequest struct {
	Max int `json:"max"`
}

// SetMaxConcurrent sets the maximum concurrent WebSocket connections.
func (h *ChatHandler) SetMaxConcurrent(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*SetMaxConcurrentRequest](r)
	if err != nil {
		h.logger.With("name", "【网关服务】").Error("failed to bind request", "error", err)
		http.Error(w, "【网关服务】无效请求参数", http.StatusBadRequest)
		return
	}

	if req.Max <= 0 {
		h.logger.With("name", "【网关服务】").Error("最大并发连接数必须大于0")
		http.Error(w, "【网关服务】最大并发连接数必须大于0", http.StatusBadRequest)
		return
	}

	if h.wsManager == nil {
		h.logger.With("name", "【网关服务】").Error("WebSocket管理器未配置")
		http.Error(w, "【网关服务】WebSocket管理器未配置", http.StatusInternalServerError)
		return
	}

	h.wsManager.SetMaxConcurrent(req.Max)

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "【网关服务】最大并发连接数更新成功",
		Data: map[string]int{
			"max_concurrent": req.Max,
		},
	})
}

// AgentStatus represents the status of an agent.
type AgentStatus struct {
	Name      string `json:"name"`
	IsActive  bool   `json:"is_active"`
	Model     string `json:"model,omitempty"`
	ChatCount int    `json:"chat_count,omitempty"`
}

// SetMaxAgentsRequest 设置最大智能体数
type SetMaxAgentsRequest struct {
	Max int `json:"max"`
}

// SetMaxAgents 设置最大智能体数
func (h *ChatHandler) SetMaxAgents(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*SetMaxAgentsRequest](r)
	if err != nil {
		h.logger.With("name", "【网关服务】").Error("绑定请求参数失败", "error", err)
		http.Error(w, "【网关服务】无效请求参数", http.StatusBadRequest)
		return
	}

	if req.Max <= 0 {
		h.logger.With("name", "【网关服务】").Error("最大智能体数必须大于0")
		http.Error(w, "【网关服务】最大智能体数必须大于0", http.StatusBadRequest)
		return
	}

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "【网关服务】最大智能体数更新成功",
		Data: map[string]int{
			"max_agents": req.Max,
		},
	})
}

// ConnectionStatus WebSocket 连接状态
type ConnectionStatus struct {
	TotalConnections int `json:"total_connections"`
	MaxConcurrent    int `json:"max_concurrent"`
}

// GetConnectionStatus WebSocket 连接状态
func (h *ChatHandler) GetConnectionStatus(w http.ResponseWriter, r *http.Request) {
	if h.wsManager == nil {
		models.WriteData(w, models.BaseResponse[*ConnectionStatus]{
			Code:    http.StatusOK,
			Message: "【网关服务】WebSocket管理器未配置成功",
			Data:    nil,
		})
		return
	}

	status := &ConnectionStatus{
		TotalConnections: h.wsManager.GetConnectionCount(),
		MaxConcurrent:    h.wsManager.GetQueueStatus().MaxConcurrent,
	}

	models.WriteData(w, models.BaseResponse[*ConnectionStatus]{
		Code:    http.StatusOK,
		Message: "【网关服务】连接状态获取成功",
		Data:    status,
	})
}
