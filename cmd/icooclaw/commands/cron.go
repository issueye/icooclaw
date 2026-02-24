package commands

import (
	"fmt"
	"time"

	"github.com/icooclaw/icooclaw/internal/scheduler"
	"github.com/icooclaw/icooclaw/internal/storage"
	"github.com/spf13/cobra"
)

var cronCmd = &cobra.Command{
	Use:   "cron",
	Short: "Cron task management",
	Long: `Manage scheduled tasks.

Subcommands:
  add <name> <cron> <msg>    Add a new scheduled task
  remove <name>              Remove a scheduled task
  list                       List all scheduled tasks`,
}

var cronAddCmd = &cobra.Command{
	Use:   "add <name> <cron> <msg>",
	Short: "Add a new scheduled task",
	Long: `Add a new scheduled task with a name, cron expression, and message.

Example:
  icooclaw cron add mytask "0 * * * *" "Hello from cron"
  icooclaw cron add daily "0 9 * * *" "Good morning!"`,
	Args: cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		cronExpr := args[1]
		message := args[2]
		addCronTask(name, cronExpr, message)
	},
}

var cronRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a scheduled task",
	Long: `Remove a scheduled task by name.

Example:
  icooclaw cron remove mytask`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		removeCronTask(name)
	},
}

var cronListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all scheduled tasks",
	Long: `List all scheduled tasks with their status and next run time.

Example:
  icooclaw cron list`,
	Run: func(cmd *cobra.Command, args []string) {
		listCronTasks()
	},
}

func init() {
	cronCmd.AddCommand(cronAddCmd)
	cronCmd.AddCommand(cronRemoveCmd)
	cronCmd.AddCommand(cronListCmd)
	rootCmd.AddCommand(cronCmd)
}

func addCronTask(name, cronExpr, message string) {
	// Validate cron expression
	cronParser := scheduler.NewCronParser()
	if !cronParser.IsValid(cronExpr) {
		fmt.Printf("Error: Invalid cron expression: %s\n", cronExpr)
		return
	}

	// Create task
	task := &storage.Task{
		Name:        name,
		CronExpr:    cronExpr,
		Message:     message,
		Channel:     "websocket", // default channel
		ChatID:      "default",
		Enabled:     true,
		NextRunAt:   time.Now(),
		Description: "Created via CLI",
	}

	// Add to scheduler
	if err := schedulerInst.AddTask(task); err != nil {
		fmt.Printf("Error: Failed to add task: %v\n", err)
		return
	}

	fmt.Printf("Task '%s' added successfully\n", name)
	fmt.Printf("  Cron: %s\n", cronExpr)
	fmt.Printf("  Message: %s\n", message)
}

func removeCronTask(name string) {
	// Remove from scheduler
	if err := schedulerInst.RemoveTask(name); err != nil {
		fmt.Printf("Error: Failed to remove task: %v\n", err)
		return
	}

	fmt.Printf("Task '%s' removed successfully\n", name)
}

func listCronTasks() {
	tasks := schedulerInst.ListTasks()

	if len(tasks) == 0 {
		fmt.Println("No scheduled tasks found.")
		return
	}

	fmt.Println("Scheduled tasks:")
	fmt.Println("")

	for _, name := range tasks {
		task := schedulerInst.GetTask(name)
		if task == nil {
			continue
		}

		status := "disabled"
		if task.Enabled {
			status = "enabled"
		}

		nextRun := "N/A"
		if next, ok := schedulerInst.GetTaskNextRun(name); ok {
			nextRun = next.Format("2006-01-02 15:04:05")
		}

		fmt.Printf("  Name: %s\n", name)
		fmt.Printf("  Status: %s\n", status)
		fmt.Printf("  Cron: %s\n", task.CronExpr)
		fmt.Printf("  Message: %s\n", task.Message)
		fmt.Printf("  Next run: %s\n", nextRun)
		fmt.Printf("  Last run: %s\n", task.LastRunAt.Format("2006-01-02 15:04:05"))
		fmt.Println("")
	}
}
