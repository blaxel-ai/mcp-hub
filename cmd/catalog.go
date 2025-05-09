package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/blaxel-ai/mcp-hub/internal/hub"
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
	catalogCmd.Flags().StringVarP(&configPath, "config", "c", "", "The path to the config files")
	catalogCmd.Flags().BoolVarP(&push, "push", "p", false, "Push the images to the registry")
	catalogCmd.Flags().StringVarP(&registry, "registry", "r", "ghcr.io/blaxel-ai/hub", "The registry to push the images to")
	catalogCmd.Flags().StringVarP(&mcp, "mcp", "m", "", "The MCP to import, if not provided")
	catalogCmd.Flags().BoolVarP(&skipBuild, "skip-build", "s", true, "Skip building the image")
	catalogCmd.Flags().StringVarP(&tag, "tag", "t", "latest", "The tag to use for the image")
	catalogCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug mode, will not save the catalog")
	rootCmd.AddCommand(catalogCmd)
}

func runCatalog(cmd *cobra.Command, args []string) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: No .env file found or error loading it: %v", err)
	}

	if configPath == "" {
		configPath = "hub"
	}
	if mcp == "" {
		log.Printf("MCP is required")
		os.Exit(1)
	}

	// We set debug to true to avoid saving the catalog in control plane
	debug = true
	skipBuild = true

	hub := hub.Hub{}
	handleError("read config file", hub.Read(configPath))
	handleError("validate config file", hub.ValidateWithDefaultValues())

	repository := hub.Repositories[mcp]
	c, err := processRepository(mcp, repository)
	if err != nil {
		log.Printf("Failed to process repository %s: %v", mcp, err)
		os.Exit(1)
	}
	artifact := c.Artifacts[0]
	json, _ := json.MarshalIndent(artifact, "", "  ")
	fmt.Printf("%s", string(json))
}
