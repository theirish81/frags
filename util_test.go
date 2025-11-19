package frags

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const json1 = `{"p1": "v1", "p2": "v2"}`
const json2 = `{"p3": "v3", "p4": "v4"}`

func TestProgMap_UnmarshalJSON(t *testing.T) {
	progmap := ProgMap{}
	err := progmap.UnmarshalJSON([]byte(json1))
	assert.Nil(t, err)
	err = progmap.UnmarshalJSON([]byte(json2))
	assert.Nil(t, err)
	assert.Equal(t, ProgMap{"p1": "v1", "p2": "v2", "p3": "v3", "p4": "v4"}, progmap)
}
