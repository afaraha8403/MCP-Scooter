package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/mcp-scooter/scooter/internal/cli/client"
	"github.com/mcp-scooter/scooter/internal/cli/errors"
	"github.com/mcp-scooter/scooter/internal/cli/output"
	"github.com/spf13/cobra"
)

var (
	autoActivate bool
)

var callCmd = &cobra.Command{
	Use:   "call <server>.<tool> [args...]",
	Short: "Call an MCP tool",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c := client.NewControlClient("http://localhost:6200", "", 0)
		
		var fmtMode output.OutputFormat = output.FormatText
		if jsonOutput {
			fmtMode = output.FormatJSON
		} else if rawOutput {
			fmtMode = output.FormatRaw
		}
		formatter := output.NewFormatter(fmtMode, true)

		target := args[0]
		parts := strings.Split(target, ".")
		if len(parts) != 2 {
			fmt.Printf("Error: Invalid target format. Use server.tool\n")
			os.Exit(1)
		}
		serverName, toolName := parts[0], parts[1]

		// Parse arguments
		toolArgs := make(map[string]interface{})
		for _, arg := range args[1:] {
			kv := strings.SplitN(arg, "=", 2)
			if len(kv) == 2 {
				toolArgs[kv[0]] = kv[1]
			}
		}

		// Call tool
		res, err := c.CallTool(serverName, toolName, toolArgs, profile)
		if err != nil {
			classified := errors.Classify(err)
			
			// Handle auto-activation if needed
			if autoActivate && classified.Kind == errors.ErrorKindNotFound {
				err = c.ActivateTool(serverName, profile)
				if err == nil {
					// Retry call
					res, err = c.CallTool(serverName, toolName, toolArgs, profile)
				}
			}
			
			if err != nil {
				fmt.Println(formatter.FormatError(errors.Classify(err)))
				os.Exit(1)
			}
		}
		
		fmt.Println(formatter.FormatResult(output.NewCallResult(res)))
	},
}

func init() {
	rootCmd.AddCommand(callCmd)
	callCmd.Flags().BoolVar(&autoActivate, "auto-activate", true, "automatically activate server if not active")
}
