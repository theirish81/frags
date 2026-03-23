/*
 * Copyright (C) 2026 Simone Pezzano
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

package mcpauth

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/theirish81/frags/log"
	"golang.org/x/oauth2"
)

type RedirectProtocol string

const (
	httpProtocol  RedirectProtocol = "http://"
	httpsProtocol RedirectProtocol = "https://"
)

// DiscoveryResources holds the results of the OAuth discovery phase.
type DiscoveryResources struct {
	ResourceMetaURL           string                     `json:"resource_meta_url"`
	ProtectedResourceMetadata *ProtectedResourceMetadata `json:"protected_resource_metadata"`
	AuthServerMetadata        *AuthServerMetadata        `json:"auth_server_metadata"`
}

func (i *DiscoveryResources) MarshalBinary() ([]byte, error) {
	return json.Marshal(i)
}

// OAuthProviderConfig is the configuration for OAuthProvider.
// Pointer fields are optional; their zero-value defaults are documented on each getter method.
type OAuthProviderConfig struct {
	Name                 string
	MCPEndpoint          string
	ClientID             *string
	ClientSecret         *string
	RedirectProtocol     *RedirectProtocol
	RedirectHost         *string
	RedirectCallbackPath *string
	ClientName           *string
	// When nil, scopes are taken from ProtectedResourceMetadata.ScopesSupported, falling back to ["repo", "read:user"]
	Scopes         []string
	State          *string
	Verifier       *string
	NonInteractive bool
	HTTPClient     *http.Client
}

func (c *OAuthProviderConfig) McpFingerprint() string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(c.MCPEndpoint+"|"+c.clientID())))
}

func (c *OAuthProviderConfig) redirectHost() string {
	return derefOr(c.RedirectHost, "localhost:9999")
}

func (c *OAuthProviderConfig) redirectCallbackPath() string {
	return derefOr(c.RedirectCallbackPath, "/callback")
}

func (c *OAuthProviderConfig) clientName() string {
	return derefOr(c.ClientName, "frags-oauth-client")
}

func (c *OAuthProviderConfig) redirectProtocol() RedirectProtocol {
	if c.RedirectProtocol != nil {
		return *c.RedirectProtocol
	}
	return httpProtocol
}

func (c *OAuthProviderConfig) redirectURI() string {
	return string(c.redirectProtocol()) + c.redirectHost() + c.redirectCallbackPath()
}

func (c *OAuthProviderConfig) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	c.HTTPClient = NewDefaultHttpClient()
	return c.HTTPClient
}

func (c *OAuthProviderConfig) clientID() string {
	return derefOr(c.ClientID, "")
}

func (c *OAuthProviderConfig) clientSecret() string {
	return derefOr(c.ClientSecret, "")
}

func (c *OAuthProviderConfig) verifier() string {
	if c.Verifier != nil {
		return *c.Verifier
	}
	v, _ := RandBase64(32)
	c.Verifier = &v
	return v
}

func (c *OAuthProviderConfig) state() string {
	if c.State != nil {
		return *c.State
	}
	s, _ := RandBase64(16)
	c.State = &s
	return s
}

// resolveScopes returns the effective scope list: config → resource metadata → default.
func (c *OAuthProviderConfig) resolveScopes(prm *ProtectedResourceMetadata) []string {
	if len(c.Scopes) > 0 {
		return c.Scopes
	}
	if len(prm.ScopesSupported) > 0 {
		return prm.ScopesSupported
	}
	return []string{"repo", "read:user"}
}

// OAuthProvider implements AuthProvider using the standard MCP OAuth 2.1 flow:
//
//  1. Probe the server unauthenticated → capture WWW-Authenticate header.
//  2. Fetch OAuth Protected Resource Metadata.
//  3. Fetch Authorization Server Metadata.
//  4. Dynamic Client Registration — skipped; ClientID must be pre-configured.
//  5. Authorization Code + PKCE — opens browser, listens for local callback (local flow only).
//  6. Token exchange — code + PKCE verifier → access_token + refresh_token.
type OAuthProvider struct {
	cfg        OAuthProviderConfig
	ts         oauth2.TokenSource
	logger     log.StreamerLogger
	oauthCache OauthCache
}

// NewOAuthProvider returns an OAuthProvider ready to authenticate.
func NewOAuthProvider(cfg OAuthProviderConfig, logger *log.StreamerLogger) *OAuthProvider {
	px := &OAuthProvider{cfg: cfg, logger: *logger}
	px.WithCache(&NopCache{})
	return px
}

func NewEmptyOauthProvider(nonInteractive bool) *OAuthProvider {
	px := &OAuthProvider{
		cfg: OAuthProviderConfig{NonInteractive: nonInteractive},
	}
	px.WithCache(&NopCache{})
	return px
}

func (p *OAuthProvider) New(config OAuthProviderConfig, logger *log.StreamerLogger) GenericOauthProvider {
	config.NonInteractive = p.cfg.NonInteractive
	return NewOAuthProvider(config, logger).WithCache(p.oauthCache)
}

func (p *OAuthProvider) WithCache(tokenCache OauthCache) GenericOauthProvider {
	p.oauthCache = tokenCache
	return p
}

func (p *OAuthProvider) Config() *OAuthProviderConfig {
	return &p.cfg
}

func (*OAuthProvider) Name() string { return "" }

// Discover runs steps 1–3 of the OAuth flow and returns the discovered resources.
// requiresAuth is false when the server accepts unauthenticated connections.
func (p *OAuthProvider) Discover(ctx context.Context) (*DiscoveryResources, bool, error) {
	resourceMetaURL, err := p.probe(ctx)
	if err != nil {
		return nil, true, fmt.Errorf("probe: %w", err)
	}
	if resourceMetaURL == "" {
		return nil, false, nil
	}

	prm, asMeta, err := p.discoverMetadata(ctx, resourceMetaURL)
	if err != nil {
		return nil, true, fmt.Errorf("discovery: %w", err)
	}

	/*
		_, _, err = p.registerClient(ctx, asMeta)
		if err != nil {
			return nil, true, fmt.Errorf("client registration: %w", err)
		}
	*/

	return &DiscoveryResources{
		ResourceMetaURL:           resourceMetaURL,
		ProtectedResourceMetadata: prm,
		AuthServerMetadata:        asMeta,
	}, true, nil
}

