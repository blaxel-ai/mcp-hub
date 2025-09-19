package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/blaxel-ai/mcp-hub/internal/catalog"
	"github.com/blaxel-ai/mcp-hub/internal/docker"
	"github.com/blaxel-ai/mcp-hub/internal/git"
	"github.com/blaxel-ai/mcp-hub/internal/hub"
	"github.com/blaxel-ai/mcp-hub/internal/smithery"
	"github.com/spf13/cobra"
)

const (
	tmpDir       = "tmp"
	githubPrefix = "https://github.com/"
	dockerfile   = "Dockerfile"
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import MCPs from a config file",
	Long:  `import is a CLI tool to import MCPs from a config file`,
	Run:   runImport,
}

func init() {
	importCmd.Flags().StringVarP(&configPath, "config", "c", "", "The path to the config files")
	importCmd.Flags().BoolVarP(&push, "push", "p", false, "Push the images to the registry")
	importCmd.Flags().StringVarP(&registry, "registry", "r", "ghcr.io/blaxel-ai/hub", "The registry to push the images to")
	importCmd.Flags().StringVarP(&mcp, "mcp", "m", "", "The MCP to import, if not provided, all MCPs will be imported")
	importCmd.Flags().BoolVarP(&skipBuild, "skip-build", "s", false, "Skip building the image")
	importCmd.Flags().StringVarP(&tag, "tag", "t", "latest", "The tag to use for the image")
	importCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug mode, will not save the catalog")
	rootCmd.AddCommand(importCmd)
}

func runImport(cmd *cobra.Command, args []string) {
	if configPath == "" {
		configPath = "hub"
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
		_, err := processRepository(name, repository)
		if err != nil {
			log.Printf("Failed to process repository %s: %v", name, err)
			os.Exit(1)
		}
	}
}

func processRepository(name string, repository *hub.Repository) (*catalog.Catalog, error) {
	var repoPath string
	var shouldCleanupRepo bool
	imageName := fmt.Sprintf("%s:%s", strings.ToLower(name), tag)
	templateImageName := fmt.Sprintf("%s:latest", strings.ToLower(name))

	if repository.Path != "" {
		repoPath = repository.Path
	} else if repository.Repository != "" {
		// Has repository URL, clone it
		repoPath = fmt.Sprintf("%s/%s/%s", tmpDir, strings.TrimPrefix(repository.Repository, githubPrefix), repository.Branch)
		shouldCleanupRepo = true
	} else {
		// No repository URL, create a temporary directory
		repoPath = fmt.Sprintf("%s/%s", tmpDir, name)
		shouldCleanupRepo = true
	}

	if shouldCleanupRepo {
		defer git.DeleteRepository(repoPath)
	}

	if repository.Disabled {
		c := catalog.Catalog{}
		handleError("load catalog", c.Load(name, repository, templateImageName, &smithery.SmitheryConfig{}))
		if !debug {
			handleError("save catalog", c.Save())
		}
		return &c, nil
	}

	if repository.Path == "" && repository.Repository != "" {
		// Only clone if we have a repository URL
		if _, err := git.CloneRepository(repoPath, repository.Branch, repository.Repository); err != nil {
			return nil, fmt.Errorf("clone repository: %w", err)
		}
	} else if repository.Path == "" && repository.Repository == "" {
		// Create empty directory for repositories without a source repo
		if err := os.MkdirAll(repoPath, 0755); err != nil {
			return nil, fmt.Errorf("create temporary directory: %w", err)
		}
	}

	var cfg *smithery.SmitheryConfig

	if repository.Smithery != nil {
		cfg = repository.Smithery
		parsedCommand, err := smithery.ExecuteCommandFunction(cfg.StartCommand.CommandFunction, cfg.StartCommand.ConfigSchema.Properties)
		if err != nil {
			return nil, fmt.Errorf("execute command function: %w", err)
		}
		parsedCommand.Type = cfg.StartCommand.Type
		cfg.ParsedCommand = parsedCommand
	} else {
		tmpCfg, err := smithery.Parse(filepath.Join(repoPath, repository.SmitheryPath))
		if err != nil {
			return nil, fmt.Errorf("parse smithery file: %w", err)
		}
		cfg = &tmpCfg
	}

	buildTo := fmt.Sprintf("%s/%s", strings.ToLower(registry), imageName)
	if !skipBuild {
		deps := manageDeps(repository)
		if err := buildAndPushImage(cfg, name, repository.SmitheryPath, repoPath, strings.TrimSuffix(repository.Dockerfile, "/Dockerfile"), buildTo, deps); err != nil {
			return nil, fmt.Errorf("build and push image: %w", err)
		}
	}

	c := catalog.Catalog{}
	handleError("load catalog", c.Load(name, repository, templateImageName, cfg))
	if !debug {
		handleError("save catalog", c.Save())
	}
	return &c, nil
}

func buildAndPushImage(cfg *smithery.SmitheryConfig, name string, smitheryPath string, repoPath string, dockerfileDir string, imageName string, deps []string) error {
	dockerfilePath, err := docker.Inject(
		context.Background(),
		name,
		repoPath,
		dockerfileDir,
		dockerfile,
		cfg.ParsedCommand.Entrypoint(),
		deps,
	)
	if err != nil {
		return fmt.Errorf("inject command: %w", err)
	}

	tmpDockerfilePath, err := docker.BuildImage(context.Background(), imageName, smitheryPath, dockerfileDir, dockerfilePath)
	if err != nil {
		return fmt.Errorf("build image: %w", err)
	}

	if err := os.Remove(tmpDockerfilePath); err != nil {
		return fmt.Errorf("remove tmp dockerfile: %w", err)
	}

	if err := os.Remove(filepath.Join(repoPath, "super-gateway")); err != nil {
		return fmt.Errorf("remove super-gateway: %w", err)
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
	os.RemoveAll(catalog.CatalogDir)
	handleError("create catalog directory", os.MkdirAll(catalog.CatalogDir, 0755))
}

func manageDeps(repository *hub.Repository) []string {
	deps := []string{}
	switch repository.PackageManager {
	case hub.PackageManagerAPK:
		if !repository.HasNPM {
			return append([]string{"apk add --no-cache node npm git"}, deps...)
		}
		return append([]string{"apk add --no-cache git"}, deps...)
	case hub.PackageManagerAPT:
		if !repository.HasNPM {
			return append([]string{"apt-get update", "apt-get install -y nodejs npm git"}, deps...)
		}
		return append([]string{"apt-get update", "apt-get install -y git"}, deps...)
	default:
		log.Fatalf("Unsupported package manager: %s", repository.PackageManager)
		return []string{}
	}
}
