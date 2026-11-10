package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const selfPath = "/app/super-gateway"

func isSelfStub(path string) bool { return path == selfPath }

func TestRunsNestedGateway(t *testing.T) {
	cases := []struct {
		name string
		cmd  []string
		want bool
	}{
		{"plain stdio server", []string{"./signoz-mcp-server"}, false},
		{"node server", []string{"node", "/app/build/index.js", "--tools=web_search_exa"}, false},
		{"platform wrapper around this gateway", []string{platformWrapper, "/app", selfPath, "--port", "80", "--transport", "http-stream", "--stdio", "./signoz-mcp-server"}, true},
		{"this gateway directly", []string{selfPath, "--port", "80", "--stdio", "node", "index.js"}, true},
		{"this gateway without --stdio", []string{selfPath, "--help"}, false},
		{"another binary named super-gateway", []string{"/tmp/evil/super-gateway", "--port", "80", "--stdio", "x"}, false},
		{"wrapper at another path", []string{"/tmp/metamorph-wrapper", "/app", selfPath, "--stdio", "x"}, false},
		{"wrapper around another binary", []string{platformWrapper, "/app", "/tmp/evil/super-gateway", "--stdio", "x"}, false},
		{"gateway named only in an argument value", []string{"node", "index.js", "--name", selfPath, "--stdio"}, false},
		{"truncated wrapper command", []string{platformWrapper, "/app"}, false},
		{"empty", nil, false},
	}
	for _, tc := range cases {
		if got := runsNestedGateway(tc.cmd, isSelfStub); got != tc.want {
			t.Errorf("%s: runsNestedGateway(%v) = %v, want %v", tc.name, tc.cmd, got, tc.want)
		}
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
	if !runsNestedGateway(got, isSelfStub) {
		t.Fatal("the platform's single-string command must be detected as a nested gateway")
	}
}

func TestNormalizeCommandPartsKeepsSeparateArgs(t *testing.T) {
	in := []string{"node", "/app/build/index.js", "--tools=a,b"}
	if got := normalizeCommandParts(in); !reflect.DeepEqual(got, in) {
		t.Fatalf("normalizeCommandParts = %v, want %v", got, in)
	}
}
