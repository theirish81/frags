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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
)

// ResourceData is a piece of data the LLM can use.
type ResourceData struct {
	Identifier        string
	MediaType         string
	ByteContent       []byte
	StructuredContent *any
	In                ResourceDestination
	Var               *string
}

// SetContent sets the content of the resource data. If the input is structured content, then the value is stored in
// the StructuredContent field and is JSON-marshaled to the ByteContent field. If it's a raw type or a slice of bytes,
// then the ByteContent field is set directly and StructuredContent is nil. The objective is when StructuredContent
// has value, then ByteContent contains its JSON representation, but when a raw value needs to be stored, then
// StructuredContent is nil and ByteContent contains the raw value.
func (r *ResourceData) SetContent(data any) error {
	switch toConcreteValue(reflect.ValueOf(data)).Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		switch t := data.(type) {
		// In case this is an array of bytes, we keep it as byte response, there's no structure in this case
		case []uint8:
			r.StructuredContent = nil
			r.ByteContent = t
		default:
			// otherwise we assume this to be a structured content
			var err error
			r.StructuredContent = &data
			if r.ByteContent, err = json.Marshal(data); err != nil {
				return err
			}
		}
	default:
		// any other data type, and we hope for the best
		r.StructuredContent = nil
		r.ByteContent = []byte(fmt.Sprintf("%v", data))
	}
	return nil
}

// ResourceLoader is a generic interface for loading resources.
type ResourceLoader interface {
	LoadResource(identifier string, params map[string]string) (ResourceData, error)
}

// FileResourceLoader loads resources from the file system.
type FileResourceLoader struct {
	basePath string
}

// NewFileResourceLoader creates a new FileResourceLoader.
func NewFileResourceLoader(basePath string) *FileResourceLoader {
	return &FileResourceLoader{basePath: basePath}
}

// LoadResource loads a resource from the file system.
func (l *FileResourceLoader) LoadResource(identifier string, _ map[string]string) (ResourceData, error) {
	resource := ResourceData{Identifier: identifier, MediaType: GetMediaType(identifier)}
	fileData, err := os.ReadFile(filepath.Join(l.basePath, identifier))
	if err != nil {
		return ResourceData{}, err
	}
	resource.ByteContent = fileData
	return resource, nil
}

// BytesLoader "loads" and returns resources that have already been preloaded into memory
type BytesLoader struct {
	resources map[string]ResourceData
}

// NewBytesLoader creates a new BytesLoader.
func NewBytesLoader() *BytesLoader {
	return &BytesLoader{resources: make(map[string]ResourceData)}
}

// SetResource sets a resource in the loader's internal map.
func (l *BytesLoader) SetResource(resourceData ResourceData) {
	l.resources[resourceData.Identifier] = resourceData
}

// LoadResource returns a resource from the loader's internal map.
func (l *BytesLoader) LoadResource(identifier string, _ map[string]string) (ResourceData, error) {
	if resource, ok := l.resources[identifier]; ok {
		return resource, nil
	} else {
		return ResourceData{}, errors.New("resource not found: " + identifier)
	}
}

// MultiResourceLoader loads resources from multiple loaders, based on a selector parameter.
type MultiResourceLoader struct {
	loaders map[string]ResourceLoader
}

// NewMultiResourceLoader creates a new MultiResourceLoader.
func NewMultiResourceLoader() *MultiResourceLoader {
	return &MultiResourceLoader{loaders: make(map[string]ResourceLoader)}
}

// SetLoader sets a loader for a specific identifier.
func (l *MultiResourceLoader) SetLoader(identifier string, loader ResourceLoader) {
	l.loaders[identifier] = loader
}

// LoadResource loads a resource from a specific loader, based on a selector parameter.
func (l *MultiResourceLoader) LoadResource(identifier string, params map[string]string) (ResourceData, error) {
	loaderSelector, ok := params["loader"]
	if !ok {
		return ResourceData{Identifier: identifier}, errors.New("no loader selector provided")
	}
	if loader, ok := l.loaders[loaderSelector]; ok {
		return loader.LoadResource(identifier, params)
	}
	return ResourceData{Identifier: identifier}, errors.New("no loader found for resource")
}

// DummyResourceLoader is a dummy resource loader that returns empty resources, for testing purposes.
type DummyResourceLoader struct{}

// NewDummyResourceLoader creates a new DummyResourceLoader.
func NewDummyResourceLoader() *DummyResourceLoader {
	return &DummyResourceLoader{}
}

// LoadResource returns an empty resource.
func (l *DummyResourceLoader) LoadResource(identifier string, params map[string]string) (ResourceData, error) {
	return ResourceData{
		Identifier:  identifier,
		MediaType:   GetMediaType(identifier),
		ByteContent: make([]byte, 0),
	}, nil
}
