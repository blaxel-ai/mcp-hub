package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	testIssuer   = "https://signin.example.com"
	testResource = "https://mcp-tools-abc123.us-pdx-1.bl.run"
)

// controlPlane fakes the access-check endpoint: it answers with the status
// registered for the presented credential and counts the calls.
type controlPlane struct {
	*httptest.Server
	calls     atomic.Int32
	workspace atomic.Value
}

func newControlPlane(t *testing.T, statusByToken map[string]int) *controlPlane {
	t.Helper()
	cp := &controlPlane{}
	cp.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cp.calls.Add(1)
		cp.workspace.Store(r.Header.Get("X-Blaxel-Workspace"))
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		status, ok := statusByToken[token]
		if !ok {
			status = http.StatusUnauthorized
		}
		w.WriteHeader(status)
	}))
	t.Cleanup(cp.Close)
	return cp
}

func newOAuth(t *testing.T, accessURL string) *oauthConfig {
	t.Helper()
	cfg, err := loadOAuthConfig(func(name string) string {
		return map[string]string{
			"BL_OAUTH_ISSUER":       testIssuer,
			"BL_OAUTH_RESOURCE_URL": testResource + "/",
			"BL_OAUTH_ACCESS_URL":   accessURL,
			"BL_WORKSPACE":          "acme",
		}[name]
	})
	if err != nil || cfg == nil {
		t.Fatalf("loadOAuthConfig: %v, %v", cfg, err)
	}
	return cfg
}

// serve runs one request through the wrapped handler and reports whether
// the MCP handler behind it was reached, and with which credential headers.
func serve(cfg *oauthConfig, method, path string, headers map[string]string) (*httptest.ResponseRecorder, bool, http.Header) {
	reached := false
	var seen http.Header
	handler := cfg.wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		seen = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(method, path, nil)
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec, reached, seen
}

func jwtExpiringAt(expiry time.Time) string {
	payload, _ := json.Marshal(map[string]any{"exp": expiry.Unix()})
	return "h." + base64.RawURLEncoding.EncodeToString(payload) + ".s"
}

func TestOAuthIsOffWithoutIssuerAndRefusesToStartHalfConfigured(t *testing.T) {
	cfg, err := loadOAuthConfig(func(string) string { return "" })
	if cfg != nil || err != nil {
		t.Fatalf("expected OAuth off, got %v, %v", cfg, err)
	}
	for _, missing := range []string{"BL_OAUTH_RESOURCE_URL", "BL_OAUTH_ACCESS_URL", "BL_WORKSPACE"} {
		_, err := loadOAuthConfig(func(name string) string {
			if name == missing {
				return ""
			}
			return "set"
		})
		if err == nil || !strings.Contains(err.Error(), missing) {
			t.Errorf("missing %s: expected an error naming it, got %v", missing, err)
		}
	}
}

