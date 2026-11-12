package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// OAuth sign-in with Blaxel (opt-in per MCP server).
//
// When the control plane enables it, the workload is public at the platform
// gateway and this process is the only authentication in front of the MCP
// server. It then:
//   - tells clients where to sign in: the protected resource metadata
//     (RFC 9728) and a WWW-Authenticate challenge on 401;
//   - asks the control plane whether the caller's token may call this
//     server, and caches the answer. The only token accepted is the one the
//     control plane issued for this server when the client signed in: this
//     process runs next to code the workspace controls, so it must never be
//     handed a credential that works anywhere else.
//
// Nothing here verifies a signature: the control plane owns the keys, the
// user records and revocation, so it is the one place that decides.

const (
	oauthMetadataPath = "/.well-known/oauth-protected-resource"

	// How long an answer from the control plane is reused. Allowed answers
	// never outlive the token itself; denied ones are kept short so a user
	// who was just granted access is not locked out.
	oauthAllowTTL = 60 * time.Second
	oauthDenyTTL  = 10 * time.Second

	// ponytail: bounded maps, emptied when full. Move to an LRU if a single
	// server ever sees more distinct live credentials than this.
	oauthMaxDecisions = 10000

	// Calls to the control plane in flight at once. Anyone can send a made-up
	// token, so the number of outbound calls must not follow the inbound rate.
	// Half of them are kept for credentials that were allowed before, so a
	// flood of made-up tokens cannot keep real users from being re-checked.
	oauthMaxChecks = 32

	// How long a request waits for one of those calls before giving up.
	oauthSlotWait = 2 * time.Second

	// No credential this server accepts is anywhere near this long.
	oauthMaxTokenLength = 8192
)

type oauthVerdict int

const (
	oauthAllowed oauthVerdict = iota
	oauthUnauthenticated
	oauthForbidden
)

type oauthDecision struct {
	verdict oauthVerdict
	// When the check that produced the answer started, and until when the
	// answer is good.
	at, until time.Time
}

type oauthConfig struct {
	issuer      string // authorization server clients sign in to
	resourceURL string // public base URL of this MCP server, no trailing slash
	accessURL   string // control plane endpoint answering "may this credential access the workspace"
	workspace   string

	client *http.Client
	now    func() time.Time
	// How long an allowed answer is good for, and so how often a request or
	// connection that stays open is checked again.
	recheckEvery time.Duration
	// Slots for calls to the control plane: one pool for credentials never
	// allowed before, one for those that were.
	checks, checksKnown chan struct{}
	slotWait            time.Duration
	// Unix second of the last logged failure.
	lastFailureLog atomic.Int64

	mu sync.Mutex
	// Allowed and refused answers live apart: a flood of made-up tokens fills
	// and empties the refused map only, and never evicts a real user.
	allowed map[[sha256.Size]byte]oauthDecision
	refused map[[sha256.Size]byte]oauthDecision
}

