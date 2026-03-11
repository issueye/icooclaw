package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"icooclaw/pkg/agent"
	"icooclaw/pkg/bus"
	"icooclaw/pkg/channels/consts"
	"icooclaw/pkg/gateway/models"
	"icooclaw/pkg/gateway/websocket"
	"icooclaw/pkg/storage"
)

// ChatHandler handles chat-related HTTP and WebSocket requests.
type ChatHandler struct {
	logger        *slog.Logger
	storage       *storage.Storage
	wsManager     *websocket.Manager
	bus           *bus.MessageBus
	agentLoop     *agent.Loop
	agentRegistry *agent.AgentRegistry
}

// NewChatHandler creates a new ChatHandler.
func NewChatHandler(
	logger *slog.Logger,
	storage *storage.Storage,
	wsManager *websocket.Manager,
	bus *bus.MessageBus,
	agentLoop *agent.Loop,
	agentRegistry *agent.AgentRegistry,
) *ChatHandler {
	return &ChatHandler{
		logger:        logger,
		storage:       storage,
		wsManager:     wsManager,
		bus:           bus,
		agentLoop:     agentLoop,
		agentRegistry: agentRegistry,
	}
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

// WithAgentLoop sets the agent loop.
func (h *ChatHandler) WithAgentLoop(l *agent.Loop) *ChatHandler {
	h.agentLoop = l
	return h
}

// WithAgentRegistry sets the agent registry.
func (h *ChatHandler) WithAgentRegistry(r *agent.AgentRegistry) *ChatHandler {
	h.agentRegistry = r
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
	if h.agentLoop != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()

		response, err := h.agentLoop.ProcessDirectWithChannel(
			ctx,
			req.Content,
			req.SessionID,
			consts.WEBSOCKET,
			req.SessionID,
		)
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
				Content:   response,
				Timestamp: time.Now().Unix(),
			},
		})
		return
	}

	// Fallback: publish to bus
	if h.bus != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		msg := bus.InboundMessage{
			Channel:   consts.WEBSOCKET,
			SessionID: req.SessionID,
			Sender:    bus.SenderInfo{ID: "http", Name: "HTTP Client"},
			Text:      req.Content,
			Timestamp: time.Now(),
		}

		if err := h.bus.PublishInbound(ctx, msg); err != nil {
			h.logger.With("name", "【网关服务】").Error("发布消息失败", "error", err)
			http.Error(w, "【网关服务】发布消息失败", http.StatusInternalServerError)
			return
		}

		models.WriteData(w, models.BaseResponse[any]{
			Code:    http.StatusOK,
			Message: "message queued",
			Data: map[string]string{
				"session_id": req.SessionID,
				"status":     "queued",
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
	if h.agentLoop != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
		defer cancel()

		response, err := h.agentLoop.ProcessDirectWithChannel(
			ctx,
			req.Content,
			req.SessionID,
			"http-stream",
			req.SessionID,
		)
		if err != nil {
			h.writeSSE(w, "error", map[string]string{"error": "处理消息失败: " + err.Error()})
			flusher.Flush()
			return
		}

		// Send content event
		h.writeSSE(w, "content", map[string]string{
			"session_id": req.SessionID,
			"content":    response,
		})
		flusher.Flush()
	} else {
		// No agent loop configured
		h.writeSSE(w, "error", map[string]string{"error": "服务未配置：缺少智能体"})
		flusher.Flush()
	}

	// Send end event
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

// GetAgentStatus returns the status of all agents.
func (h *ChatHandler) GetAgentStatus(w http.ResponseWriter, r *http.Request) {
	if h.agentRegistry == nil {
		models.WriteData(w, models.BaseResponse[any]{
			Code:    http.StatusOK,
			Message: "【网关服务】智能体注册未配置成功",
			Data:    nil,
		})
		return
	}

	agentNames := h.agentRegistry.List()
	statuses := make([]*AgentStatus, 0, len(agentNames))

	for _, name := range agentNames {
		ag, ok := h.agentRegistry.Get(name)
		if !ok {
			continue
		}

		status := &AgentStatus{
			Name:     name,
			IsActive: true,
		}

		if ag != nil {
			status.Model = ag.Config().Model
		}

		statuses = append(statuses, status)
	}

	models.WriteData(w, models.BaseResponse[[]*AgentStatus]{
		Code:    http.StatusOK,
		Message: "【网关服务】智能体状态获取成功",
		Data:    statuses,
	})
}

// SetMaxAgentsRequest represents a request to set max agents.
type SetMaxAgentsRequest struct {
	Max int `json:"max"`
}

// SetMaxAgents sets the maximum number of agents.
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

// ConnectionStatus represents the WebSocket connection status.
type ConnectionStatus struct {
	TotalConnections int `json:"total_connections"`
	MaxConcurrent    int `json:"max_concurrent"`
}

// GetConnectionStatus returns the WebSocket connection status.
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
