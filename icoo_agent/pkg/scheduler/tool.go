// Package scheduler provides task scheduling for icooclaw.
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"icooclaw/pkg/tools"
)

// SchedulerTool provides a tool for managing scheduled tasks.
type SchedulerTool struct {
	scheduler *Scheduler
	logger    *slog.Logger
}

// NewSchedulerTool creates a new scheduler tool.
func NewSchedulerTool(scheduler *Scheduler, logger *slog.Logger) *SchedulerTool {
	return &SchedulerTool{
		scheduler: scheduler,
		logger:    logger,
	}
}

// Name returns the tool name.
func (t *SchedulerTool) Name() string {
	return "scheduler"
}

// Description returns the tool description.
func (t *SchedulerTool) Description() string {
	return "管理定时任务。列出、启用、禁用或按需运行任务。"
}

// Parameters returns the tool parameters.
func (t *SchedulerTool) Parameters() map[string]any {
	return map[string]any{
		"action": map[string]any{
			"type":        "string",
			"description": "要执行的操作: list, run, enable, disable",
		},
		"task_id": map[string]any{
			"type":        "string",
			"description": "任务ID，用于 run/enable/disable 操作",
		},
	}
}

// Execute executes the scheduler tool.
func (t *SchedulerTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	action, _ := args["action"].(string)
	taskID, _ := args["task_id"].(string)

	switch action {
	case "list":
		return t.listTasks()
	case "run":
		return t.runTask(taskID)
	case "enable":
		return t.enableTask(taskID)
	case "disable":
		return t.disableTask(taskID)
	default:
		return &tools.Result{
			Success: false,
			Error:   fmt.Errorf("未知操作: %s", action),
		}
	}
}

func (t *SchedulerTool) listTasks() *tools.Result {
	tasks := t.scheduler.ListTasks()

	var result string
	result = fmt.Sprintf("找到 %d 个定时任务:\n\n", len(tasks))

	for _, task := range tasks {
		status := "已启用"
		if !task.Enabled {
			status = "已禁用"
		}

		result += fmt.Sprintf("- **%s** (%s)\n", task.Name, task.ID)
		result += fmt.Sprintf("  调度: %s\n", task.Schedule)
		result += fmt.Sprintf("  状态: %s\n", status)
		result += fmt.Sprintf("  上次运行: %s\n", task.LastRun.Format(time.RFC3339))
		result += fmt.Sprintf("  下次运行: %s\n\n", task.NextRun.Format(time.RFC3339))
	}

	return &tools.Result{Success: true, Content: result}
}

func (t *SchedulerTool) runTask(taskID string) *tools.Result {
	if taskID == "" {
		return &tools.Result{Success: false, Error: fmt.Errorf("需要提供 task_id")}
	}

	if err := t.scheduler.RunTask(taskID); err != nil {
		return &tools.Result{Success: false, Error: err}
	}

	return &tools.Result{Success: true, Content: fmt.Sprintf("任务 %s 已触发", taskID)}
}

func (t *SchedulerTool) enableTask(taskID string) *tools.Result {
	if taskID == "" {
		return &tools.Result{Success: false, Error: fmt.Errorf("需要提供 task_id")}
	}

	if err := t.scheduler.EnableTask(taskID); err != nil {
		return &tools.Result{Success: false, Error: err}
	}

	return &tools.Result{Success: true, Content: fmt.Sprintf("任务 %s 已启用", taskID)}
}

func (t *SchedulerTool) disableTask(taskID string) *tools.Result {
	if taskID == "" {
		return &tools.Result{Success: false, Error: fmt.Errorf("需要提供 task_id")}
	}

	if err := t.scheduler.DisableTask(taskID); err != nil {
		return &tools.Result{Success: false, Error: err}
	}

	return &tools.Result{Success: true, Content: fmt.Sprintf("任务 %s 已禁用", taskID)}
}

// RegisterSchedulerTool registers the scheduler tool.
func RegisterSchedulerTool(registry *tools.Registry, scheduler *Scheduler, logger *slog.Logger) {
	registry.Register(NewSchedulerTool(scheduler, logger))
}
