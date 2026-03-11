package handlers

import (
	"log/slog"
	"net/http"

	channelconsts "icooclaw/pkg/channels/consts"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/gateway/models"
	"icooclaw/pkg/storage"
)

type MessageHandler struct {
	logger  *slog.Logger
	storage *storage.Storage
}

func NewMessageHandler(logger *slog.Logger, storage *storage.Storage) *MessageHandler {
	return &MessageHandler{logger: logger, storage: storage}
}

func (h *MessageHandler) Page(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*QueryMessageRequest](r)
	if err != nil {
		h.logger.Error("绑定分页请求失败", "error", err)
		http.Error(w, "绑定分页请求失败", http.StatusBadRequest)
		return
	}

	// 构建 storage.QueryMessage
	query := &storage.QueryMessage{
		Page:    req.Page,
		Role:    req.Role,
		KeyWord: req.KeyWord,
	}

	// 如果提供了 session_id，构建 session_key
	if req.SessionID != "" {
		channel := req.Channel
		if channel == "" {
			channel = channelconsts.WEBSOCKET
		}
		query.SessionID = consts.GetSessionKey(channel, req.SessionID)
	}

	messages, err := h.storage.Message().Page(query)
	if err != nil {
		h.logger.Error("获取消息列表失败", "error", err)
		http.Error(w, "获取消息列表失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.ResQueryMessage]{
		Code:    http.StatusOK,
		Message: "消息列表获取成功",
		Data:    messages,
	})
}

// QueryMessageRequest 消息查询请求
type QueryMessageRequest struct {
	Page      storage.Page `json:"page"`
	SessionID string       `json:"session_id"`
	Channel   string       `json:"channel"`
	Role      string       `json:"role"`
	KeyWord   string       `json:"key_word"`
}

// Save 保存消息
func (h *MessageHandler) Save(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Message](r)
	if err != nil {
		h.logger.Error("绑定保存消息请求失败", "error", err)
		http.Error(w, "绑定保存消息请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Message().Save(req)
	if err != nil {
		h.logger.Error("保存消息失败", "error", err)
		http.Error(w, "保存消息失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Message]{
		Code:    http.StatusOK,
		Message: "消息保存成功",
		Data:    req,
	})
}

func (h *MessageHandler) Create(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Message](r)
	if err != nil {
		h.logger.Error("绑定创建消息请求失败", "error", err)
		http.Error(w, "绑定创建消息请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Message().Save(req)
	if err != nil {
		h.logger.Error("创建消息失败", "error", err)
		http.Error(w, "创建消息失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Message]{
		Code:    http.StatusOK,
		Message: "消息创建成功",
		Data:    req,
	})
}

func (h *MessageHandler) Update(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Message](r)
	if err != nil {
		h.logger.Error("绑定更新消息请求失败", "error", err)
		http.Error(w, "绑定更新消息请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Message().Save(req)
	if err != nil {
		h.logger.Error("更新消息失败", "error", err)
		http.Error(w, "更新消息失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Message]{
		Code:    http.StatusOK,
		Message: "消息更新成功",
		Data:    req,
	})
}

func (h *MessageHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定删除消息请求失败", "error", err)
		http.Error(w, "绑定删除消息请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Message().Delete(id)
	if err != nil {
		h.logger.Error("删除消息失败", "error", err)
		http.Error(w, "删除消息失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "消息删除成功",
	})
}

func (h *MessageHandler) GetBySessionID(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*GetBySessionIDRequest](r)
	if err != nil {
		h.logger.Error("绑定获取消息请求失败", "error", err)
		http.Error(w, "绑定获取消息请求失败", http.StatusBadRequest)
		return
	}

	if req.SessionID == "" {
		h.logger.Error("会话ID不能为空")
		http.Error(w, "会话ID不能为空", http.StatusBadRequest)
		return
	}

	// 默认使用 websocket 渠道
	if req.Channel == "" {
		req.Channel = channelconsts.WEBSOCKET
	}

	// 构建 session_key: channel:sessionID
	sessionKey := consts.GetSessionKey(req.Channel, req.SessionID)

	messages, err := h.storage.Message().Get(sessionKey, 100)
	if err != nil {
		h.logger.Error("获取消息失败", "error", err)
		http.Error(w, "获取消息失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[[]*storage.Message]{
		Code:    http.StatusOK,
		Message: "消息获取成功",
		Data:    messages,
	})
}

// GetBySessionIDRequest 按会话ID获取消息请求
type GetBySessionIDRequest struct {
	Channel   string `json:"channel"`
	SessionID string `json:"session_id"`
}

func (h *MessageHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定获取消息请求失败", "error", err)
		http.Error(w, "绑定获取消息请求失败", http.StatusBadRequest)
		return
	}

	message, err := h.storage.Message().GetByID(id)
	if err != nil {
		h.logger.Error("获取消息失败", "error", err)
		http.Error(w, "获取消息失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Message]{
		Code:    http.StatusOK,
		Message: "消息获取成功",
		Data:    message,
	})
}
