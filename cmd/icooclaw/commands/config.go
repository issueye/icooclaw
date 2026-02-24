package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management",
	Long: `Manage configuration settings.

Subcommands:
  get <key>             Get a configuration value
  set <key> <value>    Set a configuration value`,
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Long: `Get a configuration value by key.

Example:
  icooclaw config get log.level
  icooclaw config get agents.default_provider`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		getConfig(key)
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value by key.
Note: This only sets the value in memory. To persist, save to config file.

Example:
  icooclaw config set log.level debug
  icooclaw config set agents.default_provider openai`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		value := args[1]
		setConfig(key, value)
	},
}

func init() {
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	rootCmd.AddCommand(configCmd)
}

func getConfig(key string) {
	value := viper.Get(key)
	if value == nil {
		fmt.Printf("Key not found: %s\n", key)
		return
	}

	fmt.Printf("%s = %v\n", key, value)
}

func setConfig(key, value string) {
	// Try to determine the type
	switch value {
	case "true", "false":
		viper.Set(key, value == "true")
	case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
		// Check if it's a number
		viper.Set(key, value)
	default:
		// Try to parse as other types
		viper.Set(key, value)
	}

	fmt.Printf("Set %s = %s\n", key, value)
	fmt.Println("Note: This change is only in memory. Restart the application for changes to take effect.")
}
