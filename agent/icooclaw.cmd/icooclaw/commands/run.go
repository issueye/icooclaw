package commands

import (
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <message>",
	Short: "Run a single message",
	Long: `Send a single message to the AI agent and print the response.

This command is useful for quick one-off interactions with the AI agent.

Examples:
  icooclaw run "Hello, how are you?"
  icooclaw run "Please summarize the following: ..."`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// 检查组件初始化
		// if err := checkInitialized(); err != nil {
		// 	return fmt.Errorf("run: %w", err)
		// }

		// // 合并参数为消息
		// message := joinArgs(args)
		// return runSingleMessage(message)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}

// joinArgs 合并命令行参数
func joinArgs(args []string) string {
	result := ""
	for i, arg := range args {
		if i > 0 {
			result += " "
		}
		result += arg
	}
	return result
}

// runSingleMessage 执行单条消息
func runSingleMessage(message string) error {
	// ctx, cancel := getContext()
	// defer cancel()

	// // 处理信号
	// handleSignals(cancel)

	// // 启动 Agent
	// go agentInstance.Run(ctx, messageBus)

	// logger.Info("Sending message to agent", "content", truncateMessage(message, 100))

	// // 处理消息
	// resp, err := agentInstance.ProcessMessage(ctx, message)
	// if err != nil {
	// 	return fmt.Errorf("process message: %w", err)
	// }

	// fmt.Println("\nResponse:")
	// fmt.Println(resp)

	// // 清理
	// cleanup()
	return nil
}

// truncateMessage 截断消息用于日志显示
func truncateMessage(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
