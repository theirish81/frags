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
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package mcpauth

import (
	"context"
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

// OAuthProviderConfig is the configuration for OAuthProvider.
type OAuthProviderConfig struct {
	MCPEndpoint string

	ClientID *string

	ClientSecret *string

	RedirectProtocol *RedirectProtocol

	RedirectHost *string

	RedirectCallbackPath *string

	ClientName *string

	// Scopes overrides the scopes requested during authorization.
	// When nil, scopes are taken from the resource metadata's scopes_supported
	// field; if that is also empty, ["repo", "read:user"] is used as a fallback.
	Scopes []string

	HTTPClient *http.Client
}

func (c *OAuthProviderConfig) redirectHost() string {
	if c.RedirectHost != nil {
		return *c.RedirectHost
	}
	return "localhost:9999"
}

func (c *OAuthProviderConfig) redirectCallbackPath() string {
	if c.RedirectCallbackPath != nil {
		return *c.RedirectCallbackPath
	}
	return "/callback"
}

func (c *OAuthProviderConfig) clientName() string {
	if c.ClientName != nil {
		return *c.ClientName
	}
	return "frags-oauth-client"
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
	return http.DefaultClient
}

func (c *OAuthProviderConfig) clientID() string {
	if c.ClientID != nil {
		return *c.ClientID
	}
	return ""
}

func (c *OAuthProviderConfig) clientSecret() string {
	if c.ClientSecret != nil {
		return *c.ClientSecret
	}
	return ""
}

// OAuthProvider implements AuthProvider using the standard MCP OAuth 2.1 flow:
//
//  1. Probe the server unauthenticated → capture WWW-Authenticate header.
//  2. Fetch OAuth Protected Resource Metadata
//  3. Fetch Authorization Server Metadata
//  4. Dynamic Client Registration - skipped if ClientID is pre-set.
//  5. Authorization Code + PKCE — opens browser, listens for local callback.
//  6. Token exchange — code + PKCE verifier → access_token + refresh_token.
type OAuthProvider struct {
	cfg    OAuthProviderConfig
	tok    TokenResult
	ts     oauth2.TokenSource
	logger log.StreamerLogger
}

// NewOAuthProvider returns an OAuthProvider ready to authenticate.
func NewOAuthProvider(cfg OAuthProviderConfig, logger *log.StreamerLogger) *OAuthProvider {
	return &OAuthProvider{cfg: cfg, logger: *logger}
}

func (p *OAuthProvider) New(config OAuthProviderConfig, logger *log.StreamerLogger) GenericOauthProvider {
	return NewOAuthProvider(config, logger)
}

// Authenticate implements AuthProvider.
func (p *OAuthProvider) Authenticate(ctx context.Context) (*http.Client, error) {
	// 1. Probe.
	resourceMetaURL, err := p.probe(ctx)
	if err != nil {
		return nil, fmt.Errorf("probe: %w", err)
	}
	if resourceMetaURL == "" {
		// No auth required — return a plain client with a zero TokenResult.
		return p.cfg.httpClient(), nil
	}

	// 2 & 3. Discover metadata.
	prm, asMeta, err := p.discoverMetadata(ctx, resourceMetaURL)
	if err != nil {
		return nil, fmt.Errorf("discovery: %w", err)
	}

	// 4. Register client (or use pre-configured ClientID).
	clientID, clientSecret, err := p.registerClient(ctx, asMeta)
	if err != nil {
		return nil, fmt.Errorf("client registration: %w", err)
	}

	// 5 & 6. Authorization Code + PKCE.
	oauthTok, err := p.runFlow(ctx, prm, asMeta, clientID, clientSecret)
	if err != nil {
		return nil, fmt.Errorf("oauth flow: %w", err)
	}

	// Persist result so Token() can return it.
	p.tok = TokenResult{
		AccessToken:  oauthTok.AccessToken,
		RefreshToken: oauthTok.RefreshToken,
		TokenType:    oauthTok.TokenType,
		Expiry:       oauthTok.Expiry,
	}

	// Build a live token source that handles refresh transparently.
	conf := p.oauthConfig(asMeta, clientID, clientSecret, nil)
	p.ts = conf.TokenSource(ctx, oauthTok)

	return oauth2.NewClient(ctx, p.ts), nil
}

func (p *OAuthProvider) Token() TokenResult {
	if p.ts != nil {
		if t, err := p.ts.Token(); err == nil {
			p.tok.AccessToken = t.AccessToken
			p.tok.RefreshToken = t.RefreshToken
			p.tok.Expiry = t.Expiry
		}
	}
	return p.tok
}

// probe does a single unauthenticated MCP connect to trigger the 401 and capture the WWW-Authenticate header
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
		return "", nil // server accepted unauthenticated connection
	}
	if rt.captured == nil {
		return "", fmt.Errorf("connection failed but no 401/403 was captured: %w", connErr)
	}

	wwwAuth := rt.captured.Header.Get("WWW-Authenticate")
	p.logger.Debug(log.NewEvent(log.AuthEventType, log.McpComponent).WithMessage("WWW-Authenticate"))

	// Parse resource_metadata="<url>" out of the header value.
	for _, part := range strings.Split(wwwAuth, ",") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, `resource_metadata="`) {
			u := strings.TrimPrefix(part, `resource_metadata="`)
			return strings.TrimSuffix(u, `"`), nil
		}
	}

	// Standard fallback: /.well-known/oauth-protected-resource on the server host.
	parsed, _ := url.Parse(p.cfg.MCPEndpoint)
	return parsed.Scheme + "://" + parsed.Host + "/.well-known/oauth-protected-resource", nil
}

