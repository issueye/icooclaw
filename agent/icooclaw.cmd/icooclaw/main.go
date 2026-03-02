package main

import (
	"icooclaw.cmd/icooclaw/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		// Error is already handled in commands
	}
}
