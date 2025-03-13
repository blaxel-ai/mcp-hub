package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

func BuildImage(ctx context.Context, imageName string, repoPath string) error {
	fmt.Println("Building image", imageName, "in directory", repoPath)
	cmd := exec.Command("docker", "build", "-t", imageName, ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = repoPath
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
