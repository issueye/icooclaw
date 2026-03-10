package handlers

import (
	"log/slog"
	"net/http"

	"icooclaw.core/storage"
	"icooclaw.gateway/models"
)

type ParamHandler struct {
	logger  *slog.Logger
	storage *storage.Storage
}

func NewParamHandler(logger *slog.Logger, storage *storage.Storage) *ParamHandler {
	return &ParamHandler{logger: logger, storage: storage}
}

// Page 分页获取参数配置
func (h *ParamHandler) Page(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.QueryParamConfig](r)
	if err != nil {
		h.logger.Error("绑定分页请求失败", "error", err)
		http.Error(w, "绑定分页请求失败", http.StatusBadRequest)
		return
	}

	result, err := h.storage.ParamConfig().Page(req)
	if err != nil {
		h.logger.Error("获取参数配置列表失败", "error", err)
		http.Error(w, "获取参数配置列表失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.ResQueryParamConfig]{
		Code:    http.StatusOK,
		Message: "参数配置列表获取成功",
		Data:    result,
	})
}

// Create 创建参数配置
func (h *ParamHandler) Create(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.ParamConfig](r)
	if err != nil {
		h.logger.Error("绑定创建参数配置请求失败", "error", err)
		http.Error(w, "绑定创建参数配置请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.ParamConfig().Create(req)
	if err != nil {
		h.logger.Error("创建参数配置失败", "error", err)
		http.Error(w, "创建参数配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.ParamConfig]{
		Code:    http.StatusOK,
		Message: "参数配置创建成功",
		Data:    req,
	})
}

// Update 更新参数配置
func (h *ParamHandler) Update(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.ParamConfig](r)
	if err != nil {
		h.logger.Error("绑定更新参数配置请求失败", "error", err)
		http.Error(w, "绑定更新参数配置请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.ParamConfig().Update(req)
	if err != nil {
		h.logger.Error("更新参数配置失败", "error", err)
		http.Error(w, "更新参数配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.ParamConfig]{
		Code:    http.StatusOK,
		Message: "参数配置更新成功",
		Data:    req,
	})
}

// Delete 删除参数配置
func (h *ParamHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定删除参数配置请求失败", "error", err)
		http.Error(w, "绑定删除参数配置请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.ParamConfig().Delete(id)
	if err != nil {
		h.logger.Error("删除参数配置失败", "error", err)
		http.Error(w, "删除参数配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "参数配置删除成功",
	})
}

// GetByID 通过 ID 获取参数配置
func (h *ParamHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定获取参数配置请求失败", "error", err)
		http.Error(w, "绑定获取参数配置请求失败", http.StatusBadRequest)
		return
	}

	config, err := h.storage.ParamConfig().GetByID(id)
	if err != nil {
		h.logger.Error("获取参数配置失败", "error", err)
		http.Error(w, "获取参数配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.ParamConfig]{
		Code:    http.StatusOK,
		Message: "参数配置获取成功",
		Data:    config,
	})
}

// GetByKey 通过键获取参数配置
func (h *ParamHandler) GetByKey(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[struct {
		Key string `json:"key"`
	}](r)
	if err != nil {
		h.logger.Error("绑定获取参数配置请求失败", "error", err)
		http.Error(w, "绑定获取参数配置请求失败", http.StatusBadRequest)
		return
	}

	config, err := h.storage.ParamConfig().GetByKey(req.Key)
	if err != nil {
		h.logger.Error("获取参数配置失败", "error", err)
		http.Error(w, "获取参数配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.ParamConfig]{
		Code:    http.StatusOK,
		Message: "参数配置获取成功",
		Data:    config,
	})
}

// GetAll 获取所有参数配置
func (h *ParamHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	configs, err := h.storage.ParamConfig().GetAll()
	if err != nil {
		h.logger.Error("获取所有参数配置失败", "error", err)
		http.Error(w, "获取所有参数配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[[]storage.ParamConfig]{
		Code:    http.StatusOK,
		Message: "参数配置列表获取成功",
		Data:    configs,
	})
}

// GetByGroup 按分组获取参数配置
func (h *ParamHandler) GetByGroup(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[struct {
		Group string `json:"group"`
	}](r)
	if err != nil {
		h.logger.Error("绑定获取参数配置请求失败", "error", err)
		http.Error(w, "绑定获取参数配置请求失败", http.StatusBadRequest)
		return
	}

	configs, err := h.storage.ParamConfig().GetByGroup(req.Group)
	if err != nil {
		h.logger.Error("获取参数配置失败", "error", err)
		http.Error(w, "获取参数配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[[]storage.ParamConfig]{
		Code:    http.StatusOK,
		Message: "参数配置列表获取成功",
		Data:    configs,
	})
}

// SetDefaultModel 设置默认模型（便捷接口）
func (h *ParamHandler) SetDefaultModel(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[struct {
		ProviderID uint   `json:"provider_id"`
		Model      string `json:"model"`
	}](r)
	if err != nil {
		h.logger.Error("绑定设置默认模型请求失败", "error", err)
		http.Error(w, "绑定设置默认模型请求失败", http.StatusBadRequest)
		return
	}

	if req.Model == "" {
		http.Error(w, "模型不能为空", http.StatusBadRequest)
		return
	}

	// 存储到 param_configs 表
	param := &storage.ParamConfig{
		Key:         "agent.default_model",
		Value:       req.Model,
		Description: "AI Agent 默认使用的模型",
		Group:       "agent",
		Enabled:     true,
	}

	err = h.storage.ParamConfig().CreateOrUpdate(param)
	if err != nil {
		h.logger.Error("设置默认模型失败", "error", err)
		http.Error(w, "设置默认模型失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "默认模型设置成功",
		Data: map[string]interface{}{
			"provider_id": req.ProviderID,
			"model":       req.Model,
		},
	})
}

// GetDefaultModel 获取默认模型（便捷接口）
func (h *ParamHandler) GetDefaultModel(w http.ResponseWriter, r *http.Request) {
	config, err := h.storage.ParamConfig().GetByKey("agent.default_model")
	if err != nil {
		models.WriteData(w, models.BaseResponse[any]{
			Code:    http.StatusOK,
			Message: "未设置默认模型",
			Data:    nil,
		})
		return
	}

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "默认模型获取成功",
		Data: map[string]interface{}{
			"model": config.Value,
		},
	})
}
