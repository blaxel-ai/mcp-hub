package catalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"

	"github.com/blaxel-ai/mcp-hub/internal/hub"
	"github.com/blaxel-ai/mcp-hub/internal/smithery"
)

const (
	CatalogDir = "catalog"
)

type Artifact struct {
	Name            string     `json:"name"`
	Image           string     `json:"image"`
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
	HiddenSecrets   []string   `json:"hiddenSecrets"`
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
	Default     string `json:"default,omitempty"`
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

func (c *Catalog) SaveArtifact(artifact Artifact) error {
	jsonData, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return err
	}

	apiURL := os.Getenv("BL_API_URL")
	username := os.Getenv("BL_ADMIN_USERNAME")
	password := os.Getenv("BL_ADMIN_PASSWORD")

	url := fmt.Sprintf("%s/admin/store/mcp/%s", apiURL, artifact.Name)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.SetBasicAuth(username, password)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed to save artifact: HTTP %d", resp.StatusCode)
	}

	return nil
}

func (c *Catalog) Save() error {
	for _, artifact := range c.Artifacts {
		err := c.SaveArtifact(artifact)
		if err != nil {
			fmt.Printf("error saving artifact %s: %s\n", artifact.Name, err)
			return err
		}
		fmt.Printf("saved artifact %s\n", artifact.Name)
	}
	return nil
}

func (c *Catalog) Load(name string, hub *hub.Repository, registry string, imageName string, smithery *smithery.SmitheryConfig) error {
	if hub.Disabled {
		c.AddArtifact(Artifact{
			Name:            name,
			Image:           fmt.Sprintf("%s/%s", registry, imageName),
			DisplayName:     hub.DisplayName,
			Description:     hub.Description,
			LongDescription: hub.LongDescription,
			Icon:            hub.Icon,
			Categories:      hub.Categories,
			URL:             hub.URL,
			Enterprise:      hub.Enterprise,
			ComingSoon:      hub.ComingSoon,
			Integration:     hub.Integration,
		})
		return nil
	}
	secrets := make(map[string]Field)
	config := make(map[string]Field)
	hidden := make(map[string]bool)
	for _, doNotShow := range hub.DoNotShow {
		hidden[doNotShow] = true
	}
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
			Hidden:      hidden[secret],
			Default:     p.Default,
		}
	}

	for name, property := range smithery.StartCommand.ConfigSchema.Properties {
		if _, ok := secrets[name]; ok {
			continue
		}
		if slices.Contains(hub.HiddenSecrets, name) {
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
			Default:     property.Default,
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
		Image:           fmt.Sprintf("%s/%s", registry, imageName),
		DisplayName:     hub.DisplayName,
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
		Enterprise:    hub.Enterprise,
		ComingSoon:    hub.ComingSoon,
		Integration:   hub.Integration,
		HiddenSecrets: hub.HiddenSecrets,
	}
	c.AddArtifact(artifact)
	return nil
}
