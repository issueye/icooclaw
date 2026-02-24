package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	servePort  int
	serveHost  string
	enableWS   bool
	enableHook bool
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "启动 Web 服务器 (WebSocket/Webhook)",
	Long: `启动 Web 服务器以处理 WebSocket 连接和 Webhook。

此命令启动配置文件中配置的通道:
  - WebSocket: 配置的 host:port
  - Webhook: 配置的 host:port

服务器需要至少在配置中启用一个通道。`,
	Run: func(cmd *cobra.Command, args []string) {
		runServe()
	},
}

func init() {
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8080, "监听端口 (用于 WebSocket)")
	serveCmd.Flags().StringVar(&serveHost, "host", "0.0.0.0", "绑定地址")
	serveCmd.Flags().BoolVar(&enableWS, "ws", true, "启用 WebSocket 端点")
	serveCmd.Flags().BoolVar(&enableHook, "webhook", true, "启用 Webhook 端点")

	rootCmd.AddCommand(serveCmd)
}

func runServe() {
	ctx, cancel := getContext()
	defer cancel()

	// Handle signals
	handleSignals(cancel)

	// Start agent
	go agentInstance.Run(ctx, messageBus)

	// Start enabled channels
	if err := channelManager.StartAll(); err != nil {
		logger.Error("Failed to start channels", "error", err)
		fmt.Printf("Error: Failed to start channels: %v\n", err)
		return
	}

	// List started channels
	channels := channelManager.List()
	if len(channels) == 0 {
		fmt.Println("警告: 没有启用的通道。请在配置中至少启用一个通道。")
	} else {
		fmt.Println("已启动的通道:")
		for _, name := range channels {
			fmt.Printf("  - %s\n", name)
		}
	}

	fmt.Println("服务器正在运行。按 Ctrl+C 停止。")

	// Wait for context cancellation
	<-ctx.Done()

	// Cleanup
	cleanup()
}
