package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"syscall"
	"time"
)

// The platform can launch the gateway twice in one VM. The second copy used to
// spawn its own MCP child, wait out the 30 s readiness check, fail to bind the
// MCP port and exit 1, which stopped the whole VM even though the first copy
// was serving; the replacement VM then hit the same race. Claiming both ports
// and serving the health port before spawning the child makes a duplicate
// detectable at once, and lets it stay out of the way instead of taking down
// the healthy gateway with it.

const (
	gatewayName         = "super-gateway"
	gatewayIdentityPath = "/gateway"

	duplicateProbeAttempts     = 5
	duplicateProbeInterval     = time.Second
	duplicateHealthInterval    = 5 * time.Second
	duplicateHealthMaxFailures = 3
)

// gatewayIdentity is served on the health port so another launch can tell a
// gateway owning this MCP port apart from any other service on port+1.
type gatewayIdentity struct {
	Gateway string `json:"gateway"`
	MCPPort int    `json:"mcpPort"`
}

// gatewayListeners holds the ports claimed before the MCP child starts.
// health is nil when the health port was not claimed; that has never been fatal.
type gatewayListeners struct {
	main   net.Listener
	health net.Listener
}

// claimGatewayPorts binds the MCP port and, when claimHealth is set, port+1.
func claimGatewayPorts(port int, claimHealth bool) (*gatewayListeners, error) {
	mainListener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}
	listeners := &gatewayListeners{main: mainListener}
	if !claimHealth {
		return listeners, nil
	}
	healthListener, err := net.Listen("tcp", fmt.Sprintf(":%d", port+1))
	if err != nil {
		log.Printf("Failed to start health server: %v", err)
		return listeners, nil
	}
	listeners.health = healthListener
	return listeners, nil
}

// healthPortFree reports whether port+1 may be claimed for health checks. An
// HTTP upstream child listening there must keep it, as it did before the
// gateway claimed its ports up front.
func healthPortFree(port int, upstream *HTTPUpstreamConfig) bool {
	if upstream == nil || upstream.URL == nil {
		return true
	}
	upstreamPort := upstream.URL.Port()
	if upstreamPort == "" {
		upstreamPort = "80"
		if upstream.URL.Scheme == "https" {
			upstreamPort = "443"
		}
	}
	return upstreamPort != strconv.Itoa(port+1)
}

func newHealthMux(gateway *Gateway, port int) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", gateway.HandleHealth)
	mux.HandleFunc(gatewayIdentityPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(gatewayIdentity{Gateway: gatewayName, MCPPort: port})
	})
	return mux
}

func isAddrInUse(err error) bool {
	return errors.Is(err, syscall.EADDRINUSE)
}

// runningGatewayOwnsPort reports whether a live gateway serving this MCP port
// answers on port+1, and the MCP port is still held.
func runningGatewayOwnsPort(ctx context.Context, client *http.Client, port int) bool {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d%s", port+1, gatewayIdentityPath), nil)
	if err != nil {
		return false
	}
	response, err := client.Do(request) // #nosec G704 -- fixed loopback URL; transport refuses non-loopback dials.
	if err != nil {
		return false
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return false
	}
	var identity gatewayIdentity
	if err := json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(&identity); err != nil {
		return false
	}
	if identity.Gateway != gatewayName || identity.MCPPort != port {
		return false
	}
	conn, err := (&net.Dialer{Timeout: time.Second}).DialContext(ctx, "tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// findRunningGateway retries briefly so a duplicate started alongside the
// first gateway does not give up before that gateway serves its health port.
func findRunningGateway(ctx context.Context, client *http.Client, port int, attempts int, interval time.Duration) bool {
	for attempt := 1; attempt <= attempts; attempt++ {
		if runningGatewayOwnsPort(ctx, client, port) {
			return true
		}
		if attempt == attempts {
			break
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(interval):
		}
	}
	return false
}

// waitWhileGatewayRunning blocks while the other gateway keeps owning the
// port, and returns once it fails maxFailures checks in a row or ctx ends.
func waitWhileGatewayRunning(ctx context.Context, client *http.Client, port int, interval time.Duration, maxFailures int) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	failures := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		if runningGatewayOwnsPort(ctx, client, port) {
			failures = 0
			continue
		}
		failures++
		if failures >= maxFailures {
			return
		}
	}
}

// claimPortsOrDeferToRunningGateway claims the gateway ports. When the MCP port
// is already held by another gateway, this process is a duplicate launch: it
// idles while that gateway keeps running and exits non-zero only once it is
// gone, so the platform restarts the workload for a real failure only.
func claimPortsOrDeferToRunningGateway(port int, claimHealth bool) *gatewayListeners {
	listeners, err := claimGatewayPorts(port, claimHealth)
	if err == nil {
		return listeners
	}
	if !isAddrInUse(err) {
		log.Fatalf("Failed to start server: %v", err)
	}
	client := newLoopbackHTTPClient(2 * time.Second)
	ctx := context.Background()
	if !findRunningGateway(ctx, client, port, duplicateProbeAttempts, duplicateProbeInterval) {
		log.Fatalf("Failed to start server: %v", err)
	}
	log.Printf("Port %d is already served by another gateway; this is a duplicate launch, not starting a second MCP server", port)
	waitWhileGatewayRunning(ctx, client, port, duplicateHealthInterval, duplicateHealthMaxFailures)
	log.Fatalf("The gateway serving port %d is gone; exiting so the workload restarts", port)
	return nil
}
