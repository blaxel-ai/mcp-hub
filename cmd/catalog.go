package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/beamlit/mcp-hub/internal/builder"
	"github.com/beamlit/mcp-hub/internal/errors"
	"github.com/beamlit/mcp-hub/internal/hub"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var catalogCmd = &cobra.Command{
	Use:   "catalog",
	Short: "Show a MCP server configuration",
	Long:  `catalog is a CLI tool to show a MCP server configuration`,
	Run:   runCatalog,
}

func init() {
	catalogCmd.Flags().StringVarP(&configPath, "config", "c", "hub", "The path to the config files")
	catalogCmd.Flags().StringVarP(&mcp, "mcp", "m", "", "The MCP to import, if not provided")
	catalogCmd.Flags().StringVarP(&tag, "tag", "t", "latest", "The tag to use for the image")
	catalogCmd.Flags().StringVarP(&registry, "registry", "r", "ghcr.io/beamlit/mcp-hub", "The registry to use for the image")
	rootCmd.AddCommand(catalogCmd)
}

func runCatalog(cmd *cobra.Command, args []string) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: No .env file found or error loading it: %v", err)
	}

	if mcp == "" {
		log.Printf("MCP is required")
		os.Exit(1)
	}

	fmt.Printf("Start catalog for %s\n", mcp)
	hub := hub.Hub{}
	errors.HandleError("read config file", hub.Read(configPath))
	errors.HandleError("validate config file", hub.ValidateWithDefaultValues())

	repository := hub.Repositories[mcp]
	buildInstance := builder.NewBuild(tag, registry, true)
	defer buildInstance.Clean()
	c, err := buildInstance.CloneRepository(mcp, repository)
	if err != nil {
		log.Printf("Failed to process repository %s: %v", mcp, err)
		os.Exit(1)
	}
	artifact := c.Artifacts[0]
	json, _ := json.MarshalIndent(artifact, "", "  ")
	fmt.Printf("%s", string(json))
}
