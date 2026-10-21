package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPInitializeRejectsInvalidParamsBeforeChildInitialization(t *testing.T) {
	for _, params := range []string{
		`null`, `[]`, `"invalid"`, `{}`,
		`{"protocolVersion":null,"capabilities":{},"clientInfo":{"name":"test","version":"1"}}`,
		`{"protocolVersion":42,"capabilities":{},"clientInfo":{"name":"test","version":"1"}}`,
		`{"protocolVersion":"","capabilities":{},"clientInfo":{"name":"test","version":"1"}}`,
		`{"protocolVersion":"2024-11-05","clientInfo":{"name":"test","version":"1"}}`,
		`{"protocolVersion":"2024-11-05","capabilities":null,"clientInfo":{"name":"test","version":"1"}}`,
		`{"protocolVersion":"2024-11-05","capabilities":[],"clientInfo":{"name":"test","version":"1"}}`,
		`{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":null}`,
		`{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":[]}`,
		`{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test"}}`,
		`{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"","version":"1"}}`,
		`{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":1}}`,
	} {
		t.Run(params, func(t *testing.T) {
			g := NewGateway()
			request := httptest.NewRequest(http.MethodPost, "/message", strings.NewReader(`{"jsonrpc":"2.0","id":"invalid-init","method":"initialize","params":`+params+`}`))
			recorder := httptest.NewRecorder()
			g.HandleHTTPMessage(recorder, request)
			var reply struct {
				ID    string `json:"id"`
				Error struct {
					Code int `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &reply); err != nil {
				t.Fatalf("invalid response: %d %s", recorder.Code, recorder.Body.String())
			}
			if recorder.Code != http.StatusOK || reply.ID != "invalid-init" || reply.Error.Code != -32602 {
				t.Fatalf("response = %d %s, want matching ID and invalid params error", recorder.Code, recorder.Body.String())
			}
			if g.initialization != nil || len(g.sessions) != 0 || len(g.waiters) != 0 || recorder.Header().Get("Mcp-Session-Id") != "" {
				t.Fatal("invalid initialize started the child handshake, established a session, or leaked a waiter")
			}
		})
	}
}

func TestValidateInitializeParamsAllowsNegotiationAndExtensions(t *testing.T) {
	params := json.RawMessage(`{"protocolVersion":"2099-01-01","capabilities":{"roots":{"listChanged":true}},"clientInfo":{"name":"test","version":"1","title":"Test client"},"_meta":{}}`)
	if err := validateInitializeParams(params); err != nil {
		t.Fatal(err)
	}
	if err := validateInitializeParams(nil); err == nil {
		t.Fatal("missing params accepted")
	}
}
