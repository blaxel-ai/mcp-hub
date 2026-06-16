package smithery

import "testing"

func TestCommandEntrypointUsesCustomSuperGatewayArgs(t *testing.T) {
	cmd := &Command{}
	entrypoint := cmd.Entrypoint([]string{"--port", "80", "--transport", "http-stream", "--http-upstream", "http://127.0.0.1:8081/mcp", "--http-upstream-path", "/mcp", "--stdio"})
	want := `"./super-gateway","--port","80","--transport","http-stream","--http-upstream","http://127.0.0.1:8081/mcp","--http-upstream-path","/mcp","--stdio"`
	if entrypoint != want {
		t.Fatalf("entrypoint = %s, want %s", entrypoint, want)
	}
}

func TestCommandEntrypointDefaultPreservesStdioMode(t *testing.T) {
	cmd := &Command{}
	entrypoint := cmd.Entrypoint()
	want := `"./super-gateway","--transport","http-stream","--port","80","--stdio"`
	if entrypoint != want {
		t.Fatalf("entrypoint = %s, want %s", entrypoint, want)
	}
}
