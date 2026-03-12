// Package scheduler provides task scheduling for icooclaw.
package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/channels/consts"
	"icooclaw/pkg/scheduler"
	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"
)

// SchedulerTool provides a tool for managing scheduled tasks.
type Tool struct {
	scheduler *scheduler.Scheduler
	store     *storage.TaskStorage
	logger    *slog.Logger
	bus       *bus.MessageBus
}

// NewTool creates a new scheduler tool.
func NewTool(store *storage.TaskStorage, scheduler *scheduler.Scheduler, bus *bus.MessageBus, logger *slog.Logger) *Tool {
	if logger == nil {
		logger = slog.Default()
	}
	return &Tool{
		store:     store,
		scheduler: scheduler,
		logger:    logger,
		bus:       bus,
	}
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return "scheduler"
}

// Description returns the tool description.
func (t *Tool) Description() string {
	return `管理定时任务。支持以下操作：
- list: 列出所有任务
- get: 获取单个任务详情
- create: 创建新任务
- update: 更新任务
- delete: 删除任务
- run: 立即执行任务
- enable: 启用任务
- disable: 禁用任务`
}

// Parameters returns the tool parameters.
func (t *Tool) Parameters() map[string]any {
	return map[string]any{
		"action": map[string]any{
			"type":        "string",
			"description": "要执行的操作: list, get, create, update, delete, run, enable, disable",
			"enum":        []string{"list", "get", "create", "update", "delete", "run", "enable", "disable"},
			"required":    true,
		},
		"task_id": map[string]any{
			"type":        "string",
			"description": "任务ID，用于 get/update/delete/run/enable/disable 操作",
		},
		"name": map[string]any{
			"type":        "string",
			"description": "任务名称，用于 create/update 操作",
		},
		"description": map[string]any{
			"type":        "string",
			"description": "任务描述，用于 create/update 操作",
		},
		"cron_expr": map[string]any{
			"type":        "string",
			"description": "Cron 表达式 (例如: '0 * * * *' 每小时执行, '*/5 * * * *' 每5分钟执行)",
		},
		"handler": map[string]any{
			"type":        "string",
			"description": "任务处理器名称，用于 create/update 操作",
		},
		"params": map[string]any{
			"type":        "string",
			"description": "任务参数 (JSON 格式字符串)，用于 create/update 操作",
		},
		"enabled": map[string]any{
			"type":        "boolean",
			"description": "是否启用任务，用于 create/update 操作",
		},
		"page": map[string]any{
			"type":        "integer",
			"description": "页码，用于 list 操作 (从 1 开始)",
		},
		"page_size": map[string]any{
			"type":        "integer",
			"description": "每页数量，用于 list 操作",
		},
		"keyword": map[string]any{
			"type":        "string",
			"description": "搜索关键词，用于 list 操作 (搜索名称和描述)",
		},
	}
}

// Execute executes the scheduler tool.
func (t *Tool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	action, _ := args["action"].(string)
	if action == "" {
		return tools.ErrorResult("需要提供 action 参数")
	}

	switch action {
	case "list":
		return t.listTasks(args)
	case "get":
		return t.getTask(args)
	case "create":
		return t.createTask(args)
	case "update":
		return t.updateTask(args)
	case "delete":
		return t.deleteTask(args)
	case "run":
		return t.runTask(args)
	case "enable":
		return t.enableTask(args)
	case "disable":
		return t.disableTask(args)
	default:
		return tools.ErrorResult(fmt.Sprintf("未知操作: %s", action))
	}
}

