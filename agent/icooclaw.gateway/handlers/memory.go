package handlers

import (
	"log/slog"
	"net/http"

	"icooclaw.core/storage"
	"icooclaw.gateway/models"
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

	err = h.storage.Memory().Create(req)
	if err != nil {
		h.logger.Error("创建记忆失败", "error", err)
		http.Error(w, "创建记忆失败", http.StatusInternalServerError)
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
		h.logger.Error("绑定更新记忆请求失败", "error", err)
		http.Error(w, "绑定更新记忆请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Memory().Update(req)
	if err != nil {
		h.logger.Error("更新记忆失败", "error", err)
		http.Error(w, "更新记忆失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Memory]{
		Code:    http.StatusOK,
		Message: "记忆更新成功",
		Data:    req,
	})
}

func (h *MemoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[uint](r)
	if err != nil {
		h.logger.Error("绑定删除记忆请求失败", "error", err)
		http.Error(w, "绑定删除记忆请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Memory().Delete(req)
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
	req, err := models.Bind[uint](r)
	if err != nil {
		h.logger.Error("绑定获取记忆请求失败", "error", err)
		http.Error(w, "绑定获取记忆请求失败", http.StatusBadRequest)
		return
	}

	memory, err := h.storage.Memory().GetByID(req)
	if err != nil {
		h.logger.Error("获取记忆失败", "error", err)
		http.Error(w, "获取记忆失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Memory]{
		Code:    http.StatusOK,
		Message: "记忆获取成功",
		Data:    memory,
	})
}

func (h *MemoryHandler) Pin(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[uint](r)
	if err != nil {
		h.logger.Error("绑定置顶记忆请求失败", "error", err)
		http.Error(w, "绑定置顶记忆请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Memory().Pin(req)
	if err != nil {
		h.logger.Error("置顶记忆失败", "error", err)
		http.Error(w, "置顶记忆失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "记忆置顶成功",
	})
}

func (h *MemoryHandler) Unpin(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[uint](r)
	if err != nil {
		h.logger.Error("绑定取消置顶记忆请求失败", "error", err)
		http.Error(w, "绑定取消置顶记忆请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Memory().Unpin(req)
	if err != nil {
		h.logger.Error("取消置顶记忆失败", "error", err)
		http.Error(w, "取消置顶记忆失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "记忆取消置顶成功",
	})
}

func (h *MemoryHandler) SoftDelete(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[uint](r)
	if err != nil {
		h.logger.Error("绑定软删除记忆请求失败", "error", err)
		http.Error(w, "绑定软删除记忆请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Memory().SoftDelete(req)
	if err != nil {
		h.logger.Error("软删除记忆失败", "error", err)
		http.Error(w, "软删除记忆失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "记忆软删除成功",
	})
}

func (h *MemoryHandler) Restore(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[uint](r)
	if err != nil {
		h.logger.Error("绑定恢复记忆请求失败", "error", err)
		http.Error(w, "绑定恢复记忆请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Memory().Restore(req)
	if err != nil {
		h.logger.Error("恢复记忆失败", "error", err)
		http.Error(w, "恢复记忆失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "记忆恢复成功",
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

	memories, err := h.storage.Memory().Search(req.Query)
	if err != nil {
		h.logger.Error("搜索记忆失败", "error", err)
		http.Error(w, "搜索记忆失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[[]storage.Memory]{
		Code:    http.StatusOK,
		Message: "记忆搜索成功",
		Data:    memories,
	})
}