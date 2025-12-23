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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileResourceLoader(t *testing.T) {
	t.Run("successfully loads an existing file", func(t *testing.T) {
		loader := NewFileResourceLoader("./test_data")
		identifier := "story.txt"
		resource, err := loader.LoadResource(identifier, nil)

		expectedContent, readErr := os.ReadFile(filepath.Join("./test_data", identifier))
		assert.NoError(t, readErr)

		assert.NoError(t, err)
		assert.Equal(t, identifier, resource.Identifier)
		assert.Equal(t, expectedContent, resource.Data)
		assert.Equal(t, MediaText, resource.MediaType)
	})

	t.Run("returns an error for a non-existent file", func(t *testing.T) {
		loader := NewFileResourceLoader("./test_data")
		identifier := "non_existent_file.txt"
		_, err := loader.LoadResource(identifier, nil)

		assert.Error(t, err)
	})
}

func TestBytesLoader(t *testing.T) {
	t.Run("successfully loads a pre-set resource", func(t *testing.T) {
		loader := NewBytesLoader()
		expectedResource := ResourceData{
			Identifier: "in-memory-resource",
			Data:       []byte("This is some data in memory."),
			MediaType:  MediaText,
		}
		loader.SetResource(expectedResource)

		resource, err := loader.LoadResource("in-memory-resource", nil)
		assert.NoError(t, err)
		assert.Equal(t, expectedResource, resource)
	})

	t.Run("returns an error for a resource that has not been set", func(t *testing.T) {
		loader := NewBytesLoader()
		_, err := loader.LoadResource("unknown-resource", nil)

		assert.Error(t, err)
	})
}

func TestMultiResourceLoader(t *testing.T) {
	fileLoader := NewFileResourceLoader("./test_data")
	bytesLoader := NewBytesLoader()
	inMemoryResource := ResourceData{Identifier: "ram.txt", Data: []byte("data from ram"), MediaType: MediaText}
	bytesLoader.SetResource(inMemoryResource)

	multiLoader := NewMultiResourceLoader()
	multiLoader.SetLoader("fs", fileLoader)
	multiLoader.SetLoader("mem", bytesLoader)

	t.Run("successfully loads from file loader", func(t *testing.T) {
		resource, err := multiLoader.LoadResource("story.txt", map[string]string{"loader": "fs"})
		expectedContent, _ := os.ReadFile(filepath.Join("./test_data", "story.txt"))

		assert.NoError(t, err)
		assert.Equal(t, "story.txt", resource.Identifier)
		assert.Equal(t, expectedContent, resource.Data)
	})

	t.Run("successfully loads from bytes loader", func(t *testing.T) {
		resource, err := multiLoader.LoadResource("ram.txt", map[string]string{"loader": "mem"})

		assert.NoError(t, err)
		assert.Equal(t, inMemoryResource, resource)
	})

	t.Run("returns error when no loader selector is provided", func(t *testing.T) {
		_, err := multiLoader.LoadResource("story.txt", nil)
		assert.Error(t, err)
	})

	t.Run("returns error when a non-existent loader is selected", func(t *testing.T) {
		_, err := multiLoader.LoadResource("story.txt", map[string]string{"loader": "non-existent"})
		assert.Error(t, err)
	})
}
