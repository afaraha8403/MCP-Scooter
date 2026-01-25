package main

import (
	"os"

	"github.com/mcp-scooter/scooter/internal/cli/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
