// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"
	"sort"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
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

// selectableBranch appends a child that is BOTH selectable and a parent (a feature row that
// nests its consumed sketch), returning it so callers can add children. The head renders a
// node with a non-nil Select and children as an expandable, clickable tree node.
func (n *BrowserNode) selectableBranch(label, kind string, sel Selectable) *BrowserNode {
	c := n.child(label, kind)
	c.Select = sel
	return c
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

// addPartBranches builds the part's browser tree: the static Origin and Parameters folders,
// the Solid Bodies folder, then the chronological model timeline. The timeline interleaves
// user work features, sketches, and features by global creation order, nesting a consumed
// sketch under the feature that uses it (Inventor's model tree) — there is no separate
// Sketches branch, so a sketch's dependency on an earlier work plane/feature is visible
// (issue #132). Sketch/feature/datum nodes are selectable so clicking one syncs the 3D view.
func addPartBranches(root *BrowserNode, part *compdef.PartComponentDefinition) {
	addOriginBranch(root, part)
	params := root.child("Parameters", "parameters")
	for _, p := range part.Parameters().All() {
		params.child(p.Name(), "parameter")
	}
	addBodyBranch(root, part)
	addModelTimeline(root, part)
}

// timelineEntry is one node placed in the part's chronological tree: the global creation
// stamp it sorts by, and a builder that appends it (with any nested children) under root.
type timelineEntry struct {
	seq    uint64
	append func(root *BrowserNode)
}

// addModelTimeline appends the user work features, top-level sketches, and features in
// global creation order. A sketch consumed by exactly one feature (and not shared) is
// "absorbed": it is omitted here and nested under that feature instead.
func addModelTimeline(root *BrowserNode, part *compdef.PartComponentDefinition) {
	absorber := sketchAbsorbers(part)
	var entries []timelineEntry
	entries = appendWorkPlaneEntries(entries, part)
	entries = appendWorkAxisPointEntries(entries, part)
	entries = appendTopLevelSketchEntries(entries, part, absorber)
	entries = appendFeatureEntries(entries, part, absorber)

	sort.SliceStable(entries, func(i, j int) bool { return entries[i].seq < entries[j].seq })
	for _, e := range entries {
		e.append(root)
	}
}

// sketchAbsorbers maps each absorbed sketch to the single feature that nests it. A sketch is
// absorbed when exactly one feature consumes it AND it is not shared (Inventor's Share Sketch
// keeps a sketch at top level even when consumed, and lets several features consume it).
func sketchAbsorbers(part *compdef.PartComponentDefinition) map[*sketch.Sketch]*feature.PartFeature {
	consumers := map[*sketch.Sketch][]*feature.PartFeature{}
	features := part.Features()
	for i := 0; i < features.Count(); i++ {
		f := features.Item(i)
		for _, sk := range f.ConsumedSketches() {
			consumers[sk] = append(consumers[sk], f)
		}
	}
	absorber := make(map[*sketch.Sketch]*feature.PartFeature, len(consumers))
	for sk, fs := range consumers {
		if len(fs) == 1 && !sk.Shared() {
			absorber[sk] = fs[0]
		}
	}
	return absorber
}

// appendWorkPlaneEntries adds the user-created datum planes at their creation position.
// The origin coordinate-system planes live in the Origin folder, so they are skipped here.
func appendWorkPlaneEntries(entries []timelineEntry, part *compdef.PartComponentDefinition) []timelineEntry {
	planes := part.WorkPlanes()
	for i := 0; i < planes.Count(); i++ {
		wp := planes.Item(i)
		if wp.IsCoordinateSystemElement() {
			continue
		}
		entries = append(entries, timelineEntry{wp.Seq(), func(root *BrowserNode) {
			root.selectableChild(wp.Name(), "workplane", WorkPlaneHandle{Plane: wp})
		}})
	}
	return entries
}

// appendWorkAxisPointEntries adds the user-created datum axes and points at their creation
// position (the origin ones live in the Origin folder).
func appendWorkAxisPointEntries(entries []timelineEntry, part *compdef.PartComponentDefinition) []timelineEntry {
	axes := part.WorkAxes()
	for i := 0; i < axes.Count(); i++ {
		a := axes.Item(i)
		if a.IsCoordinateSystemElement() {
			continue
		}
		entries = append(entries, timelineEntry{a.Seq(), func(root *BrowserNode) {
			root.selectableChild(a.Name(), "workaxis", WorkAxisHandle{Axis: a})
		}})
	}
	points := part.WorkPoints()
	for i := 0; i < points.Count(); i++ {
		p := points.Item(i)
		if p.IsCoordinateSystemElement() {
			continue
		}
		entries = append(entries, timelineEntry{p.Seq(), func(root *BrowserNode) {
			root.selectableChild(p.Name(), "workpoint", WorkPointHandle{Point: p})
		}})
	}
	return entries
}

// appendTopLevelSketchEntries adds the sketches that are NOT absorbed under a feature
// (unconsumed, shared, or consumed by several features) at their creation position.
func appendTopLevelSketchEntries(entries []timelineEntry, part *compdef.PartComponentDefinition, absorber map[*sketch.Sketch]*feature.PartFeature) []timelineEntry {
	sketches := part.Sketches()
	for i := 0; i < sketches.Count(); i++ {
		sk := sketches.Item(i)
		if absorber[sk] != nil {
			continue
		}
		entries = append(entries, timelineEntry{sk.Seq(), func(root *BrowserNode) {
			root.selectableChild(sk.Name(), "sketch", SketchHandle{Sketch: sk})
		}})
	}
	return entries
}

// appendFeatureEntries adds each feature at its creation position, nesting the sketch(es)
// it absorbs as children so the consumed sketch reads as part of the feature.
func appendFeatureEntries(entries []timelineEntry, part *compdef.PartComponentDefinition, absorber map[*sketch.Sketch]*feature.PartFeature) []timelineEntry {
	features := part.Features()
	for i := 0; i < features.Count(); i++ {
		f := features.Item(i)
		entries = append(entries, timelineEntry{f.Seq(), func(root *BrowserNode) {
			node := root.selectableBranch(f.Name(), "feature", FeatureHandle{Feature: f})
			for _, sk := range f.ConsumedSketches() {
				if absorber[sk] == f {
					node.selectableChild(sk.Name(), "sketch", SketchHandle{Sketch: sk})
				}
			}
		}})
	}
	return entries
}

// addOriginBranch fills the Origin folder with the seven coordinate-system elements (three
// planes, three axes, the center point), all selectable — the reference inputs for
// axis/point-driven work planes (Inventor's Origin folder).
func addOriginBranch(root *BrowserNode, part *compdef.PartComponentDefinition) {
	origin := root.child("Origin", "origin")
	for _, wp := range part.OriginPlanes() {
		origin.selectableChild(wp.Name(), "workplane", WorkPlaneHandle{Plane: wp})
	}
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
