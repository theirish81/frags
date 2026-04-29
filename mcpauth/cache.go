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
	"encoding/json"
	"os"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// TokenResult is a serializable representation of OAuth2 tokens and their metadata.
// It is used to persist authentication state across application restarts.
type TokenResult struct {
	// Host is the base URL of the resource server or MCP endpoint associated with this token.
	Host string `json:"host" yaml:"host"`
	// ClientID is the OAuth2 client identifier that was used to obtain this token.
	ClientID string `json:"client_id"`
	// AccessToken is the primary credential used to authorize requests.
	AccessToken string `json:"access_token" yaml:"access_token"`
	// RefreshToken is used to obtain a new access token when the current one expires.
	RefreshToken string `json:"refresh_token" yaml:"refresh_token"`
	// TokenType indicates the type of the token (usually "Bearer").
	TokenType string `json:"token_type" yaml:"token_type"`
	// Expiry is the point in time when the AccessToken will no longer be valid.
	Expiry time.Time `json:"expiry" yaml:"expiry"`
}

// ToOauth2Token converts the TokenResult into a standard oauth2.Token pointer.
// This is required for compatibility with the golang.org/x/oauth2 library.
func (t *TokenResult) ToOauth2Token() *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		TokenType:    t.TokenType,
		Expiry:       t.Expiry,
	}
}

// FromOauth2Token populates the fields of the TokenResult using data from a standard oauth2.Token.
// It returns the updated TokenResult pointer for method chaining.
func (t *TokenResult) FromOauth2Token(tok *oauth2.Token) *TokenResult {
	if tok == nil {
		return t
	}
	t.AccessToken = tok.AccessToken
	t.RefreshToken = tok.RefreshToken
	t.TokenType = tok.TokenType
	t.Expiry = tok.Expiry
	return t
}

// IsExpired checks if the access token is currently expired or will expire within the given grace period.
func (t *TokenResult) IsExpired(grace time.Duration) bool {
	if t.Expiry.IsZero() {
		// If no expiry is set, we assume it's still valid (some servers don't provide expiry for certain token types)
		return false
	}
	return time.Now().Add(grace).After(t.Expiry)
}

// OauthCache defines the interface for persisting and retrieving OAuth2 tokens.
// Implementations handle the storage details (e.g., filesystem, memory, database).
type OauthCache interface {
	// Get retrieves a TokenResult associated with the given OAuthProviderConfig.
	// It returns a pointer to the result and true if found, nil and false otherwise.
	Get(key *OAuthProviderConfig) (*TokenResult, bool)

	// Store saves a TokenResult associated with the given OAuthProviderConfig in the cache.
	// Note: This only updates the in-memory state; call Save to persist to permanent storage.
	Store(key *OAuthProviderConfig, item TokenResult)

	// Save persists the current state of the cache to its permanent storage backend.
	Save(ctx context.Context) error
}

// FsOauthCache implements OauthCache using a JSON file on the local filesystem.
// It uses a mutex to ensure thread-safe access to the underlying map.
type FsOauthCache struct {
	filepath string
	// Items maps a unique fingerprint of the provider configuration to its TokenResult.
	Items map[string]*TokenResult `json:"items,omitempty"`
	mx    sync.Mutex
}

// NewFsOauthCache creates a new FsOauthCache instance and attempts to load existing data
// from the specified file path. If the file doesn't exist, it starts with an empty cache.
func NewFsOauthCache(filepath string) (*FsOauthCache, error) {
	cacheInstance := FsOauthCache{
		filepath: filepath,
		Items:    make(map[string]*TokenResult),
		mx:       sync.Mutex{},
	}
	if data, err := os.ReadFile(filepath); err == nil {
		if err := json.Unmarshal(data, &cacheInstance); err != nil {
			return nil, err
		}
	}
	return &cacheInstance, nil
}

// Get retrieves a token from the filesystem cache using a fingerprint of the config as the key.
func (c *FsOauthCache) Get(key *OAuthProviderConfig) (*TokenResult, bool) {
	c.mx.Lock()
	defer c.mx.Unlock()
	if item, ok := c.Items[key.McpFingerprint()]; ok {
		return item, true
	}
	return nil, false
}

// Store adds or updates a token in the filesystem cache's in-memory map.
func (c *FsOauthCache) Store(key *OAuthProviderConfig, item TokenResult) {
	c.mx.Lock()
	defer c.mx.Unlock()
	c.Items[key.McpFingerprint()] = &item
}

// Save writes the entire cache map to the configured JSON file with secure permissions (0600).
func (c *FsOauthCache) Save(_ context.Context) error {
	c.mx.Lock()
	defer c.mx.Unlock()
	data, err := json.MarshalIndent(c, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.filepath, data, 0600)
}

// InMemoryCache implements OauthCache using an in-memory map.
// Data stored here will not persist across application restarts.
type InMemoryCache struct {
	Items map[string]*TokenResult
	mx    sync.Mutex
}

// NewInMemoryCache creates a new thread-safe InMemoryCache instance.
func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{
		Items: make(map[string]*TokenResult),
	}
}

// RawGet allows retrieval of tokens using an arbitrary string name instead of a config object.
func (c *InMemoryCache) RawGet(name string) (*TokenResult, bool) {
	c.mx.Lock()
	defer c.mx.Unlock()
	if item, ok := c.Items[name]; ok {
		return item, true
	}
	return nil, false
}

// Get retrieves a token from the in-memory cache using the Name field of the provider config.
func (c *InMemoryCache) Get(key *OAuthProviderConfig) (*TokenResult, bool) {
	return c.RawGet(key.Name)
}

// RawStore allows storing tokens using an arbitrary string name instead of a config object.
func (c *InMemoryCache) RawStore(name string, item TokenResult) {
	c.mx.Lock()
	defer c.mx.Unlock()
	c.Items[name] = &item
}

// Store adds a token to the in-memory cache using the Name field of the provider config.
func (c *InMemoryCache) Store(key *OAuthProviderConfig, item TokenResult) {
	c.RawStore(key.Name, item)
}

// Save is a no-op for InMemoryCache as there is no persistent storage.
func (c *InMemoryCache) Save(_ context.Context) error {
	return nil
}

// NopCache is a "No-Operation" cache that implements OauthCache but never stores or retrieves anything.
// This is used when token persistence should be disabled.
type NopCache struct {
}

func (c *NopCache) Get(_ *OAuthProviderConfig) (*TokenResult, bool) {
	return nil, false
}

func (c *NopCache) Store(_ *OAuthProviderConfig, _ TokenResult) {
}

func (c *NopCache) Save(_ context.Context) error {
	return nil
}
