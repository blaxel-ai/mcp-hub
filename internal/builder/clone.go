package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/beamlit/mcp-hub/internal/catalog"
	"github.com/beamlit/mcp-hub/internal/errors"
	"github.com/beamlit/mcp-hub/internal/git"
	"github.com/beamlit/mcp-hub/internal/hub"
	"github.com/beamlit/mcp-hub/internal/smithery"
)

func (b *Build) CloneRepository(name string, repository *hub.Repository) (*catalog.Catalog, error) {
	var repoPath string
	imageName := GetImageName(name, b.tag)
	if repository.Path != "" {
		repoPath = repository.Path
	} else {
		repoPath = fmt.Sprintf("%s/%s/%s", tmpDir, strings.TrimPrefix(repository.Repository, githubPrefix), repository.Branch)
	}

	if repository.Disabled {
		fmt.Printf("Skipping disabled repository %s\n", name)
		c := catalog.Catalog{}
		errors.HandleError("load catalog", c.Load(name, repository, b.registry, imageName, &smithery.SmitheryConfig{}))
		if !b.debug {
			errors.HandleError("save catalog", c.Save())
		}
		return &c, nil
	}

	if repository.Path == "" {
		if _, err := os.Stat(repoPath); err == nil {
			if err := os.RemoveAll(repoPath); err != nil {
				return nil, fmt.Errorf("remove existing directory: %w", err)
			}
		}
		if _, err := git.CloneRepository(repoPath, repository.Branch, repository.Repository); err != nil {
			return nil, fmt.Errorf("clone repository: %w", err)
		}
		repository.Path = repoPath
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
		tmpCfg, err := smithery.Parse(filepath.Join(repoPath, repository.BasePath+"/smithery.yaml"))
		if err != nil {
			return nil, fmt.Errorf("parse smithery file: %w", err)
		}
		cfg = &tmpCfg
	}

	c := catalog.Catalog{}
	errors.HandleError("load catalog", c.Load(name, repository, b.registry, imageName, cfg))
	return &c, nil
}