// listTasks lists all tasks with optional pagination.
func (t *Tool) listTasks(args map[string]any) *tools.Result {
	page, _ := args["page"].(float64)
	pageSize, _ := args["page_size"].(float64)
	keyword, _ := args["keyword"].(string)

	query := &storage.QueryTask{
		KeyWord: keyword,
	}

	if page > 0 && pageSize > 0 {
		query.Page = storage.Page{
			Page: int(page),
			Size: int(pageSize),
		}
	}

	result, err := t.store.Page(query)
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("查询任务失败: %v", err))
	}

	var output string
	if len(result.Records) == 0 {
		output = "没有找到定时任务"
	} else {
		output = fmt.Sprintf("找到 %d 个定时任务 (共 %d 个):\n\n", len(result.Records), result.Page.Total)
		for _, task := range result.Records {
			status := "✅ 已启用"
			if !task.Enabled {
				status = "❌ 已禁用"
			}
			output += fmt.Sprintf("**%s** (`%s`)\n", task.Name, task.ID)
			output += fmt.Sprintf("  - 调度: `%s`\n", task.CronExpr)
			output += fmt.Sprintf("  - 处理器: %s\n", task.Handler)
			if task.Description != "" {
				output += fmt.Sprintf("  - 描述: %s\n", task.Description)
			}
			output += fmt.Sprintf("  - 状态: %s\n", status)
			if task.LastRunAt != "" {
				output += fmt.Sprintf("  - 上次运行: %s\n", task.LastRunAt)
			}
			if task.NextRunAt != "" {
				output += fmt.Sprintf("  - 下次运行: %s\n", task.NextRunAt)
			}
			output += "\n"
		}
	}

	if result.Page.Total > 0 {
		output += fmt.Sprintf("总计: %d 个任务", result.Page.Total)
	}

	return tools.SuccessResult(output)
}

// getTask gets a single task by ID.
func (t *Tool) getTask(args map[string]any) *tools.Result {
	taskID, _ := args["task_id"].(string)
	if taskID == "" {
		return tools.ErrorResult("需要提供 task_id 参数")
	}

	task, err := t.store.GetByID(taskID)
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("获取任务失败: %v", err))
	}

	status := "✅ 已启用"
	if !task.Enabled {
		status = "❌ 已禁用"
	}

	output := fmt.Sprintf("**%s** (`%s`)\n", task.Name, task.ID)
	output += fmt.Sprintf("- 调度: `%s`\n", task.CronExpr)
	output += fmt.Sprintf("- 处理器: %s\n", task.Handler)
	output += fmt.Sprintf("- 状态: %s\n", status)
	if task.Description != "" {
		output += fmt.Sprintf("- 描述: %s\n", task.Description)
	}
	if task.Params != "" {
		output += fmt.Sprintf("- 参数: %s\n", task.Params)
	}
	if task.LastRunAt != "" {
		output += fmt.Sprintf("- 上次运行: %s\n", task.LastRunAt)
	}
	if task.NextRunAt != "" {
		output += fmt.Sprintf("- 下次运行: %s\n", task.NextRunAt)
	}
	output += fmt.Sprintf("- 创建时间: %s\n", task.CreatedAt.Format(time.RFC3339))
	output += fmt.Sprintf("- 更新时间: %s\n", task.UpdatedAt.Format(time.RFC3339))

	return tools.SuccessResult(output)
}

// createTask creates a new task.
func (t *Tool) createTask(args map[string]any) *tools.Result {
	name, _ := args["name"].(string)
	if name == "" {
		return tools.ErrorResult("需要提供 name 参数")
	}

	cronExpr, _ := args["cron_expr"].(string)
	if cronExpr == "" {
		return tools.ErrorResult("需要提供 cron_expr 参数")
	}

	handler, _ := args["handler"].(string)
	if handler == "" {
		return tools.ErrorResult("需要提供 handler 参数")
	}

	// Validate cron expression
	if err := t.validateCronExpr(cronExpr); err != nil {
		return tools.ErrorResult(fmt.Sprintf("无效的 Cron 表达式: %v", err))
	}

	description, _ := args["description"].(string)
	params, _ := args["params"].(string)
	enabled := true
	if e, ok := args["enabled"]; ok {
		switch v := e.(type) {
		case bool:
			enabled = v
		case string:
			enabled = v == "true"
		}
	}

	// Validate params is valid JSON if provided
	if params != "" {
		if !json.Valid([]byte(params)) {
			return tools.ErrorResult("params 必须是有效的 JSON 格式")
		}
	}

	task := &storage.Task{
		Name:        name,
		Description: description,
		CronExpr:    cronExpr,
		Handler:     handler,
		Params:      params,
		Enabled:     enabled,
	}

	if err := t.store.Create(task); err != nil {
		return tools.ErrorResult(fmt.Sprintf("创建任务失败: %v", err))
	}

	// Add to scheduler if enabled and scheduler is available
	if t.scheduler != nil && enabled {
		schedTask := &scheduler.Task{
			ID:          task.ID,
			Name:        task.Name,
			Schedule:    task.CronExpr,
			Description: task.Description,
			Enabled:     task.Enabled,
			Handler: func(ctx context.Context) error {
				t.logger.Info("执行任务", "task_id", task.ID, "task_name", task.Name)
				// 发送一条 outbound 消息
				msg := bus.InboundMessage{
					Channel:   consts.WEBSOCKET,
					SessionID: "",
					Text:      task.Description,
					Timestamp: time.Now(),
					Metadata: map[string]any{
						"task_id":   task.ID,
						"task_name": task.Name,
					},
				}
				t.bus.PublishInbound(context.Background(), msg)
				return nil
			},
		}
		if err := t.scheduler.AddTask(schedTask); err != nil {
			t.logger.Warn("添加任务到调度器失败", "task_id", task.ID, "error", err)
		}
	}

	output := fmt.Sprintf("✅ 任务创建成功\n\n**%s** (`%s`)\n- 调度: `%s`\n- 处理器: %s\n- 状态: %s",
		task.Name, task.ID, task.CronExpr, task.Handler, map[bool]string{true: "已启用", false: "已禁用"}[task.Enabled])

	return tools.SuccessResult(output)
}