func (p *OAuthProvider) discoverMetadata(ctx context.Context, resourceMetaURL string) (*protectedResourceMetadata, *authServerMetadata, error) {
	var prm protectedResourceMetadata
	p.logger.Debug(log.NewEvent(log.AuthEventType, log.McpComponent).WithArg("resource_metadata", resourceMetaURL))
	if err := p.getJSON(ctx, resourceMetaURL, &prm); err != nil {
		return nil, nil, fmt.Errorf("resource metadata: %w", err)
	}
	if len(prm.AuthorizationServers) == 0 {
		return nil, nil, errors.New("resource metadata has no authorization_servers")
	}

	asIssuer := prm.AuthorizationServers[0]
	parsed, _ := url.Parse(asIssuer)

	wellKnown := parsed.Scheme + "://" + parsed.Host + "/.well-known/oauth-authorization-server"
	if path := strings.Trim(parsed.Path, "/"); path != "" {
		wellKnown += "/" + path
	}

	var asMeta authServerMetadata
	p.logger.Debug(log.NewEvent(log.AuthEventType, log.McpComponent).WithArg("auth_server_metadata", wellKnown))
	if err := p.getJSON(ctx, wellKnown, &asMeta); err != nil {
		// Some servers only expose OIDC discovery.
		oidc := parsed.Scheme + "://" + parsed.Host + "/.well-known/openid-configuration"
		p.logger.Warn(log.NewEvent(log.AuthEventType, log.McpComponent).WithArg("oidc_discovery", oidc).WithErr(err))
		if err2 := p.getJSON(ctx, oidc, &asMeta); err2 != nil {
			return nil, nil, fmt.Errorf("RFC8414: %v; OIDC: %v", err, err2)
		}
	}
	if asMeta.AuthorizationEndpoint == "" || asMeta.TokenEndpoint == "" {
		return nil, nil, errors.New("auth server metadata missing required endpoints")
	}
	return &prm, &asMeta, nil
}

func (p *OAuthProvider) registerClient(ctx context.Context, asMeta *authServerMetadata) (clientID, clientSecret string, err error) {
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
}

