package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type immediateReplyWriter struct {
	gateway  *Gateway
	response string
	err      error
}

func (w immediateReplyWriter) Write(p []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	var request JSONRPCMessage
	if err := json.Unmarshal(p, &request); err != nil {
		return 0, err
	}
	if request.Method == "notifications/initialized" {
		return len(p), nil
	}
	if request.ID == "readiness-check" {
		var reply JSONRPCMessage
		if err := json.Unmarshal([]byte(w.response), &reply); err != nil {
			return 0, err
		}
		reply.ID = request.ID
		w.gateway.initialization.reply <- reply
		return len(p), nil
	}
	key, _ := request.ID.(string)
	w.gateway.waitersMu.RLock()
	ch := w.gateway.waiters[key]
	w.gateway.waitersMu.RUnlock()
	if ch != nil {
		ch <- []byte(w.response)
	}
	return len(p), nil
}

func TestHTTPMessageReceivesImmediateResponse(t *testing.T) {
	t.Setenv("MCP_RESPONSE_TIMEOUT", "10ms")
	g := NewGateway()
	response := `{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}`
	g.stdinWriter = bufio.NewWriter(immediateReplyWriter{gateway: g, response: response})
	request := httptest.NewRequest(http.MethodPost, "/message", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	request.Header.Set("X-Client-ID", "client")
	recorder := httptest.NewRecorder()
	g.HandleHTTPMessage(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Body.String() != response {
		t.Fatalf("response = %d %s, want immediate JSON result", recorder.Code, recorder.Body.String())
	}
	if len(g.waiters) != 0 {
		t.Fatalf("response leaked %d waiters", len(g.waiters))
	}
}

func TestHTTPInitializeSessionUsesSuccessfulResponse(t *testing.T) {
	for _, tt := range []struct {
		name     string
		response string
		version  string
		status   int
	}{
		{"negotiated version", `{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05","capabilities":{},"serverInfo":{"name":"test","version":"1"}}}`, "2024-11-05", http.StatusOK},
		{"error", `{"jsonrpc":"2.0","id":1,"error":{"code":-32603,"message":"initialization failed"}}`, "", http.StatusOK},
		{"missing version", `{"jsonrpc":"2.0","id":1,"result":{}}`, "", http.StatusInternalServerError},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("MCP_RESPONSE_TIMEOUT", "10ms")
			g := NewGateway()
			g.stdinWriter = bufio.NewWriter(immediateReplyWriter{gateway: g, response: tt.response})
			request := httptest.NewRequest(http.MethodPost, "/message", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26"}}`))
			request.Header.Set("X-Client-ID", "client")
			recorder := httptest.NewRecorder()
			g.HandleHTTPMessage(recorder, request)
			if recorder.Code != tt.status {
				t.Fatalf("response = %d %s", recorder.Code, recorder.Body.String())
			}
			session, exists := g.sessions["client"]
			if tt.version == "" {
				if exists || recorder.Header().Get("Mcp-Session-Id") != "" {
					t.Fatal("failed initialization established a session")
				}
			} else if !exists || session.ProtocolVersion != tt.version || recorder.Header().Get("Mcp-Session-Id") != "client" {
				t.Fatalf("session = %+v, header = %q", session, recorder.Header().Get("Mcp-Session-Id"))
			}
		})
	}
}

func TestHTTPMessageCleansWaiterOnFailure(t *testing.T) {
	for _, name := range []string{"write error", "timeout", "cancelled"} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("MCP_RESPONSE_TIMEOUT", "1ms")
			g := NewGateway()
			g.stdinWriter = bufio.NewWriter(&strings.Builder{})
			request := httptest.NewRequest(http.MethodPost, "/message", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
			wantStatus := http.StatusGatewayTimeout
			if name == "write error" {
				g.stdinWriter = bufio.NewWriter(immediateReplyWriter{err: errors.New("closed")})
				wantStatus = http.StatusInternalServerError
			}
			if name == "cancelled" {
				ctx, cancel := context.WithCancel(request.Context())
				cancel()
				request = request.WithContext(ctx)
				wantStatus = http.StatusOK
			}
			recorder := httptest.NewRecorder()
			g.HandleHTTPMessage(recorder, request)
			if recorder.Code != wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, wantStatus)
			}
			if len(g.waiters) != 0 {
				t.Fatalf("failure leaked %d waiters", len(g.waiters))
			}
		})
	}
}
