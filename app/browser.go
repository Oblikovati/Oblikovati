// SPDX-License-Identifier: GPL-2.0-only

package app

import "github.com/Oblikovati/oblikovati/model/compdef"

// The browser tree reflects the active document's structure — parameters, sketches,
// and the feature history — read directly from the model each frame (no parallel
// retained tree). Node actions (rename/suppress/delete) issue commands, so they are
// undoable; here we build the structure that ImGui renders.

// BrowserNode is a node in the model browser tree. A node with a non-nil Select is
// clickable: selecting it puts that handle in the session's selection set (so e.g.
// clicking "XY Plane" selects the plane to sketch on).
type BrowserNode struct {
	Label    string
	Kind     string // "document" | "origin" | "workplane" | "parameters" | "parameter" | "sketches" | "sketch" | "feature"
	Select   Selectable
	Children []BrowserNode
}

// child appends a child node and returns the receiver for chaining within builders.
func (n *BrowserNode) child(label, kind string) *BrowserNode {
	n.Children = append(n.Children, BrowserNode{Label: label, Kind: kind})
	return &n.Children[len(n.Children)-1]
}

// selectableChild appends a child whose click selects sel.
func (n *BrowserNode) selectableChild(label, kind string, sel Selectable) {
	c := n.child(label, kind)
	c.Select = sel
}

// BuildBrowser builds the browser tree for the active document. An empty session
// yields an empty root; a part document yields parameter, sketch, and feature
// branches reflecting its component definition.
func BuildBrowser(s *Session) BrowserNode {
	doc := s.ActiveDocument()
	if doc == nil {
		return BrowserNode{Label: "(no document)", Kind: "document"}
	}
	root := BrowserNode{Label: doc.DisplayName(), Kind: "document"}
	if part, ok := doc.Content().(*compdef.PartComponentDefinition); ok {
		addPartBranches(&root, part)
	}
	return root
}

// addPartBranches adds the origin, parameters, sketches and features of a part.
func addPartBranches(root *BrowserNode, part *compdef.PartComponentDefinition) {
	origin := root.child("Origin", "origin")
	for _, wp := range part.OriginPlanes() {
		origin.selectableChild(wp.Name(), "workplane", WorkPlaneHandle{Plane: wp})
	}
	params := root.child("Parameters", "parameters")
	for _, p := range part.Parameters().All() {
		params.child(p.Name(), "parameter")
	}
	sketches := root.child("Sketches", "sketches")
	for i := 0; i < part.Sketches().Count(); i++ {
		sketches.child(part.Sketches().Item(i).Name(), "sketch")
	}
	features := part.Features()
	for i := 0; i < features.Count(); i++ {
		f := features.Item(i)
		root.child(f.Name(), "feature")
	}
}
