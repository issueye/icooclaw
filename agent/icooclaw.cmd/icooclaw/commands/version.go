package commands

import (
	"fmt"
	"runtime"

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
	Short: "Show version information",
	Long: `Show icooclaw version information.
Includes version number, commit hash, build date and build info.`,
	Run: func(cmd *cobra.Command, args []string) {
		printVersion()
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

// printVersion 打印版本信息
func printVersion() {
	fmt.Printf("icooclaw version %s\n", Version)
	fmt.Printf("  Commit:     %s\n", Commit)
	fmt.Printf("  Built:      %s\n", Date)
	fmt.Printf("  Built by:   %s\n", BuiltBy)
	fmt.Printf("  Go version: %s\n", runtime.Version())
	fmt.Printf("  OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
}
