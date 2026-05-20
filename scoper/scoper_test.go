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
package scoper

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper: marshal a node to string
func marshalNode(t *testing.T, node *KnowledgeNode) string {
	t.Helper()
	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	require.NoError(t, enc.Encode(node))
	require.NoError(t, enc.Flush())
	return buf.String()
}

// helper: assert valid XML (parseable)
func assertValidXML(t *testing.T, s string) {
	t.Helper()
	dec := xml.NewDecoder(strings.NewReader(s))
	for {
		_, err := dec.Token()
		if err != nil {
			// io.EOF is expected at end
			assert.EqualError(t, err, "EOF", "XML should be valid and complete")
			return
		}
	}
}

func TestEmptyNode(t *testing.T) {
	node := &KnowledgeNode{NodeName: "item"}
	out := marshalNode(t, node)
	assert.Equal(t, "<item></item>", strings.TrimSpace(out))
}

func TestNodeWithContent(t *testing.T) {
	node := &KnowledgeNode{
		NodeName: "item",
		Content:  "hello world",
	}
	out := marshalNode(t, node)
	assert.Contains(t, out, "<![CDATA[ hello world ]]>")
	assert.Contains(t, out, "<item")
	assert.Contains(t, out, "</item>")
}

func TestNodeWithChildren(t *testing.T) {
	node := &KnowledgeNode{
		NodeName: "parent",
		Children: []*KnowledgeNode{
			{NodeName: "child"},
		},
	}
	out := marshalNode(t, node)
	assert.Contains(t, out, "<parent>")
	assert.Contains(t, out, "<child></child>")
	assert.Contains(t, out, "</parent>")
}

func TestAllAttributes(t *testing.T) {
	node := &KnowledgeNode{
		NodeName:        "item",
		NameAttr:        "myName",
		DescriptionAttr: "myDesc",
		ContentTypeAttr: "text/plain",
	}
	out := marshalNode(t, node)
	assert.Contains(t, out, `name="myName"`)
	assert.Contains(t, out, `description="myDesc"`)
	assert.Contains(t, out, `contentType="text/plain"`)
}

func TestNoAttributesWhenEmpty(t *testing.T) {
	node := &KnowledgeNode{NodeName: "item"}
	out := marshalNode(t, node)
	assert.NotContains(t, out, "name=")
	assert.NotContains(t, out, "description=")
	assert.NotContains(t, out, "contentType=")
}

func TestPartialAttributes(t *testing.T) {
	node := &KnowledgeNode{
		NodeName: "item",
		NameAttr: "onlyName",
	}
	out := marshalNode(t, node)
	assert.Contains(t, out, `name="onlyName"`)
	assert.NotContains(t, out, "description=")
	assert.NotContains(t, out, "contentType=")
}

func TestAttributeSpecialChars(t *testing.T) {
	node := &KnowledgeNode{
		NodeName:        "item",
		NameAttr:        `quotes"and'apostrophes`,
		DescriptionAttr: "less<than>greater",
	}
	out := marshalNode(t, node)
	// xml.Encoder should escape these in attributes
	assert.NotContains(t, out, `"quotes"and'`)
	assertValidXML(t, out)
}

func TestContentWithHTMLTags(t *testing.T) {
	node := &KnowledgeNode{
		NodeName: "item",
		Content:  "<p>Hello <b>world</b></p>",
	}
	out := marshalNode(t, node)
	assert.Contains(t, out, "<![CDATA[ <p>Hello <b>world</b></p> ]]>")
}

func TestContentWithAmpersands(t *testing.T) {
	node := &KnowledgeNode{
		NodeName: "item",
		Content:  "fish & chips",
	}
	out := marshalNode(t, node)
	// inside CDATA, & must NOT be escaped
	assert.Contains(t, out, "<![CDATA[ fish & chips ]]>")
	assert.NotContains(t, out, "&amp;")
}

func TestContentWithXMLEntities(t *testing.T) {
	node := &KnowledgeNode{
		NodeName: "item",
		Content:  `<tag attr="val">text</tag>`,
	}
	out := marshalNode(t, node)
	assert.Contains(t, out, "<![CDATA[")
	// raw < and > must survive inside CDATA
	assert.Contains(t, out, "<tag")
	assert.NotContains(t, out, "&lt;")
}

