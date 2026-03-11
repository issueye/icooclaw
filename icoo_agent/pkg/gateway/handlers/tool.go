package handlers

import (
	"log/slog"
	"net/http"

	"icooclaw/pkg/gateway/models"
	"icooclaw/pkg/storage"
)

type ToolHandler struct {
	logger  *slog.Logger
	storage *storage.Storage
}

func NewToolHandler(logger *slog.Logger, storage *storage.Storage) *ToolHandler {
	return &ToolHandler{logger: logger, storage: storage}
}

func (h *ToolHandler) Page(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.QueryTool](r)
	if err != nil {
		h.logger.Error("绑定分页请求失败", "error", err)
		http.Error(w, "绑定分页请求失败", http.StatusBadRequest)
		return
	}

	tools, err := h.storage.Tool().Page(req)
	if err != nil {
		h.logger.Error("获取工具列表失败", "error", err)
		http.Error(w, "获取工具列表失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.ResQueryTool]{
		Code:    http.StatusOK,
		Message: "工具列表获取成功",
		Data:    tools,
	})
}

func (h *ToolHandler) Create(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Tool](r)
	if err != nil {
		h.logger.Error("绑定创建工具请求失败", "error", err)
		http.Error(w, "绑定创建工具请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Tool().SaveTool(req)
	if err != nil {
		h.logger.Error("创建工具失败", "error", err)
		http.Error(w, "创建工具失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Tool]{
		Code:    http.StatusOK,
		Message: "工具创建成功",
		Data:    req,
	})
}

func (h *ToolHandler) Update(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Tool](r)
	if err != nil {
		h.logger.Error("绑定更新工具请求失败", "error", err)
		http.Error(w, "绑定更新工具请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Tool().SaveTool(req)
	if err != nil {
		h.logger.Error("更新工具失败", "error", err)
		http.Error(w, "更新工具失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Tool]{
		Code:    http.StatusOK,
		Message: "工具更新成功",
		Data:    req,
	})
}

func (h *ToolHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定删除工具请求失败", "error", err)
		http.Error(w, "绑定删除工具请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Tool().DeleteTool(id)
	if err != nil {
		h.logger.Error("删除工具失败", "error", err)
		http.Error(w, "删除工具失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "工具删除成功",
	})
}

func (h *ToolHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定获取工具请求失败", "error", err)
		http.Error(w, "绑定获取工具请求失败", http.StatusBadRequest)
		return
	}

	tool, err := h.storage.Tool().GetTool(id)
	if err != nil {
		h.logger.Error("获取工具失败", "error", err)
		http.Error(w, "获取工具失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Tool]{
		Code:    http.StatusOK,
		Message: "工具获取成功",
		Data:    tool,
	})
}

func (h *ToolHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	tools, err := h.storage.Tool().ListTools()
	if err != nil {
		h.logger.Error("获取所有工具失败", "error", err)
		http.Error(w, "获取所有工具失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[[]*storage.Tool]{
		Code:    http.StatusOK,
		Message: "工具列表获取成功",
		Data:    tools,
	})
}

func (h *ToolHandler) GetEnabled(w http.ResponseWriter, r *http.Request) {
	tools, err := h.storage.Tool().ListEnabledTools()
	if err != nil {
		h.logger.Error("获取启用工具失败", "error", err)
		http.Error(w, "获取启用工具失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[[]*storage.Tool]{
		Code:    http.StatusOK,
		Message: "启用工具列表获取成功",
		Data:    tools,
	})
}