package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	configPath string
	push       bool
	registry   string
	mcp        string
	skipBuild  bool
	tag        string
	debug      bool
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
