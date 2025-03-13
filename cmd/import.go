package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	importCmd.Flags().StringVarP(&registry, "registry", "r", "ghcr.io/beamlit/hub", "The registry to push the images to")
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
	imageName := fmt.Sprintf("%s:%s", strings.ToLower(name), tag)
	if repository.Path != "" {
		repoPath = repository.Path
	} else {
		repoPath = fmt.Sprintf("%s/%s/%s", tmpDir, strings.TrimPrefix(repository.Repository, githubPrefix), repository.Branch)
		defer git.DeleteRepository(repoPath)
	}

	if repository.Disabled {
		c := catalog.Catalog{}
		handleError("load catalog", c.Load(name, repository, imageName, &smithery.SmitheryConfig{}))
		if !debug {
			handleError("save catalog", c.Save())
		}
		return &c, nil
	}

	if repository.Path == "" {
		if _, err := git.CloneRepository(repoPath, repository.Branch, repository.Repository); err != nil {
			return nil, fmt.Errorf("clone repository: %w", err)
		}

		if err := handleSourceFiles(repoPath); err != nil {
			return nil, fmt.Errorf("handle source files: %w", err)
		}

		// Modify package.json after cloning
		packageJsonPath := filepath.Join(repoPath, "package.json")
		if err := updatePackageJson(packageJsonPath); err != nil {
			return nil, fmt.Errorf("update package.json: %w", err)
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
	handleError("load catalog", c.Load(name, repository, buildTo, cfg))
	if !debug {
		handleError("save catalog", c.Save())
	}
	return &c, nil
}

func buildAndPushImage(cfg *smithery.SmitheryConfig, name string, smitheryPath string, repoPath string, dockerfileDir string, imageName string, deps []string) error {
	// Handle src directory and file copying
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
	deps := []string{
		"npm install -g pnpm",
		"pnpm install https://github.com/beamlit/supergateway",
		"pnpm install ws",
		"pnpm install -D @types/ws",
		"pnpm install uuid",
	}
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

func updatePackageJson(path string) error {
	// Read the existing package.json
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read package.json: %w", err)
	}

	var pkg map[string]interface{}
	if err := json.Unmarshal(content, &pkg); err != nil {
		return fmt.Errorf("parse package.json: %w", err)
	}

	// Update scripts
	scripts, ok := pkg["scripts"].(map[string]interface{})
	if !ok {
		scripts = make(map[string]interface{})
		pkg["scripts"] = scripts
	}
	scripts["build"] = json.RawMessage(`"tsc && node -e \"require('fs').chmodSync('build/loader.js', '755')\""`)

	// Write back to file
	updatedContent, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal package.json: %w", err)
	}

	if err := os.WriteFile(path, updatedContent, 0644); err != nil {
		return fmt.Errorf("write package.json: %w", err)
	}

	return nil
}

func handleSourceFiles(repoPath string) error {
	// Create or ensure src directory exists
	srcPath := filepath.Join(repoPath, "src")
	if err := os.MkdirAll(srcPath, 0755); err != nil {
		return fmt.Errorf("create src directory: %w", err)
	}

	// Check if index.js exists in root and needs to be moved
	rootIndexPath := filepath.Join(repoPath, "index.ts")
	if _, err := os.Stat(rootIndexPath); err == nil {
		// Move index.js to src directory
		if err := moveFile(rootIndexPath, filepath.Join(srcPath, "index.ts")); err != nil {
			return fmt.Errorf("move index.ts to src: %w", err)
		}
	}

	// Copy files from internal/copy_files to src
	copyFilesDir := "internal/copy_files"
	entries, err := os.ReadDir(copyFilesDir)
	if err != nil {
		return fmt.Errorf("read copy_files directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		srcFile := filepath.Join(copyFilesDir, entry.Name())
		dstFile := filepath.Join(srcPath, entry.Name())
		if err := copyFile(srcFile, dstFile); err != nil {
			return fmt.Errorf("copy %s: %w", entry.Name(), err)
		}
	}

	return nil
}

func moveFile(src, dst string) error {
	if err := copyFile(src, dst); err != nil {
		return err
	}
	return os.Remove(src)
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Copy file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}
