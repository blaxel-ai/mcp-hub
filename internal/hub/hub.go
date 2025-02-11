package hub

import (
	"errors"
	"fmt"
	"os"
	"reflect"

	"gopkg.in/yaml.v2"
)

type Hub struct {
	Repositories map[string]*Repository `yaml:"repositories"`
}

type PackageManager string

const (
	PackageManagerNPM PackageManager = "npm"
	PackageManagerAPK PackageManager = "apk"
	PackageManagerAPT PackageManager = "apt"
)

type Repository struct {
	Repository      string                   `yaml:"repository" mendatory:"true"`
	SmitheryPath    string                   `yaml:"smitheryPath" mendatory:"false" default:"smithery.yaml"`
	Dockerfile      string                   `yaml:"dockerfile" mendatory:"false" default:"Dockerfile"`
	PackageManager  PackageManager           `yaml:"packageManager" mendatory:"false" default:"npm"`
	Branch          string                   `yaml:"branch" mendatory:"false" default:"main"`
	DisplayName     string                   `yaml:"displayName" mendatory:"true"`
	Icon            string                   `yaml:"icon" mendatory:"true"`
	Description     string                   `yaml:"description" mendatory:"true"`
	LongDescription string                   `yaml:"longDescription" mendatory:"true"`
	Overrider       []map[string]interface{} `yaml:"overrider"`
	Tags            []string                 `yaml:"tags"`
	Categories      []string                 `yaml:"categories"`
}

func (h *Hub) Read(path string) error {
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(yamlFile, h)
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
			if mandatory, ok := field.Tag.Lookup("mendatory"); ok && mandatory == "true" {
				if value.IsZero() {
					errs = append(errs, fmt.Errorf("field %s is required in repository %s", field.Name, name))
				}
			}

			// Apply default values for empty fields
			if defaultVal, ok := field.Tag.Lookup("default"); ok && value.IsZero() {
				switch value.Kind() {
				case reflect.String:
					value.SetString(defaultVal)
				}
			}
		}
	}

	return errors.Join(errs...)
}
