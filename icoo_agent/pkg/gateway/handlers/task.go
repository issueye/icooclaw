package handlers

import (
	"log/slog"
	"net/http"

	"icooclaw/pkg/gateway/models"
	"icooclaw/pkg/scheduler"
	"icooclaw/pkg/storage"
)

type TaskHandler struct {
	logger   *slog.Logger
	storage  *storage.Storage
	schedule *scheduler.Scheduler
}

func NewTaskHandler(logger *slog.Logger, storage *storage.Storage, schedule *scheduler.Scheduler) *TaskHandler {
	return &TaskHandler{logger: logger, storage: storage, schedule: schedule}
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

// Save doc 保存任务
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

	// 添加到调度器
	task := &scheduler.Task{
		ID:          req.ID,
		Name:        req.Name,
		Schedule:    req.CronExpr,
		Enabled:     req.Enabled,
		Description: req.Description,
		Params:      req.Params,
		Channel:     req.Channel,
	}
	h.schedule.AddTask(task)

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

	// 更新调度器任务
	task := &scheduler.Task{
		ID:          req.ID,
		Name:        req.Name,
		Schedule:    req.CronExpr,
		Enabled:     req.Enabled,
		Description: req.Description,
		Params:      req.Params,
		Channel:     req.Channel,
	}
	// 先删除旧任务，再添加新任务
	h.schedule.RemoveTask(req.ID)
	h.schedule.AddTask(task)

	models.WriteData(w, models.BaseResponse[*storage.Task]{
		Code:    http.StatusOK,
		Message: "任务更新成功",
		Data:    req,
	})
}

func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定删除任务请求失败", "error", err)
		http.Error(w, "绑定删除任务请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Task().Delete(id)
	if err != nil {
		h.logger.Error("删除任务失败", "error", err)
		http.Error(w, "删除任务失败", http.StatusInternalServerError)
		return
	}

	// 删除调度器任务
	h.schedule.RemoveTask(id)

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "任务删除成功",
	})
}

func (h *TaskHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定获取任务请求失败", "error", err)
		http.Error(w, "绑定获取任务请求失败", http.StatusBadRequest)
		return
	}

	task, err := h.storage.Task().GetByID(id)
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

// ToggleEnabled 切换任务状态
func (h *TaskHandler) ToggleEnabled(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定切换任务状态请求失败", "error", err)
		http.Error(w, "绑定切换任务状态请求失败", http.StatusBadRequest)
		return
	}

	// 切换任务状态
	task, err := h.storage.Task().ToggleEnabled(id)
	if err != nil {
		h.logger.Error("切换任务状态失败", "error", err)
		http.Error(w, "切换任务状态失败", http.StatusInternalServerError)
		return
	}

	// 切换调度器任务状态
	if task.Enabled {
		// 如果任务已启用，检查是否已存在调度器任务
		schedulerTask, err := h.storage.Task().GetByID(id)
		if err != nil {
			h.logger.Error("获取调度器任务失败", slog.Any("id", id), slog.Any("error", err.Error()))
			http.Error(w, "获取调度器任务失败", http.StatusInternalServerError)
			return
		}

		// 如果任务不存在，添加到调度器
		if schedulerTask == nil {
			// 任务不存在，添加到调度器
			schedulerTask := &scheduler.Task{
				ID:          task.ID,
				Name:        task.Name,
				Schedule:    task.CronExpr,
				Enabled:     task.Enabled,
				Description: task.Description,
				Params:      task.Params,
				Channel:     task.Channel,
			}
			h.schedule.AddTask(schedulerTask)
		}
	} else {
		// 如果任务已禁用，删除调度器任务
		h.schedule.RemoveTask(task.ID)
	}

	// 返回响应
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