func TestContentWithNewlines(t *testing.T) {
	node := &KnowledgeNode{
		NodeName: "item",
		Content:  "line one\nline two\nline three",
	}
	out := marshalNode(t, node)
	assert.Contains(t, out, "line one\nline two\nline three")
}

func TestContentWithUnicode(t *testing.T) {
	node := &KnowledgeNode{
		NodeName: "item",
		Content:  "日本語 & émojis 🎉",
	}
	out := marshalNode(t, node)
	assert.Contains(t, out, "日本語 & émojis 🎉")
}

func TestEmptyContent(t *testing.T) {
	// empty Content should NOT emit a CDATA block
	node := &KnowledgeNode{NodeName: "item", Content: ""}
	out := marshalNode(t, node)
	assert.NotContains(t, out, "CDATA")
}

func TestClosingTagNotDoubled(t *testing.T) {
	node := &KnowledgeNode{
		NodeName: "item",
		Content:  "some content",
	}
	out := marshalNode(t, node)
	assert.Equal(t, 1, strings.Count(out, "</item>"), "closing tag must appear exactly once")
}

func TestClosingTagNotDoubledDeep(t *testing.T) {
	node := &KnowledgeNode{
		NodeName: "root",
		Children: []*KnowledgeNode{
			{NodeName: "child", Content: "text"},
			{NodeName: "child", Content: "more text"},
		},
	}
	out := marshalNode(t, node)
	assert.Equal(t, 1, strings.Count(out, "</root>"))
	assert.Equal(t, 2, strings.Count(out, "</child>"))
}

func TestDeepNesting(t *testing.T) {
	node := &KnowledgeNode{
		NodeName: "l1",
		Children: []*KnowledgeNode{
			{
				NodeName: "l2",
				Children: []*KnowledgeNode{
					{
						NodeName: "l3",
						Content:  "deep",
					},
				},
			},
		},
	}
	out := marshalNode(t, node)
	assert.Contains(t, out, "<l1>")
	assert.Contains(t, out, "<l2>")
	assert.Contains(t, out, "<![CDATA[ deep ]]>")
	assert.Contains(t, out, "</l3>")
	assert.Contains(t, out, "</l2>")
	assert.Contains(t, out, "</l1>")
	// order matters — l3 closes before l2, l2 before l1
	assert.Less(t, strings.Index(out, "</l3>"), strings.Index(out, "</l2>"))
	assert.Less(t, strings.Index(out, "</l2>"), strings.Index(out, "</l1>"))
}

func TestMultipleSiblings(t *testing.T) {
	node := &KnowledgeNode{
		NodeName: "root",
		Children: []*KnowledgeNode{
			{NodeName: "a", Content: "alpha"},
			{NodeName: "b", Content: "beta"},
			{NodeName: "c"},
		},
	}
	out := marshalNode(t, node)
	assert.Contains(t, out, "<![CDATA[ alpha ]]>")
	assert.Contains(t, out, "<![CDATA[ beta ]]>")
	assert.Contains(t, out, "<c></c>")
	// sibling order preserved
	assert.Less(t, strings.Index(out, "<a"), strings.Index(out, "<b"))
	assert.Less(t, strings.Index(out, "<b"), strings.Index(out, "<c"))
}

func TestChildWithAttributes(t *testing.T) {
	node := &KnowledgeNode{
		NodeName: "root",
		Children: []*KnowledgeNode{
			{
				NodeName:        "child",
				NameAttr:        "kidName",
				ContentTypeAttr: "text/html",
				Content:         "<b>bold</b>",
			},
		},
	}
	out := marshalNode(t, node)
	assert.Contains(t, out, `name="kidName"`)
	assert.Contains(t, out, `contentType="text/html"`)
	assert.Contains(t, out, "<![CDATA[ <b>bold</b> ]]>")
}

func TestOutputIsWellFormedXML(t *testing.T) {
	node := &KnowledgeNode{
		NodeName:        "root",
		NameAttr:        "test",
		DescriptionAttr: "a & b",
		Children: []*KnowledgeNode{
			{NodeName: "child", Content: "<em>hi</em>"},
			{NodeName: "empty"},
		},
	}
	out := marshalNode(t, node)
	assertValidXML(t, out)
}
