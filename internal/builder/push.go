package builder

import (
	"context"
	"fmt"

	"github.com/beamlit/mcp-hub/internal/docker"
	"github.com/beamlit/mcp-hub/internal/hub"
)

func (b *Build) Push(name string, repository *hub.Repository) error {
	imageName := b.getImageName(name)
	if err := docker.PushImage(context.Background(), imageName); err != nil {
		return fmt.Errorf("push image: %w", err)
	}
	return nil
}
