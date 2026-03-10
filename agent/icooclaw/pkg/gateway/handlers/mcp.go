package handlers

import (
	"log/slog"
	"net/http"

	"icooclaw/pkg/gateway/models"
	"icooclaw/pkg/storage"
)

type MCPHandler struct {
	logger  *slog.Logger
	storage *storage.Storage
}

func NewMCPHandler(logger *slog.Logger, storage *storage.Storage) *MCPHandler {
	return &MCPHandler{logger: logger, storage: storage}
}

func (h *MCPHandler) Page(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.QueryMCP](r)
	if err != nil {
		h.logger.Error("绑定分页请求失败", "error", err)
		http.Error(w, "绑定分页请求失败", http.StatusBadRequest)
		return
	}

	configs, err := h.storage.MCP().Page(req)
	if err != nil {
		h.logger.Error("获取MCP配置列表失败", "error", err)
		http.Error(w, "获取MCP配置列表失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.ResQueryMCP]{
		Code:    http.StatusOK,
		Message: "MCP配置列表获取成功",
		Data:    configs,
	})
}

// Save 保存MCP配置
func (h *MCPHandler) Save(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.MCPConfig](r)
	if err != nil {
		h.logger.Error("绑定保存MCP配置请求失败", "error", err)
		http.Error(w, "绑定保存MCP配置请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.MCP().Save(req)
	if err != nil {
		h.logger.Error("保存MCP配置失败", "error", err)
		http.Error(w, "保存MCP配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.MCPConfig]{
		Code:    http.StatusOK,
		Message: "MCP配置创建成功",
		Data:    req,
	})
}

func (h *MCPHandler) Create(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.MCPConfig](r)
	if err != nil {
		h.logger.Error("绑定创建MCP配置请求失败", "error", err)
		http.Error(w, "绑定创建MCP配置请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.MCP().Save(req)
	if err != nil {
		h.logger.Error("创建MCP配置失败", "error", err)
		http.Error(w, "创建MCP配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.MCPConfig]{
		Code:    http.StatusOK,
		Message: "MCP配置创建成功",
		Data:    req,
	})
}

func (h *MCPHandler) Update(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.MCPConfig](r)
	if err != nil {
		h.logger.Error("绑定更新MCP配置请求失败", "error", err)
		http.Error(w, "绑定更新MCP配置请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.MCP().Save(req)
	if err != nil {
		h.logger.Error("更新MCP配置失败", "error", err)
		http.Error(w, "更新MCP配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.MCPConfig]{
		Code:    http.StatusOK,
		Message: "MCP配置更新成功",
		Data:    req,
	})
}

func (h *MCPHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定删除MCP配置请求失败", "error", err)
		http.Error(w, "绑定删除MCP配置请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.MCP().Delete(id)
	if err != nil {
		h.logger.Error("删除MCP配置失败", "error", err)
		http.Error(w, "删除MCP配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "MCP配置删除成功",
	})
}

func (h *MCPHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定获取MCP配置请求失败", "error", err)
		http.Error(w, "绑定获取MCP配置请求失败", http.StatusBadRequest)
		return
	}

	config, err := h.storage.MCP().Get(id)
	if err != nil {
		h.logger.Error("获取MCP配置失败", "error", err)
		http.Error(w, "获取MCP配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.MCPConfig]{
		Code:    http.StatusOK,
		Message: "MCP配置获取成功",
		Data:    config,
	})
}

func (h *MCPHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	configs, err := h.storage.MCP().List()
	if err != nil {
		h.logger.Error("获取所有MCP配置失败", "error", err)
		http.Error(w, "获取所有MCP配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[[]*storage.MCPConfig]{
		Code:    http.StatusOK,
		Message: "MCP配置列表获取成功",
		Data:    configs,
	})
}
