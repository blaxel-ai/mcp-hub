package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseHTTPUpstreamConfig(t *testing.T) {
	t.Run("defaults public path from upstream URL", func(t *testing.T) {
		config, err := ParseHTTPUpstreamConfig("http://127.0.0.1:8081/mcp", "")
		if err != nil {
			t.Fatalf("ParseHTTPUpstreamConfig returned error: %v", err)
		}
		if config.PublicPath != "/mcp" {
			t.Fatalf("PublicPath = %q, want /mcp", config.PublicPath)
		}
	})

	t.Run("rejects non loopback hosts", func(t *testing.T) {
		for _, rawURL := range []string{
			"http://example.com/mcp",
			"http://169.254.169.254/mcp",
			"http://127.0.0.1.evil.test/mcp",
			"http://localhost.evil.test/mcp",
		} {
			_, err := ParseHTTPUpstreamConfig(rawURL, "")
			if err == nil || !strings.Contains(err.Error(), "loopback") {
				t.Fatalf("%s: expected loopback error, got %v", rawURL, err)
			}
		}
	})

	t.Run("accepts IPv6 loopback", func(t *testing.T) {
		config, err := ParseHTTPUpstreamConfig("http://[::1]:8081/mcp", "")
		if err != nil {
			t.Fatalf("ParseHTTPUpstreamConfig returned error: %v", err)
		}
		if config.URL.Hostname() != "::1" {
			t.Fatalf("hostname = %q, want ::1", config.URL.Hostname())
		}
	})

	t.Run("rejects userinfo", func(t *testing.T) {
		_, err := ParseHTTPUpstreamConfig("http://token@127.0.0.1:8081/mcp", "")
		if err == nil || !strings.Contains(err.Error(), "userinfo") {
			t.Fatalf("expected userinfo error, got %v", err)
		}
	})

	t.Run("rejects root public path", func(t *testing.T) {
		_, err := ParseHTTPUpstreamConfig("http://127.0.0.1:8081/mcp", "/")
		if err == nil || !strings.Contains(err.Error(), "non-root") {
			t.Fatalf("expected non-root path error, got %v", err)
		}
	})

	t.Run("rejects https scheme", func(t *testing.T) {
		_, err := ParseHTTPUpstreamConfig("https://127.0.0.1:8081/mcp", "")
		if err == nil || !strings.Contains(err.Error(), "http scheme") {
			t.Fatalf("expected http scheme error, got %v", err)
		}
	})
}

func TestHTTPUpstreamProxy(t *testing.T) {
	upstreamCalled := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalled = true
		if r.URL.Path != "/upstream-mcp" {
			t.Fatalf("upstream path = %q, want /upstream-mcp", r.URL.Path)
		}
		if r.URL.RawQuery != "clientId=abc" {
			t.Fatalf("upstream query = %q, want clientId=abc", r.URL.RawQuery)
		}
		for _, header := range []string{"Authorization", "Cookie", "Forwarded", "Proxy-Authorization", "X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto", "X-Real-Ip", "X-Blaxel-Workspace", "Cf-Connecting-Ip"} {
			if got := r.Header.Get(header); got != "" {
				t.Fatalf("%s was forwarded as %q", header, got)
			}
		}
		if got := r.Header.Get("Mcp-Session-Id"); got != "session-1" {
			t.Fatalf("Mcp-Session-Id = %q, want session-1", got)
		}
		if got := r.Header.Get("MCP-Protocol-Version"); got != "2024-11-05" {
			t.Fatalf("MCP-Protocol-Version = %q, want 2024-11-05", got)
		}
		w.Header().Set("Mcp-Session-Id", "upstream-session")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Set-Cookie", "private=true")
		w.Header().Set("X-Blaxel-Region", "private")
		w.Header().Set("Cf-Ray", "private")
		w.Header().Set("X-Forwarded-Host", "private")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{}}`))
	}))
	defer upstream.Close()

	config, err := ParseHTTPUpstreamConfig(upstream.URL+"/upstream-mcp", "/mcp")
	if err != nil {
		t.Fatalf("ParseHTTPUpstreamConfig returned error: %v", err)
	}
	handler := NewGateway().HandleHTTPUpstream(config)

	req := httptest.NewRequest(http.MethodPost, "/mcp?clientId=abc", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("Cookie", "private=true")
	req.Header.Set("Forwarded", "for=203.0.113.1")
	req.Header.Set("Proxy-Authorization", "Basic private")
	req.Header.Set("X-Forwarded-For", "203.0.113.1")
	req.Header.Set("X-Forwarded-Host", "external.example")
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Real-Ip", "203.0.113.1")
	req.Header.Set("X-Blaxel-Workspace", "workspace")
	req.Header.Set("Cf-Connecting-Ip", "203.0.113.1")
	req.Header.Set("Mcp-Session-Id", "session-1")
	req.Header.Set("MCP-Protocol-Version", "2024-11-05")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	response := recorder.Result()
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)

	if !upstreamCalled {
		t.Fatal("upstream was not called")
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.StatusCode, string(body))
	}
	if got := response.Header.Get("Mcp-Session-Id"); got != "upstream-session" {
		t.Fatalf("response Mcp-Session-Id = %q, want upstream-session", got)
	}
	for _, header := range []string{"Set-Cookie", "X-Blaxel-Region", "Cf-Ray", "X-Forwarded-Host"} {
		if got := response.Header.Get(header); got != "" {
			t.Fatalf("response %s was forwarded as %q", header, got)
		}
	}
	if !strings.Contains(string(body), `"jsonrpc":"2.0"`) {
		t.Fatalf("unexpected body: %s", string(body))
	}
}

func TestHTTPUpstreamProxyRejectsOtherPathsAndMethods(t *testing.T) {
	upstreamCalled := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalled = true
	}))
	defer upstream.Close()

	config, err := ParseHTTPUpstreamConfig(upstream.URL+"/mcp", "/mcp")
	if err != nil {
		t.Fatalf("ParseHTTPUpstreamConfig returned error: %v", err)
	}
	handler := NewGateway().HandleHTTPUpstream(config)

	for _, path := range []string{"/not-mcp", "/mcp/extra", "/mcp2"} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, path, strings.NewReader("{}")))

		if recorder.Code != http.StatusNotFound {
			t.Fatalf("%s status = %d, want 404", path, recorder.Code)
		}
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/mcp", nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET /mcp status = %d, want 405", recorder.Code)
	}

	if upstreamCalled {
		t.Fatal("upstream should not be called for non-MCP paths or methods")
	}
}

func TestDialLoopbackOnlyRejectsNonLoopback(t *testing.T) {
	_, err := dialLoopbackOnly(context.Background(), "tcp", "169.254.169.254:80")
	if err == nil || !strings.Contains(err.Error(), "non-loopback") {
		t.Fatalf("expected non-loopback error, got %v", err)
	}
}
