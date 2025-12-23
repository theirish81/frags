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
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSafeMap_StoreLoad(t *testing.T) {
	sm := NewSafeMap[string, int]()

	// Store and Load existing
	sm.Store("key1", 100)
	val, ok := sm.Load("key1")
	assert.True(t, ok)
	assert.Equal(t, 100, val)

	sm.Store("key2", 200)
	val, ok = sm.Load("key2")
	assert.True(t, ok)
	assert.Equal(t, 200, val)

	// Load non-existent
	val, ok = sm.Load("key3")
	assert.False(t, ok)
	assert.Equal(t, 0, val) // Default value for int
}

func TestSafeMap_Iter(t *testing.T) {
	sm := NewSafeMap[string, string]()
	sm.Store("a", "apple")
	sm.Store("b", "banana")
	sm.Store("c", "cherry")

	expected := map[string]string{
		"a": "apple",
		"b": "banana",
		"c": "cherry",
	}

	iterMap := sm.Iter()
	assert.Equal(t, expected, iterMap)

	// Verify it's a copy - modifying iterMap should not affect sm.data
	iterMap["a"] = "apricot"
	val, ok := sm.Load("a")
	assert.True(t, ok)
	assert.Equal(t, "apple", val)
}

func TestSafeMap_ConcurrentAccess(t *testing.T) {
	sm := NewSafeMap[int, int]()
	numWriters := 10
	writesPerWriter := 100

	var wg sync.WaitGroup
	wg.Add(numWriters)

	for i := 0; i < numWriters; i++ {
		go func(writerID int) {
			defer wg.Done()
			for j := 0; j < writesPerWriter; j++ {
				key := writerID*writesPerWriter + j
				sm.Store(key, key*2)
				// Optionally, perform some loads concurrently
				if j%10 == 0 {
					_, _ = sm.Load(key)
				}
			}
		}(i)
	}
	wg.Wait()

	// Verify all writes are present and correct
	expectedTotalKeys := numWriters * writesPerWriter
	iterMap := sm.Iter()
	assert.Equal(t, expectedTotalKeys, len(iterMap))

	for i := 0; i < numWriters; i++ {
		for j := 0; j < writesPerWriter; j++ {
			key := i*writesPerWriter + j
			val, ok := sm.Load(key)
			assert.True(t, ok)
			assert.Equal(t, key*2, val)
		}
	}
}

func TestSafeMap_ConcurrentReads(t *testing.T) {
	sm := NewSafeMap[string, string]()
	sm.Store("pre_set_key", "pre_set_value")

	numReaders := 100
	readsPerReader := 1000

	var wg sync.WaitGroup
	wg.Add(numReaders)

	for i := 0; i < numReaders; i++ {
		go func(readerID int) {
			defer wg.Done()
			for j := 0; j < readsPerReader; j++ {
				val, ok := sm.Load("pre_set_key")
				assert.True(t, ok)
				assert.Equal(t, "pre_set_value", val)
			}
		}(i)
	}
	wg.Wait()
}

func TestSafeMap_MixedTypes(t *testing.T) {
	sm := NewSafeMap[int, string]()
	sm.Store(1, "one")
	sm.Store(2, "two")

	val, ok := sm.Load(1)
	assert.True(t, ok)
	assert.Equal(t, "one", val)

	val, ok = sm.Load(2)
	assert.True(t, ok)
	assert.Equal(t, "two", val)
}

// Test with concurrent writes and reads using string keys for variety
func TestSafeMap_ConcurrentMixedStringKeys(t *testing.T) {
	sm := NewSafeMap[string, int]()
	numGoroutines := 20
	operationsPerGoroutine := 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(gID int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("goroutine_%d_op_%d", gID, j)
				value := gID + j

				// Store operation
				sm.Store(key, value)

				// Load operation for previously stored or random key
				if j%5 == 0 && gID > 0 { // Try to load from other goroutines
					prevGoroutineKey := fmt.Sprintf("goroutine_%d_op_%d", gID-1, j)
					sm.Load(prevGoroutineKey)
				}
				sm.Load(key)
			}
		}(i)
	}
	wg.Wait()

	// Verify final state
	expectedTotalKeys := numGoroutines * operationsPerGoroutine
	finalMap := sm.Iter()
	assert.Equal(t, expectedTotalKeys, len(finalMap))

	for i := 0; i < numGoroutines; i++ {
		for j := 0; j < operationsPerGoroutine; j++ {
			key := fmt.Sprintf("goroutine_%d_op_%d", i, j)
			expectedValue := i + j
			val, ok := sm.Load(key)
			assert.True(t, ok)
			assert.Equal(t, val, expectedValue)
		}
	}
}
