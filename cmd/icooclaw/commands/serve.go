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
	Short: "Start the web server (WebSocket/Webhook)",
	Long: `Start the web server to handle WebSocket connections and webhooks.

This command starts the channels configured in config:
  - WebSocket: configured host:port
  - Webhook: configured host:port

The server requires at least one channel to be enabled in the configuration.`,
	Run: func(cmd *cobra.Command, args []string) {
		runServe()
	},
}

func init() {
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8080, "Port to listen on (for WebSocket)")
	serveCmd.Flags().StringVar(&serveHost, "host", "0.0.0.0", "Host to bind to")
	serveCmd.Flags().BoolVar(&enableWS, "ws", true, "Enable WebSocket endpoint")
	serveCmd.Flags().BoolVar(&enableHook, "webhook", true, "Enable Webhook endpoint")

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
		fmt.Println("Warning: No channels are enabled. Please enable at least one channel in config.")
	} else {
		fmt.Println("Channels started:")
		for _, name := range channels {
			fmt.Printf("  - %s\n", name)
		}
	}

	fmt.Println("Server running. Press Ctrl+C to stop.")

	// Wait for context cancellation
	<-ctx.Done()

	// Cleanup
	cleanup()
}
