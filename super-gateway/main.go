package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// JSONRPCMessage represents a JSON-RPC 2.0 message
type JSONRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   interface{}     `json:"error,omitempty"`
}

// Client represents a connected WebSocket client
type Client struct {
	ID   string
	Conn *websocket.Conn
	Send chan []byte
}

// Gateway manages the MCP server subprocess and WebSocket connections
type Gateway struct {
	cmd           *exec.Cmd
	clients       map[string]*Client
	clientsMu     sync.RWMutex
	register      chan *Client
	unregister    chan *Client
	broadcast     chan []byte
	upgrader      websocket.Upgrader
	stdinWriter   *bufio.Writer
	stdoutScanner *bufio.Scanner
	stderrScanner *bufio.Scanner
}

func NewGateway() *Gateway {
	return &Gateway{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for simplicity
			},
			Subprotocols: []string{"mcp"}, // Add MCP subprotocol support
		},
	}
}

// StartMCPServer starts the MCP server subprocess
func (g *Gateway) StartMCPServer(cmdParts []string) error {
	if len(cmdParts) == 0 {
		return fmt.Errorf("empty command")
	}

	// Debug: Log exactly what we received
	log.Printf("Received %d command parts:", len(cmdParts))
	for i, part := range cmdParts {
		log.Printf("  [%d]: %q", i, part)
	}

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

	log.Printf("Final command parts: %v", cmdParts)
	log.Printf("Command executable: %s", cmdParts[0])
	log.Printf("Command arguments: %v", cmdParts[1:])

	g.cmd = exec.Command(cmdParts[0], cmdParts[1:]...)
	g.cmd.Env = os.Environ()

	// Set up pipes
	stdin, err := g.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	g.stdinWriter = bufio.NewWriter(stdin)

	stdout, err := g.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	g.stdoutScanner = bufio.NewScanner(stdout)

	stderr, err := g.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	g.stderrScanner = bufio.NewScanner(stderr)

	// Start the process
	if err := g.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	log.Printf("Started MCP server with PID: %d", g.cmd.Process.Pid)

	// Handle stdout
	go func() {
		for g.stdoutScanner.Scan() {
			line := g.stdoutScanner.Text()
			if strings.TrimSpace(line) == "" {
				continue
			}

			var msg JSONRPCMessage
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				log.Printf("Failed to parse JSON from child: %s", line)
				continue
			}

			log.Printf("Child → WebSocket: %s", line)

			// In the Node.js version, it sends using wsTransport?.send(jsonMsg, jsonMsg.id)
			// This means it uses the message's own ID to route back to the correct client
			if msg.ID != nil {
				if idStr, ok := msg.ID.(string); ok && strings.Contains(idStr, ":") {
					parts := strings.SplitN(idStr, ":", 2)
					clientID := parts[0]
					originalID := parts[1]

					// Parse the original ID back to int if it was a number
					var restoredID interface{} = originalID
					if num, err := strconv.Atoi(originalID); err == nil {
						restoredID = num
					}
					msg.ID = restoredID

					// Send to specific client
					g.clientsMu.RLock()
					if client, ok := g.clients[clientID]; ok {
						if data, err := json.Marshal(msg); err == nil {
							select {
							case client.Send <- data:
							default:
								// Client's send channel is full, close it
								g.clientsMu.RUnlock()
								g.unregister <- client
								continue
							}
						}
					}
					g.clientsMu.RUnlock()
					continue
				}
			}

			// If no ID or not a routed message, broadcast to all clients
			if data, err := json.Marshal(msg); err == nil {
				g.broadcast <- data
			}
		}
	}()

	// Handle stderr
	go func() {
		for g.stderrScanner.Scan() {
			log.Printf("Child stderr: %s", g.stderrScanner.Text())
		}
	}()

	// Monitor process exit
	go func() {
		if err := g.cmd.Wait(); err != nil {
			log.Printf("MCP server exited with error: %v", err)
		} else {
			log.Printf("MCP server exited normally")
		}
		os.Exit(1)
	}()

	return nil
}

// SendToMCP sends a message to the MCP server
func (g *Gateway) SendToMCP(msg JSONRPCMessage, clientID string) error {
	// Modify the ID to include the client ID
	if msg.ID != nil {
		msg.ID = fmt.Sprintf("%s:%v", clientID, msg.ID)
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	log.Printf("WebSocket → Child: %s", string(data))

	if _, err := g.stdinWriter.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write to stdin: %w", err)
	}

	return g.stdinWriter.Flush()
}

