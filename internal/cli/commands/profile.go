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

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage Scooter profiles",
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	Run: func(cmd *cobra.Command, args []string) {
		c := client.NewControlClient("http://localhost:6200", "", 0)
		
		var fmtMode output.OutputFormat = output.FormatText
		if jsonOutput {
			fmtMode = output.FormatJSON
		}
		formatter := output.NewFormatter(fmtMode, true)

		profiles, err := c.ListProfiles()
		if err != nil {
			fmt.Println(formatter.FormatError(errors.Classify(err)))
			os.Exit(1)
		}
		
		if jsonOutput {
			data, _ := json.MarshalIndent(profiles, "", "  ")
			fmt.Println(string(data))
		} else {
			color.Cyan("Scooter Profiles:")
			for _, p := range profiles {
				fmt.Printf("  - %s\n", p.ID)
			}
		}
	},
}

var profileShowCmd = &cobra.Command{
	Use:   "show [id]",
	Short: "Show profile details",
	Run: func(cmd *cobra.Command, args []string) {
		c := client.NewControlClient("http://localhost:6200", "", 0)
		
		var fmtMode output.OutputFormat = output.FormatText
		if jsonOutput {
			fmtMode = output.FormatJSON
		}
		formatter := output.NewFormatter(fmtMode, true)

		id := profile
		if len(args) > 0 {
			id = args[0]
		}

		p, err := c.GetProfile(id)
		if err != nil {
			fmt.Println(formatter.FormatError(errors.Classify(err)))
			os.Exit(1)
		}
		
		if jsonOutput {
			data, _ := json.MarshalIndent(p, "", "  ")
			fmt.Println(string(data))
		} else {
			color.Cyan("Profile: %s", p.ID)
			fmt.Printf("  Remote Auth Mode: %s\n", p.RemoteAuthMode)
			fmt.Printf("  Remote URL:       %s\n", p.RemoteServerURL)
			fmt.Printf("  Env Vars:         %v\n", p.Env)
			fmt.Printf("  Allowed Tools:    %v\n", p.AllowTools)
		}
	},
}

func init() {
	rootCmd.AddCommand(profileCmd)
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileShowCmd)
}
