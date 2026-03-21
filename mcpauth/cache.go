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
)

type OauthCache interface {
	Get(key *OAuthProviderConfig) (*TokenResult, bool)
	Store(key *OAuthProviderConfig, item TokenResult)
	Save(ctx context.Context) error
}

type FsOauthCache struct {
	filepath string
	Items    map[string]*TokenResult `json:"items,omitempty"`
	mx       sync.Mutex
}

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

func (c *FsOauthCache) Get(key *OAuthProviderConfig) (*TokenResult, bool) {
	c.mx.Lock()
	defer c.mx.Unlock()
	if item, ok := c.Items[key.McpFingerprint()]; ok {
		return item, true
	}
	return nil, false
}

func (c *FsOauthCache) Store(key *OAuthProviderConfig, item TokenResult) {
	c.mx.Lock()
	defer c.mx.Unlock()
	c.Items[key.McpFingerprint()] = &item
}

func (c *FsOauthCache) Save(_ context.Context) error {
	c.mx.Lock()
	defer c.mx.Unlock()
	data, err := json.MarshalIndent(c, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.filepath, data, 0600)
}

type InMemoryCache struct {
	Items map[string]*TokenResult
	mx    sync.Mutex
}

func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{
		Items: make(map[string]*TokenResult),
	}
}

func (c *InMemoryCache) RawGet(name string) (*TokenResult, bool) {
	c.mx.Lock()
	defer c.mx.Unlock()
	if item, ok := c.Items[name]; ok {
		return item, true
	}
	return nil, false
}

func (c *InMemoryCache) Get(key *OAuthProviderConfig) (*TokenResult, bool) {
	return c.RawGet(key.Name)
}

func (c *InMemoryCache) RawStore(name string, item TokenResult) {
	c.mx.Lock()
	defer c.mx.Unlock()
	c.Items[name] = &item
}

func (c *InMemoryCache) Store(key *OAuthProviderConfig, item TokenResult) {
	c.RawStore(key.Name, item)
}

func (c *InMemoryCache) Save(_ context.Context) error {
	return nil
}

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
