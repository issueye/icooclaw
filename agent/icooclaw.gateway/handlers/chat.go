package handlers

import (
	"log/slog"
	"net/http"

	"icooclaw.core/ws"
)

type ChatHandler struct {
	logger   *slog.Logger
	wsManager *ws.Manager
}

func NewChatHandler(logger *slog.Logger, wsManager *ws.Manager) *ChatHandler {
	return &ChatHandler{
		logger:   logger,
		wsManager: wsManager,
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