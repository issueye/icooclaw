package handlers

import (
	"log/slog"
	"net/http"

	"icooclaw/pkg/gateway/models"
	"icooclaw/pkg/storage"
)

type ProviderHandler struct {
	logger  *slog.Logger
	storage *storage.Storage
}

func NewProviderHandler(logger *slog.Logger, storage *storage.Storage) *ProviderHandler {
	return &ProviderHandler{logger: logger, storage: storage}
}

func (h *ProviderHandler) Page(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.QueryProvider](r)
	if err != nil {
		h.logger.Error("绑定分页请求失败", "error", err)
		http.Error(w, "绑定分页请求失败", http.StatusBadRequest)
		return
	}

	configs, err := h.storage.Provider().Page(req)
	if err != nil {
		h.logger.Error("获取Provider配置列表失败", "error", err)
		http.Error(w, "获取Provider配置列表失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.ResQueryProvider]{
		Code:    http.StatusOK,
		Message: "Provider配置列表获取成功",
		Data:    configs,
	})
}

func (h *ProviderHandler) Save(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Provider](r)
	if err != nil {
		h.logger.Error("绑定保存Provider配置请求失败", "error", err)
		http.Error(w, "绑定保存Provider配置请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Provider().Save(req)
	if err != nil {
		h.logger.Error("保存Provider配置失败", "error", err)
		http.Error(w, "保存Provider配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Provider]{
		Code:    http.StatusOK,
		Message: "Provider配置保存成功",
		Data:    req,
	})
}

func (h *ProviderHandler) Create(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Provider](r)
	if err != nil {
		h.logger.Error("绑定创建Provider配置请求失败", "error", err)
		http.Error(w, "绑定创建Provider配置请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Provider().Save(req)
	if err != nil {
		h.logger.Error("创建Provider配置失败", "error", err)
		http.Error(w, "创建Provider配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Provider]{
		Code:    http.StatusOK,
		Message: "Provider配置创建成功",
		Data:    req,
	})
}

func (h *ProviderHandler) Update(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Provider](r)
	if err != nil {
		h.logger.Error("绑定更新Provider配置请求失败", "error", err)
		http.Error(w, "绑定更新Provider配置请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Provider().Save(req)
	if err != nil {
		h.logger.Error("更新Provider配置失败", "error", err)
		http.Error(w, "更新Provider配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Provider]{
		Code:    http.StatusOK,
		Message: "Provider配置更新成功",
		Data:    req,
	})
}

func (h *ProviderHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定删除Provider配置请求失败", "error", err)
		http.Error(w, "绑定删除Provider配置请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Provider().Delete(id)
	if err != nil {
		h.logger.Error("删除Provider配置失败", "error", err)
		http.Error(w, "删除Provider配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "Provider配置删除成功",
	})
}

func (h *ProviderHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定获取Provider配置请求失败", "error", err)
		http.Error(w, "绑定获取Provider配置请求失败", http.StatusBadRequest)
		return
	}

	config, err := h.storage.Provider().GetByName(id)
	if err != nil {
		h.logger.Error("获取Provider配置失败", "error", err)
		http.Error(w, "获取Provider配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Provider]{
		Code:    http.StatusOK,
		Message: "Provider配置获取成功",
		Data:    config,
	})
}

func (h *ProviderHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	configs, err := h.storage.Provider().List()
	if err != nil {
		h.logger.Error("获取所有Provider配置失败", "error", err)
		http.Error(w, "获取所有Provider配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[[]*storage.Provider]{
		Code:    http.StatusOK,
		Message: "Provider配置列表获取成功",
		Data:    configs,
	})
}

func (h *ProviderHandler) GetEnabled(w http.ResponseWriter, r *http.Request) {
	configs, err := h.storage.Provider().List()
	if err != nil {
		h.logger.Error("获取启用Provider配置失败", "error", err)
		http.Error(w, "获取启用Provider配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[[]*storage.Provider]{
		Code:    http.StatusOK,
		Message: "启用Provider配置列表获取成功",
		Data:    configs,
	})
}
