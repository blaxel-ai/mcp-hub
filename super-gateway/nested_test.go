package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const selfPath = "/app/super-gateway"

func isSelfStub(path string) bool { return path == selfPath }

func TestUnwrapNestedGateway(t *testing.T) {
	cases := []struct {
		name string
		cmd  []string
		want []string
	}{
		{"plain stdio server", []string{"./signoz-mcp-server"}, nil},
		{"node server", []string{"node", "/app/build/index.js", "--tools=web_search_exa"}, nil},
		{"platform wrapper around this gateway", []string{platformWrapper, "/app", selfPath, "--port", "80", "--transport", "http-stream", "--stdio", "./signoz-mcp-server"}, []string{"./signoz-mcp-server"}},
		{"this gateway directly", []string{selfPath, "--port", "80", "--stdio", "node", "index.js"}, []string{"node", "index.js"}},
		{"nested flags are dropped", []string{selfPath, "--port", "80", "--authentication", "--http-upstream", "http://127.0.0.1:9/mcp", "--stdio", "server"}, []string{"server"}},
		{"this gateway without --stdio", []string{selfPath, "--help"}, nil},
		{"this gateway with nothing after --stdio", []string{selfPath, "--stdio"}, nil},
		{"another binary named super-gateway", []string{"/tmp/evil/super-gateway", "--port", "80", "--stdio", "x"}, nil},
		{"wrapper at another path", []string{"/tmp/metamorph-wrapper", "/app", selfPath, "--stdio", "x"}, nil},
		{"wrapper around another binary", []string{platformWrapper, "/app", "/tmp/evil/super-gateway", "--stdio", "x"}, nil},
		{"gateway named only in an argument value", []string{"node", "index.js", "--name", selfPath, "--stdio", "x"}, nil},
		{"truncated wrapper command", []string{platformWrapper, "/app"}, nil},
		{"empty", nil, nil},
	}
	for _, tc := range cases {
		got, ok := unwrapNestedGateway(tc.cmd, isSelfStub)
		if ok != (tc.want != nil) || !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: unwrapNestedGateway(%v) = %v, %v; want %v", tc.name, tc.cmd, got, ok, tc.want)
		}
	}
}

func TestResolveStdioCommandKeepsPlainCommand(t *testing.T) {
	in := []string{"node", "/app/build/index.js", "--tools=a"}
	if got := resolveStdioCommand(in); !reflect.DeepEqual(got, in) {
		t.Fatalf("resolveStdioCommand = %v, want %v", got, in)
	}
}

func TestIsGatewayBinary(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Skipf("os.Executable: %v", err)
	}
	if !isGatewayBinary(self) {
		t.Fatal("the running executable must be recognised")
	}

	// A copy of the binary is a different file, and so is not this gateway.
	data, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(t.TempDir(), "super-gateway")
	if err := os.WriteFile(other, data, 0o755); err != nil {
		t.Fatal(err)
	}
	if isGatewayBinary(other) {
		t.Fatal("a different file named super-gateway must not be recognised")
	}
	if isGatewayBinary(filepath.Join(t.TempDir(), "missing")) {
		t.Fatal("a missing path must not be recognised")
	}
}

func TestNormalizeCommandPartsSplitsSingleString(t *testing.T) {
	got := normalizeCommandParts([]string{"/bin/metamorph-wrapper /app /app/super-gateway --port 80 --transport http-stream --stdio ./signoz-mcp-server"})
	want := []string{"/bin/metamorph-wrapper", "/app", "/app/super-gateway", "--port", "80", "--transport", "http-stream", "--stdio", "./signoz-mcp-server"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeCommandParts = %v, want %v", got, want)
	}
	if server, ok := unwrapNestedGateway(got, isSelfStub); !ok || !reflect.DeepEqual(server, []string{"./signoz-mcp-server"}) {
		t.Fatalf("the platform's single-string command must unwrap to its server, got %v, %v", server, ok)
	}
}

func TestNormalizeCommandPartsKeepsSeparateArgs(t *testing.T) {
	in := []string{"node", "/app/build/index.js", "--tools=a,b"}
	if got := normalizeCommandParts(in); !reflect.DeepEqual(got, in) {
		t.Fatalf("normalizeCommandParts = %v, want %v", got, in)
	}
}

func TestGatewayFlagArgsStopAtStdio(t *testing.T) {
	args := []string{"--transport", "http-stream", "--port", "80", "--stdio", platformWrapper, "/app", selfPath, "--port", "9", "--authentication", "--http-upstream", "http://127.0.0.1:9/mcp", "--stdio", "server"}
	want := []string{"--transport", "http-stream", "--port", "80"}
	if got := gatewayFlagArgs(args); !reflect.DeepEqual(got, want) {
		t.Fatalf("gatewayFlagArgs = %v, want %v", got, want)
	}
	noStdio := []string{"--port", "80"}
	if got := gatewayFlagArgs(noStdio); !reflect.DeepEqual(got, noStdio) {
		t.Fatalf("gatewayFlagArgs without --stdio = %v, want %v", got, noStdio)
	}
}
