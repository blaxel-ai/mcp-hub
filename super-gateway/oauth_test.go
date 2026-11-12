package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
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
	resource  atomic.Value
}

func newControlPlane(t *testing.T, statusByToken map[string]int) *controlPlane {
	t.Helper()
	cp := &controlPlane{}
	cp.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cp.calls.Add(1)
		cp.workspace.Store(r.Header.Get("X-Blaxel-Workspace"))
		cp.resource.Store(r.Header.Get("X-Blaxel-Resource"))
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
	if got := cp.resource.Load(); got != testResource {
		t.Errorf("access check named server %q, want %q", got, testResource)
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
	now := time.Unix(1_800_000_000, 0)
	cfg.now = func() time.Time { return now }
	var logs bytes.Buffer
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	for range 2 {
		rec, reached, _ := serve(cfg, http.MethodPost, "/mcp", map[string]string{"Authorization": "Bearer token"})
		if rec.Code != http.StatusServiceUnavailable || reached {
			t.Fatalf("status %d, reached %v", rec.Code, reached)
		}
	}
	if cp.calls.Load() != 2 {
		t.Errorf("an unknown answer must not be cached, got %d calls", cp.calls.Load())
	}
	// The caller only sees a 503: the reason is logged, once per second and
	// without the credential.
	if got := strings.Count(logs.String(), "access check failed"); got != 1 {
		t.Errorf("%d log lines for two failures in the same second, want 1: %s", got, logs.String())
	}
	if !strings.Contains(logs.String(), "500") || strings.Contains(logs.String(), "token") {
		t.Errorf("log must give the reason and never the credential: %s", logs.String())
	}
	now = now.Add(time.Second)

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

// openStream runs a request that stays open behind the sign-in check, waits
// until the check let it through, and reports when the check ends it.
func openStream(t *testing.T, cfg *oauthConfig, token string) (ended chan struct{}) {
	t.Helper()
	started := make(chan struct{})
	ended = make(chan struct{})
	handler := cfg.wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
		close(ended)
	}))
	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	go handler.ServeHTTP(httptest.NewRecorder(), req)
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("the request was not let through")
	}
	return ended
}

func stillOpen(ended chan struct{}, during time.Duration) bool {
	select {
	case <-ended:
		return false
	case <-time.After(during):
		return true
	}
}