func (p *OAuthProvider) SetToken(tok *oauth2.Token, resources *DiscoveryResources) {
	p.ts = NewFragsTokenSource(tok, p, resources, p.oauthCache, p.logger)
}

// Authenticate runs the full local OAuth flow (browser redirect + local callback server).
func (p *OAuthProvider) Authenticate(ctx context.Context) (*http.Client, error) {
	resources, requiresAuth, err := p.Discover(ctx)
	if err != nil {
		return nil, fmt.Errorf("discover: %w", err)
	}
	if !requiresAuth {
		return p.cfg.httpClient(), nil
	}
	refreshed := false
	p.logger.Debug(log.NewEvent(log.AuthEventType, log.McpComponent).WithMessage("checking cache"))
	if cache, ok := p.oauthCache.Get(&p.cfg); ok {
		p.logger.Debug(log.NewEvent(log.AuthEventType, log.McpComponent).WithMessage("cache hit"))
		p.SetToken(&oauth2.Token{AccessToken: cache.AccessToken, RefreshToken: cache.RefreshToken, Expiry: cache.Expiry}, resources)
		token, err := p.Token()
		if err == nil {
			// no error means that either:
			// a) the token was still valid
			// b) the refresh token succeeded
			refreshed = true
			cache.AccessToken = token.AccessToken
			cache.RefreshToken = token.RefreshToken

			// if the access token is still valid and not refreshed, then the token has zeroed expiry
			if !token.Expiry.IsZero() {
				cache.Expiry = token.Expiry
			}
			cache.TokenType = token.TokenType

			cache.Host = p.cfg.MCPEndpoint
			cache.ClientID = p.cfg.clientID()
			p.oauthCache.Store(&p.cfg, *cache)
			if err := p.oauthCache.Save(ctx); err != nil {
				return nil, fmt.Errorf("cache save: %w", err)
			}
		}
	}
	// if:
	// a) the refresh token mechanism could not run (missing cached refresh token) or refresh failed
	// AND
	// b) provider can operate i n interactive mode (we're running on a client computer, such as a CLI utility)
	// then we trigger open a browser for 3-legged oauth2
	if !refreshed && !p.cfg.NonInteractive {
		// runs the interactive flow
		oauthTok, err := p.runFlow(ctx, resources)
		if err != nil {
			return nil, fmt.Errorf("oauth flow: %w", err)
		}

		p.SetToken(oauthTok, resources)
		tr := *(&TokenResult{}).FromOauth2Token(oauthTok)
		// adding some meta-information here
		tr.Host = p.cfg.MCPEndpoint
		tr.ClientID = p.cfg.clientID()
		refreshed = true
		p.oauthCache.Store(&p.cfg, tr)
		if err := p.oauthCache.Save(ctx); err != nil {
			return nil, fmt.Errorf("cache save: %w", err)
		}
	}

	if !refreshed {
		// if the token was not refreshed, it means that:
		// a) the token was not refreshed, nor the interactive mechanism went anywhere
		// b) the token was not refreshed, and the interactive mode was disabled
		// this means we can't proceed any further
		return nil, fmt.Errorf("no valid token found")
	}
	return oauth2.NewClient(ctx, p.ts), nil
}

