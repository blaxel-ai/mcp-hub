package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// Run the real gateway entrypoint in a subprocess so its signal handler and
// child restart loop are exercised without allowing os.Exit to end the tests.
func TestGatewayProcessHelper(t *testing.T) {
	switch os.Getenv("GATEWAY_PROCESS_TEST_MODE") {
	case "gateway":
		for i, arg := range os.Args {
			if arg == "--" {
				os.Args = append([]string{os.Args[0]}, os.Args[i+1:]...)
				break
			}
		}
		main()
		os.Exit(0)
	case "stdio":
		pidFile, err := os.OpenFile(os.Getenv("GATEWAY_PROCESS_TEST_PIDS"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			os.Exit(2)
		}
		_, _ = fmt.Fprintln(pidFile, os.Getpid())
		_ = pidFile.Close()
		initialized, notified := false, false
		scanner := bufio.NewScanner(os.Stdin)
		encoder := json.NewEncoder(os.Stdout)
		for scanner.Scan() {
			var msg JSONRPCMessage
			if json.Unmarshal(scanner.Bytes(), &msg) != nil {
				os.Exit(3)
			}
			response := map[string]interface{}{"jsonrpc": "2.0", "id": msg.ID}
			switch msg.Method {
			case "initialize":
				if initialized {
					response["error"] = map[string]interface{}{"code": 0, "message": `duplicate "initialize" received`}
				} else {
					initialized = true
					response["result"] = map[string]interface{}{"protocolVersion": "2024-11-05", "capabilities": map[string]interface{}{"tools": map[string]interface{}{}}, "serverInfo": map[string]interface{}{"name": "strict-process-test", "version": strconv.Itoa(os.Getpid())}}
				}
			case "notifications/initialized":
				if !initialized || notified {
					os.Exit(4)
				}
				notified = true
				continue
			case "tools/list":
				if !notified {
					response["error"] = map[string]interface{}{"code": -32000, "message": "handshake incomplete"}
				} else {
					response["result"] = map[string]interface{}{"tools": []interface{}{map[string]interface{}{"name": "pid_" + strconv.Itoa(os.Getpid()), "inputSchema": map[string]interface{}{"type": "object"}}}}
				}
			case "test/exit":
				os.Exit(0)
			default:
				os.Exit(5)
			}
			if encoder.Encode(response) != nil {
				os.Exit(6)
			}
		}
		os.Exit(0)
	}
}

type processTestLog struct {
	sync.Mutex
	data []byte
}

func (b *processTestLog) Write(p []byte) (int, error) {
	b.Lock()
	defer b.Unlock()
	b.data = append(b.data, p...)
	const limit = 64 * 1024
	if len(b.data) > limit {
		b.data = append([]byte(nil), b.data[len(b.data)-limit:]...)
	}
	return len(p), nil
}
func (b *processTestLog) String() string { b.Lock(); defer b.Unlock(); return string(b.data) }

func TestGatewayReinitializesRestartedProcess(t *testing.T) {
	for _, skip := range []bool{false, true} {
		t.Run(fmt.Sprintf("skip_readiness_%t", skip), func(t *testing.T) {
			executable, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			port := listener.Addr().(*net.TCPAddr).Port
			_ = listener.Close()
			pidPath := t.TempDir() + "/children"
			cmd := exec.Command(executable, "-test.run=^TestGatewayProcessHelper$", "--", "--port", strconv.Itoa(port), "--transport", "http-stream", "--stdio", "env", "GATEWAY_PROCESS_TEST_MODE=stdio", executable, "-test.run=^TestGatewayProcessHelper$")
			// Inherit runtime essentials, but never allow a caller's gateway settings
			// to redirect the integration test to a different transport or port.
			for _, entry := range os.Environ() {
				key, _, _ := strings.Cut(entry, "=")
				if key != "PORT" && key != "TRANSPORT" && key != "SKIP_READINESS_CHECK" && !strings.HasPrefix(key, "GATEWAY_PROCESS_TEST_") {
					cmd.Env = append(cmd.Env, entry)
				}
			}
			cmd.Env = append(cmd.Env, "GATEWAY_PROCESS_TEST_MODE=gateway", "GATEWAY_PROCESS_TEST_PIDS="+pidPath, "SKIP_READINESS_CHECK="+strconv.FormatBool(skip))
			logs := new(processTestLog)
			cmd.Stdout = logs
			cmd.Stderr = logs
			if err = cmd.Start(); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			t.Cleanup(func() {
				_ = cmd.Process.Signal(syscall.SIGTERM)
				select {
				case <-done:
				case <-time.After(5 * time.Second):
					_ = cmd.Process.Kill()
					<-done
				}
				// Also clean up children if startup failed before the gateway registered
				// its signal handler, or if a regression broke graceful shutdown.
				data, _ := os.ReadFile(pidPath)
				for _, line := range strings.Fields(string(data)) {
					if pid, e := strconv.Atoi(line); e == nil {
						if p, e := os.FindProcess(pid); e == nil {
							_ = p.Kill()
						}
					}
				}
				if strings.Contains(logs.String(), "WARNING: DATA RACE") {
					t.Error("race detected in subprocess")
				}
				if t.Failed() {
					t.Log(logs.String())
				}
			})
			client := &http.Client{Timeout: 3 * time.Second}
			defer client.CloseIdleConnections()
			url := "http://127.0.0.1:" + strconv.Itoa(port)
			deadline := time.Now().Add(10 * time.Second)
			for {
				response, e := client.Get(url + "/")
				if e == nil {
					_, _ = io.Copy(io.Discard, response.Body)
					_ = response.Body.Close()
					if response.StatusCode == http.StatusOK {
						break
					}
				}
				if time.Now().After(deadline) {
					t.Fatal("gateway did not start")
				}
				time.Sleep(20 * time.Millisecond)
			}
			rpc := func(method string, id interface{}, params interface{}) (map[string]interface{}, error) {
				msg := map[string]interface{}{"jsonrpc": "2.0", "method": method}
				if id != nil {
					msg["id"] = id
				}
				if params != nil {
					msg["params"] = params
				}
				body, e := json.Marshal(msg)
				if e != nil {
					return nil, e
				}
				response, e := client.Post(url+"/mcp", "application/json", bytes.NewReader(body))
				if e != nil {
					return nil, e
				}
				defer response.Body.Close()
				if response.StatusCode < 200 || response.StatusCode >= 300 {
					return nil, fmt.Errorf("HTTP %d", response.StatusCode)
				}
				if id == nil {
					_, e = io.Copy(io.Discard, response.Body)
					return nil, e
				}
				var result map[string]interface{}
				e = json.NewDecoder(response.Body).Decode(&result)
				if e == nil && result["error"] != nil {
					e = fmt.Errorf("RPC error: %v", result["error"])
				}
				return result, e
			}
			handshake := func() (string, error) {
				result, e := rpc("initialize", "restart-client", map[string]interface{}{"protocolVersion": "2024-11-05", "capabilities": map[string]interface{}{}, "clientInfo": map[string]interface{}{"name": "restart-test", "version": "1"}})
				if e != nil {
					return "", e
				}
				raw, e := json.Marshal(result["result"])
				if e != nil {
					return "", e
				}
				var initialized struct {
					ServerInfo struct {
						Version string `json:"version"`
					} `json:"serverInfo"`
				}
				if e = json.Unmarshal(raw, &initialized); e != nil {
					return "", e
				}
				return initialized.ServerInfo.Version, nil
			}
			checkTools := func(pid string) {
				t.Helper()
				if _, e := rpc("notifications/initialized", nil, nil); e != nil {
					t.Fatal(e)
				}
				result, e := rpc("tools/list", 2, map[string]interface{}{})
				if e != nil {
					t.Fatal(e)
				}
				raw, _ := json.Marshal(result["result"])
				var catalog struct {
					Tools []struct {
						Name string `json:"name"`
					} `json:"tools"`
				}
				if e = json.Unmarshal(raw, &catalog); e != nil {
					t.Fatal(e)
				}
				if len(catalog.Tools) != 1 || catalog.Tools[0].Name != "pid_"+pid {
					t.Fatalf("wrong process tools: %s", raw)
				}
			}
			first, err := handshake()
			if err != nil || first == "" {
				t.Fatalf("initial handshake: pid=%q err=%v", first, err)
			}
			checkTools(first)
			if _, err = rpc("test/exit", nil, nil); err != nil {
				t.Fatal(err)
			}
			// Wait for the actual replacement PID, rather than replaying requests
			// during the intentional restart backoff.
			deadline = time.Now().Add(10 * time.Second)
			for {
				data, _ := os.ReadFile(pidPath)
				if len(strings.Fields(string(data))) >= 2 && strings.Contains(logs.String(), "MCP server restarted successfully") {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("child was not restarted")
				}
				time.Sleep(20 * time.Millisecond)
			}
			second, err := handshake()
			if err != nil {
				t.Fatal(err)
			}
			if second == "" || second == first {
				t.Fatalf("cached dead child initialization: before=%q after=%q", first, second)
			}
			checkTools(second)
		})
	}
}
