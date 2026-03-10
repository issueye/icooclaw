package handlers

import (
	"log/slog"
	"net/http"

	"icooclaw/pkg/gateway/models"
	"icooclaw/pkg/storage"
)

type MemoryHandler struct {
	logger  *slog.Logger
	storage *storage.Storage
}

func NewMemoryHandler(logger *slog.Logger, storage *storage.Storage) *MemoryHandler {
	return &MemoryHandler{logger: logger, storage: storage}
}

func (h *MemoryHandler) Page(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.QueryMemory](r)
	if err != nil {
		h.logger.Error("绑定分页请求失败", "error", err)
		http.Error(w, "绑定分页请求失败", http.StatusBadRequest)
		return
	}

	memories, err := h.storage.Memory().Page(req)
	if err != nil {
		h.logger.Error("获取记忆列表失败", "error", err)
		http.Error(w, "获取记忆列表失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.ResQueryMemory]{
		Code:    http.StatusOK,
		Message: "记忆列表获取成功",
		Data:    memories,
	})
}

func (h *MemoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Memory](r)
	if err != nil {
		h.logger.Error("绑定创建记忆请求失败", "error", err)
		http.Error(w, "绑定创建记忆请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Memory().Save(req)
	if err != nil {
		h.logger.Error("保存记忆失败", "error", err)
		http.Error(w, "保存记忆失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Memory]{
		Code:    http.StatusOK,
		Message: "记忆创建成功",
		Data:    req,
	})
}

func (h *MemoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Memory](r)
	if err != nil {
		h.logger.Error("绑定保存记忆请求失败", "error", err)
		http.Error(w, "绑定保存记忆请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Memory().Save(req)
	if err != nil {
		h.logger.Error("保存记忆失败", "error", err)
		http.Error(w, "保存记忆失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Memory]{
		Code:    http.StatusOK,
		Message: "记忆更新成功",
		Data:    req,
	})
}

func (h *MemoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定删除记忆请求失败", "error", err)
		http.Error(w, "绑定删除记忆请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Memory().Delete(id)
	if err != nil {
		h.logger.Error("删除记忆失败", "error", err)
		http.Error(w, "删除记忆失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "记忆删除成功",
	})
}

func (h *MemoryHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定获取记忆请求失败", "error", err)
		http.Error(w, "绑定获取记忆请求失败", http.StatusBadRequest)
		return
	}

	memory, err := h.storage.Memory().Get(id, 100)
	if err != nil {
		h.logger.Error("获取记忆失败", "error", err)
		http.Error(w, "获取记忆失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[[]*storage.Memory]{
		Code:    http.StatusOK,
		Message: "记忆获取成功",
		Data:    memory,
	})
}

func (h *MemoryHandler) Search(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[struct {
		Query string `json:"query"`
	}](r)
	if err != nil {
		h.logger.Error("绑定搜索记忆请求失败", "error", err)
		http.Error(w, "绑定搜索记忆请求失败", http.StatusBadRequest)
		return
	}

	memories, err := h.storage.Memory().Page(&storage.QueryMemory{Query: req.Query})
	if err != nil {
		h.logger.Error("搜索记忆失败", "error", err)
		http.Error(w, "搜索记忆失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.ResQueryMemory]{
		Code:    http.StatusOK,
		Message: "记忆搜索成功",
		Data:    memories,
	})
}
