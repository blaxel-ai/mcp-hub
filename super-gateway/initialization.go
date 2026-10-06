package main

import (
	"bufio"
	"encoding/json"
	"fmt"
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
	request := JSONRPCMessage{JSONRPC: "2.0", ID: "readiness-check", Method: "initialize", Params: json.RawMessage(`{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"super-gateway","version":"1.0.0"}}`)}
	if state.err = g.writeInitialization(state, request); state.err != nil {
		return
	}
	// Never retry initialize on a live process: a slow response does not imply
	// that the first request was ignored. Bound the single in-flight handshake.
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()
	select {
	case state.result = <-state.reply:
		if state.result.Error != nil {
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
	case <-state.exited:
		state.err = fmt.Errorf("MCP server exited during initialization")
	case <-timer.C:
		state.err = fmt.Errorf("timeout waiting for MCP server initialization")
	}
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
