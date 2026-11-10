package main

import (
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// normalizeCommandParts splits a command that arrived as one string, or with
// arguments concatenated into one, into separate arguments.
func normalizeCommandParts(cmdParts []string) []string {
	// Handle the case where the entire command is passed as a single string
	// This happens when Docker CMD is injected as a single argument
	if len(cmdParts) == 1 && strings.Contains(cmdParts[0], " ") {
		// Split the single string into command and arguments
		// This handles cases like "node /app/build/index.js --tools=..."
		cmdParts = strings.Fields(cmdParts[0])
		log.Printf("Detected single string command, split into: %v", cmdParts)
	} else if len(cmdParts) > 1 {
		// Check if any argument (except the first) contains spaces and should be split
		// This handles cases where arguments are incorrectly concatenated
		newCmdParts := []string{cmdParts[0]} // Keep the command as-is
		for i := 1; i < len(cmdParts); i++ {
			if strings.Contains(cmdParts[i], " ") && !strings.HasPrefix(cmdParts[i], "--") {
				// This argument contains spaces and isn't a flag, split it
				log.Printf("Splitting argument [%d]: %q", i, cmdParts[i])
				splitArgs := strings.Fields(cmdParts[i])
				newCmdParts = append(newCmdParts, splitArgs...)
			} else {
				newCmdParts = append(newCmdParts, cmdParts[i])
			}
		}
		if len(newCmdParts) != len(cmdParts) {
			log.Printf("Arguments were split from %d to %d parts", len(cmdParts), len(newCmdParts))
			cmdParts = newCmdParts
		}
	}
	return cmdParts
}

// platformWrapper is the platform's process wrapper, at its fixed path in the guest.
const platformWrapper = "/bin/metamorph-wrapper"

// runsNestedGateway reports whether the command starts this same gateway
// binary, either directly or behind the platform wrapper, with a --stdio
// after it.
//
// The platform can append its own full command, e.g.
//
//	/bin/metamorph-wrapper /app /app/super-gateway --port 80 ... --stdio <server>
//
// to the image ENTRYPOINT, which is itself super-gateway ... --stdio. The outer
// gateway then waits on a child that never answers MCP on stdout, while the
// inner one serves port 80; after the 30 s readiness timeout the outer one
// fails to bind port 80, exits 1 and stops the VM, in a loop.
//
// Handing over is only safe to the same gateway binary, which enforces the
// same routing and header rules. Any other command stays a supervised child,
// whatever its arguments are named.
func runsNestedGateway(cmdParts []string, isSelf func(string) bool) bool {
	if len(cmdParts) == 0 {
		return false
	}
	gatewayIndex := 0
	if cmdParts[0] == platformWrapper {
		// metamorph-wrapper <app dir> <command> [args...]
		gatewayIndex = 2
	}
	if gatewayIndex >= len(cmdParts) || !isSelf(cmdParts[gatewayIndex]) {
		return false
	}
	for _, later := range cmdParts[gatewayIndex+1:] {
		if later == "--stdio" {
			return true
		}
	}
	return false
}

// isGatewayBinary reports whether path names the running gateway executable.
func isGatewayBinary(path string) bool {
	self, err := os.Executable()
	if err != nil {
		return false
	}
	resolved, err := exec.LookPath(path)
	if err != nil {
		return false
	}
	selfInfo, err := os.Stat(self)
	if err != nil {
		return false
	}
	pathInfo, err := os.Stat(resolved)
	if err != nil {
		return false
	}
	return os.SameFile(selfInfo, pathInfo)
}

// execNestedGateway replaces this process with the command when that command
// runs its own gateway, so only one gateway serves the ports. It returns when
// the command is a plain MCP server, or when exec fails, in which case the
// gateway carries on as before.
func execNestedGateway(stdioCmd []string) {
	cmdParts := normalizeCommandParts(stdioCmd)
	if !runsNestedGateway(cmdParts, isGatewayBinary) {
		return
	}
	path, err := exec.LookPath(cmdParts[0])
	if err != nil {
		log.Printf("Command runs its own gateway but %q was not found: %v; wrapping it anyway", cmdParts[0], err)
		return
	}
	log.Printf("Command runs its own gateway; handing over to it instead of wrapping it: %v", cmdParts)
	if err := syscall.Exec(path, cmdParts, os.Environ()); err != nil { // #nosec G204 -- the command is this container's own configured MCP command.
		log.Printf("Failed to hand over to the nested gateway: %v; wrapping it anyway", err)
	}
}