// An answer the control plane could not give (outage, too many checks in
// flight) is not a refusal: the connection survives a blip, and is only cut
// once the answer has been unknown for a whole period.
func TestOAuthUnknownAnswerDoesNotCutAnOpenConnectionAtOnce(t *testing.T) {
	var failing atomic.Bool
	var failures atomic.Int32
	cp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if failing.Load() {
			failures.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer cp.Close()
	waitForFailure := func() {
		t.Helper()
		seen := failures.Load()
		for deadline := time.Now().Add(5 * time.Second); failures.Load() == seen; {
			if time.Now().After(deadline) {
				t.Fatal("the open connection was never checked again")
			}
			time.Sleep(time.Millisecond)
		}
	}
	const period = 400 * time.Millisecond

	t.Run("a blip is survived", func(t *testing.T) {
		cfg := newOAuth(t, cp.URL)
		cfg.recheckEvery = period
		failing.Store(false)
		ended := openStream(t, cfg, "blip")
		failing.Store(true)
		waitForFailure()
		failing.Store(false)
		if !stillOpen(ended, 3*period) {
			t.Fatal("connection cut although the control plane never refused the credential")
		}
	})

	t.Run("a lasting outage fails closed", func(t *testing.T) {
		cfg := newOAuth(t, cp.URL)
		cfg.recheckEvery = period
		failing.Store(false)
		ended := openStream(t, cfg, "outage")
		failing.Store(true)
		waitForFailure()
		if !stillOpen(ended, period/4) {
			t.Fatal("connection cut on the first unknown answer")
		}
		if stillOpen(ended, 5*period) {
			t.Fatal("connection still open after the answer stayed unknown for more than a period")
		}
	})
}

// The answer held for a token lapses when the token expires. An open
// connection is checked again then, not a full period later.
func TestOAuthOpenConnectionIsCheckedAgainWhenTheTokenExpires(t *testing.T) {
	expiry := time.Now().Add(2 * time.Second).Truncate(time.Second)
	token := jwtExpiringAt(expiry)
	cp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if time.Now().Before(expiry) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer cp.Close()
	cfg := newOAuth(t, cp.URL)
	cfg.recheckEvery = time.Minute

	ended := openStream(t, cfg, token)
	if !stillOpen(ended, time.Until(expiry)-200*time.Millisecond) {
		t.Fatal("connection cut before the token expired")
	}
	if stillOpen(ended, 3*time.Second) {
		t.Fatal("connection outlived its token by seconds: the check waited for the fixed period")
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

	t.Run("calls in flight are capped, and strangers cannot starve users who are in", func(t *testing.T) {
		release := make(chan struct{})
		var inFlight atomic.Int32
		var blocking atomic.Bool
		cp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if blocking.Load() && strings.HasPrefix(r.Header.Get("Authorization"), "Bearer stranger-") {
				inFlight.Add(1)
				<-release
			}
			w.WriteHeader(http.StatusNoContent)
		}))
		defer cp.Close()
		defer close(release)
		cfg := newOAuth(t, cp.URL)
		cfg.slotWait = 50 * time.Millisecond
		now := time.Unix(1_800_000_000, 0)
		var clock sync.Mutex
		cfg.now = func() time.Time { clock.Lock(); defer clock.Unlock(); return now }

		// A user who is in, then whose answer lapses.
		if verdict, err := cfg.authorize(context.Background(), "member"); err != nil || verdict != oauthAllowed {
			t.Fatalf("member: %v, %v", verdict, err)
		}
		clock.Lock()
		now = now.Add(cfg.recheckEvery + time.Second)
		clock.Unlock()

		// Strangers fill every slot open to them.
		blocking.Store(true)
		strangers := cap(cfg.checks)
		for i := 0; i < strangers; i++ {
			go func() { _, _ = cfg.authorize(context.Background(), fmt.Sprintf("stranger-%d", i)) }()
		}
		deadline := time.Now().Add(5 * time.Second)
		for inFlight.Load() < int32(strangers) {
			if time.Now().After(deadline) {
				t.Fatalf("only %d calls reached the control plane", inFlight.Load())
			}
			time.Sleep(time.Millisecond)
		}

		// One more stranger waits for a slot, then fails closed.
		rec, reached, _ := serve(cfg, http.MethodPost, "/mcp", map[string]string{"Authorization": "Bearer stranger-one-too-many"})
		if rec.Code != http.StatusServiceUnavailable || reached {
			t.Fatalf("overflow: status %d, reached %v, want 503 and not reached", rec.Code, reached)
		}
		if got := inFlight.Load(); got != int32(strangers) {
			t.Fatalf("%d calls in flight, want %d", got, strangers)
		}
		// The member is checked again all the same: their slot is reserved.
		rec, reached, _ = serve(cfg, http.MethodPost, "/mcp", map[string]string{"Authorization": "Bearer member"})
		if rec.Code != http.StatusOK || !reached {
			t.Fatalf("member during the flood: status %d, reached %v", rec.Code, reached)
		}
	})
}

// The control plane cannot always answer. A user who was allowed stays in
// for one more period, never past their token's expiry; a stranger does not
// get in on an unknown answer.
func TestOAuthKeepsARecentAllowedAnswerThroughAnOutage(t *testing.T) {
	var failing atomic.Bool
	cp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if failing.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer cp.Close()
	cfg := newOAuth(t, cp.URL)
	now := time.Unix(1_800_000_000, 0)
	cfg.now = func() time.Time { return now }
	expiring := jwtExpiringAt(now.Add(cfg.recheckEvery + 20*time.Second))
	status := func(token string) int {
		rec, _, _ := serve(cfg, http.MethodPost, "/mcp", map[string]string{"Authorization": "Bearer " + token})
		return rec.Code
	}

	for _, token := range []string{"member", expiring} {
		if got := status(token); got != http.StatusOK {
			t.Fatalf("before the outage: %d", got)
		}
	}
	failing.Store(true)
	now = now.Add(cfg.recheckEvery + time.Second)
	if got := status("member"); got != http.StatusOK {
		t.Fatalf("a user who was in, during the outage: %d, want 200", got)
	}
	if got := status(expiring); got != http.StatusOK {
		t.Fatalf("a token still valid, during the outage: %d, want 200", got)
	}
	if got := status("stranger"); got != http.StatusServiceUnavailable {
		t.Fatalf("a stranger, during the outage: %d, want 503", got)
	}
	now = now.Add(20 * time.Second)
	if got := status(expiring); got != http.StatusServiceUnavailable {
		t.Fatalf("an expired token must not ride the outage: %d, want 503", got)
	}
	now = now.Add(cfg.recheckEvery)
	if got := status("member"); got != http.StatusServiceUnavailable {
		t.Fatalf("the answer has been unknown for more than a period: %d, want 503", got)
	}
}

