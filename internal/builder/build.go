package builder

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/beamlit/mcp-hub/internal/docker"
	"github.com/beamlit/mcp-hub/internal/files"
	"github.com/beamlit/mcp-hub/internal/hub"
)

func GetImageName(name string, tag string) string {
	if os.Getenv("BL_ENV") == "prod" {
		return fmt.Sprintf("prod/%s:%s", strings.ToLower(name), tag)
	} else {
		return fmt.Sprintf("dev/%s:%s", strings.ToLower(name), tag)
	}
}

func (b *Build) Build(name string, repository *hub.Repository) error {
	switch repository.Language {
	case "typescript":
		err := b.prepareTypescript(name, repository)
		if err != nil {
			return fmt.Errorf("prepare typescript: %w", err)
		}
	case "python":
		err := b.preparePython(name, repository)
		if err != nil {
			return fmt.Errorf("prepare python: %w", err)
		}
	default:
		return fmt.Errorf("unsupported language: %s", repository.Language)
	}

	buildArgs := map[string]string{}
	if repository.BasePath != "" {
		buildArgs["BUILD_PATH"] = "/" + repository.BasePath
	}
	if repository.DistPath != "" {
		buildArgs["DIST_PATH"] = repository.DistPath
	}
	err := docker.BuildImage(context.Background(), b.registry, GetImageName(name, b.tag), repository.Path, buildArgs)
	if err != nil {
		return fmt.Errorf("build image: %w", err)
	}

	return nil
}

func (b *Build) preparePython(name string, repository *hub.Repository) error {
	srcPath := repository.Path
	if repository.SrcPath != "" {
		srcPath = filepath.Join(repository.Path, repository.SrcPath)
	}
	err := files.CopyFile("envs/python/Dockerfile", filepath.Join(repository.Path, "Dockerfile"))
	if err != nil {
		return fmt.Errorf("copy dockerfile: %w", err)
	}
	err = files.CopyFile("envs/python/transport.py", filepath.Join(srcPath, "transport.py"))
	if err != nil {
		return fmt.Errorf("copy transport.py: %w", err)
	}
	err = files.AddLineToStartOfFile(
		filepath.Join(srcPath, "__init__.py"),
		"from .transport import websocket_server",
	)
	if err != nil {
		return fmt.Errorf("add line to start of file: %w", err)
	}
	err = files.CreateFileIfNotExists(
		filepath.Join(srcPath, "__main__.py"),
		"from . import main\n\nif __name__ == '__main__':\n    main()",
	)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}

	// Replace ${ENTRYPOINT} with specific Python entrypoint in Dockerfile
	dockerfilePath := filepath.Join(repository.Path, "Dockerfile")
	dockerfileContent, err := os.ReadFile(dockerfilePath)
	if err != nil {
		return fmt.Errorf("read dockerfile: %w", err)
	}

	if repository.Entrypoint == "" {
		return fmt.Errorf("entrypoint is not set, required for python")
	}
	newContent := strings.ReplaceAll(
		string(dockerfileContent),
		"${ENTRYPOINT}",
		fmt.Sprintf("\"%s\"", strings.Join(strings.Split(repository.Entrypoint, " "), "\", \"")),
	)

	err = os.WriteFile(dockerfilePath, []byte(newContent), 0644)
	if err != nil {
		return fmt.Errorf("write dockerfile: %w", err)
	}
	return nil
}

func (b *Build) prepareTypescript(name string, repository *hub.Repository) error {
	basePath := repository.Path
	if repository.BasePath != "" {
		basePath = filepath.Join(repository.Path, repository.BasePath)
	}

	srcPath := repository.Path
	if repository.SrcPath != "" {
		srcPath = filepath.Join(repository.Path, repository.SrcPath)
	}

	packageJson, err := os.ReadFile(filepath.Join(basePath, "package.json"))
	if err != nil {
		return fmt.Errorf("read package.json: %w", err)
	}
	type PackageJson struct {
		Type string `json:"type"`
	}
	var pj PackageJson
	err = json.Unmarshal(packageJson, &pj)
	if err != nil {
		return fmt.Errorf("unmarshal package.json: %w", err)
	}

	err = files.CopyFile("envs/typescript/Dockerfile", filepath.Join(repository.Path, "Dockerfile"))
	if err != nil {
		return fmt.Errorf("copy dockerfile: %w", err)
	}
	if pj.Type == "module" {
		err = files.CopyMergeDir("envs/typescript/esm", srcPath)
	} else {
		return fmt.Errorf("unsupported package.json type: %s, only module is supported", pj.Type)
	}
	if err != nil {
		return fmt.Errorf("copy overrides: %w", err)
	}
	return nil
}
