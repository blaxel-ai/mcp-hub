package main

import (
	"bufio"
	"encoding/json"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

func TestInitializationRecoversAfterReadinessDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := NewGateway()
		var count atomic.Int32
		g.stdinWriter = bufio.NewWriter(rpcWriter(func(p []byte) (int, error) {
			var msg JSONRPCMessage
			if err := json.Unmarshal(p, &msg); err != nil {
				return 0, err
			}
			if msg.Method == "initialize" {
				count.Add(1)
				go func() {
					time.Sleep(35 * time.Second)
					g.initialization.reply <- JSONRPCMessage{JSONRPC: "2.0", Result: json.RawMessage(`{"protocolVersion":"2024-11-05"}`)}
				}()
			}
			return len(p), nil
		}))
		if err := g.WaitForReady(30 * time.Second); err == nil {
			t.Fatal("expected readiness timeout")
		}
		time.Sleep(10 * time.Second)
		if err := g.WaitForReady(time.Second); err != nil {
			t.Fatalf("late success rejected: %v", err)
		}
		if count.Load() != 1 {
			t.Fatalf("initialize count = %d", count.Load())
		}
	})
}

func TestInitializationNegotiatesAfterVersionRejection(t *testing.T) {
	g := NewGateway()
	var versions []string
	g.stdinWriter = bufio.NewWriter(rpcWriter(func(p []byte) (int, error) {
		var msg JSONRPCMessage
		if err := json.Unmarshal(p, &msg); err != nil {
			return 0, err
		}
		if msg.Method == "initialize" {
			var params struct {
				ProtocolVersion string `json:"protocolVersion"`
			}
			if err := json.Unmarshal(msg.Params, &params); err != nil {
				return 0, err
			}
			versions = append(versions, params.ProtocolVersion)
			reply := JSONRPCMessage{JSONRPC: "2.0", ID: msg.ID}
			if params.ProtocolVersion != "2025-03-26" {
				reply.Error = map[string]interface{}{"code": -32602, "message": "Unsupported protocol version"}
			} else {
				reply.Result = json.RawMessage(`{"protocolVersion":"2025-03-26"}`)
			}
			g.initialization.reply <- reply
		}
		return len(p), nil
	}))
	if err := g.WaitForReady(time.Second); err != nil {
		t.Fatal(err)
	}
	// A successfully negotiated child must never receive another initialize.
	if err := g.WaitForReady(time.Second); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, version := range versions {
		if seen[version] {
			t.Fatalf("repeated version %s", version)
		}
		seen[version] = true
	}
	if versions[len(versions)-1] != "2025-03-26" {
		t.Fatalf("negotiated versions: %v", versions)
	}
}

func TestInitializationDoesNotRetryOtherErrors(t *testing.T) {
	g := NewGateway()
	var count atomic.Int32
	g.stdinWriter = bufio.NewWriter(rpcWriter(func(p []byte) (int, error) {
		var msg JSONRPCMessage
		if err := json.Unmarshal(p, &msg); err != nil {
			return 0, err
		}
		if msg.Method == "initialize" {
			count.Add(1)
			g.initialization.reply <- JSONRPCMessage{JSONRPC: "2.0", Error: map[string]interface{}{"code": -32602, "message": "invalid client metadata"}}
		}
		return len(p), nil
	}))
	for i := 0; i < 2; i++ {
		if err := g.WaitForReady(time.Second); err == nil {
			t.Fatal("expected initialize error")
		}
	}
	if count.Load() != 1 {
		t.Fatalf("retried terminal error %d times", count.Load())
	}
}
