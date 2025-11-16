package frags

import (
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

type MultiResourceLoader struct {
	loaders []ResourceLoader
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
