package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

func BuildImage(ctx context.Context, registry string, imageName string, repoPath string, buildArgs map[string]string) error {
	fmt.Println("Building image", imageName, "in directory", repoPath)
	cmd := exec.Command(
		"docker", "build",
		"--platform", "linux/amd64",
		"-t", fmt.Sprintf("%s/%s", registry, imageName),
		".",
	)
	for k, v := range buildArgs {
		cmd.Args = append(cmd.Args, "--build-arg", fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = repoPath
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