// runFlow does the Authorization Code + PKCE flow
func (p *OAuthProvider) runFlow(ctx context.Context, prm *protectedResourceMetadata, asMeta *authServerMetadata, clientID, clientSecret string) (*oauth2.Token, error) {
	verifier, err := randBase64(32)
	if err != nil {
		return nil, err
	}
	state, err := randBase64(16)
	if err != nil {
		return nil, err
	}

	scopes := p.cfg.Scopes
	if len(scopes) == 0 {
		scopes = prm.ScopesSupported
	}
	if len(scopes) == 0 {
		scopes = []string{"repo", "read:user"} // sensible default for GitHub
	}

	conf := p.oauthConfig(asMeta, clientID, clientSecret, scopes)
	authURL := conf.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", s256(verifier)),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("resource", p.cfg.MCPEndpoint), // RFC 8707
	)

	// Start local callback server.
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	ln, err := net.Listen("tcp", p.cfg.redirectHost())
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", p.cfg.redirectHost(), err)
	}
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != p.cfg.redirectCallbackPath() {
				http.NotFound(w, r)
				return
			}
			q := r.URL.Query()
			if e := q.Get("error"); e != "" {
				errCh <- fmt.Errorf("oauth error: %s — %s", e, q.Get("error_description"))
				_, _ = fmt.Fprintln(w, "Authentication failed — you may close this tab.")
				return
			}
			if q.Get("state") != state {
				errCh <- errors.New("state mismatch — possible CSRF")
				_, _ = fmt.Fprintln(w, "State mismatch — you may close this tab.")
				return
			}
			code := q.Get("code")
			if code == "" {
				errCh <- errors.New("callback missing 'code'")
				_, _ = fmt.Fprintln(w, "Missing code — you may close this tab.")
				return
			}
			_, _ = fmt.Fprintln(w, "Authenticated! You may close this tab.")
			codeCh <- code
		}),
	}
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	defer func() {
		_ = srv.Shutdown(context.Background())
	}()
	p.logger.Debug(log.NewEvent(log.AuthEventType, log.McpComponent).WithMessage("opening browser").WithArg("url", authURL))
	openBrowser(authURL)
	p.logger.Debug(log.NewEvent(log.AuthEventType, log.McpComponent).WithMessage("waiting for callback").WithArg("url", p.cfg.redirectURI()))

	var code string
	select {
	case code = <-codeCh:
		p.logger.Debug(log.NewEvent(log.AuthEventType, log.McpComponent).WithMessage("callback received"))
	case err = <-errCh:
		return nil, fmt.Errorf("callback: %w", err)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	p.logger.Debug(log.NewEvent(log.AuthEventType, log.McpComponent).WithMessage("exchanging code for token"))
	token, err := conf.Exchange(ctx, code,
		oauth2.SetAuthURLParam("code_verifier", verifier),
		oauth2.SetAuthURLParam("resource", p.cfg.MCPEndpoint), // RFC 8707
	)
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}
	return token, nil
}

func (p *OAuthProvider) oauthConfig(asMeta *authServerMetadata, clientID, clientSecret string, scopes []string) *oauth2.Config {
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

// capturingTransport wraps a RoundTripper and captures the first 401/403 response  so we can read its headers before
// the SDK consumes/discards them.
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

// ─── OAuth metadata structs ───────────────────────────────────────────────────

type protectedResourceMetadata struct {
	Resource             string   `json:"resource"`
	AuthorizationServers []string `json:"authorization_servers"`
	ScopesSupported      []string `json:"scopes_supported"`
}

type authServerMetadata struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	RegistrationEndpoint  string `json:"registration_endpoint"`
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

type NopOauthAuthProvider struct{}

func (NopOauthAuthProvider) Authenticate(ctx context.Context) (*http.Client, error) {
	return nil, errors.New("unreachable")
}
func (NopOauthAuthProvider) Token() TokenResult {
	return TokenResult{}
}
func (NopOauthAuthProvider) New(config OAuthProviderConfig, logger *log.StreamerLogger) GenericOauthProvider {
	return NewOAuthProvider(config, logger)
}
