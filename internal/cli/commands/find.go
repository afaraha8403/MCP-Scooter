package commands

import (
	"fmt"
	"os"

	"github.com/mcp-scooter/scooter/internal/cli/client"
	"github.com/mcp-scooter/scooter/internal/cli/errors"
	"github.com/mcp-scooter/scooter/internal/cli/output"
	"github.com/spf13/cobra"
)

var findCmd = &cobra.Command{
	Use:   "find <query>",
	Short: "Search for tools by capability or server name",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c := client.NewControlClient("http://localhost:6200", "", 0)
		
		var fmtMode output.OutputFormat = output.FormatText
		if jsonOutput {
			fmtMode = output.FormatJSON
		}
		formatter := output.NewFormatter(fmtMode, true)

		query := args[0]
		entries, err := c.FindTools(query)
		if err != nil {
			fmt.Println(formatter.FormatError(errors.Classify(err)))
			os.Exit(1)
		}
		
		formatter.FormatServers(entries)
	},
}

func init() {
	rootCmd.AddCommand(findCmd)
}
