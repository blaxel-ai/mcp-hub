package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
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

// runsNestedGateway reports whether the command starts another super-gateway,
// directly or through a wrapper such as the platform's metamorph-wrapper:
// some argument names the gateway binary and a --stdio follows it.
//
// The platform can append its own full command, e.g.
//
//	/bin/metamorph-wrapper /app /app/super-gateway --port 80 ... --stdio <server>
//
// to the image ENTRYPOINT, which is itself super-gateway ... --stdio. The outer
// gateway then waits on a child that never answers MCP on stdout, while the
// inner one serves port 80; after the 30 s readiness timeout the outer one
// fails to bind port 80, exits 1 and stops the VM, in a loop.
func runsNestedGateway(cmdParts []string) bool {
	for i, part := range cmdParts {
		base := filepath.Base(part)
		if base != "super-gateway" && base != "supergateway" {
			continue
		}
		for _, later := range cmdParts[i+1:] {
			if later == "--stdio" {
				return true
			}
		}
	}
	return false
}

// execNestedGateway replaces this process with the command when that command
// runs its own gateway, so only one gateway serves the ports. It returns when
// the command is a plain MCP server, or when exec fails, in which case the
// gateway carries on as before.
func execNestedGateway(stdioCmd []string) {
	cmdParts := normalizeCommandParts(stdioCmd)
	if !runsNestedGateway(cmdParts) {
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
