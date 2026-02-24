package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "配置管理",
	Long: `管理配置设置。

子命令:
  get <key>             获取配置值
  set <key> <value>    设置配置值`,
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "获取配置值",
	Long: `通过键获取配置值。

示例:
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
	Short: "设置配置值",
	Long: `通过键设置配置值。
注意: 这只设置内存中的值。要持久化，请保存到配置文件。

示例:
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
		fmt.Printf("未找到键: %s\n", key)
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

	fmt.Printf("已设置 %s = %s\n", key, value)
	fmt.Println("注意: 此更改仅在内存中。重启应用程序以使更改生效。")
}
