package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mcp-hub-importer",
	Short: "Import MCPs from a directory",
	Long: `mcp-hub-importer is a CLI tool to import MCPs from a config file.
It supports validating and importing MCP configurations.`,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// handleError is a helper function for consistent error handling across commands
func handleError(operation string, err error) {
	if err != nil {
		log.Fatalf("Failed to %s: %v", operation, err)
	}
}
