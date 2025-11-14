package frags

import (
	"os"
	"path/filepath"
)

type Resource struct {
	Identifier string
	Data       []byte
	MediaType  string
}

type ResourceLoader interface {
	LoadResource(identifier string) (Resource, error)
}

type FileResourceLoader struct {
	basePath string
}

func NewFileResourceLoader(basePath string) *FileResourceLoader {
	return &FileResourceLoader{basePath: basePath}
}

func (l *FileResourceLoader) LoadResource(identifier string) (Resource, error) {
	resource := Resource{Identifier: identifier, MediaType: GetMediaType(identifier)}
	fileData, err := os.ReadFile(filepath.Join(l.basePath, identifier))
	if err != nil {
		return Resource{}, err
	}
	resource.Data = fileData
	return resource, nil
}

type DummyResourceLoader struct{}

func NewDummyResourceLoader() *DummyResourceLoader {
	return &DummyResourceLoader{}
}
func (l *DummyResourceLoader) LoadResource(identifier string) (Resource, error) {
	return Resource{
		Identifier: identifier,
		MediaType:  GetMediaType(identifier),
		Data:       make([]byte, 0),
	}, nil
}
