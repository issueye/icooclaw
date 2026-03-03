package handlers

import (
	"log/slog"
	"net/http"

	"icooclaw.core/storage"
	"icooclaw.gateway/models"
)

type TaskHandler struct {
	logger  *slog.Logger
	storage *storage.Storage
}

func NewTaskHandler(logger *slog.Logger, storage *storage.Storage) *TaskHandler {
	return &TaskHandler{logger: logger, storage: storage}
}

func (h *TaskHandler) Page(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.QueryTask](r)
	if err != nil {
		h.logger.Error("绑定分页请求失败", "error", err)
		http.Error(w, "绑定分页请求失败", http.StatusBadRequest)
		return
	}

	tasks, err := h.storage.Task().Page(req)
	if err != nil {
		h.logger.Error("获取任务列表失败", "error", err)
		http.Error(w, "获取任务列表失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.ResQueryTask]{
		Code:    http.StatusOK,
		Message: "任务列表获取成功",
		Data:    tasks,
	})
}

// 保存任务
func (h *TaskHandler) Save(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Task](r)
	if err != nil {
		h.logger.Error("绑定保存任务请求失败", "error", err)
		http.Error(w, "绑定保存任务请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Task().CreateOrUpdate(req)
	if err != nil {
		h.logger.Error("保存任务失败", "error", err)
		http.Error(w, "保存任务失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Task]{
		Code:    http.StatusOK,
		Message: "任务保存成功",
		Data:    req,
	})
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Task](r)
	if err != nil {
		h.logger.Error("绑定创建任务请求失败", "error", err)
		http.Error(w, "绑定创建任务请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Task().Create(req)
	if err != nil {
		h.logger.Error("创建任务失败", "error", err)
		http.Error(w, "创建任务失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Task]{
		Code:    http.StatusOK,
		Message: "任务创建成功",
		Data:    req,
	})
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Task](r)
	if err != nil {
		h.logger.Error("绑定更新任务请求失败", "error", err)
		http.Error(w, "绑定更新任务请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Task().Update(req)
	if err != nil {
		h.logger.Error("更新任务失败", "error", err)
		http.Error(w, "更新任务失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Task]{
		Code:    http.StatusOK,
		Message: "任务更新成功",
		Data:    req,
	})
}

func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[uint](r)
	if err != nil {
		h.logger.Error("绑定删除任务请求失败", "error", err)
		http.Error(w, "绑定删除任务请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Task().Delete(req)
	if err != nil {
		h.logger.Error("删除任务失败", "error", err)
		http.Error(w, "删除任务失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "任务删除成功",
	})
}

func (h *TaskHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[uint](r)
	if err != nil {
		h.logger.Error("绑定获取任务请求失败", "error", err)
		http.Error(w, "绑定获取任务请求失败", http.StatusBadRequest)
		return
	}

	task, err := h.storage.Task().GetByID(req)
	if err != nil {
		h.logger.Error("获取任务失败", "error", err)
		http.Error(w, "获取任务失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Task]{
		Code:    http.StatusOK,
		Message: "任务获取成功",
		Data:    task,
	})
}

func (h *TaskHandler) ToggleEnabled(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[uint](r)
	if err != nil {
		h.logger.Error("绑定切换任务状态请求失败", "error", err)
		http.Error(w, "绑定切换任务状态请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Task().ToggleEnabled(req)
	if err != nil {
		h.logger.Error("切换任务状态失败", "error", err)
		http.Error(w, "切换任务状态失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "任务状态切换成功",
	})
}

func (h *TaskHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.storage.Task().GetAll()
	if err != nil {
		h.logger.Error("获取所有任务失败", "error", err)
		http.Error(w, "获取所有任务失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[[]storage.Task]{
		Code:    http.StatusOK,
		Message: "任务列表获取成功",
		Data:    tasks,
	})
}

func (h *TaskHandler) GetEnabled(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.storage.Task().GetEnabled()
	if err != nil {
		h.logger.Error("获取启用任务失败", "error", err)
		http.Error(w, "获取启用任务失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[[]storage.Task]{
		Code:    http.StatusOK,
		Message: "启用任务列表获取成功",
		Data:    tasks,
	})
}
