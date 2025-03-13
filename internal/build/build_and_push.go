package build

import (
	"context"
	"fmt"
	"strings"

	"github.com/beamlit/mcp-hub/internal/docker"
	"github.com/beamlit/mcp-hub/internal/files"
	"github.com/beamlit/mcp-hub/internal/hub"
)

func (b *Build) BuildAndPushImage(name string, repository *hub.Repository) error {
	imageName := fmt.Sprintf("%s:%s", strings.ToLower(name), b.tag)
	switch repository.Language {
	case "typescript":
		err := b.prepareTypescript(name, repository)
		if err != nil {
			return fmt.Errorf("prepare typescript: %w", err)
		}
	default:
		return fmt.Errorf("unsupported language: %s", repository.Language)
	}

	err := docker.BuildImage(context.Background(), imageName, repository.Path)
	if err != nil {
		return fmt.Errorf("build image: %w", err)
	}

	return nil
}

func (b *Build) prepareTypescript(name string, repository *hub.Repository) error {
	err := files.CopyMergeDir("envs/typescript", repository.Path)
	if err != nil {
		return fmt.Errorf("copy dockerfile: %w", err)
	}
	return nil
}

// 	if push {
// 		if err := docker.PushImage(context.Background(), imageName); err != nil {
// 			return fmt.Errorf("push image: %w", err)
// 		}
// 	}

// 	return nil
// }

// 	buildTo := fmt.Sprintf("%s/%s", strings.ToLower(registry), imageName)
// 	if !skipBuild {
// 		if err := buildAndPushImage(cfg, name, repository.SmitheryPath, repoPath, strings.TrimSuffix(repository.Dockerfile, "/Dockerfile"), buildTo, deps); err != nil {
// 			return nil, fmt.Errorf("build and push image: %w", err)
// 		}
// 	}

// 	fmt.Println(buildTo)
