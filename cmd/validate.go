package cmd

import (
	"fmt"

	"github.com/beamlit/mcp-hub/internal/hub"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the MCP hub",
	Run:   runValidate,
}

func init() {
	validateCmd.Flags().StringVarP(&configPath, "config", "c", "", "The path to the config file")
	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) {
	if configPath == "" {
		cmd.Help()
		return
	}

	hub := hub.Hub{}
	handleError("read config file", hub.Read(configPath))
	handleError("validate config file", hub.ValidateWithDefaultValues())

	// Print validation success message
	fmt.Println("Configuration validated successfully")
}
