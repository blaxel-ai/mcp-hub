package smithery

import (
	"fmt"
	"os"

	"github.com/dop251/goja"
	"gopkg.in/yaml.v2"
)

func Parse(path string, overrider []map[string]interface{}) (SmitheryConfig, error) {

	file, err := os.Open(path)
	if err != nil {
		return SmitheryConfig{}, err
	}

	defer file.Close()

	var smithery SmitheryConfig
	err = yaml.NewDecoder(file).Decode(&smithery)
	if err != nil {
		return SmitheryConfig{}, err
	}

	parsedCommand, err := ExecuteCommandFunction(smithery.StartCommand.CommandFunction, smithery.StartCommand.ConfigSchema.Properties)
	if err != nil {
		return SmitheryConfig{}, err
	}
	parsedCommand.Type = smithery.StartCommand.Type
	if err := smithery.ApplyOverrides(overrider); err != nil {
		return SmitheryConfig{}, fmt.Errorf("failed to apply overrides: %w", err)
	}

	smithery.ParsedCommand = parsedCommand

	return smithery, nil
}

func ExecuteCommandFunction(commandFn string, config map[string]Property) (*Command, error) {
	vm := goja.New()

	// Convert Property struct to a simpler map for JavaScript
	jsConfig := make(map[string]string)
	for key, prop := range config {
		// For now, we'll use the Default value if it exists, otherwise empty string
		jsConfig[key] = prop.Default
		if jsConfig[key] == "" {
			jsConfig[key] = "$" + key
		}
	}

	// Inject the simplified config into VM
	if err := vm.Set("config", jsConfig); err != nil {
		return nil, fmt.Errorf("failed to set config: %w", err)
	}

	// Execute the JS function
	v, err := vm.RunString(fmt.Sprintf(`(%s)(config)`, commandFn))
	if err != nil {
		return nil, fmt.Errorf("failed to execute command function: %w", err)
	}

	// Convert the JavaScript object to a Go map first
	jsObj := v.Export().(map[string]interface{})

	// Create the Command struct manually
	cmd := &Command{
		Command: jsObj["command"].(string),
		Args:    make([]string, 0),
		Env:     make(map[string]string),
	}

	// Convert args array
	if args, ok := jsObj["args"].([]interface{}); ok {
		for _, arg := range args {
			cmd.Args = append(cmd.Args, arg.(string))
		}
	}

	// Convert env map
	if env, ok := jsObj["env"].(map[string]interface{}); ok {
		for key, value := range env {
			cmd.Env[key] = fmt.Sprint(value) // Use fmt.Sprint to safely convert any type to string
		}
	}

	return cmd, nil
}
