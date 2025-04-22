package builder

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/beamlit/mcp-hub/internal/hub"
)

func (b *Build) BuildAndPushUKC(name string, repository *hub.Repository) error {
	imageName := GetImageName(name, b.tag)

	buildArgs := map[string]string{}
	if repository.BasePath != "" {
		buildArgs["BUILD_PATH"] = "/" + repository.BasePath
	}
	if repository.DistPath != "" {
		buildArgs["DIST_PATH"] = repository.DistPath
	}

	fmt.Println("Building image", imageName, "in directory", repository.Path)
	destination := fmt.Sprintf("index.unikraft.io/blaxel/%s", imageName)
	cmd := exec.Command(
		"kraft", "pkg",
		"--arch", "x86_64",
		"--plat", "kraftcloud",
		"--name", destination,
		"--rootfs", "Dockerfile",
		"--runtime", "index.unikraft.io/official/base-compat:latest",
		"--push",
		".",
	)
	fmt.Println("cmd", cmd.String())
	for k, v := range buildArgs {
		cmd.Args = append(cmd.Args, "--build-arg", fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = repository.Path
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
