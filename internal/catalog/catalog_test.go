package catalog

import (
	"testing"

	"github.com/blaxel-ai/mcp-hub/internal/hub"
	"github.com/blaxel-ai/mcp-hub/internal/smithery"
)

func TestCatalogLoadHTTPUpstream(t *testing.T) {
	repo := &hub.Repository{
		DisplayName:     "Dummy MCP",
		Icon:            "icon",
		Description:     "description",
		LongDescription: "long description",
		HTTPUpstream:    &hub.HTTPUpstream{URL: "http://127.0.0.1:8081/mcp"},
	}
	cfg := &smithery.SmitheryConfig{
		ParsedCommand: &smithery.Command{
			Command: "go",
			Args:    []string{"run", "./cmd/dummy_mcp"},
			Env:     map[string]string{},
		},
		StartCommand: smithery.StartCommand{
			ConfigSchema: smithery.ConfigSchema{Properties: map[string]smithery.Property{}},
		},
	}

	catalog := &Catalog{}
	if err := catalog.Load("dummy", repo, "ghcr.io/blaxel-ai/hub/dummy:latest", cfg); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(catalog.Artifacts) != 1 {
		t.Fatalf("len(Artifacts) = %d, want 1", len(catalog.Artifacts))
	}
	artifact := catalog.Artifacts[0]
	if artifact.Transport != "http-stream" {
		t.Fatalf("Transport = %q, want http-stream", artifact.Transport)
	}
	wantArgs := []string{"--port", "80", "--transport", "http-stream", "--http-upstream", "http://127.0.0.1:8081/mcp", "--http-upstream-path", "/mcp", "--stdio"}
	if len(artifact.Entrypoint.SuperGatewayArgs) != len(wantArgs) {
		t.Fatalf("len(SuperGatewayArgs) = %d, want %d (%v)", len(artifact.Entrypoint.SuperGatewayArgs), len(wantArgs), artifact.Entrypoint.SuperGatewayArgs)
	}
	for i := range wantArgs {
		if artifact.Entrypoint.SuperGatewayArgs[i] != wantArgs[i] {
			t.Fatalf("SuperGatewayArgs[%d] = %q, want %q (all args: %v)", i, artifact.Entrypoint.SuperGatewayArgs[i], wantArgs[i], artifact.Entrypoint.SuperGatewayArgs)
		}
	}
}
