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
)

type OauthCache interface {
	Get(key string) (*TokenResult, bool)
	Store(key string, item TokenResult)
	Save(ctx context.Context) error
}

type FsOauthCache struct {
	filepath string
	Items    map[string]*TokenResult `json:"items,omitempty"`
}

func NewFsOauthCache(filepath string) (*FsOauthCache, error) {
	cacheInstance := FsOauthCache{
		filepath: filepath,
		Items:    make(map[string]*TokenResult),
	}
	if data, err := os.ReadFile(filepath); err == nil {
		if err := json.Unmarshal(data, &cacheInstance); err != nil {
			return nil, err
		}
	}
	return &cacheInstance, nil
}

func (c *FsOauthCache) Get(key string) (*TokenResult, bool) {
	if item, ok := c.Items[key]; ok {
		return item, true
	}
	return nil, false
}

func (c *FsOauthCache) Store(key string, item TokenResult) {
	c.Items[key] = &item
}

func (c *FsOauthCache) Save(_ context.Context) error {
	data, err := json.MarshalIndent(c, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.filepath, data, 0600)
}

type NopCache struct {
}

func (c *NopCache) Get(key string) (*TokenResult, bool) {
	return nil, false
}

func (c *NopCache) Store(key string, item TokenResult) {
}

func (c *NopCache) Save(_ context.Context) error {
	return nil
}
