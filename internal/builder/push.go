package builder

import (
	"context"
	"fmt"

	"github.com/beamlit/mcp-hub/internal/docker"
	"github.com/beamlit/mcp-hub/internal/hub"
)

func (b *Build) Push(name string, repository *hub.Repository) error {
	imageName := GetImageName(name, b.tag)
	if err := docker.PushImage(context.Background(), imageName); err != nil {
		return fmt.Errorf("push image: %w", err)
	}
	return nil
}
