package schema

import (
	"testing"

	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/assert"
	"github.com/theirish81/frags/util"
)

func TestFromAny_MapstructureTypeArray(t *testing.T) {
	data := map[string]any{
		"type": []any{"object", "null"},
		"properties": map[string]any{
			"name": map[string]any{
				"type": []any{"string", "null"},
			},
			"age": map[string]any{
				"type": "integer",
			},
		},
	}

	s, err := FromAny(data)
	assert.NoError(t, err)

	assert.Equal(t, Type("object"), s.Type)
	assert.Equal(t, util.Ptr(true), s.Nullable)

	assert.Equal(t, Type("string"), s.Properties["name"].Type)
	assert.Equal(t, util.Ptr(true), s.Properties["name"].Nullable)

	assert.Equal(t, Type("integer"), s.Properties["age"].Type)
	assert.Nil(t, s.Properties["age"].Nullable)
}

func TestFromAny_CopierTypeConverter(t *testing.T) {
	type SourceSchema struct {
		Type       []string
		Nullable   *bool
		Properties map[string]SourceSchema
	}

	src := SourceSchema{
		Type: []string{"object", "null"},
		Properties: map[string]SourceSchema{
			"name": {Type: []string{"string", "null"}},
			"age":  {Type: []string{"integer"}},
		},
	}

	dst := &Schema{}
	err := copier.CopyWithOption(dst, src, copier.Option{
		Converters: CopyConverters(),
	})

	assert.NoError(t, err)
	assert.Equal(t, Type("object"), dst.Type)
	assert.Equal(t, Type("string"), dst.Properties["name"].Type)
	assert.Equal(t, Type("integer"), dst.Properties["age"].Type)
}

func TestCopyConverter_StringToType(t *testing.T) {
	type SourceSchema struct {
		Type string
	}

	src := SourceSchema{Type: "object"}
	dst := &Schema{}

	err := copier.CopyWithOption(dst, src, copier.Option{
		Converters: CopyConverters(),
	})

	assert.NoError(t, err)
	assert.Equal(t, Type("object"), dst.Type)
}
