package handlers

import (
	"log/slog"
	"net/http"

	"icooclaw/pkg/gateway/models"
	"icooclaw/pkg/storage"
)

type ParamHandler struct {
	logger  *slog.Logger
	storage *storage.Storage
}

func NewParamHandler(logger *slog.Logger, storage *storage.Storage) *ParamHandler {
	return &ParamHandler{logger: logger, storage: storage}
}

func (h *ParamHandler) Page(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.QueryParam](r)
	if err != nil {
		h.logger.Error("绑定分页请求失败", "error", err)
		http.Error(w, "绑定分页请求失败", http.StatusBadRequest)
		return
	}

	result, err := h.storage.Param().Page(req)
	if err != nil {
		h.logger.Error("获取参数配置列表失败", "error", err)
		http.Error(w, "获取参数配置列表失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.ResQueryParam]{
		Code:    http.StatusOK,
		Message: "参数配置列表获取成功",
		Data:    result,
	})
}

func (h *ParamHandler) Create(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.ParamConfig](r)
	if err != nil {
		h.logger.Error("绑定创建参数配置请求失败", "error", err)
		http.Error(w, "绑定创建参数配置请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Param().Save(req)
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

func (h *ParamHandler) Update(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.ParamConfig](r)
	if err != nil {
		h.logger.Error("绑定更新参数配置请求失败", "error", err)
		http.Error(w, "绑定更新参数配置请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Param().Save(req)
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

func (h *ParamHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定删除参数配置请求失败", "error", err)
		http.Error(w, "绑定删除参数配置请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Param().Delete(id)
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

func (h *ParamHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定获取参数配置请求失败", "error", err)
		http.Error(w, "绑定获取参数配置请求失败", http.StatusBadRequest)
		return
	}

	config, err := h.storage.Param().Get(id)
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

func (h *ParamHandler) GetByKey(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[struct {
		Key string `json:"key"`
	}](r)
	if err != nil {
		h.logger.Error("绑定获取参数配置请求失败", "error", err)
		http.Error(w, "绑定获取参数配置请求失败", http.StatusBadRequest)
		return
	}

	config, err := h.storage.Param().Get(req.Key)
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

func (h *ParamHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	configs, err := h.storage.Param().List()
	if err != nil {
		h.logger.Error("获取所有参数配置失败", "error", err)
		http.Error(w, "获取所有参数配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[[]*storage.ParamConfig]{
		Code:    http.StatusOK,
		Message: "参数配置列表获取成功",
		Data:    configs,
	})
}

func (h *ParamHandler) GetByGroup(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[struct {
		Group string `json:"group"`
	}](r)
	if err != nil {
		h.logger.Error("绑定获取参数配置请求失败", "error", err)
		http.Error(w, "绑定获取参数配置请求失败", http.StatusBadRequest)
		return
	}

	configs, err := h.storage.Param().ListByGroup(req.Group)
	if err != nil {
		h.logger.Error("获取参数配置失败", "error", err)
		http.Error(w, "获取参数配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[[]*storage.ParamConfig]{
		Code:    http.StatusOK,
		Message: "参数配置列表获取成功",
		Data:    configs,
	})
}

func (h *ParamHandler) SetDefaultModel(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[struct {
		Model string `json:"model"`
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

	param := &storage.ParamConfig{
		Key:         "agent.default_model",
		Value:       req.Model,
		Description: "AI Agent 默认使用的模型",
		Group:       "agent",
		Enabled:     true,
	}

	err = h.storage.Param().Save(param)
	if err != nil {
		h.logger.Error("设置默认模型失败", "error", err)
		http.Error(w, "设置默认模型失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "默认模型设置成功",
		Data: map[string]interface{}{
			"model": req.Model,
		},
	})
}

func (h *ParamHandler) GetDefaultModel(w http.ResponseWriter, r *http.Request) {
	config, err := h.storage.Param().Get("agent.default_model")
	if err != nil || config == nil {
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