package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func Inject(ctx context.Context, name string, path string, smitheryDir string, dockerfileDir string, cmd string, deps []string) (string, error) {
	dockerFilePath := filepath.Join(path, smitheryDir, dockerfileDir)
	os.Remove(fmt.Sprintf("%s.tmp", dockerFilePath))
	if smitheryDir == "@mcp-hub" {
		// Use the current working directory to construct the full path to the source file
		sourcePath := filepath.Join("dockerfiles", fmt.Sprintf("%s.Dockerfile", strings.ToLower(name)))

		// Open source file
		sourceFile, err := os.Open(sourcePath)
		if err != nil {
			return "", fmt.Errorf("failed to open source file: %w", err)
		}
		defer sourceFile.Close()

		// Create destination file
		destPath := filepath.Join(path, "Dockerfile.tmp")
		destFile, err := os.Create(destPath)
		if err != nil {
			return "", fmt.Errorf("failed to create destination file: %w", err)
		}
		defer destFile.Close()

		// Copy the contents
		_, err = io.Copy(destFile, sourceFile)
		if err != nil {
			return "", fmt.Errorf("failed to copy file: %w", err)
		}
		return destPath, nil
	}

	dockerFile, err := os.Open(dockerFilePath)
	if err != nil {
		return "", err
	}
	defer dockerFile.Close()

	dockerFileBytes, err := io.ReadAll(dockerFile)
	if err != nil {
		return "", err
	}

	dockerFileString := string(dockerFileBytes)
	var lines []string

	// First pass: find the last CMD and ENTRYPOINT
	for _, line := range strings.Split(dockerFileString, "\n") {
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	lines[len(lines)-1] = ""
	for _, dep := range deps {
		lines = append(lines, fmt.Sprintf("RUN %s", dep))
	}
	lines = append(lines, fmt.Sprintf("ENTRYPOINT [%s]", cmd))
	destPath := fmt.Sprintf("%s.tmp", dockerFilePath)
	return destPath, os.WriteFile(destPath, []byte(strings.Join(lines, "\n")), 0644)
}