// AuthLink builds the authorization URL for deferred / SaaS flows.
func (p *OAuthProvider) AuthLink(resources *DiscoveryResources) (string, error) {
	conf := p.OauthConfig(
		resources.AuthServerMetadata,
		p.cfg.clientID(), p.cfg.clientSecret(),
		p.cfg.resolveScopes(resources.ProtectedResourceMetadata),
	)
	authURL := conf.AuthCodeURL(p.cfg.state(),
		oauth2.SetAuthURLParam("code_challenge", s256(p.cfg.verifier())),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("resource", p.cfg.MCPEndpoint),
	)
	return authURL, nil
}

// Exchange trades an authorization code for tokens (deferred / SaaS flows).
func (p *OAuthProvider) Exchange(ctx context.Context, code string, resources *DiscoveryResources) (*oauth2.Token, error) {
	conf := p.OauthConfig(
		resources.AuthServerMetadata,
		p.cfg.clientID(), p.cfg.clientSecret(),
		p.cfg.resolveScopes(resources.ProtectedResourceMetadata),
	)
	return conf.Exchange(ctx, code,
		oauth2.SetAuthURLParam("code_verifier", p.cfg.verifier()),
		oauth2.SetAuthURLParam("resource", p.cfg.MCPEndpoint),
	)
}

// Token returns the most recent token, refreshing from the token source if available.
func (p *OAuthProvider) Token() (TokenResult, error) {
	if p.ts != nil {
		if t, err := p.ts.Token(); err == nil {
			tr := TokenResult{}
			return *tr.FromOauth2Token(t), nil
		} else {
			return TokenResult{}, err
		}
	}
	return TokenResult{}, errors.New("no token source")
}

// OauthConfig builds an oauth2.Config for the given authorization server.
func (p *OAuthProvider) OauthConfig(asMeta *AuthServerMetadata, clientID, clientSecret string, scopes []string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  p.cfg.redirectURI(),
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  asMeta.AuthorizationEndpoint,
			TokenURL: asMeta.TokenEndpoint,
		},
	}
}

// probe makes an unauthenticated MCP connection to trigger a 401/403 and extract
// the resource_metadata URL from the WWW-Authenticate header.
// Returns an empty string when no authentication is required.
func (p *OAuthProvider) probe(ctx context.Context) (string, error) {
	rt := &capturingTransport{inner: p.cfg.httpClient().Transport}
	if rt.inner == nil {
		rt.inner = http.DefaultTransport
	}

	probeClient := mcp.NewClient(&mcp.Implementation{Name: "probe", Version: "0"}, nil)
	_, connErr := probeClient.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:   p.cfg.MCPEndpoint,
		HTTPClient: &http.Client{Transport: rt},
	}, nil)

	if connErr == nil {
		return "", nil
	}
	if rt.captured == nil {
		return "", fmt.Errorf("connection failed but no 401/403 was captured: %w", connErr)
	}

	p.logger.Debug(log.NewEvent(log.AuthEventType, log.McpComponent).WithMessage("WWW-Authenticate"))

	for _, part := range strings.Split(rt.captured.Header.Get("WWW-Authenticate"), ",") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, `resource_metadata="`) {
			return strings.TrimSuffix(strings.TrimPrefix(part, `resource_metadata="`), `"`), nil
		}
	}

	// Fallback to the standard well-known path on the server host.
	parsed, _ := url.Parse(p.cfg.MCPEndpoint)
	return parsed.Scheme + "://" + parsed.Host + "/.well-known/oauth-protected-resource", nil
}

