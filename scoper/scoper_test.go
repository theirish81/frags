package scoper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNode(t *testing.T) {
	node := Node("gino", "").Name("some content").ContentType("application/json").Description("some description")
	node.AppendChild(Node("child1", "some other content"))
	data := node.String()
	assert.Equal(t, "<gino name=\"some content\" description=\"some description\" contentType=\"application/json\">\n <child1><![CDATA[ some other content ]]></child1>\n</gino>", data)
}
