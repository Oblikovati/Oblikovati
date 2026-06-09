// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/model/compdef"
)

// The browser tree reflects the active document's structure — parameters, sketches,
// and the feature history — read directly from the model each frame (no parallel
// retained tree). Node actions (rename/suppress/delete) issue commands, so they are
// undoable; here we build the structure that ImGui renders.

// BrowserNode is a node in the model browser tree. A node with a non-nil Select is
// clickable: selecting it puts that handle in the session's selection set (so e.g.
// clicking "XY Plane" selects the plane to sketch on).
type BrowserNode struct {
	Label    string
	Kind     string // "document" | "origin" | "workplane" | "workaxis" | "workpoint" | "parameters" | "parameter" | "bodies" | "body" | "sketches" | "sketch" | "feature"
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

// addPartBranches adds the origin, parameters, solid bodies, sketches and features of a
// part. Sketch/feature/body nodes are selectable so clicking one syncs the 3D view (and
// a viewport pick lights up the matching node).
func addPartBranches(root *BrowserNode, part *compdef.PartComponentDefinition) {
	origin := root.child("Origin", "origin")
	for _, wp := range part.OriginPlanes() {
		origin.selectableChild(wp.Name(), "workplane", WorkPlaneHandle{Plane: wp})
	}
	addOriginAxesAndPoint(origin, part)
	params := root.child("Parameters", "parameters")
	for _, p := range part.Parameters().All() {
		params.child(p.Name(), "parameter")
	}
	addBodyBranch(root, part)
	addUserWorkPlanes(root, part)
	addUserWorkAxesAndPoints(root, part)
	sketches := root.child("Sketches", "sketches")
	for i := 0; i < part.Sketches().Count(); i++ {
		sk := part.Sketches().Item(i)
		sketches.selectableChild(sk.Name(), "sketch", SketchHandle{Sketch: sk})
	}
	features := part.Features()
	for i := 0; i < features.Count(); i++ {
		f := features.Item(i)
		root.selectableChild(f.Name(), "feature", FeatureHandle{Feature: f})
	}
}

// addUserWorkPlanes adds the part's user-created datum planes (offset, midplane, …) as
// top-level selectable nodes, so a plane made from the ribbon shows in the tree and can
// be re-picked (to sketch on, or as input to another datum). The origin coordinate-system
// planes live in the Origin folder, so they are skipped here.
func addUserWorkPlanes(root *BrowserNode, part *compdef.PartComponentDefinition) {
	planes := part.WorkPlanes()
	for i := 0; i < planes.Count(); i++ {
		if wp := planes.Item(i); !wp.IsCoordinateSystemElement() {
			root.selectableChild(wp.Name(), "workplane", WorkPlaneHandle{Plane: wp})
		}
	}
}

// addOriginAxesAndPoint completes the Origin folder with the X/Y/Z axes and the center
// point (the planes are added by the caller), so the whole origin coordinate frame is
// selectable — the reference inputs for axis/point-driven work planes (Inventor's Origin
// folder holds all seven elements).
func addOriginAxesAndPoint(origin *BrowserNode, part *compdef.PartComponentDefinition) {
	axes := part.WorkAxes()
	for i := 0; i < axes.Count(); i++ {
		if a := axes.Item(i); a.IsCoordinateSystemElement() {
			origin.selectableChild(a.Name(), "workaxis", WorkAxisHandle{Axis: a})
		}
	}
	points := part.WorkPoints()
	for i := 0; i < points.Count(); i++ {
		if p := points.Item(i); p.IsCoordinateSystemElement() {
			origin.selectableChild(p.Name(), "workpoint", WorkPointHandle{Point: p})
		}
	}
}

// addUserWorkAxesAndPoints adds user-created datum axes and points as top-level
// selectable nodes (the origin ones live in the Origin folder).
func addUserWorkAxesAndPoints(root *BrowserNode, part *compdef.PartComponentDefinition) {
	axes := part.WorkAxes()
	for i := 0; i < axes.Count(); i++ {
		if a := axes.Item(i); !a.IsCoordinateSystemElement() {
			root.selectableChild(a.Name(), "workaxis", WorkAxisHandle{Axis: a})
		}
	}
	points := part.WorkPoints()
	for i := 0; i < points.Count(); i++ {
		if p := points.Item(i); !p.IsCoordinateSystemElement() {
			root.selectableChild(p.Name(), "workpoint", WorkPointHandle{Point: p})
		}
	}
}

// addBodyBranch adds the "Solid Bodies" folder. Bodies carry no name, so they are
// numbered Solid1, Solid2, … in body order (Inventor's convention). The folder is
// omitted when the part has produced no bodies yet.
func addBodyBranch(root *BrowserNode, part *compdef.PartComponentDefinition) {
	bodies := part.SurfaceBodies().All()
	if len(bodies) == 0 {
		return
	}
	folder := root.child("Solid Bodies", "bodies")
	for i, b := range bodies {
		folder.selectableChild(fmt.Sprintf("Solid%d", i+1), "body", BodyHandle{Body: b})
	}
}
