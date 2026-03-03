package commands

import (
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
	Short: "Start web server (WebSocket/Webhook)",
	Long: `Start a web server to handle WebSocket connections and Webhooks.

This command starts channels configured in the configuration file:
  - WebSocket: configured host:port
  - Webhook: configured host:port

At least one channel must be enabled in the configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// if err := checkInitialized(); err != nil {
		// 	return fmt.Errorf("serve: %w", err)
		// }
		// return runServe()
		return nil
	},
}

func init() {
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8080, "listen port (for WebSocket)")
	serveCmd.Flags().StringVar(&serveHost, "host", "0.0.0.0", "bind address")
	serveCmd.Flags().BoolVar(&enableWS, "ws", true, "enable WebSocket endpoint")
	serveCmd.Flags().BoolVar(&enableHook, "webhook", true, "enable Webhook endpoint")

	rootCmd.AddCommand(serveCmd)
}

// runServe 运行服务
func runServe() error {
	// ctx, cancel := getContext()
	// defer cancel()

	// // 处理信号
	// handleSignals(cancel)

	// // 启动 Agent
	// go agentInstance.Run(ctx, messageBus)

	// // 启动所有启用的通道
	// if err := channelManager.StartAll(); err != nil {
	// 	return fmt.Errorf("start channels: %w", err)
	// }

	// // 列出已启动的通道
	// channels := channelManager.List()
	// if len(channels) == 0 {
	// 	fmt.Println("Warning: No channels enabled. Please enable at least one channel in configuration.")
	// } else {
	// 	fmt.Println("Started channels:")
	// 	for _, name := range channels {
	// 		fmt.Printf("  - %s\n", name)
	// 	}
	// }

	// fmt.Println("Server running. Press Ctrl+C to stop.")

	// // 等待上下文取消
	// <-ctx.Done()

	// // 清理
	// cleanup()
	return nil
}
