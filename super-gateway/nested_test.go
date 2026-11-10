package main

import (
	"reflect"
	"testing"
)

func TestRunsNestedGateway(t *testing.T) {
	cases := []struct {
		name string
		cmd  []string
		want bool
	}{
		{"plain stdio server", []string{"./signoz-mcp-server"}, false},
		{"node server", []string{"node", "/app/build/index.js", "--tools=web_search_exa"}, false},
		{"platform wrapper around gateway", []string{"/bin/metamorph-wrapper", "/app", "/app/super-gateway", "--port", "80", "--transport", "http-stream", "--stdio", "./signoz-mcp-server"}, true},
		{"gateway directly", []string{"./super-gateway", "--port", "80", "--stdio", "node", "index.js"}, true},
		{"gateway binary name without --stdio", []string{"/app/super-gateway", "--help"}, false},
		{"gateway only named in a later argument value", []string{"node", "index.js", "--name", "super-gateway"}, false},
	}
	for _, tc := range cases {
		if got := runsNestedGateway(tc.cmd); got != tc.want {
			t.Errorf("%s: runsNestedGateway(%v) = %v, want %v", tc.name, tc.cmd, got, tc.want)
		}
	}
}

func TestNormalizeCommandPartsSplitsSingleString(t *testing.T) {
	got := normalizeCommandParts([]string{"/bin/metamorph-wrapper /app /app/super-gateway --port 80 --transport http-stream --stdio ./signoz-mcp-server"})
	want := []string{"/bin/metamorph-wrapper", "/app", "/app/super-gateway", "--port", "80", "--transport", "http-stream", "--stdio", "./signoz-mcp-server"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeCommandParts = %v, want %v", got, want)
	}
	if !runsNestedGateway(got) {
		t.Fatal("the platform's single-string command must be detected as a nested gateway")
	}
}

func TestNormalizeCommandPartsKeepsSeparateArgs(t *testing.T) {
	in := []string{"node", "/app/build/index.js", "--tools=a,b"}
	if got := normalizeCommandParts(in); !reflect.DeepEqual(got, in) {
		t.Fatalf("normalizeCommandParts = %v, want %v", got, in)
	}
}
