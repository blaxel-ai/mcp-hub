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
			case "test/oversized-response":
				response["result"] = map[string]interface{}{"blob": strings.Repeat("x", maxChildLineSize+1)}
			case "test/oversized-stderr":
				_, _ = fmt.Fprintln(os.Stderr, strings.Repeat("x", maxChildLineSize+1))
				// Keep logging past the pipe buffer, as a real server would.
				for i := 0; i < 256; i++ {
					_, _ = fmt.Fprintln(os.Stderr, strings.Repeat("y", 1024))
				}
				response["result"] = map[string]interface{}{}
			case "test/close-stdout":
				_ = os.Stdout.Close()
				time.Sleep(time.Hour)
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

// processGateway is a gateway subprocess wrapping the strict stdio helper above.
type processGateway struct {
	t       *testing.T
	url     string
	client  *http.Client
	logs    *processTestLog
	pidPath string
}

func startProcessGateway(t *testing.T, skipReadiness bool) *processGateway {
	t.Helper()
	// The free port can be taken before the gateway binds it; retry on a new one.
	for attempt := 1; ; attempt++ {
		if gateway := tryStartProcessGateway(t, skipReadiness); gateway != nil || attempt == 3 {
			if gateway == nil {
				t.Fatal("gateway port was taken on every attempt")
			}
			return gateway
		}
	}
}

func tryStartProcessGateway(t *testing.T, skipReadiness bool) *processGateway {
	t.Helper()
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
	cmd.Env = append(cmd.Env, "GATEWAY_PROCESS_TEST_MODE=gateway", "GATEWAY_PROCESS_TEST_PIDS="+pidPath, "SKIP_READINESS_CHECK="+strconv.FormatBool(skipReadiness))
	logs := new(processTestLog)
	cmd.Stdout = logs
	cmd.Stderr = logs
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	exited := make(chan struct{})
	go func() { _ = cmd.Wait(); close(exited) }()
	gateway := &processGateway{t: t, url: "http://127.0.0.1:" + strconv.Itoa(port), client: &http.Client{Timeout: 3 * time.Second}, logs: logs, pidPath: pidPath}
	t.Cleanup(func() {
		gateway.client.CloseIdleConnections()
		_ = cmd.Process.Signal(syscall.SIGTERM)
		select {
		case <-exited:
		case <-time.After(5 * time.Second):
			_ = cmd.Process.Kill()
			<-exited
		}
		// Also clean up children if startup failed before the gateway registered
		// its signal handler, or if a regression broke graceful shutdown.
		for _, pid := range gateway.childPIDs() {
			if p, e := os.FindProcess(pid); e == nil {
				_ = p.Kill()
			}
		}
		if strings.Contains(logs.String(), "WARNING: DATA RACE") {
			t.Error("race detected in subprocess")
		}
		if t.Failed() {
			t.Log(logs.String())
		}
	})
	deadline := time.Now().Add(10 * time.Second)
	for {
		response, e := gateway.client.Get(gateway.url + "/")
		if e == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return gateway
			}
		}
		select {
		case <-exited:
			if strings.Contains(logs.String(), "address already in use") {
				return nil
			}
			t.Fatal("gateway exited during startup")
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("gateway did not start")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func (g *processGateway) childPIDs() []int {
	data, _ := os.ReadFile(g.pidPath)
	var pids []int
	for _, line := range strings.Fields(string(data)) {
		if pid, e := strconv.Atoi(line); e == nil {
			pids = append(pids, pid)
		}
	}
	return pids
}

func (g *processGateway) rpc(method string, id interface{}, params interface{}) (map[string]interface{}, error) {
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
	response, e := g.client.Post(g.url+"/mcp", "application/json", bytes.NewReader(body))
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

// handshake initializes a client and returns the serving child's PID.
func (g *processGateway) handshake() (string, error) {
	result, e := g.rpc("initialize", "restart-client", map[string]interface{}{"protocolVersion": "2024-11-05", "capabilities": map[string]interface{}{}, "clientInfo": map[string]interface{}{"name": "restart-test", "version": "1"}})
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

func (g *processGateway) checkTools(pid string) {
	g.t.Helper()
	if _, e := g.rpc("notifications/initialized", nil, nil); e != nil {
		g.t.Fatal(e)
	}
	result, e := g.rpc("tools/list", 2, map[string]interface{}{})
	if e != nil {
		g.t.Fatal(e)
	}
	raw, _ := json.Marshal(result["result"])
	var catalog struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if e = json.Unmarshal(raw, &catalog); e != nil {
		g.t.Fatal(e)
	}
	if len(catalog.Tools) != 1 || catalog.Tools[0].Name != "pid_"+pid {
		g.t.Fatalf("wrong process tools: %s", raw)
	}
}

// waitForRestart waits for the actual replacement PID, rather than replaying
// requests during the intentional restart backoff.
func (g *processGateway) waitForRestart() {
	g.t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		if len(g.childPIDs()) >= 2 && strings.Contains(g.logs.String(), "MCP server restarted successfully") {
			return
		}
		if time.Now().After(deadline) {
			g.t.Fatal("child was not restarted")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestGatewayReinitializesRestartedProcess(t *testing.T) {
	for _, skip := range []bool{false, true} {
		t.Run(fmt.Sprintf("skip_readiness_%t", skip), func(t *testing.T) {
			gateway := startProcessGateway(t, skip)
			first, err := gateway.handshake()
			if err != nil || first == "" {
				t.Fatalf("initial handshake: pid=%q err=%v", first, err)
			}
			gateway.checkTools(first)
			if _, err = gateway.rpc("test/exit", nil, nil); err != nil {
				t.Fatal(err)
			}
			gateway.waitForRestart()
			second, err := gateway.handshake()
			if err != nil {
				t.Fatal(err)
			}
			if second == "" || second == first {
				t.Fatalf("cached dead child initialization: before=%q after=%q", first, second)
			}
			gateway.checkTools(second)
		})
	}
}

func TestGatewayOversizedResponseFailsOnlyThatRequest(t *testing.T) {
	gateway := startProcessGateway(t, false)
	pid, err := gateway.handshake()
	if err != nil {
		t.Fatal(err)
	}
	_, err = gateway.rpc("test/oversized-response", 7, nil)
	if err == nil || !strings.Contains(err.Error(), "exceeded the gateway limit") {
		t.Fatalf("expected an oversized-response error, got %v", err)
	}
	gateway.checkTools(pid)
}

func TestGatewayKeepsServingAfterOversizedStderrLine(t *testing.T) {
	gateway := startProcessGateway(t, false)
	pid, err := gateway.handshake()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = gateway.rpc("test/oversized-stderr", 7, nil); err != nil {
		t.Fatal(err)
	}
	gateway.checkTools(pid)
}

func TestGatewayRestartsChildThatClosesStdout(t *testing.T) {
	gateway := startProcessGateway(t, false)
	first, err := gateway.handshake()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = gateway.rpc("test/close-stdout", nil, nil); err != nil {
		t.Fatal(err)
	}
	gateway.waitForRestart()
	second, err := gateway.handshake()
	if err != nil {
		t.Fatal(err)
	}
	if second == first {
		t.Fatalf("still served by the child with closed stdout: pid=%q", first)
	}
	gateway.checkTools(second)
}
