package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version info variables - can be set via ldflags during build
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
	BuiltBy = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Long: `显示 icooclaw 的版本信息。
包括版本号、提交哈希、构建日期和构建信息。`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("icooclaw 版本 %s\n", Version)
		fmt.Printf("  提交: %s\n", Commit)
		fmt.Printf("  日期: %s\n", Date)
		fmt.Printf("  构建者: %s\n", BuiltBy)
		os.Exit(0)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
