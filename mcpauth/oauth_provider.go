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
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"

	"github.com/theirish81/doauth"
	"github.com/theirish81/frags/httpfactory"
	"github.com/theirish81/frags/log"
	"golang.org/x/oauth2"
)

// OAuthProviderConfig is the configuration for OAuthProvider.
// It embeds doauth.Config for standard OAuth2 parameters and adds frags-specific fields.
type OAuthProviderConfig struct {
	doauth.Config

	// State is an optional pre-configured CSRF token. If nil, a random one will be generated.
	State *string

	// NonInteractive, when true, prevents the provider from attempting to open a browser
	// for the 3-legged OAuth flow. Use this for server-side or headless environments.
	NonInteractive bool
	// HTTPClient is an optional custom client to be used for all OAuth2-related network requests.
	HTTPClient *http.Client
}

// ensureDefaults populates missing configuration fields with sensible default values.
func (c *OAuthProviderConfig) ensureDefaults() {
	if c.RedirectURL == "" {
		// Default local callback URL for the frags CLI and local tools.
		c.RedirectURL = "http://localhost:9999/callback"
	}
	if c.ClientID == "" {
		// Default client ID identifying frags to the authorization server.
		c.ClientID = "frags-client"
	}
}

// McpFingerprint generates a unique hash based on the configuration's base URL and client ID.
// This is used by caches to uniquely identify the authentication session.
func (c *OAuthProviderConfig) McpFingerprint() string {
	return c.Config.McpFingerprint()
}

// httpClient returns the configured HTTP client or a default one from the httpfactory.
func (c *OAuthProviderConfig) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	// Fallback to the global frags http client factory.
	c.HTTPClient = httpfactory.Instance.HttpClient()
	return c.HTTPClient
}

// OAuthProvider implements the AuthProvider interface using the standard MCP OAuth 2.1 flow.
// It leverages the github.com/theirish81/doauth library for discovery and local flow management.
type OAuthProvider struct {
	cfg        OAuthProviderConfig
	ts         oauth2.TokenSource
	logger     log.StreamerLogger
	oauthCache OauthCache
	// mx protects access to all mutable fields: cfg, ts, and oauthCache.
	// Note: Mutexes must never be copied. Each instance created by the factory (New)
	// gets its own fresh, zero-valued mutex.
	mx sync.RWMutex
}

// NewOAuthProvider creates a new OAuthProvider instance with the provided configuration and logger.
// It initializes the provider with a NopCache by default.
func NewOAuthProvider(cfg OAuthProviderConfig, logger *log.StreamerLogger) *OAuthProvider {
	cfg.ensureDefaults()
	px := &OAuthProvider{cfg: cfg, logger: *logger}
	px.WithCache(&NopCache{})
	return px
}

// NewEmptyOauthProvider creates an OAuthProvider with no initial configuration except for the
// interactive mode setting. This is typically used as a prototype for creating configured providers.
func NewEmptyOauthProvider(nonInteractive bool) *OAuthProvider {
	px := &OAuthProvider{
		cfg: OAuthProviderConfig{NonInteractive: nonInteractive},
	}
	// Still ensure defaults for the prototype.
	px.cfg.ensureDefaults()
	px.WithCache(&NopCache{})
	return px
}

// New implements GenericOauthProvider by creating a new configured instance from the prototype.
func (p *OAuthProvider) New(config OAuthProviderConfig, logger *log.StreamerLogger) GenericOauthProvider {
	// Inherit non-interactive setting from the prototype.
	config.NonInteractive = p.SafeConfig().NonInteractive

	// Inherit the cache implementation from the prototype.
	p.mx.RLock()
	cache := p.oauthCache
	p.mx.RUnlock()

	// Return a fresh instance (with a fresh mutex).
	return NewOAuthProvider(config, logger).WithCache(cache)
}

// WithCache attaches a token cache to the provider for persistent authentication state.
func (p *OAuthProvider) WithCache(tokenCache OauthCache) GenericOauthProvider {
	p.mx.Lock()
	defer p.mx.Unlock()
	p.oauthCache = tokenCache
	return p
}

// Config returns a pointer to the provider's current configuration.
// Warning: Accessing the returned struct is not thread-safe if the provider is being modified.
func (p *OAuthProvider) Config() *OAuthProviderConfig {
	p.mx.RLock()
	defer p.mx.RUnlock()
	return &p.cfg
}

// SafeConfig returns a copy of the current configuration.
func (p *OAuthProvider) SafeConfig() OAuthProviderConfig {
	p.mx.RLock()
	defer p.mx.RUnlock()
	return p.cfg
}

