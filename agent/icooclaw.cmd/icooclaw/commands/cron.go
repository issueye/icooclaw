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
		listCronTasks()
		return nil
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

	// 创建任务
	task := &storage.Task{
		Name:        name,
		CronExpr:    cronExpr,
		Message:     message,
		Channel:     "websocket", // 默认通道
		ChatID:      "default",
		Enabled:     true,
		NextRunAt:   time.Now(),
		Description: "Created via CLI",
	}

	// 添加到调度器
	if err := schedulerInst.AddTaskRunner(task); err != nil {
		return fmt.Errorf("add task: %w", err)
	}

	fmt.Printf("Task '%s' added successfully\n", name)
	fmt.Printf("  Cron: %s\n", cronExpr)
	fmt.Printf("  Message: %s\n", message)
	return nil
}

// removeCronTask 删除定时任务
func removeCronTask(name string) error {
	if err := schedulerInst.RemoveTask(name); err != nil {
		return fmt.Errorf("remove task: %w", err)
	}

	fmt.Printf("Task '%s' removed successfully\n", name)
	return nil
}

// listCronTasks 列出定时任务
func listCronTasks() {
	tasks := schedulerInst.ListTasks()

	if len(tasks) == 0 {
		fmt.Println("No scheduled tasks found.")
		return
	}

	fmt.Println("Scheduled tasks:")
	fmt.Println()

	for _, name := range tasks {
		task := schedulerInst.GetTask(name)
		if task == nil {
			continue
		}

		status := "disabled"
		if task.Enabled {
			status = "enabled"
		}

		nextRun := "none"
		if next, ok := schedulerInst.GetTaskNextRun(name); ok {
			nextRun = next.Format("2006-01-02 15:04:05")
		}

		fmt.Printf("  Name: %s\n", name)
		fmt.Printf("  Status: %s\n", status)
		fmt.Printf("  Cron: %s\n", task.CronExpr)
		fmt.Printf("  Message: %s\n", task.Message)
		fmt.Printf("  Next run: %s\n", nextRun)
		fmt.Printf("  Last run: %s\n", task.LastRunAt.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}
}
