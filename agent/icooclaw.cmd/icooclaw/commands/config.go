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
  get <key>           Get configuration value
  set <key> <value>   Set configuration value`,
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get configuration value",
	Long: `Get configuration value by key.

Examples:
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
	Short: "Set configuration value",
	Long: `Set configuration value by key.
Note: This only sets the value in memory. To persist, save to config file.

Examples:
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
	// 尝试推断类型
	switch value {
	case "true":
		viper.Set(key, true)
	case "false":
		viper.Set(key, false)
	default:
		viper.Set(key, value)
	}

	fmt.Printf("Set %s = %s\n", key, value)
	fmt.Println("Note: This change is in memory only. Restart the application for changes to take effect.")
}