// Name returns an empty string as OAuthProvider's name is usually managed via the config.
func (*OAuthProvider) Name() string { return "" }

// Discover retrieves OAuth2 metadata from the server, either via standard discovery paths
// (RFC 8414) or by probing the resource endpoint (RFC 9728).
// If endpoints are already explicitly configured, discovery is bypassed.
func (p *OAuthProvider) Discover(ctx context.Context) (*doauth.Metadata, bool, error) {
	cfg := p.SafeConfig()

	// Bypass network discovery if we already have the necessary endpoints.
	if cfg.AuthorizationURL != "" && cfg.TokenURL != "" {
		return &doauth.Metadata{
			AuthorizationURL: cfg.AuthorizationURL,
			TokenURL:         cfg.TokenURL,
			ScopesSupported:  cfg.Scopes,
		}, true, nil
	}

	auth, err := p.newAuthenticator()
	if err != nil {
		return nil, false, err
	}

	// Perform discovery or probing.
	requiresAuth, err := auth.Discover(ctx)
	if err != nil {
		return nil, false, err
	}
	return auth.GetMetadata(), requiresAuth, nil
}

// SetToken initializes the provider's token source with the provided token and metadata.
func (p *OAuthProvider) SetToken(tok *oauth2.Token, resources *doauth.Metadata) {
	p.mx.Lock()
	defer p.mx.Unlock()
	p.ts = NewFragsTokenSource(tok, p, resources, p.oauthCache, p.logger)
}

// Authenticate performs the full authentication flow.
func (p *OAuthProvider) Authenticate(ctx context.Context) (*http.Client, error) {
	resources, requiresAuth, err := p.Discover(ctx)
	if err != nil {
		return nil, fmt.Errorf("discover: %w", err)
	}

	cfg := p.SafeConfig()
	if !requiresAuth {
		return cfg.httpClient(), nil
	}

	refreshed := false
	p.logger.Debug(log.NewEvent(log.AuthEventType, log.McpComponent).WithMessage("checking cache"))

	p.mx.RLock()
	cache := p.oauthCache
	p.mx.RUnlock()

	// Step 1: Check if we have a cached token.
	if cachedResult, ok := cache.Get(&cfg); ok {
		p.logger.Debug(log.NewEvent(log.AuthEventType, log.McpComponent).WithMessage("cache hit"))
		p.SetToken(&oauth2.Token{
			AccessToken:  cachedResult.AccessToken,
			RefreshToken: cachedResult.RefreshToken,
			Expiry:       cachedResult.Expiry,
		}, resources)

		// Verify if the token is valid or successfully refreshed.
		token, err := p.Token()
		if err == nil {
			refreshed = true
			cachedResult.AccessToken = token.AccessToken
			cachedResult.RefreshToken = token.RefreshToken
			if !token.Expiry.IsZero() {
				cachedResult.Expiry = token.Expiry
			}
			cachedResult.TokenType = token.TokenType
			cachedResult.Host = cfg.BaseURL
			cachedResult.ClientID = cfg.ClientID

			// Update the cache with the (potentially refreshed) token.
			cache.Store(&cfg, *cachedResult)
			if err := cache.Save(ctx); err != nil {
				return nil, fmt.Errorf("cache save: %w", err)
			}
		}
	}

	// Step 2: If no valid token found, run interactive flow if allowed.
	if !refreshed && !cfg.NonInteractive {
		oauthTok, err := p.runFlow(ctx, resources)
		if err != nil {
			return nil, fmt.Errorf("oauth flow: %w", err)
		}

		p.SetToken(oauthTok, resources)
		tr := *(&TokenResult{}).FromOauth2Token(oauthTok)
		tr.Host = cfg.BaseURL
		tr.ClientID = cfg.ClientID
		refreshed = true

		cache.Store(&cfg, tr)
		if err := cache.Save(ctx); err != nil {
			return nil, fmt.Errorf("cache save: %w", err)
		}
	}

	if !refreshed {
		return nil, fmt.Errorf("no valid token found and interactive authentication disabled or failed")
	}

	// Return an oauth2 client. NewClient handles the transport configuration.
	ctx = context.WithValue(ctx, oauth2.HTTPClient, cfg.httpClient())
	p.mx.RLock()
	ts := p.ts
	p.mx.RUnlock()
	return oauth2.NewClient(ctx, ts), nil
}

