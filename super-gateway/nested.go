package main

import (
	"log"
	"os"
	"os/exec"
	"strings"
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

// unwrapNestedGateway returns the MCP server command when the stdio command is
// itself this gateway, either directly or behind the platform wrapper:
//
//	[/bin/metamorph-wrapper <app dir>] <this gateway> [flags...] --stdio <server...>
//
// The platform can append such a command to the image ENTRYPOINT, which is
// itself super-gateway ... --stdio. Run as given, the outer gateway waits on a
// child that never answers MCP on stdout while the inner one serves port 80;
// after the 30 s readiness timeout the outer one fails to bind port 80, exits 1
// and stops the VM, in a loop.
//
// The gateway keeps running itself with its own flags and supervises <server>
// directly. Nothing else from the command is executed and the nested gateway's
// flags are ignored, so the command cannot change how requests reach the guest.
// isSelf reports whether a path names this gateway's executable.
func unwrapNestedGateway(cmdParts []string, isSelf func(string) bool) ([]string, bool) {
	if len(cmdParts) == 0 {
		return nil, false
	}
	gatewayIndex := 0
	if cmdParts[0] == platformWrapper {
		// metamorph-wrapper <app dir> <command> [args...]
		gatewayIndex = 2
	}
	if gatewayIndex >= len(cmdParts) || !isSelf(cmdParts[gatewayIndex]) {
		return nil, false
	}
	for i := gatewayIndex + 1; i < len(cmdParts); i++ {
		if cmdParts[i] == "--stdio" {
			if i+1 == len(cmdParts) {
				return nil, false
			}
			return cmdParts[i+1:], true
		}
	}
	return nil, false
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

// resolveStdioCommand returns the command the gateway should supervise:
// the MCP server inside a nested gateway command, or the command as given.
func resolveStdioCommand(stdioCmd []string) []string {
	cmdParts := normalizeCommandParts(stdioCmd)
	server, ok := unwrapNestedGateway(cmdParts, isGatewayBinary)
	if !ok {
		return stdioCmd
	}
	log.Printf("Command runs this gateway again; supervising its MCP server directly instead of nesting gateways: %v", server)
	return server
}

// gatewayFlagArgs returns the arguments that configure this gateway: those
// before the first --stdio. Everything after it is the subprocess command.
func gatewayFlagArgs(args []string) []string {
	for i, arg := range args {
		if arg == "--stdio" {
			return args[:i]
		}
	}
	return args
}
