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
	_ = e.EncodeToken(start)
	if k.Content != "" {
		_ = e.EncodeToken(xml.Directive("[CDATA[ " + k.Content + " ]]"))
	}
	for _, child := range k.Children {
		_ = child.MarshalXML(e, xml.StartElement{})
	}
	_ = e.EncodeToken(start.End())
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