// AuthLink generates the authorization URL for external or deferred authentication flows.
func (p *OAuthProvider) AuthLink(resources *doauth.Metadata, state string) (authURL string, finalState string, verifier string, err error) {
	p.updateEndpoints(resources)

	auth, err := p.newAuthenticator()
	if err != nil {
		return "", "", "", err
	}

	var authOpts []doauth.AuthURLOption
	cfg := p.SafeConfig()

	if state != "" {
		authOpts = append(authOpts, doauth.WithState(state))
	} else if cfg.State != nil {
		authOpts = append(authOpts, doauth.WithState(*cfg.State))
	}

	return auth.GetAuthURL(authOpts...)
}

// Exchange trades an authorization code for an OAuth2 token.
func (p *OAuthProvider) Exchange(ctx context.Context, code string, state string, verifier string, resources *doauth.Metadata) (*oauth2.Token, error) {
	p.updateEndpoints(resources)

	auth, err := p.newAuthenticator()
	if err != nil {
		return nil, err
	}

	return auth.Exchange(ctx, code, state, verifier)
}

// Token returns the current valid token.
func (p *OAuthProvider) Token() (TokenResult, error) {
	p.mx.RLock()
	ts := p.ts
	p.mx.RUnlock()

	if ts != nil {
		if t, err := ts.Token(); err == nil {
			tr := TokenResult{}
			return *tr.FromOauth2Token(t), nil
		} else {
			return TokenResult{}, err
		}
	}
	return TokenResult{}, fmt.Errorf("no token source available")
}

// runFlow executes the local 3-legged OAuth flow.
func (p *OAuthProvider) runFlow(ctx context.Context, resources *doauth.Metadata) (*oauth2.Token, error) {
	cfg := p.SafeConfig()
	u, err := url.Parse(cfg.RedirectURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redirect URL: %w", err)
	}
	port, _ := strconv.Atoi(u.Port())
	if port == 0 {
		port = 80
		if u.Scheme == "https" {
			port = 443
		}
	}

	flow := doauth.NewLocalFlow(
		doauth.WithPort(port),
		doauth.WithCallbackPath(u.Path),
		doauth.WithLocalFlowLogger(p.logger.Logger()),
	)

	p.updateEndpoints(resources)

	auth, err := p.newAuthenticator()
	if err != nil {
		return nil, err
	}

	var authOpts []doauth.AuthURLOption
	// Re-fetch config as it might have been updated by updateEndpoints
	cfg = p.SafeConfig()
	if cfg.State != nil {
		authOpts = append(authOpts, doauth.WithState(*cfg.State))
	}

	authURL, state, verifier, err := auth.GetAuthURL(authOpts...)
	if err != nil {
		return nil, err
	}

	if err := flow.OpenBrowser(authURL); err != nil {
		return nil, fmt.Errorf("failed to open browser: %w", err)
	}

	res, err := flow.WaitForCode(ctx)
	if err != nil {
		return nil, err
	}

	if res.State != state {
		return nil, fmt.Errorf("security error: state mismatch")
	}

	return auth.Exchange(ctx, res.Code, state, verifier)
}

// newAuthenticator creates a new doauth.Authenticator instance.
func (p *OAuthProvider) newAuthenticator() (*doauth.Authenticator, error) {
	cfg := p.SafeConfig()
	return doauth.NewAuthenticator(
		cfg.Config,
		doauth.WithHTTPClient(cfg.httpClient()),
		doauth.WithLogger(p.logger.Logger()),
	)
}

// updateEndpoints ensures that the configuration endpoints match discovered metadata.
func (p *OAuthProvider) updateEndpoints(resources *doauth.Metadata) {
	p.mx.Lock()
	defer p.mx.Unlock()
	if p.cfg.AuthorizationURL == "" {
		p.cfg.AuthorizationURL = resources.AuthorizationURL
	}
	if p.cfg.TokenURL == "" {
		p.cfg.TokenURL = resources.TokenURL
	}
}

// ─── NopOauthAuthProvider ────────────────────────────────────────────────────

type NopOauthAuthProvider struct{}

func (NopOauthAuthProvider) Authenticate(_ context.Context) (*http.Client, error) {
	return httpfactory.Instance.HttpClient(), nil
}

func (NopOauthAuthProvider) Token() (TokenResult, error) { return TokenResult{}, nil }

func (NopOauthAuthProvider) New(_ OAuthProviderConfig, _ *log.StreamerLogger) GenericOauthProvider {
	return NopOauthAuthProvider{}
}

func (NopOauthAuthProvider) Config() *OAuthProviderConfig {
	return &OAuthProviderConfig{}
}

func (p NopOauthAuthProvider) WithCache(_ OauthCache) GenericOauthProvider {
	return &p
}
