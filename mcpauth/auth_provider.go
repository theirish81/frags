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
 * along with this program.  See the LICENSE file for more details.
 */

// Package mcpauth provides authentication mechanisms for Model Context Protocol (MCP) servers.
// It supports various authentication flows, including 3-legged OAuth2 and static token injection.
package mcpauth

import (
	"context"
	"net/http"

	"github.com/theirish81/frags/log"
)

// AuthProvider defines the standard interface for all authentication mechanisms in frags.
// Any component that needs to provide an authenticated http.Client must implement this.
type AuthProvider interface {
	// Authenticate performs the necessary authentication steps (e.g., discovery, token exchange,
	// cache lookup) and returns an *http.Client configured with a Transport that automatically
	// injects the required authentication headers (and handles transparent token refresh if supported).
	// The provided context should be used for all network operations.
	Authenticate(ctx context.Context) (*http.Client, error)

	// Token returns the current state of the authentication tokens.
	// This is useful for inspection, debugging, or persisting tokens for future sessions.
	// If no token is currently available, it may return an error or a zero-valued TokenResult.
	Token() (TokenResult, error)
}

// GenericOauthProvider extends AuthProvider with methods specifically required for
// managing OAuth2-based authentication providers, such as creating new instances
// with specific configurations or attaching token caches.
type GenericOauthProvider interface {
	AuthProvider

	// Config returns the current configuration of the provider.
	Config() *OAuthProviderConfig

	// New creates a new instance of the provider using the supplied OAuthProviderConfig
	// and StreamerLogger. This is typically used to spawn per-server authentication
	// handlers from a template or prototype provider.
	New(config OAuthProviderConfig, logger *log.StreamerLogger) GenericOauthProvider

	// WithCache attaches a specific OauthCache implementation to the provider.
	// This allows the provider to persist and reuse tokens across sessions.
	WithCache(tokenCache OauthCache) GenericOauthProvider
}

// DerefOr is a utility function that safely dereferences a string pointer.
// If the pointer is nil, it returns the provided fallback string.
// This is frequently used when mapping configuration structs (which often use pointers
// for optional fields) to library-specific configuration types.
func DerefOr(p *string, fallback string) string {
	if p != nil {
		return *p
	}
	return fallback
}
