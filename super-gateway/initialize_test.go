package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type rpcWriter func([]byte) (int, error)

func (f rpcWriter) Write(p []byte) (int, error) { return f(p) }

// A strict stdio server accepts initialize once for the lifetime of its process.
func TestInitializeAfterReadiness(t *testing.T) {
	g := NewGateway()
	client := &Client{ID: "client", Send: make(chan []byte, 8)}
	g.clients[client.ID] = client
	initializes := 0
	g.stdinWriter = bufio.NewWriter(rpcWriter(func(p []byte) (int, error) {
		var request JSONRPCMessage
		if err := json.Unmarshal(p, &request); err != nil {
			return 0, err
		}
		if request.Method != "initialize" {
			return len(p), nil
		}
		initializes++
		reply := JSONRPCMessage{JSONRPC: "2.0", ID: request.ID, Result: json.RawMessage(`{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"serverInfo":{"name":"strict","version":"1"}}`)}
		if initializes > 1 {
			reply.Result = nil
			reply.Error = map[string]interface{}{"code": 0, "message": `duplicate "initialize" received`}
		}
		if request.ID == "readiness-check" {
			g.initialization.reply <- reply
		} else {
			reply.ID = strings.TrimPrefix(request.ID.(string), "client:")
			data, _ := json.Marshal(reply)
			client.Send <- data
		}
		return len(p), nil
	}))
	if err := g.WaitForReady(time.Second); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"first", "second"} {
		if err := g.SendToMCP(JSONRPCMessage{JSONRPC: "2.0", ID: id, Method: "initialize", Params: json.RawMessage(`{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1"}}`)}, client.ID); err != nil {
			t.Fatal(err)
		}
		select {
		case data := <-client.Send:
			var reply JSONRPCMessage
			if err := json.Unmarshal(data, &reply); err != nil {
				t.Fatal(err)
			}
			if reply.Error != nil {
				t.Fatalf("initialize failed: %s", data)
			}
			if reply.ID != id {
				t.Fatalf("id = %v, want %s", reply.ID, id)
			}
		case <-time.After(time.Second):
			t.Fatal("missing initialize reply")
		}
	}
	if initializes != 1 {
		t.Fatalf("initialize count = %d, want 1", initializes)
	}
}

func TestConcurrentInitializeWithoutReadiness(t *testing.T) {
	g := NewGateway()
	var initializes, notifications atomic.Int32
	g.stdinWriter = bufio.NewWriter(rpcWriter(func(p []byte) (int, error) {
		var msg JSONRPCMessage
		if err := json.Unmarshal(p, &msg); err != nil {
			return 0, err
		}
		switch msg.Method {
		case "initialize":
			initializes.Add(1)
			g.initialization.reply <- JSONRPCMessage{JSONRPC: "2.0", Result: json.RawMessage(`{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"serverInfo":{"name":"strict","version":"1"},"instructions":"preserve me"}`)}
		case "notifications/initialized":
			notifications.Add(1)
		}
		return len(p), nil
	}))
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		clientID := fmt.Sprintf("client-%d", i)
		id := interface{}("001")
		if i%2 == 0 {
			id = float64(i)
		}
		ch := make(chan []byte, 1)
		g.waiters[fmt.Sprintf("%s:%v", clientID, id)] = ch
	}
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			clientID := fmt.Sprintf("client-%d", i)
			id := interface{}("001")
			if i%2 == 0 {
				id = float64(i)
			}
			if err := g.SendToMCP(JSONRPCMessage{JSONRPC: "2.0", ID: id, Method: "initialize"}, clientID); err != nil {
				t.Error(err)
				return
			}
			ch := g.waiters[fmt.Sprintf("%s:%v", clientID, id)]
			select {
			case data := <-ch:
				var msg JSONRPCMessage
				if err := json.Unmarshal(data, &msg); err != nil {
					t.Error(err)
					return
				}
				if msg.ID != id || msg.Error != nil || !strings.Contains(string(msg.Result), "preserve me") {
					t.Errorf("unexpected reply: %s", data)
				}
			case <-time.After(time.Second):
				t.Error("missing response")
			}
			if err := g.SendToMCP(JSONRPCMessage{JSONRPC: "2.0", Method: "notifications/initialized"}, clientID); err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	if initializes.Load() != 1 || notifications.Load() != 1 {
		t.Fatalf("initialize=%d initialized=%d, want 1 each", initializes.Load(), notifications.Load())
	}
}

func TestSlowInitializationIsNotRetried(t *testing.T) {
	g := NewGateway()
	var initializes atomic.Int32
	received := make(chan struct{})
	g.stdinWriter = bufio.NewWriter(rpcWriter(func(p []byte) (int, error) {
		var msg JSONRPCMessage
		if err := json.Unmarshal(p, &msg); err != nil {
			return 0, err
		}
		if msg.Method == "initialize" {
			if initializes.Add(1) == 1 {
				close(received)
			}
		}
		return len(p), nil
	}))
	// The readiness caller timing out must not restart the child's handshake.
	if err := g.WaitForReady(10 * time.Millisecond); err == nil {
		t.Fatal("expected timeout")
	}
	<-received
	done := make(chan error, 1)
	go func() { done <- g.WaitForReady(3 * time.Second) }()
	time.Sleep(1100 * time.Millisecond) // Cross the old one-second retry interval.
	g.initialization.reply <- JSONRPCMessage{JSONRPC: "2.0", Result: json.RawMessage(`{"protocolVersion":"2024-11-05"}`)}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if initializes.Load() != 1 {
		t.Fatalf("sent %d initialize requests", initializes.Load())
	}
}

func TestInitializationRejectsExitedChildAndResetsForReplacement(t *testing.T) {
	g := NewGateway()
	var initializes atomic.Int32
	g.stdinWriter = bufio.NewWriter(rpcWriter(func(p []byte) (int, error) {
		var msg JSONRPCMessage
		if err := json.Unmarshal(p, &msg); err != nil {
			return 0, err
		}
		if msg.Method == "initialize" {
			version := initializes.Add(1)
			g.initialization.reply <- JSONRPCMessage{JSONRPC: "2.0", Result: json.RawMessage(fmt.Sprintf(`{"protocolVersion":"2024-11-05","serverInfo":{"version":"%d"}}`, version))}
		}
		return len(p), nil
	}))
	if err := g.WaitForReady(time.Second); err != nil {
		t.Fatal(err)
	}
	close(g.initialization.exited)
	for i := 0; i < 20; i++ {
		if err := g.WaitForReady(time.Second); err == nil {
			t.Fatal("accepted exited child")
		}
	}
	g.initialization = newStdioInitialization(g.stdinWriter)
	reply, err := g.initializeStdio(time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if initializes.Load() != 2 || !strings.Contains(string(reply.Result), `"version":"2"`) {
		t.Fatalf("replacement reused old result: %s", reply.Result)
	}
}
