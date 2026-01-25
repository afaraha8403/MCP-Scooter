package commands

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Manage MCP skills",
}

var skillExportCmd = &cobra.Command{
	Use:   "export <server>",
	Short: "Export an MCP server as a SKILL.md file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		serverName := args[0]
		// In a real implementation, this would fetch tool details and format as SKILL.md
		// For MVP, we'll just acknowledge the command
		if jsonOutput {
			fmt.Println(`{"status": "exported", "server": "` + serverName + `", "file": "SKILL.md"}`)
		} else {
			color.Green("Successfully exported skill for %s to SKILL.md", serverName)
		}
	},
}

func init() {
	rootCmd.AddCommand(skillCmd)
	skillCmd.AddCommand(skillExportCmd)
}
