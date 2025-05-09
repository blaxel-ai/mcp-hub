package builder

import (
	"context"
	"fmt"

	"github.com/blaxel-ai/mcp-hub/internal/docker"
	"github.com/blaxel-ai/mcp-hub/internal/hub"
)

func (b *Build) Push(name string, repository *hub.Repository) error {
	imageName := GetImageName(name, b.tag)
	if err := docker.PushImage(context.Background(), b.registry, imageName); err != nil {
		return fmt.Errorf("push image: %w", err)
	}
	return nil
}
