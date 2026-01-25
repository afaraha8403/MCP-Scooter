package commands

import (
	"os"

	"github.com/mcp-scooter/scooter/internal/cli/inference"
	"github.com/spf13/cobra"
)

var (
	cfgFile    string
	profile    string
	logLevel   string
	jsonOutput bool
	rawOutput  bool
	directMode bool
	timeout    int
)

var rootCmd = &cobra.Command{
	Use:   "scooter",
	Short: "MCP Scooter CLI - The universal OS for Model Context Protocol",
	Long: `MCP Scooter is a lightweight, native desktop application that serves as the 
universal "Operating System" for the Model Context Protocol (MCP). 
This CLI allows you to interact with the Scooter daemon or run in direct mode.`,
}

func Execute() error {
	// Simple command inference - prepend inferred command to args
	if len(os.Args) > 1 {
		inferredCmd, _ := inference.InferCommand(os.Args[1:])
		if inferredCmd != "" {
			// Insert the inferred command after the program name
			newArgs := make([]string, 0, len(os.Args)+1)
			newArgs = append(newArgs, os.Args[0])
			newArgs = append(newArgs, inferredCmd)
			newArgs = append(newArgs, os.Args[1:]...)
			os.Args = newArgs
		}
	}
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/mcp-scooter/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&profile, "profile", "default", "profile to use")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output in JSON format")
	rootCmd.PersistentFlags().BoolVar(&rawOutput, "raw", false, "raw output (no formatting)")
	rootCmd.PersistentFlags().BoolVar(&directMode, "direct", false, "direct mode (no daemon, for headless servers)")
	rootCmd.PersistentFlags().IntVar(&timeout, "timeout", 30000, "request timeout in milliseconds")
}
