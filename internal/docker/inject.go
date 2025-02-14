package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
)

func Inject(ctx context.Context, dockerFilePath string, cmd string, deps []string) error {
	os.Remove(fmt.Sprintf("%s.tmp", dockerFilePath))
	dockerFile, err := os.Open(dockerFilePath)
	if err != nil {
		return err
	}
	defer dockerFile.Close()

	dockerFileBytes, err := io.ReadAll(dockerFile)
	if err != nil {
		return err
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
	return os.WriteFile(fmt.Sprintf("%s.tmp", dockerFilePath), []byte(strings.Join(lines, "\n")), 0644)
}
