package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// OAuth sign-in with Blaxel (opt-in per MCP server).
//
// When the control plane enables it, the workload is public at the platform
// gateway and this process is the only authentication in front of the MCP
// server. It then:
//   - tells clients where to sign in: the protected resource metadata
//     (RFC 9728) and a WWW-Authenticate challenge on 401;
//   - asks the control plane whether the caller's credential (OAuth access
//     token, API key or Blaxel token) may access the workspace, and caches
//     the answer.
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

	// ponytail: one bounded map, emptied when full. Move to an LRU if a
	// single server ever sees more distinct live credentials than this.
	oauthMaxDecisions = 10000
)

type oauthVerdict int

const (
	oauthAllowed oauthVerdict = iota
	oauthUnauthenticated
	oauthForbidden
)

type oauthDecision struct {
	verdict oauthVerdict
	until   time.Time
}

type oauthConfig struct {
	issuer      string // authorization server clients sign in to
	resourceURL string // public base URL of this MCP server, no trailing slash
	accessURL   string // control plane endpoint answering "may this credential access the workspace"
	workspace   string

	client *http.Client
	now    func() time.Time

	mu        sync.Mutex
	decisions map[[sha256.Size]byte]oauthDecision
}

// loadOAuthConfig returns nil when OAuth sign-in is off. Once it is on, a
// missing setting is fatal: the gateway in front lets everyone through, so
// starting without authentication would expose the server.
func loadOAuthConfig(getenv func(string) string) (*oauthConfig, error) {
	issuer := getenv("BL_OAUTH_ISSUER")
	if issuer == "" {
		return nil, nil
	}
	cfg := &oauthConfig{
		issuer:      issuer,
		resourceURL: strings.TrimRight(getenv("BL_OAUTH_RESOURCE_URL"), "/"),
		accessURL:   getenv("BL_OAUTH_ACCESS_URL"),
		workspace:   getenv("BL_WORKSPACE"),
		client:      &http.Client{Timeout: 5 * time.Second},
		now:         time.Now,
		decisions:   make(map[[sha256.Size]byte]oauthDecision),
	}
	for name, value := range map[string]string{
		"BL_OAUTH_RESOURCE_URL": cfg.resourceURL,
		"BL_OAUTH_ACCESS_URL":   cfg.accessURL,
		"BL_WORKSPACE":          cfg.workspace,
	} {
		if value == "" {
			return nil, fmt.Errorf("BL_OAUTH_ISSUER is set but %s is missing", name)
		}
	}
	return cfg, nil
}

// wrap puts authentication in front of every route of the gateway.
func (c *oauthConfig) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS preflights carry no credential.
		if r.Method == http.MethodOptions {
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
		verdict, err := c.authorize(r.Context(), token)
		if err != nil {
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
		next.ServeHTTP(w, r)
	})
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
		if token, ok := strings.CutPrefix(r.Header.Get(header), "Bearer "); ok && token != "" {
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
	decision, found := c.decisions[key]
	c.mu.Unlock()
	if found && now.Before(decision.until) {
		return decision.verdict, nil
	}

	// ponytail: concurrent first requests with one credential each ask the
	// control plane. Add singleflight if that burst ever shows up.
	verdict, err := c.askControlPlane(ctx, token)
	if err != nil {
		return 0, err
	}

	ttl := oauthDenyTTL
	if verdict == oauthAllowed {
		ttl = oauthAllowTTL
		if expiry, ok := jwtExpiry(token); ok && expiry.Sub(now) < ttl {
			ttl = expiry.Sub(now)
		}
	}
	if ttl > 0 {
		c.mu.Lock()
		if len(c.decisions) >= oauthMaxDecisions {
			clear(c.decisions)
		}
		c.decisions[key] = oauthDecision{verdict: verdict, until: now.Add(ttl)}
		c.mu.Unlock()
	}
	return verdict, nil
}

func (c *oauthConfig) askControlPlane(ctx context.Context, token string) (oauthVerdict, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.accessURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Blaxel-Workspace", c.workspace)

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return oauthAllowed, nil
	case resp.StatusCode == http.StatusUnauthorized:
		return oauthUnauthenticated, nil
	case resp.StatusCode == http.StatusForbidden:
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
