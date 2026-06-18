package hub

import (
	"strings"
	"testing"
)

func TestHTTPUpstreamValidateWithDefaultValues(t *testing.T) {
	t.Run("defaults allowed path from URL", func(t *testing.T) {
		upstream := &HTTPUpstream{URL: "http://127.0.0.1:8081/mcp"}
		if err := upstream.ValidateWithDefaultValues(); err != nil {
			t.Fatalf("ValidateWithDefaultValues returned error: %v", err)
		}
		if upstream.AllowedPath != "/mcp" {
			t.Fatalf("AllowedPath = %q, want /mcp", upstream.AllowedPath)
		}
	})

	t.Run("rejects non loopback host", func(t *testing.T) {
		for _, rawURL := range []string{
			"http://example.com/mcp",
			"http://169.254.169.254/mcp",
			"http://127.0.0.1.evil.test/mcp",
			"http://localhost.evil.test/mcp",
		} {
			upstream := &HTTPUpstream{URL: rawURL}
			err := upstream.ValidateWithDefaultValues()
			if err == nil || !strings.Contains(err.Error(), "loopback") {
				t.Fatalf("%s: expected loopback error, got %v", rawURL, err)
			}
		}
	})

	t.Run("rejects userinfo", func(t *testing.T) {
		upstream := &HTTPUpstream{URL: "http://token@127.0.0.1:8081/mcp"}
		err := upstream.ValidateWithDefaultValues()
		if err == nil || !strings.Contains(err.Error(), "userinfo") {
			t.Fatalf("expected userinfo error, got %v", err)
		}
	})

	t.Run("rejects query and fragment", func(t *testing.T) {
		for _, rawURL := range []string{
			"http://127.0.0.1:8081/mcp?token=secret",
			"http://127.0.0.1:8081/mcp#secret",
		} {
			upstream := &HTTPUpstream{URL: rawURL}
			err := upstream.ValidateWithDefaultValues()
			if err == nil || !strings.Contains(err.Error(), "query or fragment") {
				t.Fatalf("%s: expected query or fragment error, got %v", rawURL, err)
			}
		}
	})

	t.Run("rejects root allowed path", func(t *testing.T) {
		upstream := &HTTPUpstream{URL: "http://127.0.0.1:8081/mcp", AllowedPath: "/"}
		err := upstream.ValidateWithDefaultValues()
		if err == nil || !strings.Contains(err.Error(), "must not be /") {
			t.Fatalf("expected root path error, got %v", err)
		}
	})

	t.Run("rejects allowed path query and fragment", func(t *testing.T) {
		for _, allowedPath := range []string{"/mcp?token=secret", "/mcp#secret"} {
			upstream := &HTTPUpstream{URL: "http://127.0.0.1:8081/mcp", AllowedPath: allowedPath}
			err := upstream.ValidateWithDefaultValues()
			if err == nil || !strings.Contains(err.Error(), "query or fragment") {
				t.Fatalf("%s: expected query or fragment error, got %v", allowedPath, err)
			}
		}
	})
}

func TestHTTPUpstreamSuperGatewayArgs(t *testing.T) {
	upstream := &HTTPUpstream{URL: "http://localhost:8081/mcp"}
	args, err := upstream.SuperGatewayArgs()
	if err != nil {
		t.Fatalf("SuperGatewayArgs returned error: %v", err)
	}
	want := []string{"--port", "80", "--transport", "http-stream", "--http-upstream", "http://localhost:8081/mcp", "--http-upstream-path", "/mcp", "--stdio"}
	if len(args) != len(want) {
		t.Fatalf("len(args) = %d, want %d (%v)", len(args), len(want), args)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("args[%d] = %q, want %q (all args: %v)", i, args[i], want[i], args)
		}
	}
}

func TestHubValidateHTTPUpstreamDefaultsTransport(t *testing.T) {
	h := &Hub{Repositories: map[string]*Repository{
		"dummy": {
			DisplayName:     "Dummy",
			Icon:            "icon",
			Description:     "description",
			LongDescription: "long description",
			HTTPUpstream:    &HTTPUpstream{URL: "http://127.0.0.1:8081/mcp"},
		},
	}}
	if err := h.ValidateWithDefaultValues(); err != nil {
		t.Fatalf("ValidateWithDefaultValues returned error: %v", err)
	}
	if got := h.Repositories["dummy"].Transport; got != "http-stream" {
		t.Fatalf("Transport = %q, want http-stream", got)
	}
}
