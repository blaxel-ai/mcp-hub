package smithery

import (
	"fmt"
	"reflect"
	"strings"
)

type SmitheryConfig struct {
	ParsedCommand *Command     `yaml:"parsedConfig,omitempty"`
	Build         *Build       `yaml:"build,omitempty"`
	StartCommand  StartCommand `yaml:"startCommand"`
}

type Build struct {
	Dockerfile      *string `yaml:"dockerfile,omitempty"`
	DockerBuildPath *string `yaml:"dockerBuildPath,omitempty"`
}

type Command struct {
	Type    string            `json:"type"`
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

func (c *Command) Entrypoint() string {
	switch c.Type {
	case "stdio":
		return "\"npx\",\"-y\",\"supergateway\",\"--stdio\""
	case "sse":
		return "\"npx\",\"-y\",\"supergateway\",\"--sse\""
	}
	return ""
}

type StartCommand struct {
	Type            string       `yaml:"type"`
	ConfigSchema    ConfigSchema `yaml:"configSchema"`
	CommandFunction string       `yaml:"commandFunction"`
}

type ConfigSchema struct {
	Type       string              `yaml:"type"`
	Required   []string            `yaml:"required"`
	Properties map[string]Property `yaml:"properties"`
}

type Property struct {
	Type        string `yaml:"type"`
	Default     string `yaml:"default"`
	Description string `yaml:"description"`
}

func (c *SmitheryConfig) ApplyOverrides(overrides []map[string]interface{}) error {
	for _, override := range overrides {
		for key, value := range override {
			parts := strings.Split(key, ".")
			current := reflect.ValueOf(c).Elem()

			// Navigate through all parts except the last one
			for i := 0; i < len(parts)-1; i++ {
				if !current.IsValid() {
					return fmt.Errorf("invalid path: %s", key)
				}

				// Get the field first
				field := current.FieldByName(Title(parts[i]))
				if !field.IsValid() {
					return fmt.Errorf("field not found: %s in path %s", parts[i], key)
				}

				// Handle pointer types
				if field.Kind() == reflect.Ptr {
					if field.IsNil() {
						// Initialize nil pointer
						field.Set(reflect.New(field.Type().Elem()))
					}
					field = field.Elem()
				}

				fmt.Printf("Current field: %s, Type: %s, Kind: %s\n", parts[i], field.Type().Name(), field.Kind())
				current = field
			}

			// Handle the last field
			if !current.IsValid() {
				return fmt.Errorf("invalid path: %s", key)
			}

			lastField := current.FieldByName(Title(parts[len(parts)-1]))
			if !lastField.IsValid() {
				return fmt.Errorf("field not found: %s in path %s", parts[len(parts)-1], key)
			}

			// Handle pointer for the last field
			if lastField.Kind() == reflect.Ptr {
				if lastField.IsNil() {
					lastField.Set(reflect.New(lastField.Type().Elem()))
				}
				lastField = lastField.Elem()
			}

			// Set the value based on the field's type
			switch lastField.Kind() {
			case reflect.String:
				lastField.SetString(fmt.Sprint(value))
			default:
				return fmt.Errorf("unsupported field type %v for field %s", lastField.Kind(), key)
			}
		}
	}
	return nil
}
