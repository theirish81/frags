/*
 * Copyright (C) 2025 Simone Pezzano
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

package frags

import (
	"encoding/json"
	"sync"
)

// safeUnmarshalDataStructure unmarshals data into the runner's data structure in a thread-safe manner.
func (r *Runner[T]) safeUnmarshalDataStructure(data []byte) error {
	r.marshalingMutex.Lock()
	defer r.marshalingMutex.Unlock()
	return MergeJSONInto(r.dataStructure, data)
}

// safeMarshalDataStructure marshals the runner's data structure in a thread-safe manner. Note that the output
// could either be JSON or K format, depending on the runner's configuration.
func (r *Runner[T]) safeMarshalDataStructure(indent bool) ([]byte, error) {
	r.marshalingMutex.Lock()
	defer r.marshalingMutex.Unlock()
	if indent {
		return json.MarshalIndent(r.dataStructure, "", "  ")
	}
	return json.Marshal(r.dataStructure)
}

type SafeMap[K comparable, V any] struct {
	mu   sync.RWMutex
	data map[K]V
}

func NewSafeMap[K comparable, V any]() *SafeMap[K, V] {
	return &SafeMap[K, V]{
		data: make(map[K]V),
	}
}

// Store is strongly typed: K and V are enforced at compile time.
func (sm *SafeMap[K, V]) Store(key K, value V) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.data[key] = value
}

func (sm *SafeMap[K, V]) Load(key K) (V, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	value, ok := sm.data[key]
	return value, ok
}

func (sm *SafeMap[K, V]) Iter() map[K]V {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	cpy := make(map[K]V)
	for k, v := range sm.data {
		cpy[k] = v
	}
	return cpy
}
