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

	"github.com/theirish81/frags/httpfactory"
)

// StaticTokenProvider is an AuthProvider for situations where you already have
// a token (e.g. a GitHub PAT, a saved access token from a previous OAuth run,
// or a service-account API key). No browser flow is performed.
//
// Example:
//
//	provider := NewStaticTokenProvider("ghp_abc123", "")
//	httpClient, err := provider.Authenticate(ctx)
type StaticTokenProvider struct {
	accessToken  string
	refreshToken string
	tokenType    string
	headers      http.Header
	expiry       time.Time
}

// NewStaticTokenProvider returns an AuthProvider that injects the given access
// token as a Bearer header. refreshToken and expiry are purely informational —
// no automatic refresh is attempted (use OAuthProvider for that).
func NewStaticTokenProvider(accessToken, refreshToken string, headers http.Header) *StaticTokenProvider {
	hx := http.Header{}
	if headers != nil {
		hx = headers
	}
	return &StaticTokenProvider{
		accessToken:  accessToken,
		refreshToken: refreshToken,
		tokenType:    "Bearer",
		headers:      hx,
	}
}

// Authenticate implements AuthProvider for static token provider
func (p *StaticTokenProvider) Authenticate(_ context.Context) (*http.Client, error) {
	h := p.headers.Clone()
	h.Set("Authorization", "Bearer "+p.accessToken)
	return httpfactory.Instance.Builder().WithHeaders(h).Build()
}

// Token implements AuthProvider.
func (p *StaticTokenProvider) Token() (TokenResult, error) {
	return TokenResult{
		AccessToken:  p.accessToken,
		RefreshToken: p.refreshToken,
		TokenType:    p.tokenType,
		Expiry:       p.expiry,
	}, nil
}
