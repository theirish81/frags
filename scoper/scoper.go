/*
 * Copyright (C) 2026 Simone Pezzano
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
	"encoding/xml"
)

type KnowledgeNode struct {
	NodeName        string
	NameAttr        string
	DescriptionAttr string
	ContentTypeAttr string
	Content         string
	Children        []*KnowledgeNode
}

func (k *KnowledgeNode) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = k.NodeName
	start.Attr = make([]xml.Attr, 0)
	if k.NameAttr != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "name"}, Value: k.NameAttr})
	}
	if k.DescriptionAttr != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "description"}, Value: k.DescriptionAttr})
	}
	if k.ContentTypeAttr != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "contentType"}, Value: k.ContentTypeAttr})
	}
	if k.Content != "" {
		_ = e.EncodeElement(struct {
			S string `xml:",innerxml"`
		}{
			S: "<![CDATA[ " + k.Content + " ]]>",
		}, start)
	} else {
		_ = e.EncodeToken(start)
		for _, child := range k.Children {
			_ = child.MarshalXML(e, xml.StartElement{})
		}
		_ = e.EncodeToken(start.End())
	}
	return nil
}

func (k *KnowledgeNode) XML() string {
	data, _ := xml.MarshalIndent(k, "", " ")
	return string(data)
}

func (k *KnowledgeNode) String() string {
	return k.XML()
}
func (k *KnowledgeNode) AppendChild(child *KnowledgeNode) {
	k.Children = append(k.Children, child)
}

func (k *KnowledgeNode) Name(name string) *KnowledgeNode {
	k.NameAttr = name
	return k
}
func (k *KnowledgeNode) Description(description string) *KnowledgeNode {
	k.DescriptionAttr = description
	return k
}

func (k *KnowledgeNode) ContentType(typ string) *KnowledgeNode {
	k.ContentTypeAttr = typ
	return k
}

func Node(nodeName string, content string) *KnowledgeNode {
	return &KnowledgeNode{
		NodeName: nodeName,
		Content:  content,
	}
}
