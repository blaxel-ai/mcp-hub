package cmd

import (
	"log"
	"os"

	"github.com/blaxel-ai/mcp-hub/internal/builder"
	"github.com/blaxel-ai/mcp-hub/internal/errors"
	"github.com/blaxel-ai/mcp-hub/internal/hub"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Build & Start the MCP server",
	Long:  `start is a CLI tool to build & start the MCP server`,
	Run:   runStart,
}

func init() {
	startCmd.Flags().StringVarP(&configPath, "config", "c", "hub", "The path to the config files")
	startCmd.Flags().StringVarP(&registry, "registry", "r", "ghcr.io/blaxel-ai/hub", "The registry to push the images to")
	startCmd.Flags().StringVarP(&mcp, "mcp", "m", "", "The MCP to import, if not provided")
	startCmd.Flags().StringVarP(&tag, "tag", "t", "latest", "The tag to use for the image")
	startCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug mode, will not save the catalog")
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: No .env file found or error loading it: %v", err)
	}

	if mcp == "" {
		log.Printf("MCP is required")
		os.Exit(1)
	}

	// We set debug to true to avoid saving the catalog in control plane
	debug = true

	hub := hub.Hub{}
	errors.HandleError("read config file", hub.Read(configPath))
	errors.HandleError("validate config file", hub.ValidateWithDefaultValues())

	repository := hub.Repositories[mcp]
	if repository == nil {
		log.Printf("Repository %s not found", mcp)
		os.Exit(1)
	}
	buildInstance := builder.NewBuild(tag, registry, debug)
	defer buildInstance.Clean()

	c, err := buildInstance.CloneRepository(mcp, repository)
	if err != nil {
		log.Printf("Failed to process repository %s: %v", mcp, err)
		os.Exit(1)
	}
	err = buildInstance.Build(mcp, repository)
	if err != nil {
		log.Printf("Failed to build image for repository %s: %v", mcp, err)
		os.Exit(1)
	}

	err = buildInstance.Start(mcp, repository, c)

	if err != nil {
		log.Printf("Failed to start image for repository %s: %v", mcp, err)
		os.Exit(1)
	}
	if !debug {
		errors.HandleError("save catalog", c.Save())
	}
}

// camelToEnvVar converts a camelCase string to an ENV_VAR format (uppercase with underscores)
func camelToEnvVar(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && 'A' <= r && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToUpper(result.String())
}
