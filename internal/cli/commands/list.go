package commands

import (
	"fmt"
	"os"

	"github.com/mcp-scooter/scooter/internal/cli/client"
	"github.com/mcp-scooter/scooter/internal/cli/errors"
	"github.com/mcp-scooter/scooter/internal/cli/output"
	"github.com/spf13/cobra"
)

var (
	listActive bool
	listSchema bool
)

var listCmd = &cobra.Command{
	Use:   "list [server]",
	Short: "List available servers or tools in a server",
	Run: func(cmd *cobra.Command, args []string) {
		// Initialize client
		// For now, assume default daemon address
		c := client.NewControlClient("http://localhost:6200", "", 0)
		
		var fmtMode output.OutputFormat = output.FormatText
		if jsonOutput {
			fmtMode = output.FormatJSON
		}
		formatter := output.NewFormatter(fmtMode, true)

		if len(args) == 0 {
			// List servers
			entries, err := c.FindTools("")
			if err != nil {
				fmt.Println(formatter.FormatError(errors.Classify(err)))
				os.Exit(1)
			}
			formatter.FormatServers(entries)
		} else {
			// List tools in server
			// TODO: Add server-specific tool listing endpoint
			// For now, let's list all tools (inefficient but works for MVP)
			_ = args[0] // serverName - will be used when filtering is implemented
			tools, err := c.ListTools()
			if err != nil {
				fmt.Println(formatter.FormatError(errors.Classify(err)))
				os.Exit(1)
			}
			
			// In a real implementation, we'd want the daemon to handle this filtering
			formatter.FormatTools(tools)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVar(&listActive, "active", false, "list only active servers")
	listCmd.Flags().BoolVar(&listSchema, "schema", false, "include full schemas")
}
