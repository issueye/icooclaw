package handlers

import (
	"log/slog"
	"net/http"

	"icooclaw/pkg/gateway/models"
	"icooclaw/pkg/storage"
)

type BindingHandler struct {
	logger  *slog.Logger
	storage *storage.Storage
}

func NewBindingHandler(logger *slog.Logger, storage *storage.Storage) *BindingHandler {
	return &BindingHandler{logger: logger, storage: storage}
}

func (h *BindingHandler) Page(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.QueryBinding](r)
	if err != nil {
		h.logger.Error("绑定分页请求失败", "error", err)
		http.Error(w, "绑定分页请求失败", http.StatusBadRequest)
		return
	}

	bindings, err := h.storage.Binding().Page(req)
	if err != nil {
		h.logger.Error("获取绑定列表失败", "error", err)
		http.Error(w, "获取绑定列表失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.ResQueryBinding]{
		Code:    http.StatusOK,
		Message: "绑定列表获取成功",
		Data:    bindings,
	})
}

func (h *BindingHandler) Create(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Binding](r)
	if err != nil {
		h.logger.Error("绑定创建绑定请求失败", "error", err)
		http.Error(w, "绑定创建绑定请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Binding().SaveBinding(req)
	if err != nil {
		h.logger.Error("创建绑定失败", "error", err)
		http.Error(w, "创建绑定失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Binding]{
		Code:    http.StatusOK,
		Message: "绑定创建成功",
		Data:    req,
	})
}

func (h *BindingHandler) Update(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Binding](r)
	if err != nil {
		h.logger.Error("绑定更新绑定请求失败", "error", err)
		http.Error(w, "绑定更新绑定请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Binding().SaveBinding(req)
	if err != nil {
		h.logger.Error("更新绑定失败", "error", err)
		http.Error(w, "更新绑定失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Binding]{
		Code:    http.StatusOK,
		Message: "绑定更新成功",
		Data:    req,
	})
}

func (h *BindingHandler) Delete(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[struct {
		Channel string `json:"channel"`
		ChatID  string `json:"chat_id"`
	}](r)
	if err != nil {
		h.logger.Error("绑定删除绑定请求失败", "error", err)
		http.Error(w, "绑定删除绑定请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Binding().DeleteBinding(req.Channel, req.ChatID)
	if err != nil {
		h.logger.Error("删除绑定失败", "error", err)
		http.Error(w, "删除绑定失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "绑定删除成功",
	})
}

func (h *BindingHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[struct {
		Channel string `json:"channel"`
		ChatID  string `json:"chat_id"`
	}](r)
	if err != nil {
		h.logger.Error("绑定获取绑定请求失败", "error", err)
		http.Error(w, "绑定获取绑定请求失败", http.StatusBadRequest)
		return
	}

	binding, err := h.storage.Binding().GetBinding(req.Channel, req.ChatID)
	if err != nil {
		h.logger.Error("获取绑定失败", "error", err)
		http.Error(w, "获取绑定失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Binding]{
		Code:    http.StatusOK,
		Message: "绑定获取成功",
		Data:    binding,
	})
}

func (h *BindingHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	bindings, err := h.storage.Binding().ListBindings()
	if err != nil {
		h.logger.Error("获取所有绑定失败", "error", err)
		http.Error(w, "获取所有绑定失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[[]*storage.Binding]{
		Code:    http.StatusOK,
		Message: "绑定列表获取成功",
		Data:    bindings,
	})
}
