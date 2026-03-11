package handlers

import (
	"log/slog"
	"net/http"

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
	req, err := models.Bind[*storage.QueryMessage](r)
	if err != nil {
		h.logger.Error("绑定分页请求失败", "error", err)
		http.Error(w, "绑定分页请求失败", http.StatusBadRequest)
		return
	}

	messages, err := h.storage.Message().Page(req)
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
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定获取消息请求失败", "error", err)
		http.Error(w, "绑定获取消息请求失败", http.StatusBadRequest)
		return
	}

	message, err := h.storage.Message().Get(id, 100)
	if err != nil {
		h.logger.Error("获取消息失败", "error", err)
		http.Error(w, "获取消息失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[[]*storage.Message]{
		Code:    http.StatusOK,
		Message: "消息获取成功",
		Data:    message,
	})
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
