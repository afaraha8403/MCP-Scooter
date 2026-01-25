package commands

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/mcp-scooter/scooter/internal/cli/client"
	"github.com/mcp-scooter/scooter/internal/cli/errors"
	"github.com/mcp-scooter/scooter/internal/cli/output"
	"github.com/spf13/cobra"
)

var activateCmd = &cobra.Command{
	Use:   "activate <server>",
	Short: "Activate an MCP server",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c := client.NewControlClient("http://localhost:6200", "", 0)
		
		var fmtMode output.OutputFormat = output.FormatText
		if jsonOutput {
			fmtMode = output.FormatJSON
		}
		formatter := output.NewFormatter(fmtMode, true)

		serverName := args[0]
		err := c.ActivateTool(serverName, profile)
		if err != nil {
			fmt.Println(formatter.FormatError(errors.Classify(err)))
			os.Exit(1)
		}
		
		if jsonOutput {
			fmt.Println(`{"status": "activated", "server": "` + serverName + `"}`)
		} else {
			color.Green("Successfully activated server: %s", serverName)
		}
	},
}

func init() {
	rootCmd.AddCommand(activateCmd)
}
