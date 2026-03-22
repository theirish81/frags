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
	"time"

	"github.com/theirish81/frags/log"
	"golang.org/x/oauth2"
)

// FragsTokenSource is a wrapper around the oauth2.TokenSource that uses OauthCache to:
// 1) serve cached tokens is the cached tokens are valid and more recent than the one it carries
// 2) refresh the token if it is expired
// c) cache the refreshed token
type FragsTokenSource struct {
	ts        oauth2.TokenSource
	cache     OauthCache
	conf      *oauth2.Config
	p         *OAuthProvider
	resources *DiscoveryResources
	log       log.StreamerLogger
	expiry    time.Time
}

// NewFragsTokenSource constructs a new FragsTokenSource. It requires the OAuthProvider and the DiscoveryResources
// because it has caching and self-refreshing capabilities
func NewFragsTokenSource(token *oauth2.Token, p *OAuthProvider, resources *DiscoveryResources, cache OauthCache, log log.StreamerLogger) *FragsTokenSource {
	conf := p.OauthConfig(resources.AuthServerMetadata, p.cfg.clientID(), p.cfg.clientSecret(), nil)
	return &FragsTokenSource{ts: conf.TokenSource(context.Background(), token),
		expiry: token.Expiry, conf: conf, cache: cache, p: p, log: log}
}

// Token returns the most recent token, either from cache, from its internal state, or by refreshing. It will update
// the case in case of a refresh.
func (f *FragsTokenSource) Token() (*oauth2.Token, error) {
	cachedToken, hasExistingToken := f.cache.Get(f.p.Config())
	if hasExistingToken && cachedToken.Expiry.After(f.expiry) {
		// if the cache has a more recent token, it means the cache has been updated by someone else. Therefore,
		// it becomes the source of truth.
		f.ts = f.conf.TokenSource(context.Background(), &oauth2.Token{AccessToken: cachedToken.AccessToken, RefreshToken: cachedToken.RefreshToken, Expiry: cachedToken.Expiry})
		f.log.Debug(log.NewEvent(log.AuthEventType, log.McpComponent).WithMessage("cache had a more recent token. Using it").WithArg("endpoint", f.p.Config().MCPEndpoint))
	}
	// we call the inner TokenSource Token().
	token, err := f.ts.Token()
	if err != nil {
		return nil, err
	}
	if !hasExistingToken || cachedToken.AccessToken != token.AccessToken || cachedToken.RefreshToken != token.RefreshToken {
		// if:
		// a) the cache had no existing token
		// b) the cache had an existing token, but it was different from the one we got from the TokenSource (it got
		// refreshed)
		// then we update the cache
		f.expiry = token.Expiry
		f.cache.Store(f.p.Config(), TokenResult{
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
			Expiry:       token.Expiry,
			ClientID:     f.p.Config().clientID(),
			Host:         f.p.Config().MCPEndpoint,
		})
		if err = f.cache.Save(context.Background()); err != nil {
			f.log.Err(log.NewEvent(log.ErrorEventType, log.McpComponent).WithMessage("error caching token").WithErr(err).WithArg("endpoint", f.p.Config().MCPEndpoint))
		} else {
			f.log.Debug(log.NewEvent(log.AuthEventType, log.McpComponent).WithMessage("token cached updated").WithArg("endpoint", f.p.Config().MCPEndpoint))
		}
	} else {
		f.log.Debug(log.NewEvent(log.AuthEventType, log.McpComponent).WithMessage("token reused").WithArg("endpoint", f.p.Config().MCPEndpoint))
	}
	return token, nil
}
