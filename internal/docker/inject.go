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
	lastCmdIndex := -1
	lastEntrypointIndex := -1

	// First pass: find the last CMD and ENTRYPOINT
	for i, line := range strings.Split(dockerFileString, "\n") {
		if strings.Contains(line, "ENTRYPOINT") {
			lastEntrypointIndex = i
		}
		if strings.Contains(line, "CMD") {
			lastCmdIndex = i
		}
		lines = append(lines, line)
	}

	// Replace only the last occurrence
	lastIndex := lastCmdIndex
	if lastEntrypointIndex > lastCmdIndex {
		lastIndex = lastEntrypointIndex
	}

	// Add RUN commands on  the last ENTRYPOINT/CMD
	if lastIndex != -1 && len(deps) > 0 {
		// Split the lines into before and after the last ENTRYPOINT/CMD
		before := lines[:lastIndex-1]
		after := lines[lastIndex-1:]

		// Add RUN commands
		for _, dep := range deps {
			before = append(before, fmt.Sprintf("RUN %s", dep))
		}

		// Combine everything back together
		lines = append(before, after...)
	}

	lines[len(lines)-1] = fmt.Sprintf("ENTRYPOINT [%s]", cmd)
	return os.WriteFile(dockerFilePath, []byte(strings.Join(lines, "\n")), 0644)
}
