package hub

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/beamlit/mcp-hub/internal/smithery"
	"gopkg.in/yaml.v2"
)

type Hub struct {
	Repositories map[string]*Repository `yaml:"repositories"`
}

type PackageManager string

const (
	PackageManagerAPK PackageManager = "apk"
	PackageManagerAPT PackageManager = "apt"
)

type Repository struct {
	Repository      string                   `yaml:"repository" mandatory:"false"`
	Path            string                   `yaml:"path" mandatory:"false"`
	BasePath        string                   `yaml:"basePath" mandatory:"false" default:""`
	SrcPath         string                   `yaml:"srcPath" mandatory:"false" default:"src"`
	DistPath        string                   `yaml:"distPath" mandatory:"false" default:"dist"`
	Entrypoint      string                   `yaml:"entrypoint" mandatory:"false"`
	Smithery        *smithery.SmitheryConfig `yaml:"smithery" mandatory:"false"`
	Language        string                   `yaml:"language" mandatory:"false" default:"typescript"`
	PackageManager  PackageManager           `yaml:"packageManager" mandatory:"false" default:"apk"`
	DoNotShow       []string                 `yaml:"doNotShow" mandatory:"false"`
	HasNPM          bool                     `yaml:"hasNPM" mandatory:"false" default:"true"`
	Branch          string                   `yaml:"branch" mandatory:"false" default:"main"`
	URL             string                   `yaml:"url" mandatory:"false"`
	DisplayName     string                   `yaml:"displayName" mandatory:"true"`
	Icon            string                   `yaml:"icon" mandatory:"true"`
	Disabled        bool                     `yaml:"disabled" mandatory:"false" default:"false"`
	Description     string                   `yaml:"description" mandatory:"true"`
	LongDescription string                   `yaml:"longDescription" mandatory:"true"`
	Enterprise      bool                     `yaml:"enterprise" mandatory:"false" default:"false"`
	ComingSoon      bool                     `yaml:"comingSoon" mandatory:"false" default:"false"`
	Secrets         []string                 `yaml:"secrets" mandatory:"false"`
	HiddenSecrets   []string                 `yaml:"hiddenSecrets" mandatory:"false"`
	OAuth           *OAuth                   `yaml:"oauth" mandatory:"false"`
	Integration     string                   `yaml:"integration" mandatory:"false"`
	Tags            []string                 `yaml:"tags"`
	Categories      []string                 `yaml:"categories"`
}

type OAuth struct {
	Type   string   `yaml:"type"`
	Scopes []string `yaml:"scopes"`
}

func (h *Hub) Read(path string) error {
	h.Repositories = make(map[string]*Repository)
	files, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		yamlFile, err := os.ReadFile(filepath.Join(path, file.Name()))
		if err != nil {
			return err
		}

		var repo Repository
		if err := yaml.Unmarshal(yamlFile, &repo); err != nil {
			return err
		}

		// Use filename without extension as repository name
		name := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
		h.Repositories[name] = &repo
	}
	return nil
}

// ValidateWithDefaultValues validates the hub and applies default values to empty fields
// This is useful to validate the hub before running the import command
func (h *Hub) ValidateWithDefaultValues() error {
	if h.Repositories == nil {
		return errors.New("repositories is required")
	}

	var errs []error

	for name, repository := range h.Repositories {
		// Use reflection to validate struct tags
		v := reflect.ValueOf(repository).Elem() // Get the element the pointer refers to
		t := v.Type()

		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			value := v.Field(i)

			// Check mandatory fields
			if mandatory, ok := field.Tag.Lookup("mandatory"); ok && mandatory == "true" {
				if value.IsZero() {
					errs = append(errs, fmt.Errorf("field %s is required in repository %s", field.Name, name))
				}
			}

			// Apply default values for empty fields
			if defaultVal, ok := field.Tag.Lookup("default"); ok && value.IsZero() {
				switch value.Kind() {
				case reflect.String:
					value.SetString(defaultVal)
				case reflect.Bool:
					value.SetBool(defaultVal == "true")
				}
			}
		}
	}

	return errors.Join(errs...)
}
