package frags

import (
	"encoding/json"
	"sync"
)

// safeUnmarshalDataStructure is a thread-safe version of json.Unmarshal, and is used to unmarshal the runner's
// data structure.
func (r *Runner[T]) safeUnmarshalDataStructure(data []byte) error {
	r.marshalingMutex.Lock()
	defer r.marshalingMutex.Unlock()
	return MergeJSONInto(r.dataStructure, data)
}

// safeMarshalDataStructure is a thread-safe version of json.Marshal, and is used to marshal the runner's
// data structure.
func (r *Runner[T]) safeMarshalDataStructure(indent bool) ([]byte, error) {
	r.marshalingMutex.Lock()
	defer r.marshalingMutex.Unlock()
	if r.kFormat {
		return []byte(ToKFormat(r.dataStructure)), nil
	}
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
