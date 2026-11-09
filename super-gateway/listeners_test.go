package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
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
		listeners, err := claimGatewayPorts(port)
		if err != nil || listeners.health == nil {
			if listeners != nil {
				_ = listeners.main.Close()
			}
			continue
		}
		_ = listeners.main.Close()
		_ = listeners.health.Close()
		return port
	}
	t.Fatal("no free consecutive ports")
	return 0
}

func TestClaimGatewayPortsReportsAddrInUse(t *testing.T) {
	port := freeConsecutivePorts(t)
	first, err := claimGatewayPorts(port)
	if err != nil {
		t.Fatalf("first claim: %v", err)
	}
	defer first.main.Close()
	defer first.health.Close()

	_, err = claimGatewayPorts(port)
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

	listeners, err := claimGatewayPorts(port)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	defer listeners.main.Close()
	if listeners.health != nil {
		t.Fatal("health listener should be nil when its port is busy")
	}
}

func healthServer(t *testing.T, healthy *atomic.Bool) int {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" && healthy.Load() {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	return server.Listener.Addr().(*net.TCPAddr).Port
}

func TestGatewayHealthy(t *testing.T) {
	var healthy atomic.Bool
	healthy.Store(true)
	port := healthServer(t, &healthy)
	client := newLoopbackHTTPClient(time.Second)

	if !gatewayHealthy(context.Background(), client, port) {
		t.Fatal("expected healthy gateway")
	}
	healthy.Store(false)
	if gatewayHealthy(context.Background(), client, port) {
		t.Fatal("expected unhealthy gateway on 500")
	}

	closed := freeConsecutivePorts(t)
	if gatewayHealthy(context.Background(), client, closed) {
		t.Fatal("expected unhealthy gateway when nothing listens")
	}
}

func TestWaitWhileGatewayHealthy(t *testing.T) {
	var healthy atomic.Bool
	healthy.Store(true)
	port := healthServer(t, &healthy)
	client := newLoopbackHTTPClient(time.Second)

	done := make(chan struct{})
	go func() {
		waitWhileGatewayHealthy(context.Background(), client, port, 10*time.Millisecond, 3)
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("returned while the gateway was healthy")
	case <-time.After(150 * time.Millisecond):
	}

	healthy.Store(false)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("did not return after the gateway became unhealthy")
	}
}
