package builder

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/blaxel-ai/mcp-hub/internal/catalog"
	"github.com/blaxel-ai/mcp-hub/internal/hub"
)

func (b *Build) Start(name string, repository *hub.Repository, c *catalog.Catalog) error {

	artifact := c.Artifacts[0]
	envKeys := []string{}
	for key := range artifact.Entrypoint.Env {
		envKeys = append(envKeys, key)
		err := b.checkEnvironmentVariable(name, artifact, key, artifact.Entrypoint.Env[key])
		if err != nil {
			log.Printf(err.Error())
			os.Exit(1)
		}
	}
	log.Printf("Starting MCP %s", name)
	err := b.dockerRun(name, artifact, envKeys)
	if err != nil {
		log.Printf("Failed to run docker command: %v", err)
		os.Exit(1)
	}

	return nil
}

func (b *Build) dockerRun(mcp string, artifact catalog.Artifact, envKeys []string) error {
	name := fmt.Sprintf("mcp-hub-%s", mcp)
	exec.Command("docker", "rm", "-f", name).Run()
	dockerRunCmd := []string{"run", "--rm", "-i", "-p", "8080:8080", "--platform", "linux/amd64", "--name", name}
	for _, key := range envKeys {
		dockerRunCmd = append(dockerRunCmd, "-e", fmt.Sprintf("%s=%s", key, os.Getenv(key)))
	}
	for _, value := range os.Environ() {
		if strings.Contains(value, "BL_") {
			dockerRunCmd = append(dockerRunCmd, "-e", value)
		}
	}
	dockerRunCmd = append(dockerRunCmd, artifact.Image)

	dockerCmd := artifact.Entrypoint.Command
	for _, arg := range artifact.Entrypoint.Args {
		if strings.HasPrefix(arg, "$") {
			// Convert camelCase to ENV_VAR format
			envVar := strings.TrimPrefix(arg, "$")
			envVarName := camelToEnvVar(envVar)
			dockerCmd += " " + os.Getenv(envVarName)
		} else {
			dockerCmd += " " + arg
		}
	}
	dockerRunCmd = append(dockerRunCmd, dockerCmd)

	cmd := exec.Command("docker", dockerRunCmd...)
	fmt.Println("dockerRunCmd", strings.Join(dockerRunCmd, " "))
	// Connect command's stdout and stderr to our process stdout and stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command and wait for it to finish
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("Failed to run docker command \"docker %s\": %v", strings.Join(dockerRunCmd, " "), err)
	}
	return nil
}

func (b *Build) checkEnvironmentVariable(mcp string, artifact catalog.Artifact, key string, val string) error {
	trimedVal := strings.Trim(val, "$")
	required := false

	if _, ok := artifact.Form.Config[trimedVal]; ok {
		required = artifact.Form.Config[trimedVal].Required
	}

	if _, ok := artifact.Form.Secrets[trimedVal]; ok {
		required = artifact.Form.Secrets[trimedVal].Required
	}

	if required && os.Getenv(key) == "" {
		return fmt.Errorf("Environment variable %s is not set and is required for the MCP %s", key, mcp)
	}
	return nil
}

func camelToEnvVar(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && 'A' <= r && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToUpper(result.String())
}
