package main

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"
)

// freeConsecutivePorts finds port and port+1 that are both free on all interfaces.
func freeConsecutivePorts(t *testing.T) int {
	t.Helper()
	for attempt := 0; attempt < 50; attempt++ {
		probe, err := net.Listen("tcp", ":0")
		if err != nil {
			t.Fatalf("listen: %v", err)
		}
		port := probe.Addr().(*net.TCPAddr).Port
		_ = probe.Close()
		listeners, err := claimGatewayPorts(port, true)
		if err != nil {
			continue
		}
		_ = listeners.main.Close()
		if listeners.health == nil {
			continue
		}
		_ = listeners.health.Close()
		return port
	}
	t.Fatal("no free consecutive ports")
	return 0
}

// startGateway claims port and port+1 and serves the health mux, like main does.
func startGateway(t *testing.T, port int) *gatewayListeners {
	t.Helper()
	listeners, err := claimGatewayPorts(port, true)
	if err != nil || listeners.health == nil {
		t.Fatalf("claim gateway ports: %v", err)
	}
	go func() { _ = http.Serve(listeners.health, newHealthMux(NewGateway(), port)) }()
	t.Cleanup(func() {
		_ = listeners.main.Close()
		_ = listeners.health.Close()
	})
	return listeners
}

func TestClaimGatewayPortsReportsAddrInUse(t *testing.T) {
	port := freeConsecutivePorts(t)
	startGateway(t, port)

	_, err := claimGatewayPorts(port, true)
	if err == nil || !isAddrInUse(err) {
		t.Fatalf("second claim error = %v, want address in use", err)
	}
}

func TestClaimGatewayPortsToleratesBusyHealthPort(t *testing.T) {
	port := freeConsecutivePorts(t)
	busy, err := net.Listen("tcp", ":"+strconv.Itoa(port+1))
	if err != nil {
		t.Fatalf("occupy health port: %v", err)
	}
	defer busy.Close()

	listeners, err := claimGatewayPorts(port, true)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	defer listeners.main.Close()
	if listeners.health != nil {
		t.Fatal("health listener should be nil when its port is busy")
	}
}

func TestClaimGatewayPortsSkipsHealthWhenAsked(t *testing.T) {
	port := freeConsecutivePorts(t)
	listeners, err := claimGatewayPorts(port, false)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	defer listeners.main.Close()
	if listeners.health != nil {
		_ = listeners.health.Close()
		t.Fatal("health port must stay free for the upstream child")
	}
}

func TestHealthPortFree(t *testing.T) {
	upstream := func(raw string) *HTTPUpstreamConfig {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		return &HTTPUpstreamConfig{URL: u}
	}
	cases := []struct {
		name     string
		upstream *HTTPUpstreamConfig
		want     bool
	}{
		{"stdio child", nil, true},
		{"upstream on another port", upstream("http://127.0.0.1:8081/mcp"), true},
		{"upstream on port+1", upstream("http://127.0.0.1:81/mcp"), false},
		{"upstream on default http port", upstream("http://127.0.0.1/mcp"), true},
	}
	for _, tc := range cases {
		if got := healthPortFree(80, tc.upstream); got != tc.want {
			t.Errorf("%s: healthPortFree = %v, want %v", tc.name, got, tc.want)
		}
	}
	if healthPortFree(79, upstream("http://127.0.0.1/mcp")) {
		t.Error("default http port 80 is port+1 for gateway port 79")
	}
}

func TestRunningGatewayOwnsPort(t *testing.T) {
	client := newLoopbackHTTPClient(time.Second)
	ctx := context.Background()

	t.Run("gateway serving the port", func(t *testing.T) {
		port := freeConsecutivePorts(t)
		startGateway(t, port)
		if !runningGatewayOwnsPort(ctx, client, port) {
			t.Fatal("expected the running gateway to be detected")
		}
	})

	t.Run("nothing listening", func(t *testing.T) {
		port := freeConsecutivePorts(t)
		if runningGatewayOwnsPort(ctx, client, port) {
			t.Fatal("expected no gateway")
		}
	})

	t.Run("unrelated service on port+1", func(t *testing.T) {
		port := freeConsecutivePorts(t)
		main, err := net.Listen("tcp", ":"+strconv.Itoa(port))
		if err != nil {
			t.Fatal(err)
		}
		defer main.Close()
		other, err := net.Listen("tcp", ":"+strconv.Itoa(port+1))
		if err != nil {
			t.Fatal(err)
		}
		defer other.Close()
		go func() {
			_ = http.Serve(other, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK) // answers 200 on every path, including /health
			}))
		}()
		if runningGatewayOwnsPort(ctx, client, port) {
			t.Fatal("an unrelated 200 responder must not count as a gateway")
		}
	})

	t.Run("gateway for another port", func(t *testing.T) {
		port := freeConsecutivePorts(t)
		main, err := net.Listen("tcp", ":"+strconv.Itoa(port))
		if err != nil {
			t.Fatal(err)
		}
		defer main.Close()
		health, err := net.Listen("tcp", ":"+strconv.Itoa(port+1))
		if err != nil {
			t.Fatal(err)
		}
		defer health.Close()
		go func() { _ = http.Serve(health, newHealthMux(NewGateway(), port+7)) }()
		if runningGatewayOwnsPort(ctx, client, port) {
			t.Fatal("identity for a different MCP port must not match")
		}
	})

	t.Run("identity answers but MCP port released", func(t *testing.T) {
		port := freeConsecutivePorts(t)
		listeners := startGateway(t, port)
		_ = listeners.main.Close()
		if runningGatewayOwnsPort(ctx, client, port) {
			t.Fatal("a gateway that no longer holds the MCP port must not count")
		}
	})
}

// A duplicate started alongside the first gateway must wait for its health
// port to answer instead of giving up on the first probe.
func TestFindRunningGatewayRetriesWhileHealthNotServed(t *testing.T) {
	port := freeConsecutivePorts(t)
	listeners, err := claimGatewayPorts(port, true)
	if err != nil || listeners.health == nil {
		t.Fatalf("claim: %v", err)
	}
	defer listeners.main.Close()
	defer listeners.health.Close()

	go func() {
		time.Sleep(300 * time.Millisecond)
		_ = http.Serve(listeners.health, newHealthMux(NewGateway(), port))
	}()

	client := newLoopbackHTTPClient(100 * time.Millisecond)
	if !findRunningGateway(context.Background(), client, port, 10, 100*time.Millisecond) {
		t.Fatal("expected the gateway to be found once its health port is served")
	}
}

func TestWaitWhileGatewayRunning(t *testing.T) {
	port := freeConsecutivePorts(t)
	listeners := startGateway(t, port)
	client := newLoopbackHTTPClient(time.Second)

	done := make(chan struct{})
	go func() {
		waitWhileGatewayRunning(context.Background(), client, port, 10*time.Millisecond, 3)
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("returned while the gateway was running")
	case <-time.After(150 * time.Millisecond):
	}

	_ = listeners.main.Close()
	_ = listeners.health.Close()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("did not return after the gateway went away")
	}
}
