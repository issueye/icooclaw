package handlers

import (
	"log/slog"
	"net/http"
)

type ChatHandler struct {
	logger *slog.Logger
}

func NewChatHandler(logger *slog.Logger) *ChatHandler {
	return &ChatHandler{logger: logger}
}

// HandleChat 处理聊天请求
func (h *ChatHandler) HandleChat(w http.ResponseWriter, r *http.Request) {
	// 通过websocket处理聊天请求
}
