package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

func PushImage(ctx context.Context, registry string, imageName string) error {
	cmd := exec.Command("docker", "push", fmt.Sprintf("%s/%s", registry, imageName))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
