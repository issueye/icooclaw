package handlers

import (
	"log/slog"
	"net/http"

	"icooclaw/pkg/gateway/models"
	"icooclaw/pkg/storage"
)

type ChannelHandler struct {
	logger  *slog.Logger
	storage *storage.Storage
}

func NewChannelHandler(logger *slog.Logger, storage *storage.Storage) *ChannelHandler {
	return &ChannelHandler{logger: logger, storage: storage}
}

func (h *ChannelHandler) Page(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.QueryChannel](r)
	if err != nil {
		h.logger.Error("绑定分页请求失败", "error", err)
		http.Error(w, "绑定分页请求失败", http.StatusBadRequest)
		return
	}

	configs, err := h.storage.Channel().Page(req)
	if err != nil {
		h.logger.Error("获取通道配置列表失败", "error", err)
		http.Error(w, "获取通道配置列表失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.ResQueryChannel]{
		Code:    http.StatusOK,
		Message: "通道配置列表获取成功",
		Data:    configs,
	})
}

func (h *ChannelHandler) Create(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Channel](r)
	if err != nil {
		h.logger.Error("绑定创建通道配置请求失败", "error", err)
		http.Error(w, "绑定创建通道配置请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Channel().SaveChannel(req)
	if err != nil {
		h.logger.Error("创建通道配置失败", "error", err)
		http.Error(w, "创建通道配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Channel]{
		Code:    http.StatusOK,
		Message: "通道配置创建成功",
		Data:    req,
	})
}

func (h *ChannelHandler) Update(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Channel](r)
	if err != nil {
		h.logger.Error("绑定更新通道配置请求失败", "error", err)
		http.Error(w, "绑定更新通道配置请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Channel().SaveChannel(req)
	if err != nil {
		h.logger.Error("更新通道配置失败", "error", err)
		http.Error(w, "更新通道配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Channel]{
		Code:    http.StatusOK,
		Message: "通道配置更新成功",
		Data:    req,
	})
}

func (h *ChannelHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定删除通道配置请求失败", "error", err)
		http.Error(w, "绑定删除通道配置请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Channel().Delete(id)
	if err != nil {
		h.logger.Error("删除通道配置失败", "error", err)
		http.Error(w, "删除通道配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "通道配置删除成功",
	})
}

func (h *ChannelHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定获取通道配置请求失败", "error", err)
		http.Error(w, "绑定获取通道配置请求失败", http.StatusBadRequest)
		return
	}

	config, err := h.storage.Channel().GetChannel(id)
	if err != nil {
		h.logger.Error("获取通道配置失败", "error", err)
		http.Error(w, "获取通道配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Channel]{
		Code:    http.StatusOK,
		Message: "通道配置获取成功",
		Data:    config,
	})
}

func (h *ChannelHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	configs, err := h.storage.Channel().ListChannels()
	if err != nil {
		h.logger.Error("获取所有通道配置失败", "error", err)
		http.Error(w, "获取所有通道配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[[]*storage.Channel]{
		Code:    http.StatusOK,
		Message: "通道配置列表获取成功",
		Data:    configs,
	})
}

func (h *ChannelHandler) GetEnabled(w http.ResponseWriter, r *http.Request) {
	configs, err := h.storage.Channel().ListEnabledChannels()
	if err != nil {
		h.logger.Error("获取启用通道配置失败", "error", err)
		http.Error(w, "获取启用通道配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[[]*storage.Channel]{
		Code:    http.StatusOK,
		Message: "启用通道配置列表获取成功",
		Data:    configs,
	})
}