// discoverMetadata retrieves all the necessary information to perform the oauth procedures
func (p *OAuthProvider) discoverMetadata(ctx context.Context, resourceMetaURL string) (*ProtectedResourceMetadata, *AuthServerMetadata, error) {
	var prm ProtectedResourceMetadata
	p.logger.Debug(log.NewEvent(log.AuthEventType, log.McpComponent).WithArg("resource_metadata", resourceMetaURL))
	if err := p.getJSON(ctx, resourceMetaURL, &prm); err != nil {
		return nil, nil, fmt.Errorf("resource metadata: %w", err)
	}
	if len(prm.AuthorizationServers) == 0 {
		return nil, nil, errors.New("resource metadata has no authorization_servers")
	}

	asMeta, err := p.fetchAuthServerMetadata(ctx, prm.AuthorizationServers[0])
	if err != nil {
		return nil, nil, err
	}
	return &prm, asMeta, nil
}

// fetchAuthServerMetadata tries RFC 8414 discovery first, then OIDC discovery as a fallback.
func (p *OAuthProvider) fetchAuthServerMetadata(ctx context.Context, issuer string) (*AuthServerMetadata, error) {
	parsed, _ := url.Parse(issuer)
	base := parsed.Scheme + "://" + parsed.Host

	rfc8414 := base + "/.well-known/oauth-authorization-server"
	if path := strings.Trim(parsed.Path, "/"); path != "" {
		rfc8414 += "/" + path
	}

	var asMeta AuthServerMetadata
	p.logger.Debug(log.NewEvent(log.AuthEventType, log.McpComponent).WithArg("auth_server_metadata", rfc8414))
	if err := p.getJSON(ctx, rfc8414, &asMeta); err != nil {
		oidc := base + "/.well-known/openid-configuration"
		p.logger.Warn(log.NewEvent(log.AuthEventType, log.McpComponent).WithArg("oidc_discovery", oidc).WithErr(err))
		if err2 := p.getJSON(ctx, oidc, &asMeta); err2 != nil {
			return nil, fmt.Errorf("RFC8414: %v; OIDC: %v", err, err2)
		}
	}
	if asMeta.AuthorizationEndpoint == "" || asMeta.TokenEndpoint == "" {
		return nil, errors.New("auth server metadata missing required endpoints")
	}
	return &asMeta, nil
}

// runFlow executes the local Authorization Code + PKCE flow:
// opens a browser, waits for the callback, then exchanges the code for tokens.
func (p *OAuthProvider) runFlow(ctx context.Context, resources *DiscoveryResources) (*oauth2.Token, error) {
	verifier := p.cfg.verifier()
	state := p.cfg.state()
	scopes := p.cfg.resolveScopes(resources.ProtectedResourceMetadata)

	conf := p.OauthConfig(resources.AuthServerMetadata, p.cfg.clientID(), p.cfg.clientSecret(), scopes)
	authURL := conf.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", s256(verifier)),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("resource", p.cfg.MCPEndpoint),
	)

	code, err := p.listenForCode(ctx, state, authURL)
	if err != nil {
		return nil, err
	}

	p.logger.Debug(log.NewEvent(log.AuthEventType, log.McpComponent).WithMessage("exchanging code for token"))
	token, err := conf.Exchange(ctx, code,
		oauth2.SetAuthURLParam("code_verifier", verifier),
		oauth2.SetAuthURLParam("resource", p.cfg.MCPEndpoint),
	)
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}
	return token, nil
}

