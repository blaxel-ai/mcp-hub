package docker

import (
	"context"
	"os"
	"os/exec"
)

func BuildImage(ctx context.Context, imageName string, dockerfile string, directory string, context string) error {
	cmd := exec.Command("docker", "build", "-t", imageName, "-f", "Dockerfile", context)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = directory
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