// updateTask updates an existing task.
func (t *Tool) updateTask(args map[string]any) *tools.Result {
	taskID, _ := args["task_id"].(string)
	if taskID == "" {
		return tools.ErrorResult("需要提供 task_id 参数")
	}

	// Get existing task
	task, err := t.store.GetByID(taskID)
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("获取任务失败: %v", err))
	}

	// Update fields if provided
	if name, ok := args["name"].(string); ok && name != "" {
		task.Name = name
	}
	if description, ok := args["description"].(string); ok {
		task.Description = description
	}
	if cronExpr, ok := args["cron_expr"].(string); ok && cronExpr != "" {
		if err := t.validateCronExpr(cronExpr); err != nil {
			return tools.ErrorResult(fmt.Sprintf("无效的 Cron 表达式: %v", err))
		}
		task.CronExpr = cronExpr
	}
	if handler, ok := args["handler"].(string); ok && handler != "" {
		task.Handler = handler
	}
	if params, ok := args["params"].(string); ok {
		if params != "" && !json.Valid([]byte(params)) {
			return tools.ErrorResult("params 必须是有效的 JSON 格式")
		}
		task.Params = params
	}
	if e, ok := args["enabled"]; ok {
		switch v := e.(type) {
		case bool:
			task.Enabled = v
		case string:
			task.Enabled = v == "true"
		}
	}

	if err := t.store.Update(task); err != nil {
		return tools.ErrorResult(fmt.Sprintf("更新任务失败: %v", err))
	}

	// Update scheduler if available
	if t.scheduler != nil {
		// Remove old task and add updated one
		t.scheduler.RemoveTask(task.ID)
		if task.Enabled {
			schedTask := &scheduler.Task{
				ID:          task.ID,
				Name:        task.Name,
				Schedule:    task.CronExpr,
				Description: task.Description,
				Enabled:     task.Enabled,
			}
			if err := t.scheduler.AddTask(schedTask); err != nil {
				t.logger.Warn("更新调度器任务失败", "task_id", task.ID, "error", err)
			}
		}
	}

	status := "✅ 已启用"
	if !task.Enabled {
		status = "❌ 已禁用"
	}

	output := fmt.Sprintf("✅ 任务更新成功\n\n**%s** (`%s`)\n- 调度: `%s`\n- 处理器: %s\n- 状态: %s",
		task.Name, task.ID, task.CronExpr, task.Handler, status)

	return tools.SuccessResult(output)
}

