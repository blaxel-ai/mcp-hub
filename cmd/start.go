package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/blaxel-ai/mcp-hub/internal/catalog"
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
	startCmd.Flags().StringVarP(&configPath, "config", "c", "", "The path to the config files")
	startCmd.Flags().BoolVarP(&push, "push", "p", false, "Push the images to the registry")
	startCmd.Flags().StringVarP(&registry, "registry", "r", "ghcr.io/blaxel-ai/hub", "The registry to push the images to")
	startCmd.Flags().StringVarP(&mcp, "mcp", "m", "", "The MCP to import, if not provided")
	startCmd.Flags().BoolVarP(&skipBuild, "skip-build", "s", false, "Skip building the image")
	startCmd.Flags().StringVarP(&tag, "tag", "t", "latest", "The tag to use for the image")
	startCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug mode, will not save the catalog")
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) {
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

	hub := hub.Hub{}
	handleError("read config file", hub.Read(configPath))
	handleError("validate config file", hub.ValidateWithDefaultValues())

	repository := hub.Repositories[mcp]
	if repository == nil {
		log.Printf("Repository %s not found", mcp)
		os.Exit(1)
	}
	c, err := processRepository(mcp, repository)
	if err != nil {
		log.Printf("Failed to process repository %s: %v", mcp, err)
		os.Exit(1)
	}
	artifact := c.Artifacts[0]
	envKeys := []string{}
	for key := range artifact.Entrypoint.Env {
		envKeys = append(envKeys, key)
		err := checkEnvironmentVariable(artifact, key, artifact.Entrypoint.Env[key])
		if err != nil {
			log.Printf(err.Error())
			os.Exit(1)
		}
	}
	log.Printf("Starting MCP %s", mcp)
	err = dockerRun(artifact, envKeys)
	if err != nil {
		log.Printf("Failed to run docker command: %v", err)
		os.Exit(1)
	}
}

func dockerRun(artifact catalog.Artifact, envKeys []string) error {
	name := fmt.Sprintf("mcp-hub-%s", mcp)
	exec.Command("docker", "rm", "-f", name).Run()
	dockerRunCmd := []string{"run", "--platform", "linux/amd64", "--rm", "-i", "-p", "1400:80", "--name", name}
	for _, key := range envKeys {
		dockerRunCmd = append(dockerRunCmd, "-e", fmt.Sprintf("%s=%s", key, os.Getenv(key)))
	}
	dockerRunCmd = append(dockerRunCmd, artifact.Image)

	dockerCmd := artifact.Entrypoint.Command
	for _, arg := range artifact.Entrypoint.Args {
		if strings.HasPrefix(arg, "$") {
			// Convert camelCase to ENV_VAR format
			envVar := strings.TrimPrefix(arg, "$")
			envVarName := camelToEnvVar(envVar)
			dockerCmd += " " + os.Getenv(envVarName)
		} else {
			dockerCmd += " " + arg
		}
	}
	dockerRunCmd = append(dockerRunCmd, dockerCmd)

	fmt.Println("Running docker command: " + strings.Join(dockerRunCmd, " "))

	cmd := exec.Command("docker", dockerRunCmd...)
	// Connect command's stdout and stderr to our process stdout and stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command and wait for it to finish
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("Failed to run docker command \"docker %s\": %v", strings.Join(dockerRunCmd, " "), err)
	}
	return nil
}

func checkEnvironmentVariable(artifact catalog.Artifact, key string, val string) error {
	trimedVal := strings.Trim(val, "$")
	required := false

	if _, ok := artifact.Form.Config[trimedVal]; ok {
		required = artifact.Form.Config[trimedVal].Required
	}

	if _, ok := artifact.Form.Secrets[trimedVal]; ok {
		required = artifact.Form.Secrets[trimedVal].Required
	}

	if required && os.Getenv(key) == "" {
		return fmt.Errorf("Environment variable %s is not set and is required for the MCP %s", key, mcp)
	}
	return nil
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
