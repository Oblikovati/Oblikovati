// SPDX-License-Identifier: GPL-2.0-only

// Package blenderxml reads and writes Blender theme XML documents (`<bpy><Theme>…`) as a
// generic element tree. The Blender schema is huge and versioned, so we never model it
// with structs: every element/attribute is kept verbatim, which lets a theme file
// round-trip through load → edit → save without losing the hundreds of attributes the
// application does not map (ADR-0032). The theme package addresses colors inside the
// tree by element path + attribute name.
package blenderxml

import (
	"bytes"
	"encoding/xml"
	"fmt"
)

// Node is one XML element: its name, attributes in document order, and child elements.
// Text content is dropped — Blender theme files carry data exclusively in attributes.
type Node struct {
	XMLName  xml.Name
	Attrs    []xml.Attr `xml:",any,attr"`
	Children []*Node    `xml:",any"`
}

// NewElement returns an empty element with the given name (the seed for a section the
// document does not have yet, e.g. the oblikovati token snapshot).
func NewElement(name string) *Node {
	return &Node{XMLName: xml.Name{Local: name}}
}

// Parse decodes a Blender theme XML document into its root element.
//
//	root, err := blenderxml.Parse(darkXML) // root.XMLName.Local == "bpy"
func Parse(data []byte) (*Node, error) {
	var root Node
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("blenderxml: parse: %w", err)
	}
	return &root, nil
}

// Marshal renders the tree back to indented XML with a trailing newline, the shape
// Blender itself writes, so a saved theme diffs cleanly against its source.
func (n *Node) Marshal() ([]byte, error) {
	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(n); err != nil {
		return nil, fmt.Errorf("blenderxml: marshal <%s>: %w", n.XMLName.Local, err)
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

// Find walks the tree by child element names, returning nil when any step is missing.
//
//	wcol := root.Find("Theme", "user_interface", "ThemeUserInterface", "wcol_regular")
func (n *Node) Find(path ...string) *Node {
	cur := n
	for _, name := range path {
		cur = cur.child(name)
		if cur == nil {
			return nil
		}
	}
	return cur
}

// child returns the first child element with the given name, or nil.
func (n *Node) child(name string) *Node {
	for _, c := range n.Children {
		if c.XMLName.Local == name {
			return c
		}
	}
	return nil
}

// Attr returns the named attribute's value and whether it is present.
func (n *Node) Attr(name string) (string, bool) {
	for _, a := range n.Attrs {
		if a.Name.Local == name {
			return a.Value, true
		}
	}
	return "", false
}

// SetAttr overwrites the named attribute in place, or appends it when absent, so a
// write-back never reorders the attributes the file was authored with.
func (n *Node) SetAttr(name, value string) {
	for i, a := range n.Attrs {
		if a.Name.Local == name {
			n.Attrs[i].Value = value
			return
		}
	}
	n.Attrs = append(n.Attrs, xml.Attr{Name: xml.Name{Local: name}, Value: value})
}

// AppendChild adds an element under n (used for the oblikovati token-snapshot section).
func (n *Node) AppendChild(c *Node) { n.Children = append(n.Children, c) }

// RemoveChild deletes every direct child with the given name (no-op when absent).
func (n *Node) RemoveChild(name string) {
	kept := n.Children[:0:0]
	for _, c := range n.Children {
		if c.XMLName.Local != name {
			kept = append(kept, c)
		}
	}
	n.Children = kept
}

// Clone returns a deep, independent copy — duplicating a theme must never share tree
// nodes with its source, or editing the copy would silently recolor the original.
func (n *Node) Clone() *Node {
	out := &Node{XMLName: n.XMLName, Attrs: append([]xml.Attr(nil), n.Attrs...)}
	out.Children = make([]*Node, len(n.Children))
	for i, c := range n.Children {
		out.Children[i] = c.Clone()
	}
	return out
}
