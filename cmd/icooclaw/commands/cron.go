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
	Short: "定时任务管理",
	Long: `管理定时任务。

子命令:
  add <名称> <cron> <消息>    添加新的定时任务
  remove <名称>              删除定时任务
  list                       列出所有定时任务`,
}

var cronAddCmd = &cobra.Command{
	Use:   "add <名称> <cron> <消息>",
	Short: "添加新的定时任务",
	Long: `添加带有名称、cron 表达式和消息的新定时任务。

示例:
  icooclaw cron add mytask "0 * * * *" "Hello from cron"
  icooclaw cron add daily "0 9 * * *" "早安!"`,
	Args: cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		cronExpr := args[1]
		message := args[2]
		addCronTask(name, cronExpr, message)
	},
}

var cronRemoveCmd = &cobra.Command{
	Use:   "remove <名称>",
	Short: "删除定时任务",
	Long: `根据名称删除定时任务。

示例:
  icooclaw cron remove mytask`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		removeCronTask(name)
	},
}

var cronListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有定时任务",
	Long: `列出所有定时任务及其状态和下次运行时间。

示例:
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
		fmt.Printf("错误: 无效的 cron 表达式: %s\n", cronExpr)
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
		Description: "通过 CLI 创建",
	}

	// Add to scheduler
	if err := schedulerInst.AddTask(task); err != nil {
		fmt.Printf("错误: 添加任务失败: %v\n", err)
		return
	}

	fmt.Printf("任务 '%s' 添加成功\n", name)
	fmt.Printf("  Cron: %s\n", cronExpr)
	fmt.Printf("  消息: %s\n", message)
}

func removeCronTask(name string) {
	// Remove from scheduler
	if err := schedulerInst.RemoveTask(name); err != nil {
		fmt.Printf("错误: 删除任务失败: %v\n", err)
		return
	}

	fmt.Printf("任务 '%s' 删除成功\n", name)
}

func listCronTasks() {
	tasks := schedulerInst.ListTasks()

	if len(tasks) == 0 {
		fmt.Println("未找到定时任务。")
		return
	}

	fmt.Println("定时任务:")
	fmt.Println("")

	for _, name := range tasks {
		task := schedulerInst.GetTask(name)
		if task == nil {
			continue
		}

		status := "已禁用"
		if task.Enabled {
			status = "已启用"
		}

		nextRun := "无"
		if next, ok := schedulerInst.GetTaskNextRun(name); ok {
			nextRun = next.Format("2006-01-02 15:04:05")
		}

		fmt.Printf("  名称: %s\n", name)
		fmt.Printf("  状态: %s\n", status)
		fmt.Printf("  Cron: %s\n", task.CronExpr)
		fmt.Printf("  消息: %s\n", task.Message)
		fmt.Printf("  下次运行: %s\n", nextRun)
		fmt.Printf("  上次运行: %s\n", task.LastRunAt.Format("2006-01-02 15:04:05"))
		fmt.Println("")
	}
}