// listenForCode starts a local HTTP server, opens the browser at authURL,
// and waits for the OAuth callback to deliver the authorization code.
func (p *OAuthProvider) listenForCode(ctx context.Context, state, authURL string) (string, error) {
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	ln, err := net.Listen("tcp", p.cfg.redirectHost())
	if err != nil {
		return "", fmt.Errorf("listen on %s: %w", p.cfg.redirectHost(), err)
	}

	srv := &http.Server{Handler: p.callbackHandler(state, codeCh, errCh)}
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	defer func() { _ = srv.Shutdown(context.Background()) }()

	p.logger.Debug(log.NewEvent(log.AuthEventType, log.McpComponent).WithMessage("opening browser").WithArg("url", authURL))
	openBrowser(authURL)
	p.logger.Debug(log.NewEvent(log.AuthEventType, log.McpComponent).WithMessage("waiting for callback").WithArg("url", p.cfg.redirectURI()))

	select {
	case code := <-codeCh:
		p.logger.Debug(log.NewEvent(log.AuthEventType, log.McpComponent).WithMessage("callback received"))
		return code, nil
	case err := <-errCh:
		return "", fmt.Errorf("callback: %w", err)
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// callbackHandler returns an http.HandlerFunc that validates the OAuth callback
// and sends the authorization code (or an error) on the provided channels.
func (p *OAuthProvider) callbackHandler(state string, codeCh chan<- string, errCh chan<- error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != p.cfg.redirectCallbackPath() {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		switch {
		case q.Get("error") != "":
			errCh <- fmt.Errorf("oauth error: %s — %s", q.Get("error"), q.Get("error_description"))
			_, _ = fmt.Fprintln(w, "Authentication failed — you may close this tab.")
		case q.Get("state") != state:
			errCh <- errors.New("state mismatch — possible CSRF")
			_, _ = fmt.Fprintln(w, "State mismatch — you may close this tab.")
		case q.Get("code") == "":
			errCh <- errors.New("callback missing 'code'")
			_, _ = fmt.Fprintln(w, "Missing code — you may close this tab.")
		default:
			_, _ = fmt.Fprintln(w, "Authenticated! You may close this tab.")
			codeCh <- q.Get("code")
		}
	}
}

// capturingTransport wraps a RoundTripper and captures the first 401/403 response
// so we can read its headers before the SDK discards them.
type capturingTransport struct {
	inner    http.RoundTripper
	captured *http.Response
}

func (rt *capturingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := rt.inner.RoundTrip(req)
	if err == nil && rt.captured == nil && (resp.StatusCode == 401 || resp.StatusCode == 403) {
		rt.captured = resp
	}
	return resp, err
}

// ─── Metadata structs ────────────────────────────────────────────────────────

type ProtectedResourceMetadata struct {
	Resource             string   `json:"resource"`
	AuthorizationServers []string `json:"authorization_servers"`
	ScopesSupported      []string `json:"scopes_supported"`
}

type AuthServerMetadata struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	RegistrationEndpoint  string `json:"registration_endpoint"`
}

// registerClient returns the pre-configured ClientID/Secret.
// Dynamic Client Registration is disabled but preserved below for future use.
func (p *OAuthProvider) registerClient(_ context.Context, _ *AuthServerMetadata) (clientID, clientSecret string, err error) {
	return p.cfg.clientID(), p.cfg.clientSecret(), nil

	/*
		if id := p.cfg.clientID(); id != "" {
			p.logger.Info(log.NewEvent(log.AuthEventType, log.McpComponent).WithMessage("using configured client_id"))
			return id, p.cfg.clientSecret(), nil
		}

		if asMeta.RegistrationEndpoint == "" {
			return "", "", errors.New("auth server has no registration_endpoint and no ClientID is configured")
		}
		p.logger.Debug(log.NewEvent(log.AuthEventType, log.McpComponent).WithMessage("dynamic client registration"))
		reqBody := dcrRequest{
			ClientName:              p.cfg.clientName(),
			RedirectURIs:            []string{p.cfg.redirectURI()},
			GrantTypes:              []string{"authorization_code"},
			ResponseTypes:           []string{"code"},
			TokenEndpointAuthMethod: "none", // public client — PKCE only, no secret
		}
		var resp dcrResponse
		if err := p.postJSON(ctx, asMeta.RegistrationEndpoint, reqBody, &resp); err != nil {
			return "", "", fmt.Errorf("DCR: %w", err)
		}
		if resp.ClientID == "" {
			return "", "", errors.New("DCR response missing client_id")
		}
		p.logger.Info(log.NewEvent(log.AuthEventType, log.McpComponent).WithMessage("registered client_id").WithArg("client_id", resp.ClientID))
		return resp.ClientID, resp.ClientSecret, nil
	*/
}

type dcrRequest struct {
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

type dcrResponse struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret,omitempty"`
}

// ─── NopOauthAuthProvider ────────────────────────────────────────────────────

type NopOauthAuthProvider struct{}

func (NopOauthAuthProvider) Authenticate(_ context.Context) (*http.Client, error) {
	return http.DefaultClient, nil
}

func (NopOauthAuthProvider) Token() (TokenResult, error) { return TokenResult{}, nil }

func (NopOauthAuthProvider) New(_ OAuthProviderConfig, _ *log.StreamerLogger) GenericOauthProvider {
	return NopOauthAuthProvider{}
}

func (p NopOauthAuthProvider) WithCache(_ OauthCache) GenericOauthProvider {
	return &p
}
func derefOr(p *string, fallback string) string {
	if p != nil {
		return *p
	}
	return fallback
}
