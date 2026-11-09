package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"syscall"
	"time"
)

// The platform can launch the gateway twice in one VM. The second copy used to
// spawn its own MCP child, wait out the 30 s readiness check, fail to bind the
// MCP port and exit 1, which stopped the whole VM even though the first copy
// was serving; the replacement VM then hit the same race. Claiming both ports
// before spawning the child makes a duplicate detectable at once, and lets it
// stay out of the way instead of taking down the healthy gateway with it.

const (
	duplicateHealthInterval    = 5 * time.Second
	duplicateHealthMaxFailures = 3
)

// gatewayListeners holds the ports claimed before the MCP child starts.
// health is nil when the health port could not be claimed; that has never
// been fatal.
type gatewayListeners struct {
	main   net.Listener
	health net.Listener
}

func claimGatewayPorts(port int) (*gatewayListeners, error) {
	mainListener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}
	healthListener, err := net.Listen("tcp", fmt.Sprintf(":%d", port+1))
	if err != nil {
		log.Printf("Failed to start health server: %v", err)
		healthListener = nil
	}
	return &gatewayListeners{main: mainListener, health: healthListener}, nil
}

func isAddrInUse(err error) bool {
	return errors.Is(err, syscall.EADDRINUSE)
}

// gatewayHealthy reports whether a gateway answers 200 on the given health port.
func gatewayHealthy(ctx context.Context, client *http.Client, healthPort int) bool {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/health", healthPort), nil)
	if err != nil {
		return false
	}
	response, err := client.Do(request) // #nosec G704 -- fixed loopback URL; transport refuses non-loopback dials.
	if err != nil {
		return false
	}
	_ = response.Body.Close()
	return response.StatusCode == http.StatusOK
}

// waitWhileGatewayHealthy blocks while another gateway keeps answering its
// health check, and returns once it fails maxFailures checks in a row or ctx ends.
func waitWhileGatewayHealthy(ctx context.Context, client *http.Client, healthPort int, interval time.Duration, maxFailures int) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	failures := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		if gatewayHealthy(ctx, client, healthPort) {
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
// is already held by a healthy gateway, this process is a duplicate launch: it
// idles while that gateway stays healthy and exits non-zero only once it stops
// answering, so the platform restarts the workload for a real failure only.
func claimPortsOrDeferToRunningGateway(port int) *gatewayListeners {
	listeners, err := claimGatewayPorts(port)
	if err == nil {
		return listeners
	}
	if !isAddrInUse(err) {
		log.Fatalf("Failed to start server: %v", err)
	}
	client := newLoopbackHTTPClient(2 * time.Second)
	if !gatewayHealthy(context.Background(), client, port+1) {
		log.Fatalf("Failed to start server: %v", err)
	}
	log.Printf("Port %d is already served by a healthy gateway; this is a duplicate launch, not starting a second MCP server", port)
	waitWhileGatewayHealthy(context.Background(), client, port+1, duplicateHealthInterval, duplicateHealthMaxFailures)
	log.Fatalf("The gateway already serving port %d stopped answering health checks; exiting so the workload restarts", port)
	return nil
}
