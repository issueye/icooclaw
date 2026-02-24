package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <msg>",
	Short: "Run a single message",
	Long: `Send a single message to the AI agent and print the response.

This command is useful for quick one-off interactions with the AI agent.

Example:
  icooclaw run "Hello, how are you?"
  icooclaw run "Summarize the following: ..."`,
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
	logger.Info("Sending message to agent", "content", message)

	resp, err := agentInstance.ProcessMessage(ctx, message)
	if err != nil {
		logger.Error("Failed to process message", "error", err)
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("\nResponse:")
	fmt.Println(resp)

	// Cleanup
	cleanup()
}
