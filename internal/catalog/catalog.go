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
	Name            string     `json:"name"`
	Enterprise      bool       `json:"enterprise"`
	ComingSoon      bool       `json:"coming_soon"`
	DisplayName     string     `json:"displayName"`
	Categories      []string   `json:"categories"`
	Integration     string     `json:"integration"`
	Description     string     `json:"description"`
	LongDescription string     `json:"longDescription"`
	Icon            string     `json:"icon"`
	URL             string     `json:"url"`
	Form            Form       `json:"form"`
	Entrypoint      Entrypoint `json:"entrypoint"`
}

type Form struct {
	Config  map[string]Field `json:"config"`
	Secrets map[string]Field `json:"secrets"`
	OAuth   *OAuth           `json:"oauth,omitempty"`
}

type Field struct {
	Description string `json:"description"`
	Label       string `json:"label"`
	Required    bool   `json:"required"`
	Hidden      bool   `json:"hidden,omitempty"`
}

type OAuth struct {
	Type  string   `json:"type"`
	Scope []string `json:"scope"`
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
	secrets := make(map[string]Field)
	config := make(map[string]Field)
	for _, secret := range hub.Secrets {

		p, ok := smithery.StartCommand.ConfigSchema.Properties[secret]
		if !ok {
			return fmt.Errorf("secret %s not found in smithery config", secret)
		}
		isRequired := false
		for _, required := range smithery.StartCommand.ConfigSchema.Required {
			if required == secret {
				isRequired = true
			}
		}
		secrets[secret] = Field{
			Description: p.Description,
			Label:       ToLabel(secret),
			Required:    isRequired,
			Hidden:      p.Default != "",
		}
	}

	for name, property := range smithery.StartCommand.ConfigSchema.Properties {
		if _, ok := secrets[name]; ok {
			continue
		}
		isRequired := false
		for _, required := range smithery.StartCommand.ConfigSchema.Required {
			if required == name {
				isRequired = true
			}
		}
		config[name] = Field{
			Description: property.Description,
			Label:       ToLabel(name),
			Required:    isRequired,
		}
	}

	var oauth *OAuth
	if hub.OAuth != nil {
		oauth = &OAuth{
			Type:  hub.OAuth.Type,
			Scope: hub.OAuth.Scopes,
		}
	}

	if hub.Integration == "" {
		hub.Integration = name
	}

	artifact := Artifact{
		Name:            name,
		DisplayName:     name,
		Description:     hub.Description,
		LongDescription: hub.LongDescription,
		Icon:            hub.Icon,
		Categories:      hub.Categories,
		URL:             hub.URL,
		Form: Form{
			Config:  config,
			Secrets: secrets,
			OAuth:   oauth,
		},
		Entrypoint: Entrypoint{
			Command: smithery.ParsedCommand.Command,
			Args:    smithery.ParsedCommand.Args,
			Env:     smithery.ParsedCommand.Env,
		},
		Enterprise:  hub.Enterprise,
		ComingSoon:  hub.ComingSoon,
		Integration: hub.Integration,
	}
	c.AddArtifact(artifact)
	return nil
}