// Two checks of one credential can finish out of order. The answer of the
// check that started last wins: a late "allowed" must not cover a refusal.
func TestOAuthLateAllowedAnswerDoesNotCoverANewerRefusal(t *testing.T) {
	releaseFirst := make(chan struct{})
	firstStarted := make(chan struct{})
	var calls atomic.Int32
	cp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			close(firstStarted)
			<-releaseFirst
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer cp.Close()
	cfg := newOAuth(t, cp.URL)
	var tick atomic.Int64
	cfg.now = func() time.Time { return time.Unix(1_800_000_000, tick.Add(1)) }

	first := make(chan oauthVerdict)
	go func() {
		verdict, _ := cfg.authorize(context.Background(), "revoked-meanwhile")
		first <- verdict
	}()
	<-firstStarted
	if verdict, err := cfg.authorize(context.Background(), "revoked-meanwhile"); err != nil || verdict != oauthUnauthenticated {
		t.Fatalf("second check: %v, %v", verdict, err)
	}
	close(releaseFirst)
	<-first

	before := calls.Load()
	if verdict, err := cfg.authorize(context.Background(), "revoked-meanwhile"); err != nil || verdict != oauthUnauthenticated {
		t.Fatalf("after the late answer: %v, %v, want the refusal to stand", verdict, err)
	}
	if calls.Load() != before {
		t.Fatal("the refusal was dropped by the late answer")
	}
}

// One setting without the others means the platform made this server
// public: that must stop the gateway, not start it without authentication.
func TestOAuthRefusesAHalfSetOrUnsafeConfiguration(t *testing.T) {
	full := map[string]string{
		"BL_OAUTH_ISSUER": testIssuer, "BL_OAUTH_RESOURCE_URL": testResource,
		"BL_OAUTH_ACCESS_URL": "https://api.example.com/v0/mcp/access", "BL_WORKSPACE": "acme",
	}
	load := func(change func(map[string]string)) (*oauthConfig, error) {
		env := map[string]string{}
		for k, v := range full {
			env[k] = v
		}
		change(env)
		return loadOAuthConfig(func(name string) string { return env[name] })
	}
	if cfg, err := load(func(map[string]string) {}); err != nil || cfg == nil {
		t.Fatalf("complete configuration: %v, %v", cfg, err)
	}
	for name, change := range map[string]func(map[string]string){
		"issuer missing":       func(env map[string]string) { delete(env, "BL_OAUTH_ISSUER") },
		"issuer blank":         func(env map[string]string) { env["BL_OAUTH_ISSUER"] = "  " },
		"only the access URL":  func(env map[string]string) { delete(env, "BL_OAUTH_ISSUER"); delete(env, "BL_OAUTH_RESOURCE_URL") },
		"access URL over http": func(env map[string]string) { env["BL_OAUTH_ACCESS_URL"] = "http://api.example.com/v0/mcp/access" },
		"access URL not a URL": func(env map[string]string) { env["BL_OAUTH_ACCESS_URL"] = "api.example.com" },
	} {
		if cfg, err := load(change); err == nil {
			t.Errorf("%s: started with %v, want an error", name, cfg)
		}
	}
}

// A bearer far longer than any credential is refused without asking anyone.
func TestOAuthRefusesAnOversizedTokenLocally(t *testing.T) {
	cp := newControlPlane(t, nil)
	cfg := newOAuth(t, cp.URL)
	rec, reached, _ := serve(cfg, http.MethodPost, "/mcp", map[string]string{"Authorization": "Bearer " + strings.Repeat("a", oauthMaxTokenLength+1)})
	if rec.Code != http.StatusUnauthorized || reached || cp.calls.Load() != 0 {
		t.Fatalf("status %d, reached %v, %d control plane calls", rec.Code, reached, cp.calls.Load())
	}

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
