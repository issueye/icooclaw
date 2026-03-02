package commands

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"icooclaw.ai/storage"
	scheduler "icooclaw.scheduler"
)

var cronCmd = &cobra.Command{
	Use:   "cron",
	Short: "Scheduled task management",
	Long: `Manage scheduled tasks.

Subcommands:
  add <name> <cron> <message>   Add a new scheduled task
  remove <name>                 Remove a scheduled task
  list                          List all scheduled tasks`,
}

var cronAddCmd = &cobra.Command{
	Use:   "add <name> <cron> <message>",
	Short: "Add a new scheduled task",
	Long: `Add a new scheduled task with name, cron expression and message.

Examples:
  icooclaw cron add mytask "0 * * * *" "Hello from cron"
  icooclaw cron add daily "0 9 * * *" "Good morning!"`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := checkInitialized(); err != nil {
			return fmt.Errorf("cron add: %w", err)
		}
		name := args[0]
		cronExpr := args[1]
		message := args[2]
		return addCronTask(name, cronExpr, message)
	},
}

var cronRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a scheduled task",
	Long: `Remove a scheduled task by name.

Examples:
  icooclaw cron remove mytask`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := checkInitialized(); err != nil {
			return fmt.Errorf("cron remove: %w", err)
		}
		name := args[0]
		return removeCronTask(name)
	},
}

var cronListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all scheduled tasks",
	Long: `List all scheduled tasks with their status and next run time.

Examples:
  icooclaw cron list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := checkInitialized(); err != nil {
			return fmt.Errorf("cron list: %w", err)
		}
		return listCronTasks()
	},
}

func init() {
	cronCmd.AddCommand(cronAddCmd)
	cronCmd.AddCommand(cronRemoveCmd)
	cronCmd.AddCommand(cronListCmd)
	rootCmd.AddCommand(cronCmd)
}

// addCronTask 添加定时任务
func addCronTask(name, cronExpr, message string) error {
	// 验证 cron 表达式
	cronParser := scheduler.NewCronParser()
	if !cronParser.IsValid(cronExpr) {
		return fmt.Errorf("invalid cron expression: %s", cronExpr)
	}

	// 计算下次执行时间
	nextRun, ok := cronParser.NextRun(cronExpr, time.Now())
	if !ok {
		return fmt.Errorf("failed to calculate next run time")
	}

	// 创建任务存储记录
	task := &storage.Task{
		Name:        name,
		CronExpr:    cronExpr,
		Message:     message,
		Channel:     "websocket", // 默认通道
		ChatID:      "default",
		Enabled:     true,
		NextRunAt:   nextRun,
		Description: "Created via CLI",
	}

	// 保存到数据库
	store := agentInstance.Storage()
	if err := store.CreateTask(task); err != nil {
		return fmt.Errorf("save task to database: %w", err)
	}

	// 如果调度器正在运行，创建 TaskRunner 并添加
	if schedulerInst != nil && schedulerInst.IsRunning() {
		taskInfo := &scheduler.TaskInfo{
			ID:          task.ID,
			Name:        task.Name,
			Description: task.Description,
			Type:        scheduler.TaskTypeCron,
			CronExpr:    task.CronExpr,
			Message:     task.Message,
			Channel:     task.Channel,
			ChatID:      task.ChatID,
			Enabled:     task.Enabled,
			NextRunAt:   task.NextRunAt,
			LastRunAt:   task.LastRunAt,
		}

		runner, err := scheduler.NewCronTaskRunner(name, taskInfo, cronExpr, logger)
		if err != nil {
			logger.Warn("Failed to create task runner", "error", err)
		} else if err := schedulerInst.AddTaskRunner(runner); err != nil {
			logger.Warn("Failed to add task runner to scheduler", "error", err)
		}
	}

	fmt.Printf("Task '%s' added successfully\n", name)
	fmt.Printf("  Cron: %s\n", cronExpr)
	fmt.Printf("  Message: %s\n", message)
	fmt.Printf("  Next run: %s\n", nextRun.Format("2006-01-02 15:04:05"))
	return nil
}

// removeCronTask 删除定时任务
func removeCronTask(name string) error {
	// 从调度器移除
	if schedulerInst != nil && schedulerInst.IsRunning() {
		if err := schedulerInst.RemoveTaskRunner(name); err != nil {
			logger.Debug("Task runner not found in scheduler", "name", name)
		}
	}

	// 从数据库删除
	store := agentInstance.Storage()
	task, err := store.GetTaskByName(name)
	if err != nil {
		return fmt.Errorf("task not found: %s", name)
	}

	if err := store.DeleteTask(task.ID); err != nil {
		return fmt.Errorf("delete task from database: %w", err)
	}

	fmt.Printf("Task '%s' removed successfully\n", name)
	return nil
}

// listCronTasks 列出定时任务
func listCronTasks() error {
	store := agentInstance.Storage()
	tasks, err := store.GetAllTasks()
	if err != nil {
		return fmt.Errorf("get tasks: %w", err)
	}

	if len(tasks) == 0 {
		fmt.Println("No scheduled tasks found.")
		return nil
	}

	fmt.Println("Scheduled tasks:")
	fmt.Println()

	for _, task := range tasks {
		status := "disabled"
		if task.Enabled {
			status = "enabled"
		}

		nextRun := "not scheduled"
		if !task.NextRunAt.IsZero() {
			nextRun = task.NextRunAt.Format("2006-01-02 15:04:05")
		}

		lastRun := "never"
		if !task.LastRunAt.IsZero() {
			lastRun = task.LastRunAt.Format("2006-01-02 15:04:05")
		}

		fmt.Printf("  Name: %s\n", task.Name)
		fmt.Printf("  Status: %s\n", status)
		fmt.Printf("  Cron: %s\n", task.CronExpr)
		fmt.Printf("  Message: %s\n", task.Message)
		fmt.Printf("  Next run: %s\n", nextRun)
		fmt.Printf("  Last run: %s\n", lastRun)
		fmt.Println()
	}

	return nil
}