// loadOAuthConfig returns nil when OAuth sign-in is off. Once it is on, a
// missing setting is fatal: the gateway in front lets everyone through, so
// starting without authentication would expose the server.
func loadOAuthConfig(getenv func(string) string) (*oauthConfig, error) {
	issuer := strings.TrimSpace(getenv("BL_OAUTH_ISSUER"))
	// Any one of the three settings means the platform made this server
	// public: an empty issuer next to the others must not read as "off".
	if issuer == "" && getenv("BL_OAUTH_RESOURCE_URL") == "" && getenv("BL_OAUTH_ACCESS_URL") == "" {
		return nil, nil
	}
	cfg := &oauthConfig{
		issuer:      issuer,
		resourceURL: strings.TrimRight(getenv("BL_OAUTH_RESOURCE_URL"), "/"),
		accessURL:   getenv("BL_OAUTH_ACCESS_URL"),
		workspace:   getenv("BL_WORKSPACE"),
		client: &http.Client{
			Timeout: 5 * time.Second,
			// No proxy from the environment: the workspace sets the container's
			// variables, and callers' credentials travel on this connection.
			Transport: &http.Transport{
				MaxConnsPerHost:     oauthMaxChecks,
				MaxIdleConnsPerHost: oauthMaxChecks,
				IdleConnTimeout:     90 * time.Second,
			},
			// The answer is the status of the access endpoint itself: a
			// redirect (to a login page, say) must never read as "allowed".
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
		now:          time.Now,
		recheckEvery: oauthAllowTTL,
		checks:       make(chan struct{}, oauthMaxChecks/2),
		checksKnown:  make(chan struct{}, oauthMaxChecks/2),
		slotWait:     oauthSlotWait,
		allowed:      make(map[[sha256.Size]byte]oauthDecision),
		refused:      make(map[[sha256.Size]byte]oauthDecision),
	}
	for name, value := range map[string]string{
		"BL_OAUTH_ISSUER":       cfg.issuer,
		"BL_OAUTH_RESOURCE_URL": cfg.resourceURL,
		"BL_OAUTH_ACCESS_URL":   cfg.accessURL,
		"BL_WORKSPACE":          cfg.workspace,
	} {
		if value == "" {
			return nil, fmt.Errorf("OAuth sign-in is configured but %s is missing", name)
		}
	}
	// Callers' credentials travel to this URL.
	access, err := url.Parse(cfg.accessURL)
	if err != nil || access.Host == "" {
		return nil, fmt.Errorf("BL_OAUTH_ACCESS_URL is not a URL")
	}
	if host := access.Hostname(); access.Scheme != "https" && host != "localhost" && !net.ParseIP(host).IsLoopback() {
		return nil, fmt.Errorf("BL_OAUTH_ACCESS_URL must use https")
	}
	return cfg, nil
}

// wrap puts authentication in front of every route of the gateway.
func (c *oauthConfig) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Browser-based MCP clients call from another origin. Nothing in
		// front of this process adds these headers, and a browser sends no
		// ambient credential here: access only ever comes from the bearer.
		header := w.Header()
		header.Set("Access-Control-Allow-Origin", "*")
		header.Set("Access-Control-Expose-Headers", "WWW-Authenticate, Mcp-Session-Id")
		// CORS preflights carry no credential.
		if r.Method == http.MethodOptions {
			header.Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			header.Set("Access-Control-Allow-Headers", "Authorization, X-Blaxel-Authorization, Content-Type, Accept, Mcp-Session-Id, MCP-Protocol-Version, X-Client-ID")
			header.Set("Access-Control-Max-Age", "600")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if resourcePath, ok := oauthMetadataResource(r); ok {
			c.serveMetadata(w, resourcePath)
			return
		}

		token := bearerToken(r)
		if token == "" {
			c.challenge(w, r, "")
			return
		}
		if len(token) > oauthMaxTokenLength {
			c.challenge(w, r, `error="invalid_token", `)
			return
		}
		verdict, err := c.authorize(r.Context(), token)
		if err != nil {
			c.logFailure(err)
			http.Error(w, "Authentication is temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		switch verdict {
		case oauthUnauthenticated:
			c.challenge(w, r, `error="invalid_token", `)
			return
		case oauthForbidden:
			http.Error(w, "This account cannot access this MCP server", http.StatusForbidden)
			return
		}

		// The credential stops here: nothing behind this point needs it.
		r.Header.Del("Authorization")
		r.Header.Del("X-Blaxel-Authorization")

		// A stream can stay open long after the answer above: keep asking, and
		// end the request once the credential is no longer accepted. WebSocket
		// handlers outlive this function, so they get the same watch to hold.
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()
		defer c.watch(token, cancel)()
		ctx = context.WithValue(ctx, oauthWatchKey{}, func(revoke func()) func() {
			return c.watch(token, revoke)
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// logFailure says why the answer is unknown, at most once a second: the
// caller only sees a 503, and a flood of requests must not flood the logs.
// The error never contains the credential.
func (c *oauthConfig) logFailure(err error) {
	now := c.now().Unix()
	if last := c.lastFailureLog.Load(); last != now && c.lastFailureLog.CompareAndSwap(last, now) {
		log.Printf("OAuth sign-in: access check failed: %v", err)
	}
}

type oauthWatchKey struct{}

// oauthWatch lets a handler whose connection outlives the request (WebSocket)
// keep the sign-in check alive: revoke is called once the credential that
// opened it stops being accepted. The returned func ends the watch. Without
// OAuth sign-in it does nothing.
func oauthWatch(r *http.Request, revoke func()) (stop func()) {
	if watch, ok := r.Context().Value(oauthWatchKey{}).(func(func()) func()); ok {
		return watch(revoke)
	}
	return func() {}
}

// watch keeps asking whether token is still accepted for as long as a
// connection stays open, and calls revoke on the first answer that is not
// "allowed".
//
// It asks when the answer it holds lapses (the token's own expiry included),
// not on a fixed period. authorize keeps a recent "allowed" through a short
// outage of the control plane, so an error here means the answer has been
// unknown for a whole period: the connection must not stay open unchecked.
func (c *oauthConfig) watch(token string, revoke func()) (stop func()) {
	done := make(chan struct{})
	var once sync.Once
	// Read the clock here, not in the goroutine: a request that ends at once
	// stops the watch before it ever looks at anything.
	wait := c.untilRecheck(token)
	go func() {
		for {
			timer := time.NewTimer(wait)
			select {
			case <-done:
				timer.Stop()
				return
			case <-timer.C:
			}
			verdict, err := c.authorize(context.Background(), token)
			if err != nil {
				c.logFailure(err)
			}
			if err != nil || verdict != oauthAllowed {
				revoke()
				return
			}
			wait = c.untilRecheck(token)
		}
	}()
	return func() { once.Do(func() { close(done) }) }
}

// untilRecheck is how long the allowed answer held for token stays good, at
// most recheckEvery and never so short that the watch spins.
func (c *oauthConfig) untilRecheck(token string) time.Duration {
	c.mu.Lock()
	decision, found := c.allowed[sha256.Sum256([]byte(token))]
	c.mu.Unlock()
	wait := c.recheckEvery
	if found {
		wait = min(wait, decision.until.Sub(c.now()))
	}
	return max(wait, min(c.recheckEvery/10, time.Second))
}

// oauthMetadataResource reports whether r asks for the metadata document,
// and the path of the resource it describes ("" for the server root).
func oauthMetadataResource(r *http.Request) (string, bool) {
	if r.Method != http.MethodGet {
		return "", false
	}
	rest, ok := strings.CutPrefix(r.URL.Path, oauthMetadataPath)
	if !ok || (rest != "" && !strings.HasPrefix(rest, "/")) {
		return "", false
	}
	return rest, true
}

func (c *oauthConfig) serveMetadata(w http.ResponseWriter, resourcePath string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"resource":                 c.resourceURL + resourcePath,
		"authorization_servers":    []string{c.issuer},
		"bearer_methods_supported": []string{"header"},
		"scopes_supported":         []string{"openid", "email", "profile", "offline_access"},
	})
}

// challenge answers 401 and points the client at the metadata document of
// the resource it asked for. params is an optional `error="...", ` prefix.
func (c *oauthConfig) challenge(w http.ResponseWriter, r *http.Request, params string) {
	path := r.URL.Path
	if path == "/" || strings.ContainsAny(path, "\"\\") {
		path = ""
	}
	w.Header().Set("WWW-Authenticate", fmt.Sprintf(
		`Bearer %sresource_metadata="%s%s%s"`, params, c.resourceURL, oauthMetadataPath, path))
	http.Error(w, "Authentication required", http.StatusUnauthorized)
}

func bearerToken(r *http.Request) string {
	for _, header := range []string{"Authorization", "X-Blaxel-Authorization"} {
		// The scheme name is case-insensitive (RFC 9110).
		scheme, token, _ := strings.Cut(r.Header.Get(header), " ")
		if strings.EqualFold(scheme, "Bearer") && token != "" {
			return token
		}
	}
	return ""
}

// authorize answers from the cache, or asks the control plane once and
// remembers the answer. An error means the answer is unknown.
func (c *oauthConfig) authorize(ctx context.Context, token string) (oauthVerdict, error) {
	key := sha256.Sum256([]byte(token))
	now := c.now()

	c.mu.Lock()
	allowed, wasAllowed := c.allowed[key]
	refused, wasRefused := c.refused[key]
	c.mu.Unlock()
	if wasAllowed && now.Before(allowed.until) {
		return oauthAllowed, nil
	}
	if wasRefused && now.Before(refused.until) {
		return refused.verdict, nil
	}

	// ponytail: concurrent first requests with one credential each ask the
	// control plane. Add singleflight if that burst ever shows up.
	verdict, err := c.askControlPlane(ctx, token, wasAllowed)
	if err != nil {
		// Unknown is not refused. A credential that was allowed keeps that
		// answer for one more period, never past its own expiry, so an
		// outage or a busy moment does not lock out users who were in.
		if wasAllowed && now.Before(c.graceEnd(token, allowed)) {
			return oauthAllowed, nil
		}
		return 0, err
	}

	ttl := oauthDenyTTL
	if verdict == oauthAllowed {
		ttl = c.recheckEvery
		if expiry, ok := jwtExpiry(token); ok && expiry.Sub(now) < ttl {
			ttl = expiry.Sub(now)
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	// Two checks of one credential can finish out of order: the answer of
	// the check that started last wins, so a late "allowed" never covers a
	// newer refusal.
	if current, ok := c.allowed[key]; ok && current.at.After(now) {
		return verdict, nil
	}
	if current, ok := c.refused[key]; ok && current.at.After(now) {
		return verdict, nil
	}
	delete(c.allowed, key)
	delete(c.refused, key)
	if ttl > 0 {
		decisions := c.refused
		if verdict == oauthAllowed {
			decisions = c.allowed
		}
		if len(decisions) >= oauthMaxDecisions {
			c.prune(decisions, now)
		}
		decisions[key] = oauthDecision{verdict: verdict, at: now, until: now.Add(ttl)}
	}
	return verdict, nil
}

// graceEnd is until when a lapsed "allowed" still stands in for an answer
// the control plane could not give.
func (c *oauthConfig) graceEnd(token string, allowed oauthDecision) time.Time {
	end := allowed.until.Add(c.recheckEvery)
	if expiry, ok := jwtExpiry(token); ok && expiry.Before(end) {
		end = expiry
	}
	return end
}

// prune makes room in a full map: answers of no further use go first, and
// only if none could go is the map emptied. c.mu is held.
func (c *oauthConfig) prune(decisions map[[sha256.Size]byte]oauthDecision, now time.Time) {
	for key, decision := range decisions {
		if !now.Before(decision.until.Add(c.recheckEvery)) {
			delete(decisions, key)
		}
	}
	if len(decisions) >= oauthMaxDecisions {
		clear(decisions)
	}
}

// askControlPlane asks whether token may call this server. known says the
// credential was allowed before: it then takes its slot from the reserved
// pool.
func (c *oauthConfig) askControlPlane(ctx context.Context, token string, known bool) (oauthVerdict, error) {
	slots := c.checks
	if known {
		slots = c.checksKnown
	}
	select {
	case slots <- struct{}{}:
	default:
		timer := time.NewTimer(c.slotWait)
		defer timer.Stop()
		select {
		case slots <- struct{}{}:
		case <-timer.C:
			return 0, fmt.Errorf("too many access checks in flight")
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}
	defer func() { <-slots }()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.accessURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Blaxel-Workspace", c.workspace)
	// The control plane only accepts a token issued for this very server.
	req.Header.Set("X-Blaxel-Resource", c.resourceURL)

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNoContent:
		return oauthAllowed, nil
	case http.StatusUnauthorized:
		return oauthUnauthenticated, nil
	case http.StatusForbidden:
		return oauthForbidden, nil
	default:
		return 0, fmt.Errorf("access check returned %d", resp.StatusCode)
	}
}

// jwtExpiry reads the `exp` claim of a JWT without verifying it. It only
// ever shortens how long an answer the control plane already gave is kept.
func jwtExpiry(token string) (time.Time, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, false
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if json.Unmarshal(payload, &claims) != nil || claims.Exp == 0 {
		return time.Time{}, false
	}
	return time.Unix(claims.Exp, 0), true
}
