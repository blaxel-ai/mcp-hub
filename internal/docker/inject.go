package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
)

func Inject(ctx context.Context, dockerFilePath string, cmd string, deps []string) error {
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

	lines[len(lines)-1] = fmt.Sprintf("ENTRYPOINT [%s]", cmd)
	return os.WriteFile(dockerFilePath, []byte(strings.Join(lines, "\n")), 0644)
}
