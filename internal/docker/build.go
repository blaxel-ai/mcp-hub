package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func BuildImage(ctx context.Context, imageName string, dockerfilePath string) error {
	directory := filepath.Dir(dockerfilePath)
	dockerfile := filepath.Base(dockerfilePath)
	fmt.Println("Building image", imageName, "with dockerfile", dockerfile, "in directory", directory)
	cmd := exec.Command("docker", "build", "-t", imageName, "-f", dockerfile, ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = directory
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
