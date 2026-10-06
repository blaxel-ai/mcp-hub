package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// The gateway is the sole client of a shared stdio process. Downstream clients
// receive its negotiated protocol, capabilities and server metadata unchanged.
// In particular, we cannot advertise one downstream client's capabilities on
// behalf of the others (e.g. sampling or roots).
type stdioInitialization struct {
	once   sync.Once
	done   chan struct{}
	reply  chan JSONRPCMessage
	exited chan struct{}
	writer *bufio.Writer
	result JSONRPCMessage
	err    error
}

func newStdioInitialization(writer *bufio.Writer) *stdioInitialization {
	return &stdioInitialization{done: make(chan struct{}), reply: make(chan JSONRPCMessage, 1), exited: make(chan struct{}), writer: writer}
}

func (g *Gateway) initializeStdio(timeout time.Duration) (JSONRPCMessage, error) {
	g.initializationMu.Lock()
	if g.initialization == nil {
		g.initialization = newStdioInitialization(g.stdinWriter)
	}
	state := g.initialization
	g.initializationMu.Unlock()
	state.once.Do(func() { go g.runInitialization(state) })
	// Once negotiated, new clients need neither another child request nor a timer.
	select {
	case <-state.done:
		return state.completedResult()
	default:
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-state.done:
		return state.completedResult()
	case <-state.exited:
		return JSONRPCMessage{}, fmt.Errorf("MCP server exited during initialization")
	case <-timer.C:
		return JSONRPCMessage{}, fmt.Errorf("timeout waiting for MCP server initialization")
	}
}

func (state *stdioInitialization) completedResult() (JSONRPCMessage, error) {
	select {
	case <-state.exited:
		return JSONRPCMessage{}, fmt.Errorf("MCP server exited during initialization")
	default:
		return state.result, state.err
	}
}

func (g *Gateway) runInitialization(state *stdioInitialization) {
	defer close(state.done)
	// Keep the existing version first for compatibility. Only an explicit
	// version rejection permits trying another version, never a slow response.
	versions := []string{"2024-11-05", "2025-11-25", "2025-06-18", "2025-03-26"}
	for _, version := range versions {
		params, _ := json.Marshal(map[string]interface{}{
			"protocolVersion": version,
			"capabilities":    map[string]interface{}{},
			"clientInfo":      map[string]string{"name": "super-gateway", "version": "1.0.0"},
		})
		request := JSONRPCMessage{JSONRPC: "2.0", ID: "readiness-check", Method: "initialize", Params: params}
		if state.err = g.writeInitialization(state, request); state.err != nil {
			return
		}
		// Callers have their own deadlines. This one worker belongs to the child
		// lifetime, allowing its original response to arrive after readiness times
		// out without ever sending duplicate initialize requests.
		select {
		case state.result = <-state.reply:
			if state.result.Error != nil {
				if isProtocolVersionRejection(state.result.Error) {
					continue
				}
				return
			}
			var result struct {
				ProtocolVersion string `json:"protocolVersion"`
			}
			if err := json.Unmarshal(state.result.Result, &result); err != nil || result.ProtocolVersion == "" {
				state.err = fmt.Errorf("invalid MCP initialize result")
				return
			}
			state.err = g.writeInitialization(state, JSONRPCMessage{JSONRPC: "2.0", Method: "notifications/initialized"})
			return
		case <-state.exited:
			state.err = fmt.Errorf("MCP server exited during initialization")
			return
		}
	}
}

func isProtocolVersionRejection(value interface{}) bool {
	data, err := json.Marshal(value)
	if err != nil {
		return false
	}
	var rpcError struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if json.Unmarshal(data, &rpcError) != nil || rpcError.Code != -32602 {
		return false
	}
	message := strings.ToLower(rpcError.Message)
	return strings.Contains(message, "protocol") && strings.Contains(message, "version") &&
		(strings.Contains(message, "unsupported") || strings.Contains(message, "not supported"))
}

func (g *Gateway) writeInitialization(state *stdioInitialization, msg JSONRPCMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	g.stdinMu.Lock()
	defer g.stdinMu.Unlock()
	if state.writer == nil {
		return fmt.Errorf("MCP stdin is unavailable")
	}
	if _, err := state.writer.Write(append(data, '\n')); err != nil {
		return err
	}
	return state.writer.Flush()
}

func (g *Gateway) deliverInitialization(reply JSONRPCMessage, clientID string) error {
	data, err := json.Marshal(reply)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%s:%v", clientID, reply.ID)
	g.waitersMu.RLock()
	waiter := g.waiters[key]
	g.waitersMu.RUnlock()
	if waiter != nil {
		select {
		case waiter <- data:
		default:
		}
		return nil
	}
	g.clientsMu.RLock()
	defer g.clientsMu.RUnlock()
	if client := g.clients[clientID]; client != nil {
		select {
		case client.Send <- data:
		default:
			return fmt.Errorf("client response queue is full")
		}
		return nil
	}
	g.sseClientsMu.RLock()
	defer g.sseClientsMu.RUnlock()
	if client := g.sseClients[clientID]; client != nil {
		select {
		case client.Send <- data:
		default:
			return fmt.Errorf("client response queue is full")
		}
	}
	return nil
}
