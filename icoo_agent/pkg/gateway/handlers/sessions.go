package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"icooclaw/pkg/channels/consts"
	"icooclaw/pkg/gateway/models"
	"icooclaw/pkg/storage"
)

type SessionHandler struct {
	logger  *slog.Logger
	storage *storage.Storage
}

func NewSessionHandler(logger *slog.Logger, storage *storage.Storage) *SessionHandler {
	return &SessionHandler{logger: logger, storage: storage}
}

// CreateSessionRequest 创建会话请求
type CreateSessionRequest struct {
	Channel  string            `json:"channel,omitempty"`  // 渠道 (默认为 "websocket")
	UserID   string            `json:"user_id,omitempty"`  // 用户ID
	ChatID   string            `json:"chat_id,omitempty"`  // 聊天ID (可选，不提供则自动生成)
	Metadata map[string]string `json:"metadata,omitempty"` // 元数据 (JSON格式)
}

// CreateSessionResponse 创建会话响应
type CreateSessionResponse struct {
	SessionID string `json:"session_id"`
	Channel   string `json:"channel"`
	ChatID    string `json:"chat_id"`
	UserID    string `json:"user_id"`
	Key       string `json:"key"`
}

// Create 创建新会话 (供前端调用)
func (h *SessionHandler) Create(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*CreateSessionRequest](r)
	if err != nil {
		h.logger.Error("绑定创建会话请求失败", "error", err)
		http.Error(w, "绑定创建会话请求失败", http.StatusBadRequest)
		return
	}

	h.logger.With("name", "【会话】").Info("创建会话", slog.Any("params", req))

	// 设置默认渠道
	if req.Channel == "" {
		req.Channel = consts.WEBSOCKET
	}

	var session storage.Session

	// 如果没有提供 ChatID，生成唯一的 ChatID
	if req.ChatID == "" {
		chatID := fmt.Sprintf("chat-%d", time.Now().UnixNano())
		session = storage.Session{
			Channel: req.Channel,
			ChatID:  chatID,
			UserID:  req.UserID,
			Title:   req.Metadata["title"],
		}
	}

	// 使用提供的 ChatID 创建会话
	session = storage.Session{
		Channel: req.Channel,
		ChatID:  req.ChatID,
		UserID:  req.UserID,
		Title:   req.Metadata["title"],
	}

	h.logger.With("name", "【会话】").Info("创建会话", slog.Any("params", session))

	err = h.storage.Session().Save(&session)
	if err != nil {
		h.logger.With("name", "【会话】").Error("创建会话失败", "error", err.Error())
		http.Error(w, "创建会话失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*CreateSessionResponse]{
		Code:    http.StatusOK,
		Message: "会话创建成功",
		Data: &CreateSessionResponse{
			SessionID: session.ID,
			Channel:   session.Channel,
			ChatID:    session.ChatID,
		},
	})
}

func (h *SessionHandler) Page(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.QuerySession](r)
	if err != nil {
		h.logger.Error("绑定分页请求失败", "error", err)
		http.Error(w, "绑定分页请求失败", http.StatusBadRequest)
		return
	}

	if req.Channel == "" {
		req.Channel = consts.WEBSOCKET
	}

	sessions, err := h.storage.Session().Page(req)
	if err != nil {
		h.logger.Error("获取会话列表失败", "error", err)
		http.Error(w, "获取会话列表失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.ResQuerySession]{
		Code:    http.StatusOK,
		Message: "会话列表获取成功",
		Data:    sessions,
	})
}

// Save 保存会话
func (h *SessionHandler) Save(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Session](r)
	if err != nil {
		h.logger.Error("绑定保存会话请求失败", "error", err)
		http.Error(w, "绑定保存会话请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Session().Save(req)
	if err != nil {
		h.logger.Error("保存会话失败", "error", err)
		http.Error(w, "保存会话失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Session]{
		Code:    http.StatusOK,
		Message: "会话保存成功",
		Data:    req,
	})
}

func (h *SessionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// 从请求体获取 ID
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定获取消息请求失败", "error", err)
		http.Error(w, "绑定获取消息请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Session().Delete(id)
	if err != nil {
		h.logger.Error("删除会话失败", "error", err)
		http.Error(w, "删除会话失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "会话删除成功",
	})
}

func (h *SessionHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定获取消息请求失败", "error", err)
		http.Error(w, "绑定获取消息请求失败", http.StatusBadRequest)
		return
	}

	session, err := h.storage.Session().Get(id)
	if err != nil {
		h.logger.Error("获取会话失败", "error", err)
		http.Error(w, "获取会话失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Session]{
		Code:    http.StatusOK,
		Message: "会话获取成功",
		Data:    session,
	})
}
