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
	return "Manage scheduled tasks. List, enable, disable, or run tasks on demand."
}

// Parameters returns the tool parameters.
func (t *SchedulerTool) Parameters() map[string]any {
	return map[string]any{
		"action": map[string]any{
			"type":        "string",
			"description": "Action to perform: list, run, enable, disable",
		},
		"task_id": map[string]any{
			"type":        "string",
			"description": "Task ID for run/enable/disable actions",
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
			Error:   fmt.Errorf("unknown action: %s", action),
		}
	}
}

func (t *SchedulerTool) listTasks() *tools.Result {
	tasks := t.scheduler.ListTasks()

	var result string
	result = fmt.Sprintf("Found %d scheduled tasks:\n\n", len(tasks))

	for _, task := range tasks {
		status := "enabled"
		if !task.Enabled {
			status = "disabled"
		}

		result += fmt.Sprintf("- **%s** (%s)\n", task.Name, task.ID)
		result += fmt.Sprintf("  Schedule: %s\n", task.Schedule)
		result += fmt.Sprintf("  Status: %s\n", status)
		result += fmt.Sprintf("  Last run: %s\n", task.LastRun.Format(time.RFC3339))
		result += fmt.Sprintf("  Next run: %s\n\n", task.NextRun.Format(time.RFC3339))
	}

	return &tools.Result{Success: true, Content: result}
}

func (t *SchedulerTool) runTask(taskID string) *tools.Result {
	if taskID == "" {
		return &tools.Result{Success: false, Error: fmt.Errorf("task_id is required")}
	}

	if err := t.scheduler.RunTask(taskID); err != nil {
		return &tools.Result{Success: false, Error: err}
	}

	return &tools.Result{Success: true, Content: fmt.Sprintf("Task %s triggered", taskID)}
}

func (t *SchedulerTool) enableTask(taskID string) *tools.Result {
	if taskID == "" {
		return &tools.Result{Success: false, Error: fmt.Errorf("task_id is required")}
	}

	if err := t.scheduler.EnableTask(taskID); err != nil {
		return &tools.Result{Success: false, Error: err}
	}

	return &tools.Result{Success: true, Content: fmt.Sprintf("Task %s enabled", taskID)}
}

func (t *SchedulerTool) disableTask(taskID string) *tools.Result {
	if taskID == "" {
		return &tools.Result{Success: false, Error: fmt.Errorf("task_id is required")}
	}

	if err := t.scheduler.DisableTask(taskID); err != nil {
		return &tools.Result{Success: false, Error: err}
	}

	return &tools.Result{Success: true, Content: fmt.Sprintf("Task %s disabled", taskID)}
}

// RegisterSchedulerTool registers the scheduler tool.
func RegisterSchedulerTool(registry *tools.Registry, scheduler *Scheduler, logger *slog.Logger) {
	registry.Register(NewSchedulerTool(scheduler, logger))
}
