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
