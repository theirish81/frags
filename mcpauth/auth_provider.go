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
	"net/http"
	"time"

	"github.com/theirish81/frags/log"
	"golang.org/x/oauth2"
)

// TokenResult holds the tokens obtained after a successful authentication.
// Store these to avoid re-running the auth sequence if reused
type TokenResult struct {
	Host         string    `json:"host" yaml:"host"`
	ClientID     string    `json:"client_id"`
	AccessToken  string    `json:"access_token" yaml:"access_token"`
	RefreshToken string    `json:"refresh_token" yaml:"refresh_token"`
	TokenType    string    `json:"token_type" yaml:"token_type"`
	Expiry       time.Time `json:"expiry" yaml:"expiry"`
}

func (t *TokenResult) ToOauth2Token() *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		TokenType:    t.TokenType,
		Expiry:       t.Expiry,
	}
}

func (t *TokenResult) FromOauth2Token(tok *oauth2.Token) *TokenResult {
	t.AccessToken = tok.AccessToken
	t.RefreshToken = tok.RefreshToken
	t.TokenType = tok.TokenType
	t.Expiry = tok.Expiry
	return t
}

// IsExpired reports whether the access token has expired (or will within grace).
func (t *TokenResult) IsExpired(grace time.Duration) bool {
	if t.Expiry.IsZero() {
		return false
	}
	return time.Now().Add(grace).After(t.Expiry)
}

type AuthProvider interface {
	// Authenticate runs the full auth flow and returns a ready-to-use *http.Client
	// whose Transport injects the bearer token (and transparently refreshes it).
	Authenticate(ctx context.Context) (*http.Client, error)

	// Token returns the current TokenResult
	Token() (TokenResult, error)
}

type GenericOauthProvider interface {
	AuthProvider
	New(config OAuthProviderConfig, logger *log.StreamerLogger) GenericOauthProvider
	WithCache(tokenCache OauthCache) GenericOauthProvider
}
