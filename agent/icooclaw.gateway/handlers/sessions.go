package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"icooclaw.core/storage"
	"icooclaw.gateway/models"
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
	Channel string `json:"channel,omitempty"` // 渠道 (默认为 "websocket")
	UserID  string `json:"user_id,omitempty"` // 用户ID
	ChatID  string `json:"chat_id,omitempty"` // 聊天ID (可选，不提供则自动生成)
}

// CreateSessionResponse 创建会话响应
type CreateSessionResponse struct {
	SessionID uint   `json:"session_id"`
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

	// 设置默认渠道
	if req.Channel == "" {
		req.Channel = "websocket"
	}

	// 如果没有提供 ChatID，使用自动生成的方式
	if req.ChatID == "" {
		session, err := h.storage.Session().GetOrCreateSession(req.Channel, "ws-auto", req.UserID)
		if err != nil {
			h.logger.Error("创建会话失败", "error", err)
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
				UserID:    session.UserID,
				Key:       session.Key,
			},
		})
		return
	}

	// 使用提供的 ChatID 创建会话
	session := &storage.Session{
		Key:     req.Channel + ":" + req.ChatID,
		Channel: req.Channel,
		ChatID:  req.ChatID,
		UserID:  req.UserID,
	}

	err = h.storage.Session().CreateOrUpdate(session)
	if err != nil {
		h.logger.Error("创建会话失败", "error", err)
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
			UserID:    session.UserID,
			Key:       session.Key,
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

	err = h.storage.Session().CreateOrUpdate(req)
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
	// chi 绑定 {id}
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.logger.Error("绑定删除会话请求失败", "error", err)
		http.Error(w, "绑定删除会话请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Session().Delete(uint(id))
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
	// chi 绑定 {id}
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.logger.Error("绑定获取会话请求失败", "error", err)
		http.Error(w, "绑定获取会话请求失败", http.StatusBadRequest)
		return
	}

	session, err := h.storage.Session().GetByID(uint(id))
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
