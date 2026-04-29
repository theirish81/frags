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
	"sync"
	"time"

	"github.com/theirish81/doauth"
	"github.com/theirish81/frags/log"
	"golang.org/x/oauth2"
)

// FragsTokenSource is a smart wrapper around oauth2.TokenSource that integrates with OauthCache.
// It ensures that:
// 1. Tokens are shared across multiple components via a central cache.
// 2. If another process or thread updates the cache, this source picks up the new token.
// 3. When a token is refreshed, the new token is automatically persisted back to the cache.
type FragsTokenSource struct {
	// ts is the underlying oauth2.TokenSource that handles the actual refresh logic.
	ts oauth2.TokenSource
	// cache is the persistence layer for tokens.
	cache OauthCache
	// conf is the OAuth2 configuration used to create new token sources.
	conf *oauth2.Config
	// p is the parent OAuthProvider, used to access current configuration.
	p *OAuthProvider
	// resources contains the discovered server endpoints.
	resources *doauth.Metadata
	// log is used for reporting synchronization and refresh events.
	log log.StreamerLogger
	// expiry is the expiration time of the token currently held by the underlying ts.
	expiry time.Time
	// mx protects access to ts and expiry.
	mx sync.Mutex
}

// NewFragsTokenSource constructs a new FragsTokenSource and initializes the underlying OAuth2 token source.
func NewFragsTokenSource(token *oauth2.Token, p *OAuthProvider, resources *doauth.Metadata, cache OauthCache, log log.StreamerLogger) *FragsTokenSource {
	// Reconstruct the oauth2.Config from the provider's current state.
	conf := &oauth2.Config{
		ClientID:     p.cfg.ClientID,
		ClientSecret: p.cfg.ClientSecret,
		RedirectURL:  p.cfg.RedirectURL,
		Scopes:       p.cfg.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  resources.AuthorizationURL,
			TokenURL: resources.TokenURL,
		},
	}

	// Create the initial token source using the provided token.
	return &FragsTokenSource{
		ts:     conf.TokenSource(context.Background(), token),
		expiry: token.Expiry,
		conf:   conf,
		cache:  cache,
		p:      p,
		log:    log,
	}
}

// Token implements the oauth2.TokenSource interface.
// It synchronizes with the cache before returning a token to ensure the most recent credential is used.
func (f *FragsTokenSource) Token() (*oauth2.Token, error) {
	// Synchronization check with the cache.
	f.mx.Lock()
	cachedToken, hasExistingToken := f.cache.Get(f.p.Config())
	if hasExistingToken && cachedToken.Expiry.After(f.expiry) {
		// Cache contains a newer token. Re-initialize the underlying source to use it.
		f.ts = f.conf.TokenSource(context.Background(), cachedToken.ToOauth2Token())
		f.expiry = cachedToken.Expiry
		f.log.Debug(log.NewEvent(log.AuthEventType, log.McpComponent).
			WithMessage("Cache synchronization: using newer token found in cache").
			WithArg("endpoint", f.p.Config().BaseURL))
	}
	ts := f.ts
	f.mx.Unlock()

	// Retrieve a token from the underlying source. This will trigger a refresh if the token is expired.
	token, err := ts.Token()
	if err != nil {
		return nil, err
	}

	// If the token was refreshed (or if it's the first time we're seeing it), update the cache.
	f.mx.Lock()
	defer f.mx.Unlock()

	// We need to re-fetch from cache to be absolutely sure we're not overwriting something newer
	// that might have landed while we were waiting for ts.Token().
	// But usually, since we just refreshed, our 'token' is the newest.

	if !hasExistingToken || cachedToken.AccessToken != token.AccessToken || cachedToken.RefreshToken != token.RefreshToken {
		f.expiry = token.Expiry

		// Map the oauth2.Token back to our serializable TokenResult.
		tr := TokenResult{
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
			Expiry:       token.Expiry,
			TokenType:    token.TokenType,
			ClientID:     f.p.Config().ClientID,
			Host:         f.p.Config().BaseURL,
		}

		f.cache.Store(f.p.Config(), tr)

		// Persist the updated cache to its storage backend.
		if err = f.cache.Save(context.Background()); err != nil {
			f.log.Err(log.NewEvent(log.ErrorEventType, log.McpComponent).
				WithMessage("Persistence error: failed to save refreshed token to cache").
				WithErr(err).
				WithArg("endpoint", f.p.Config().BaseURL))
		} else {
			f.log.Debug(log.NewEvent(log.AuthEventType, log.McpComponent).
				WithMessage("Cache updated: refreshed token successfully persisted").
				WithArg("endpoint", f.p.Config().BaseURL))
		}
	} else {
		f.log.Debug(log.NewEvent(log.AuthEventType, log.McpComponent).
			WithMessage("Token reuse: current token is still valid").
			WithArg("endpoint", f.p.Config().BaseURL))
	}

	return token, nil
}
