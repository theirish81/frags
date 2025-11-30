package frags

import (
	"errors"
	"os"
	"path/filepath"
)

// ResourceData is a piece of data the LLM can use.
type ResourceData struct {
	Identifier string
	Data       []byte
	MediaType  string
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
	resource.Data = fileData
	return resource, nil
}

type BytesLoader struct {
	resources map[string]ResourceData
}

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
		Identifier: identifier,
		MediaType:  GetMediaType(identifier),
		Data:       make([]byte, 0),
	}, nil
}
