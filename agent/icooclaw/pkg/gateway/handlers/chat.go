package handlers

import (
	"log/slog"
	"net/http"

	"icooclaw.ai/agent"
	"icooclaw.core/ws"
	"icooclaw.gateway/models"
)

type ChatHandler struct {
	logger       *slog.Logger
	wsManager    *ws.Manager
	agentManager *agent.AgentManager
}

func NewChatHandler(logger *slog.Logger, wsManager *ws.Manager, agentManager *agent.AgentManager) *ChatHandler {
	return &ChatHandler{
		logger:       logger,
		wsManager:    wsManager,
		agentManager: agentManager,
	}
}

// HandleWebSocket 处理 WebSocket 连接
func (h *ChatHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	h.wsManager.HandleWebSocket(w, r)
}

// HandleChat 处理聊天请求 (HTTP 方式，可选)
func (h *ChatHandler) HandleChat(w http.ResponseWriter, r *http.Request) {
	// 通过 HTTP 方式处理聊天请求 (非 WebSocket)
	// 可以作为备用方案
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("请使用 WebSocket 连接进行聊天"))
}

// GetQueueStatus 获取队列状态
func (h *ChatHandler) GetQueueStatus(w http.ResponseWriter, r *http.Request) {
	status := h.wsManager.GetQueueStatus()

	models.WriteData(w, models.BaseResponse[*ws.QueueStatus]{
		Code:    http.StatusOK,
		Message: "队列状态获取成功",
		Data:    status,
	})
}

// SetMaxConcurrentRequest 设置最大并发数请求
type SetMaxConcurrentRequest struct {
	Max int `json:"max"`
}

// SetMaxConcurrent 设置最大并发数
func (h *ChatHandler) SetMaxConcurrent(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*SetMaxConcurrentRequest](r)
	if err != nil {
		h.logger.Error("绑定设置最大并发数请求失败", "error", err)
		http.Error(w, "绑定设置最大并发数请求失败", http.StatusBadRequest)
		return
	}

	if req.Max <= 0 {
		http.Error(w, "最大并发数必须大于 0", http.StatusBadRequest)
		return
	}

	h.wsManager.SetMaxConcurrent(req.Max)

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "最大并发数设置成功",
		Data: map[string]int{
			"max_concurrent": req.Max,
		},
	})
}

// GetAgentStatus 获取 Agent 状态
func (h *ChatHandler) GetAgentStatus(w http.ResponseWriter, r *http.Request) {
	if h.agentManager == nil {
		models.WriteData(w, models.BaseResponse[any]{
			Code:    http.StatusOK,
			Message: "Agent Manager 未初始化",
			Data:    nil,
		})
		return
	}

	infos := h.agentManager.GetAgentInfos()
	status := map[string]any{
		"agent_count":      h.agentManager.GetAgentCount(),
		"active_count":     h.agentManager.GetActiveAgentCount(),
		"max_agents":       h.agentManager.GetMaxAgents(),
		"agent_infos":      infos,
	}

	models.WriteData(w, models.BaseResponse[map[string]any]{
		Code:    http.StatusOK,
		Message: "Agent 状态获取成功",
		Data:    status,
	})
}

// SetMaxAgentsRequest 设置最大 Agent 数量请求
type SetMaxAgentsRequest struct {
	Max int `json:"max"`
}

// SetMaxAgents 设置最大 Agent 数量
func (h *ChatHandler) SetMaxAgents(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*SetMaxAgentsRequest](r)
	if err != nil {
		h.logger.Error("绑定设置最大 Agent 数量请求失败", "error", err)
		http.Error(w, "绑定设置最大 Agent 数量请求失败", http.StatusBadRequest)
		return
	}

	if req.Max <= 0 {
		http.Error(w, "最大 Agent 数量必须大于 0", http.StatusBadRequest)
		return
	}

	if h.agentManager == nil {
		http.Error(w, "Agent Manager 未初始化", http.StatusInternalServerError)
		return
	}

	h.agentManager.SetMaxAgents(req.Max)

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "最大 Agent 数量设置成功",
		Data: map[string]int{
			"max_agents": req.Max,
		},
	})
}
