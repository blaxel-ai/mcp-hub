package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/beamlit/mcp-hub/internal/catalog"
	"github.com/beamlit/mcp-hub/internal/docker"
	"github.com/beamlit/mcp-hub/internal/git"
	"github.com/beamlit/mcp-hub/internal/hub"
	"github.com/beamlit/mcp-hub/internal/smithery"
	"github.com/spf13/cobra"
)

const (
	tmpDir        = "tmp"
	githubPrefix  = "https://github.com/"
	dockerfileDir = "Dockerfile"
)

var (
	configPath string
	push       bool
	registry   string
	mcp        string
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import MCPs from a config file",
	Long:  `import is a CLI tool to import MCPs from a config file`,
	Run:   runImport,
}

func init() {
	importCmd.Flags().StringVarP(&configPath, "config", "c", "", "The path to the config file")
	importCmd.Flags().BoolVarP(&push, "push", "p", false, "Push the images to the registry")
	importCmd.Flags().StringVarP(&registry, "registry", "r", "ghcr.io/beamlit/hub", "The registry to push the images to")
	importCmd.Flags().StringVarP(&mcp, "mcp", "m", "", "The MCP to import, if not provided, all MCPs will be imported")
	rootCmd.AddCommand(importCmd)
}

func runImport(cmd *cobra.Command, args []string) {
	if configPath == "" {
		cmd.Help()
		return
	}

	hub := hub.Hub{}
	handleError("read config file", hub.Read(configPath))
	handleError("validate config file", hub.ValidateWithDefaultValues())

	setupTempDirectory()
	defer os.RemoveAll(tmpDir)

	for name, repository := range hub.Repositories {
		if mcp != "" && mcp != name {
			continue
		}

		if err := processRepository(name, repository); err != nil {
			log.Printf("Failed to process repository %s: %v", name, err)
		}
	}
}

func processRepository(name string, repository *hub.Repository) error {
	repoPath := fmt.Sprintf("%s/%s/%s", tmpDir, strings.TrimPrefix(repository.Repository, githubPrefix), repository.Branch)
	defer git.DeleteRepository(repoPath)

	if _, err := git.CloneRepository(repoPath, repository.Branch, repository.Repository); err != nil {
		return fmt.Errorf("clone repository: %w", err)
	}

	cfg, err := smithery.Parse(filepath.Join(repoPath, repository.SmitheryPath), repository.Overrider)
	if err != nil {
		return fmt.Errorf("parse smithery file: %w", err)
	}

	imageName := fmt.Sprintf("%s/%s:latest", strings.ToLower(registry), strings.ToLower(name))
	smitheryDir := strings.TrimSuffix(repository.SmitheryPath, "/smithery.yaml")
	deps := manageDeps(repository)

	if err := buildAndPushImage(&cfg, repoPath, smitheryDir, imageName, deps); err != nil {
		return fmt.Errorf("build and push image: %w", err)
	}

	catalog := catalog.Catalog{}
	handleError("load catalog", catalog.Load(name, repository, &cfg))
	handleError("save catalog", catalog.Save())
	return nil
}

func buildAndPushImage(cfg *smithery.SmitheryConfig, repoPath, smitheryDir, imageName string, deps []string) error {
	dockerfilePath := filepath.Join(repoPath, smitheryDir, dockerfileDir)
	if err := docker.Inject(context.Background(), dockerfilePath, cfg.ParsedCommand.Entrypoint(), deps); err != nil {
		return fmt.Errorf("inject command: %w", err)
	}

	buildContext := "."
	if cfg.Build != nil && cfg.Build.DockerBuildPath != nil {
		buildContext = *cfg.Build.DockerBuildPath
	}

	if err := docker.BuildImage(context.Background(), imageName, filepath.Join(smitheryDir, dockerfileDir),
		filepath.Join(repoPath, smitheryDir), buildContext); err != nil {
		return fmt.Errorf("build image: %w", err)
	}

	if push {
		if err := docker.PushImage(context.Background(), imageName); err != nil {
			return fmt.Errorf("push image: %w", err)
		}
	}

	return nil
}

func setupTempDirectory() {
	os.RemoveAll(tmpDir)
	handleError("create temp directory", os.MkdirAll(tmpDir, 0755))
}

func manageDeps(repository *hub.Repository) []string {
	switch repository.PackageManager {
	case hub.PackageManagerNPM:
		return []string{}
	case hub.PackageManagerAPK:
		return []string{
			"apk add --no-cache node npm",
		}
	case hub.PackageManagerAPT:
		return []string{
			"apt-get update",
			"apt-get install -y nodejs npm",
		}
	default:
		log.Fatalf("Unsupported package manager: %s", repository.PackageManager)
		return []string{}
	}
}
