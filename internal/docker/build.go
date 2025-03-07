package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func BuildImage(ctx context.Context, imageName string, smitheryPath string, dockerfileDir string, dockerfilePath string) (string, error) {
	directory := filepath.Dir(dockerfilePath)
	dockerfile := filepath.Base(dockerfilePath)

	if smitheryPath != "" && strings.Contains(smitheryPath, "/") {
		directory = strings.Replace(directory, filepath.Dir(smitheryPath), "", 1)
	}
	if dockerfileDir != "" && strings.Contains(dockerfileDir, "/") && dockerfileDir != "/" {
		dockerfile = fmt.Sprintf("%s/%s", dockerfileDir, dockerfile)
	}

	fmt.Println("Building image", imageName, "with smitheryPath", smitheryPath, "with dockerfile", dockerfile, "in directory", directory)
	cmd := exec.Command("docker", "build", "-t", imageName, "-f", dockerfile, ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = directory
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return directory + "/" + dockerfile, nil
}
