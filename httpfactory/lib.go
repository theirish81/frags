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

package httpfactory

import (
	"net/http"
	"time"

	"github.com/theirish81/sesat2"
)

type Factory interface {
	Builder() *sesat2.Builder
	HttpClient() *http.Client
}

var Instance Factory = &SimpleFactory{}

type SimpleFactory struct{}

const (
	DefaultDialTimeout           = 10 * time.Second
	DefaultKeepAliveInterval     = 30 * time.Second
	DefaultTLSHandshakeTimeout   = 10 * time.Second
	DefaultResponseHeaderTimeout = 15 * time.Second
	DefaultIdleConnTimeout       = 90 * time.Second
	DefaultClientTotalTimeout    = 5 * time.Minute
)

func (f *SimpleFactory) HttpClient() *http.Client {
	c, _ := f.Builder().Build()
	return c
}

func (f *SimpleFactory) Builder() *sesat2.Builder {
	return sesat2.New().WithTimeout(DefaultClientTotalTimeout).WithDialTimeout(DefaultDialTimeout).
		WithKeepAlive(DefaultKeepAliveInterval).WithTLSHandshakeTimeout(DefaultTLSHandshakeTimeout).
		WithResponseHeaderTimeout(DefaultResponseHeaderTimeout).WithIdleConnTimeout(DefaultIdleConnTimeout)
}
