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
	Short: "Display version information",
	Long: `Display the version information for icooclaw.
This includes the version number, commit hash, build date, and build information.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("icooclaw version %s\n", Version)
		fmt.Printf("  commit: %s\n", Commit)
		fmt.Printf("  date: %s\n", Date)
		fmt.Printf("  built by: %s\n", BuiltBy)
		os.Exit(0)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