// Run starts the gateway's main loop
func (g *Gateway) Run() {
	for {
		select {
		case client := <-g.register:
			g.clientsMu.Lock()
			g.clients[client.ID] = client
			g.clientsMu.Unlock()
			log.Printf("New WebSocket connection: %s", client.ID)

		case client := <-g.unregister:
			g.clientsMu.Lock()
			if _, ok := g.clients[client.ID]; ok {
				delete(g.clients, client.ID)
				close(client.Send)
				g.clientsMu.Unlock()
				log.Printf("WebSocket connection closed: %s", client.ID)
			} else {
				g.clientsMu.Unlock()
			}

		case message := <-g.broadcast:
			g.clientsMu.RLock()
			for _, client := range g.clients {
				select {
				case client.Send <- message:
				default:
					// Client's send channel is full, close it
					g.clientsMu.RUnlock()
					g.unregister <- client
					g.clientsMu.RLock()
				}
			}
			g.clientsMu.RUnlock()
		}
	}
}

// HandleWebSocket handles WebSocket connections
func (g *Gateway) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := g.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	client := &Client{
		ID:   uuid.New().String(),
		Conn: conn,
		Send: make(chan []byte, 256),
	}

	g.register <- client

	go client.writePump()
	go client.readPump(g)
}

// readPump reads messages from the WebSocket connection
func (c *Client) readPump(g *Gateway) {
	defer func() {
		g.unregister <- c
		_ = c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512 * 1024) // 512KB max message size

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for client %s: %v", c.ID, err)
			}
			break
		}

		var msg JSONRPCMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Failed to parse message from client %s: %v", c.ID, err)
			continue
		}

		if err := g.SendToMCP(msg, c.ID); err != nil {
			log.Printf("Failed to send message to MCP from client %s: %v", c.ID, err)
		}
	}
}

// writePump writes messages to the WebSocket connection
func (c *Client) writePump() {
	defer func() { _ = c.Conn.Close() }()

	for message := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("Write error for client %s: %v", c.ID, err)
			return
		}
	}
}

// HandleHealth provides a health check endpoint
func (g *Gateway) HandleHealth(w http.ResponseWriter, r *http.Request) {
	if g.cmd == nil || g.cmd.ProcessState != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("MCP server not running"))
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func main() {
	var (
		stdioCmd []string
		port     int
	)

	// Show help if no arguments
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s --port <port> --stdio <command> [args...]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s --port 8000 --stdio npx -y @modelcontextprotocol/server-filesystem /path/to/folder\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nNote: Everything after --stdio is passed to the subprocess\n")
		os.Exit(1)
	}

	// Custom flag parsing to handle everything after --stdio as subprocess args
	args := os.Args[1:]
	portStr := "8000"

	// Find --port flag if present
	for i := 0; i < len(args); i++ {
		if args[i] == "--port" && i+1 < len(args) {
			portStr = args[i+1]
			break
		}
	}

	// Parse port
	_, _ = fmt.Sscanf(portStr, "%d", &port)

	// Override with environment variable if set
	if envPort := os.Getenv("PORT"); envPort != "" {
		_, _ = fmt.Sscanf(envPort, "%d", &port)
	}

	// Find --stdio flag and capture everything after it
	stdioIndex := -1
	for i := 0; i < len(args); i++ {
		if args[i] == "--stdio" {
			stdioIndex = i
			break
		}
	}

	if stdioIndex == -1 || stdioIndex+1 >= len(args) {
		log.Fatal("--stdio flag is required with at least one argument")
	}

	// Everything after --stdio is the command and its arguments
	stdioCmd = args[stdioIndex+1:]

	gateway := NewGateway()

	// Start the MCP server
	if err := gateway.StartMCPServer(stdioCmd); err != nil {
		log.Fatalf("Failed to start MCP server: %v", err)
	}

	// Start the gateway's main loop
	go gateway.Run()

	// Start health check on separate port (port + 1) like Node.js version
	go func() {
		healthMux := http.NewServeMux()
		healthMux.HandleFunc("/health", gateway.HandleHealth)
		log.Printf("Health check endpoint listening on port %d", port+1)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port+1), healthMux); err != nil {
			log.Printf("Failed to start health server: %v", err)
		}
	}()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutting down...")
		if gateway.cmd != nil && gateway.cmd.Process != nil {
			_ = gateway.cmd.Process.Kill()
		}
		os.Exit(0)
	}()

	log.Printf("Starting...")
	log.Printf("  - port: %d", port)
	log.Printf("  - stdio: %s", strings.Join(stdioCmd, " "))

	// In Node.js version, WebSocket is served directly on the port
	// The ws library creates a WebSocket-only server, not HTTP+WebSocket
	log.Printf("WebSocket endpoint: ws://localhost:%d", port)

	// Create a custom handler that only handles WebSocket upgrade requests
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only handle WebSocket upgrade requests
		if r.Header.Get("Upgrade") == "websocket" {
			gateway.HandleWebSocket(w, r)
		} else {
			http.Error(w, "This server only accepts WebSocket connections", http.StatusBadRequest)
		}
	})

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), handler); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
