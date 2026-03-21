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

	"github.com/theirish81/frags/log"
	"golang.org/x/oauth2"
)

type FragsTokenSource struct {
	ts    oauth2.TokenSource
	cache OauthCache
	cfg   *OAuthProviderConfig
	log   log.StreamerLogger
}

func NewFragsTokenSource(ts oauth2.TokenSource, cfg *OAuthProviderConfig, cache OauthCache, log log.StreamerLogger) *FragsTokenSource {
	return &FragsTokenSource{ts: ts, cache: cache, cfg: cfg, log: log}
}

func (f *FragsTokenSource) Token() (*oauth2.Token, error) {
	token, err := f.ts.Token()
	if err != nil {
		return nil, err
	}
	oldToken, hadOldToken := f.cache.Get(f.cfg)
	f.cache.Store(f.cfg, TokenResult{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
		ClientID:     f.cfg.clientID(),
		Host:         f.cfg.MCPEndpoint,
	})
	if !hadOldToken || oldToken.AccessToken != token.AccessToken || oldToken.RefreshToken != token.RefreshToken {
		if err = f.cache.Save(context.Background()); err != nil {
			f.log.Err(log.NewEvent(log.ErrorEventType, log.McpComponent).WithMessage("error caching token").WithErr(err).WithArg("endpoint", f.cfg.MCPEndpoint))
		} else {
			f.log.Debug(log.NewEvent(log.AuthEventType, log.McpComponent).WithMessage("token cached updated").WithArg("endpoint", f.cfg.MCPEndpoint))
		}
	} else {
		f.log.Debug(log.NewEvent(log.AuthEventType, log.McpComponent).WithMessage("token reused").WithArg("endpoint", f.cfg.MCPEndpoint))
	}
	return token, nil
}