func TestOAuthMetadataDescribesTheRequestedResource(t *testing.T) {
	cp := newControlPlane(t, nil)
	cfg := newOAuth(t, cp.URL)
	for path, resource := range map[string]string{
		"/.well-known/oauth-protected-resource/mcp": testResource + "/mcp",
		"/.well-known/oauth-protected-resource":     testResource,
	} {
		rec, reached, _ := serve(cfg, http.MethodGet, path, nil)
		if rec.Code != http.StatusOK || reached {
			t.Fatalf("%s: status %d, reached MCP handler: %v", path, rec.Code, reached)
		}
		var doc struct {
			Resource             string   `json:"resource"`
			AuthorizationServers []string `json:"authorization_servers"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
			t.Fatal(err)
		}
		if doc.Resource != resource || len(doc.AuthorizationServers) != 1 || doc.AuthorizationServers[0] != testIssuer {
			t.Errorf("%s: got %+v", path, doc)
		}
	}
	if cp.calls.Load() != 0 {
		t.Errorf("metadata must not call the control plane, got %d calls", cp.calls.Load())
	}
	// A look-alike path is not the metadata document.
	if rec, _, _ := serve(cfg, http.MethodGet, "/.well-known/oauth-protected-resources", nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("look-alike path: status %d", rec.Code)
	}
}

func TestOAuthChallengesWithoutACredentialAndNeverReachesTheServer(t *testing.T) {
	cp := newControlPlane(t, nil)
	cfg := newOAuth(t, cp.URL)

	rec, reached, _ := serve(cfg, http.MethodPost, "/mcp", nil)
	want := fmt.Sprintf(`Bearer resource_metadata="%s/.well-known/oauth-protected-resource/mcp"`, testResource)
	if rec.Code != http.StatusUnauthorized || reached || rec.Header().Get("WWW-Authenticate") != want {
		t.Fatalf("status %d, reached %v, challenge %q", rec.Code, reached, rec.Header().Get("WWW-Authenticate"))
	}
	if cp.calls.Load() != 0 {
		t.Errorf("no credential must not call the control plane")
	}

	rec, reached, _ = serve(cfg, http.MethodPost, "/mcp", map[string]string{"Authorization": "Bearer nope"})
	want = fmt.Sprintf(`Bearer error="invalid_token", resource_metadata="%s/.well-known/oauth-protected-resource/mcp"`, testResource)
	if rec.Code != http.StatusUnauthorized || reached || rec.Header().Get("WWW-Authenticate") != want {
		t.Fatalf("invalid credential: status %d, reached %v, challenge %q", rec.Code, reached, rec.Header().Get("WWW-Authenticate"))
	}

	// A browser client must be able to preflight, then read the challenge.
	preflight, reached, _ := serve(cfg, http.MethodOptions, "/mcp", nil)
	if preflight.Code != http.StatusNoContent || reached ||
		preflight.Header().Get("Access-Control-Allow-Origin") != "*" ||
		!strings.Contains(preflight.Header().Get("Access-Control-Allow-Headers"), "Authorization") {
		t.Errorf("preflight: status %d, reached %v, headers %v", preflight.Code, reached, preflight.Header())
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" ||
		!strings.Contains(rec.Header().Get("Access-Control-Expose-Headers"), "WWW-Authenticate") {
		t.Errorf("401 is not readable cross-origin: %v", rec.Header())
	}
}

func TestOAuthLetsAllowedCredentialsThroughAndDropsThem(t *testing.T) {
	cp := newControlPlane(t, map[string]int{
		"oauth-token": http.StatusNoContent,
		"bl_apikey":   http.StatusNoContent,
		"outsider":    http.StatusForbidden,
	})
	cfg := newOAuth(t, cp.URL)

	for _, headers := range []map[string]string{
		{"Authorization": "Bearer oauth-token"},
		{"X-Blaxel-Authorization": "Bearer bl_apikey"},
		{"Authorization": "bearer oauth-token"},
	} {
		rec, reached, seen := serve(cfg, http.MethodPost, "/mcp", headers)
		if rec.Code != http.StatusOK || !reached {
			t.Fatalf("%v: status %d, reached %v", headers, rec.Code, reached)
		}
		if seen.Get("Authorization") != "" || seen.Get("X-Blaxel-Authorization") != "" {
			t.Errorf("%v: credential reached the MCP handler: %v", headers, seen)
		}
	}
	if got := cp.workspace.Load(); got != "acme" {
		t.Errorf("access check sent workspace %q", got)
	}

	rec, reached, _ := serve(cfg, http.MethodPost, "/mcp", map[string]string{"Authorization": "Bearer outsider"})
	if rec.Code != http.StatusForbidden || reached || rec.Header().Get("WWW-Authenticate") != "" {
		t.Errorf("non member: status %d, reached %v, challenge %q", rec.Code, reached, rec.Header().Get("WWW-Authenticate"))
	}
}

func TestOAuthCachesAnswersButNeverPastTheTokenExpiry(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	// Expires 20 seconds after the moment the test first presents it.
	shortLived := jwtExpiringAt(now.Add(oauthAllowTTL + 21*time.Second))
	cp := newControlPlane(t, map[string]int{"opaque": http.StatusNoContent, shortLived: http.StatusNoContent})
	cfg := newOAuth(t, cp.URL)
	cfg.now = func() time.Time { return now }

	call := func(token string) int {
		rec, _, _ := serve(cfg, http.MethodPost, "/mcp", map[string]string{"Authorization": "Bearer " + token})
		return rec.Code
	}
	expectCalls := func(want int32, step string) {
		t.Helper()
		if got := cp.calls.Load(); got != want {
			t.Fatalf("%s: %d control plane calls, want %d", step, got, want)
		}
	}

	call("opaque")
	call("opaque")
	expectCalls(1, "allowed answer is reused")
	now = now.Add(oauthAllowTTL + time.Second)
	call("opaque")
	expectCalls(2, "allowed answer expires")

	call(shortLived)
	now = now.Add(19 * time.Second)
	call(shortLived)
	expectCalls(3, "token answer reused before its expiry")
	now = now.Add(2 * time.Second)
	call(shortLived)
	expectCalls(4, "token answer dropped at its expiry")

	call("denied")
	call("denied")
	expectCalls(5, "denied answer is reused")
	now = now.Add(oauthDenyTTL + time.Second)
	call("denied")
	expectCalls(6, "denied answer expires quickly")
}

// Only the access endpoint's own 204 lets a caller in: a redirect to a page
// that answers 200, or any other success status, does not.
func TestOAuthOnlyTrustsTheAccessEndpointsOwnAnswer(t *testing.T) {
	login := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer login.Close()
	redirecting := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, login.URL, http.StatusFound)
	}))
	defer redirecting.Close()

	for name, accessURL := range map[string]string{"redirect to a 200 page": redirecting.URL, "plain 200": login.URL} {
		cfg := newOAuth(t, accessURL)
		rec, reached, _ := serve(cfg, http.MethodPost, "/mcp", map[string]string{"Authorization": "Bearer anything"})
		if rec.Code != http.StatusServiceUnavailable || reached {
			t.Errorf("%s: status %d, reached %v", name, rec.Code, reached)
		}
	}
}

func TestOAuthFailsClosedWhenTheControlPlaneIsUnavailable(t *testing.T) {
	cp := newControlPlane(t, map[string]int{"token": http.StatusInternalServerError})
	cfg := newOAuth(t, cp.URL)

	for range 2 {
		rec, reached, _ := serve(cfg, http.MethodPost, "/mcp", map[string]string{"Authorization": "Bearer token"})
		if rec.Code != http.StatusServiceUnavailable || reached {
			t.Fatalf("status %d, reached %v", rec.Code, reached)
		}
	}
	if cp.calls.Load() != 2 {
		t.Errorf("an unknown answer must not be cached, got %d calls", cp.calls.Load())
	}

	cp.Close()
	rec, reached, _ := serve(cfg, http.MethodPost, "/mcp", map[string]string{"Authorization": "Bearer token"})
	if rec.Code != http.StatusServiceUnavailable || reached {
		t.Fatalf("unreachable control plane: status %d, reached %v", rec.Code, reached)
	}
}

// A stream opened with a good credential must not outlive it: once the
// control plane stops accepting the credential, open requests are cancelled
// and WebSocket handlers (which return while the socket lives on) are told.
func TestOAuthEndsOpenConnectionsOnceTheCredentialIsRefused(t *testing.T) {
	var revoked atomic.Bool
	cp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if revoked.Load() {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer cp.Close()
	cfg := newOAuth(t, cp.URL)
	cfg.recheckEvery = 5 * time.Millisecond
	// Every look at the clock is past the previous answer's lifetime.
	var clock atomic.Int64
	cfg.now = func() time.Time { return time.Unix(1_800_000_000+clock.Add(int64(2*oauthAllowTTL/time.Second)), 0) }

	streamEnded := make(chan struct{})
	socketClosed := make(chan struct{})
	handler := cfg.wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ws" {
			// Like HandleWebSocket: hand the watch to the socket and return.
			oauthWatch(r, func() { close(socketClosed) })
			return
		}
		<-r.Context().Done()
		close(streamEnded)
	}))
	request := func(path string) *http.Request {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer good")
		return req
	}
	go handler.ServeHTTP(httptest.NewRecorder(), request("/mcp"))
	handler.ServeHTTP(httptest.NewRecorder(), request("/ws"))

	select {
	case <-streamEnded:
		t.Fatal("stream ended while the credential was still accepted")
	case <-socketClosed:
		t.Fatal("socket closed while the credential was still accepted")
	case <-time.After(50 * time.Millisecond):
	}

	revoked.Store(true)
	for name, ended := range map[string]chan struct{}{"stream": streamEnded, "socket": socketClosed} {
		select {
		case <-ended:
		case <-time.After(2 * time.Second):
			t.Fatalf("%s still open after the credential was refused", name)
		}
	}
}

// Without OAuth sign-in there is nothing to watch.
func TestOAuthWatchIsANoOpWhenSignInIsOff(t *testing.T) {
	stop := oauthWatch(httptest.NewRequest(http.MethodGet, "/ws", nil), func() { t.Fatal("revoked") })
	stop()
}

// Made-up tokens are free to send: they must neither evict a real user's
// answer nor turn into an unbounded number of calls to the control plane.
func TestOAuthMadeUpTokensNeitherEvictUsersNorFloodTheControlPlane(t *testing.T) {
	t.Run("refused answers never evict an allowed one", func(t *testing.T) {
		cp := newControlPlane(t, map[string]int{"good": http.StatusNoContent})
		cfg := newOAuth(t, cp.URL)
		if verdict, err := cfg.authorize(context.Background(), "good"); err != nil || verdict != oauthAllowed {
			t.Fatalf("good: %v, %v", verdict, err)
		}
		for i := 0; i <= oauthMaxDecisions; i++ {
			if _, err := cfg.authorize(context.Background(), fmt.Sprintf("junk-%d", i)); err != nil {
				t.Fatalf("junk %d: %v", i, err)
			}
		}
		before := cp.calls.Load()
		if verdict, err := cfg.authorize(context.Background(), "good"); err != nil || verdict != oauthAllowed {
			t.Fatalf("good after the flood: %v, %v", verdict, err)
		}
		if cp.calls.Load() != before {
			t.Fatal("the allowed answer was evicted by refused ones")
		}
		if len(cfg.refused) > oauthMaxDecisions {
			t.Fatalf("refused answers are not bounded: %d", len(cfg.refused))
		}
	})

	t.Run("calls in flight are capped and the overflow fails closed", func(t *testing.T) {
		release := make(chan struct{})
		var inFlight atomic.Int32
		cp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			inFlight.Add(1)
			<-release
			w.WriteHeader(http.StatusNoContent)
		}))
		defer cp.Close()
		cfg := newOAuth(t, cp.URL)

		var wg sync.WaitGroup
		for i := 0; i < oauthMaxChecks; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = cfg.authorize(context.Background(), fmt.Sprintf("slow-%d", i))
			}()
		}
		deadline := time.Now().Add(5 * time.Second)
		for inFlight.Load() < oauthMaxChecks {
			if time.Now().After(deadline) {
				t.Fatalf("only %d calls reached the control plane", inFlight.Load())
			}
			time.Sleep(time.Millisecond)
		}

		rec, reached, _ := serve(cfg, http.MethodPost, "/mcp", map[string]string{"Authorization": "Bearer one-too-many"})
		if rec.Code != http.StatusServiceUnavailable || reached {
			t.Fatalf("overflow: status %d, reached %v, want 503 and not reached", rec.Code, reached)
		}
		if got := inFlight.Load(); got != oauthMaxChecks {
			t.Fatalf("%d calls in flight, want %d", got, oauthMaxChecks)
		}
		close(release)
		wg.Wait()
	})
}

// The real gateway process, started with the OAuth settings: nothing reaches
// the MCP child without a credential the control plane accepts.
func TestGatewayProcessRequiresSignInWhenOAuthIsOn(t *testing.T) {
	cp := newControlPlane(t, map[string]int{"member": http.StatusNoContent})
	t.Setenv("BL_OAUTH_ISSUER", testIssuer)
	t.Setenv("BL_OAUTH_RESOURCE_URL", testResource)
	t.Setenv("BL_OAUTH_ACCESS_URL", cp.URL)
	t.Setenv("BL_WORKSPACE", "acme")
	gateway := startProcessGateway(t, false)

	post := func(token string) *http.Response {
		t.Helper()
		body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"oauth-test","version":"1"}}}`
		req, _ := http.NewRequest(http.MethodPost, gateway.url+"/mcp", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := gateway.client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = resp.Body.Close() })
		return resp
	}

	if resp := post(""); resp.StatusCode != http.StatusUnauthorized ||
		!strings.Contains(resp.Header.Get("WWW-Authenticate"), testResource+oauthMetadataPath+"/mcp") {
		t.Fatalf("no credential: status %d, challenge %q", resp.StatusCode, resp.Header.Get("WWW-Authenticate"))
	}
	if resp := post("stranger"); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unknown credential: status %d", resp.StatusCode)
	}
	resp := post("member")
	var result struct {
		Result map[string]any `json:"result"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&result) != nil || result.Result == nil {
		t.Fatalf("member: status %d, result %v", resp.StatusCode, result.Result)
	}

	metadata, err := gateway.client.Get(gateway.url + oauthMetadataPath + "/mcp")
	if err != nil {
		t.Fatal(err)
	}
	defer metadata.Body.Close()
	if metadata.StatusCode != http.StatusOK {
		t.Fatalf("metadata: status %d", metadata.StatusCode)
	}
}
