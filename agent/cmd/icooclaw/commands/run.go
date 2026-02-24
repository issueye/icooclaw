package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <消息>",
	Short: "运行单条消息",
	Long: `向 AI 代理发送单条消息并打印响应。

此命令用于与 AI 代理的快速一次性交互。

示例:
  icooclaw run "你好，最近怎么样？"
  icooclaw run "请总结以下内容: ..."`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		message := ""
		for i, arg := range args {
			if i > 0 {
				message += " "
			}
			message += arg
		}
		runSingleMessage(message)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func runSingleMessage(message string) {
	ctx, cancel := getContext()
	defer cancel()

	// Handle signals
	handleSignals(cancel)

	// Start agent
	go agentInstance.Run(ctx, messageBus)

	// Send message
	logger.Info("正在发送消息给代理", "content", message)

	resp, err := agentInstance.ProcessMessage(ctx, message)
	if err != nil {
		logger.Error("处理消息失败", "error", err)
		fmt.Printf("错误: %v\n", err)
		return
	}

	fmt.Println("\n回复:")
	fmt.Println(resp)

	// Cleanup
	cleanup()
}