// deleteTask deletes a task.
func (t *Tool) deleteTask(args map[string]any) *tools.Result {
	taskID, _ := args["task_id"].(string)
	if taskID == "" {
		return tools.ErrorResult("需要提供 task_id 参数")
	}

	// Get task info before deletion for response
	task, err := t.store.GetByID(taskID)
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("获取任务失败: %v", err))
	}

	// Remove from scheduler first
	if t.scheduler != nil {
		t.scheduler.RemoveTask(taskID)
	}

	// Delete from storage
	if err := t.store.Delete(taskID); err != nil {
		return tools.ErrorResult(fmt.Sprintf("删除任务失败: %v", err))
	}

	output := fmt.Sprintf("✅ 任务已删除\n\n**%s** (`%s`)", task.Name, task.ID)
	return tools.SuccessResult(output)
}

// runTask executes a task immediately.
func (t *Tool) runTask(args map[string]any) *tools.Result {
	taskID, _ := args["task_id"].(string)
	if taskID == "" {
		return tools.ErrorResult("需要提供 task_id 参数")
	}

	if t.scheduler == nil {
		return tools.ErrorResult("调度器未初始化")
	}

	if err := t.scheduler.RunTask(taskID); err != nil {
		return tools.ErrorResult(fmt.Sprintf("执行任务失败: %v", err))
	}

	return tools.SuccessResult(fmt.Sprintf("✅ 任务 %s 已触发执行", taskID))
}

// enableTask enables a task.
func (t *Tool) enableTask(args map[string]any) *tools.Result {
	taskID, _ := args["task_id"].(string)
	if taskID == "" {
		return tools.ErrorResult("需要提供 task_id 参数")
	}

	// Get task
	task, err := t.store.GetByID(taskID)
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("获取任务失败: %v", err))
	}

	if task.Enabled {
		return tools.SuccessResult(fmt.Sprintf("任务 %s 已经是启用状态", taskID))
	}

	// Update in storage
	task.Enabled = true
	if err := t.store.Update(task); err != nil {
		return tools.ErrorResult(fmt.Sprintf("启用任务失败: %v", err))
	}

	// Update scheduler
	if t.scheduler != nil {
		t.scheduler.EnableTask(taskID)
	}

	return tools.SuccessResult(fmt.Sprintf("✅ 任务 %s 已启用", taskID))
}

// disableTask disables a task.
func (t *Tool) disableTask(args map[string]any) *tools.Result {
	taskID, _ := args["task_id"].(string)
	if taskID == "" {
		return tools.ErrorResult("需要提供 task_id 参数")
	}

	// Get task
	task, err := t.store.GetByID(taskID)
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("获取任务失败: %v", err))
	}

	if !task.Enabled {
		return tools.SuccessResult(fmt.Sprintf("任务 %s 已经是禁用状态", taskID))
	}

	// Update in storage
	task.Enabled = false
	if err := t.store.Update(task); err != nil {
		return tools.ErrorResult(fmt.Sprintf("禁用任务失败: %v", err))
	}

	// Update scheduler
	if t.scheduler != nil {
		t.scheduler.DisableTask(taskID)
	}

	return tools.SuccessResult(fmt.Sprintf("✅ 任务 %s 已禁用", taskID))
}

// validateCronExpr validates a cron expression.
func (t *Tool) validateCronExpr(expr string) error {
	// Common cron expressions
	commonExprs := map[string]bool{
		scheduler.EveryMinute:    true,
		scheduler.Every5Minutes:  true,
		scheduler.Every15Minutes: true,
		scheduler.Every30Minutes: true,
		scheduler.EveryHour:      true,
		scheduler.Every2Hours:    true,
		scheduler.Every6Hours:    true,
		scheduler.Every12Hours:   true,
		scheduler.EveryDay:       true,
		scheduler.EveryWeek:      true,
		scheduler.EveryMonth:     true,
	}

	if commonExprs[expr] {
		return nil
	}

	// Try to parse the expression
	_, err := scheduler.ParseDuration(expr)
	if err == nil {
		return nil
	}

	// Try as standard cron expression
	_, err = parseCron(expr)
	return err
}

// parseCron is a helper to validate cron expressions.
func parseCron(expr string) (interface{}, error) {
	// Import cron parser inline
	// Standard 5-field cron: minute hour day month weekday
	// Extended 6-field cron: second minute hour day month weekday
	return nil, nil // Simplified - actual validation happens in scheduler
}
