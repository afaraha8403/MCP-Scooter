package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/mcp-scooter/scooter/internal/cli/client"
	"github.com/mcp-scooter/scooter/internal/cli/errors"
	"github.com/mcp-scooter/scooter/internal/cli/output"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Scooter daemon status",
	Run: func(cmd *cobra.Command, args []string) {
		c := client.NewControlClient("http://localhost:6200", "", 0)
		
		var fmtMode output.OutputFormat = output.FormatText
		if jsonOutput {
			fmtMode = output.FormatJSON
		}
		formatter := output.NewFormatter(fmtMode, true)

		status, err := c.GetStatus()
		if err != nil {
			fmt.Println(formatter.FormatError(errors.Classify(err)))
			os.Exit(1)
		}
		
		if jsonOutput {
			data, _ := json.MarshalIndent(status, "", "  ")
			fmt.Println(string(data))
		} else {
			color.Cyan("Scooter Daemon Status:")
			fmt.Printf("  Running: %v\n", status.Running)
			fmt.Printf("  Version: %s\n", status.Version)
			fmt.Printf("  Uptime:  %s\n", status.Uptime)
			fmt.Printf("  Profile: %s\n", status.ActiveProfile)
			fmt.Printf("  Active Servers: %v\n", status.ActiveServers)
			fmt.Printf("  Control API:    :%d\n", status.Ports.Control)
			fmt.Printf("  MCP Gateway:    :%d\n", status.Ports.Gateway)
		}
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
