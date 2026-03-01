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
)

// TokenResult holds the tokens obtained after a successful authentication.
// Store these to avoid re-running the auth sequence if reused
type TokenResult struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	Expiry       time.Time
}

// IsExpired reports whether the access token has expired (or will within grace).
func (t TokenResult) IsExpired(grace time.Duration) bool {
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
	Token() TokenResult
}

type GenericOauthProvider interface {
	AuthProvider
	New(config OAuthProviderConfig, logger *log.StreamerLogger) GenericOauthProvider
}
