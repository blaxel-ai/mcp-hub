package catalog

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/beamlit/mcp-hub/internal/hub"
	"github.com/beamlit/mcp-hub/internal/smithery"
)

const (
	CatalogDir = "catalog"
)

type Artifact struct {
	DisplayName     string                `json:"displayName"`
	Repository      string                `json:"repository"`
	Description     string                `json:"description"`
	Icon            string                `json:"icon"`
	Tags            []string              `json:"tags"`
	Categories      []string              `json:"categories"`
	LongDescription string                `json:"longDescription"`
	Entrypoint      Entrypoint            `json:"entrypoint"`
	Schema          smithery.ConfigSchema `json:"schema"`
}

type Entrypoint struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

type Catalog struct {
	Artifacts []Artifact
}

func (c *Catalog) AddArtifact(artifact Artifact) {
	c.Artifacts = append(c.Artifacts, artifact)
}

func (c *Catalog) Save() error {
	os.MkdirAll(CatalogDir, 0755)
	for _, artifact := range c.Artifacts {
		json, err := json.MarshalIndent(artifact, "", "  ")
		if err != nil {
			return err
		}
		os.WriteFile(fmt.Sprintf("%s/%s.json", CatalogDir, artifact.DisplayName), json, 0644)
	}
	return nil
}

func (c *Catalog) Load(name string, hub *hub.Repository, smithery *smithery.SmitheryConfig) error {
	artifact := Artifact{
		DisplayName:     name,
		Repository:      hub.Repository,
		Description:     hub.Description,
		Icon:            hub.Icon,
		Tags:            hub.Tags,
		Categories:      hub.Categories,
		LongDescription: hub.LongDescription,
		Entrypoint: Entrypoint{
			Command: smithery.ParsedCommand.Command,
			Args:    smithery.ParsedCommand.Args,
			Env:     smithery.ParsedCommand.Env,
		},
		Schema: smithery.StartCommand.ConfigSchema,
	}
	c.AddArtifact(artifact)
	return nil
}
