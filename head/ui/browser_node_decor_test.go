//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// featurePartSession returns a session whose active part has two extrude features, plus the
// browser node for the first ("Extrusion1") — the fixture for the in-place rename.
func featurePartSession(t *testing.T) (*app.Session, app.BrowserNode) {
	t.Helper()
	s := app.NewSession()
	pd, err := compdef.AddPart(s.Workspace(), "p.opd", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	def := pd.Content().(*compdef.PartComponentDefinition)
	addBox(def, 0)
	addBox(def, 6)
	def.Recompute()
	f, ok := def.Features().ByName("Extrusion1")
	if !ok {
		t.Fatal("expected an Extrusion1 feature")
	}
	return s, app.BrowserNode{Label: "Extrusion1", Kind: "feature", Select: app.FeatureHandle{Feature: f}}
}

func addBox(def *compdef.PartComponentDefinition, x float64) {
	sk := def.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(x, 0))
	c1 := sk.Points().Add(math.P2(x+2, 0))
	c2 := sk.Points().Add(math.P2(x+2, 2))
	c3 := sk.Points().Add(math.P2(x, 2))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 3 })
}

func TestRenameStateHelpers(t *testing.T) {
	_, n := featurePartSession(t)
	browserRename = browserRenameState{} // reset
	if renaming(n) {
		t.Error("nothing should be renaming initially")
	}
	if !isRenameableNode(n) {
		t.Error("a feature node should be renameable")
	}
	if got := nodeName(n); got != "Extrusion1" {
		t.Errorf("nodeName = %q, want Extrusion1", got)
	}
	beginRename(n)
	if !renaming(n) {
		t.Error("beginRename should mark the node as renaming")
	}
	if got := bufString(browserRename.buf[:]); got != "Extrusion1" {
		t.Errorf("rename buffer seeded with %q, want Extrusion1", got)
	}
}

// TestApplyRenameCommits: applyRename renames the feature (stable id) and closes the editor.
func TestApplyRenameCommits(t *testing.T) {
	s, n := featurePartSession(t)
	beginRename(n)
	setBuf(browserRename.buf[:], "Mounting Boss")
	applyRename(s, n)
	if browserRename.target != nil {
		t.Error("applyRename should close the editor")
	}
	if got := n.Select.(app.FeatureHandle).Feature.Name(); got != "Mounting Boss" {
		t.Errorf("feature name = %q, want Mounting Boss", got)
	}
}

// TestApplyRenameRejectsDuplicate: a name already used by another feature is rejected by the
// backend and the original name is kept (the document-unique invariant, #1264).
func TestApplyRenameRejectsDuplicate(t *testing.T) {
	s, n := featurePartSession(t)
	beginRename(n)
	setBuf(browserRename.buf[:], "Extrusion2") // already taken by the second feature
	applyRename(s, n)
	if got := n.Select.(app.FeatureHandle).Feature.Name(); got != "Extrusion1" {
		t.Errorf("duplicate rename should be rejected; name = %q, want Extrusion1", got)
	}
}

// TestIsRenameableNodeRejectsNonRenameable: a plain folder node is not renameable.
func TestIsRenameableNodeRejectsNonRenameable(t *testing.T) {
	n := app.BrowserNode{Label: "Solid Bodies", Kind: "bodies"}
	if isRenameableNode(n) {
		t.Error("a non-renameable node must not be renameable")
	}
	if got := nodeName(n); got != "Solid Bodies" {
		t.Errorf("nodeName of a non-handle node = %q, want its label", got)
	}
}

// TestRenameSketchAndWorkFeatures: sketches and user work features rename through the browser
// dispatch, while the grounded origin datums are rejected (#1264).
func TestRenameSketchAndWorkFeatures(t *testing.T) {
	s := app.NewSession()
	pd, err := compdef.AddPart(s.Workspace(), "w.opd", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	def := pd.Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches().Add(sketch.XYPlane())

	skNode := app.BrowserNode{Kind: "sketch", Select: app.SketchHandle{Sketch: sk}}
	if !isRenameableNode(skNode) {
		t.Fatal("a sketch node should be renameable")
	}
	if err := renameNode(s, skNode, "Profile"); err != nil {
		t.Fatalf("rename sketch: %v", err)
	}
	if sk.Name() != "Profile" {
		t.Errorf("sketch name = %q, want Profile", sk.Name())
	}

	// The origin centre point is a grounded coordinate-system datum: not renameable, and the
	// backend rejects a rename attempt.
	center, _ := def.WorkGeometry().WorkPointByRef(feature.OriginCenter)
	xPlane, _ := def.WorkGeometry().WorkPlaneByRef(feature.OriginXYPlane)
	xAxis, _ := def.WorkGeometry().AxisByRef(feature.OriginXAxis)
	originNodes := []app.BrowserNode{
		{Kind: "workpoint", Select: app.WorkPointHandle{Point: center}},
		{Kind: "workplane", Select: app.WorkPlaneHandle{Plane: xPlane}},
		{Kind: "workaxis", Select: app.WorkAxisHandle{Axis: xAxis}},
	}
	for _, n := range originNodes {
		if isRenameableNode(n) {
			t.Errorf("origin datum %T must not be renameable", n.Select)
		}
	}
	if err := s.RenameWorkPoint(center, "Nope"); err == nil {
		t.Error("renaming a coordinate-system datum should error")
	}
}

// TestNodeNameAndRenameDispatchAllTypes exercises nodeName and renameNode for every renameable
// handle type, so the per-type dispatch is covered end to end (#1264).
func TestNodeNameAndRenameDispatchAllTypes(t *testing.T) {
	s := app.NewSession()
	pd, err := compdef.AddPart(s.Workspace(), "a.opd", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	def := pd.Content().(*compdef.PartComponentDefinition)
	addBox(def, 0)
	sk2 := def.Sketches().Add(sketch.XYPlane())
	sk3 := def.Sketches3D().Add()
	wp := def.WorkPlanes().AddByPlaneAndOffset(feature.OriginXYPlane, func() float64 { return 2 })
	wa := def.WorkAxes().AddByPlaneIntersection(feature.OriginXYPlane, feature.OriginXZPlane)
	wpt := def.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(1, 1, 1) })
	def.Recompute()
	feat, _ := def.Features().ByName("Extrusion1")

	cases := []struct {
		node    app.BrowserNode
		newName string
		current func() string
	}{
		{app.BrowserNode{Select: app.FeatureHandle{Feature: feat}}, "Feat", feat.Name},
		{app.BrowserNode{Select: app.SketchHandle{Sketch: sk2}}, "Sk2", sk2.Name},
		{app.BrowserNode{Select: app.Sketch3DHandle{Sketch3D: sk3}}, "Sk3", sk3.Name},
		{app.BrowserNode{Select: app.WorkPlaneHandle{Plane: wp}}, "WP", wp.Name},
		{app.BrowserNode{Select: app.WorkAxisHandle{Axis: wa}}, "WA", wa.Name},
		{app.BrowserNode{Select: app.WorkPointHandle{Point: wpt}}, "WPt", wpt.Name},
	}
	for _, c := range cases {
		before := c.current()
		if !isRenameableNode(c.node) {
			t.Errorf("node %T should be renameable", c.node.Select)
		}
		if got := nodeName(c.node); got != before {
			t.Errorf("nodeName = %q, want %q", got, before)
		}
		if err := renameNode(s, c.node, c.newName); err != nil {
			t.Errorf("renameNode to %q: %v", c.newName, err)
		}
		if got := c.current(); got != c.newName {
			t.Errorf("after rename = %q, want %q", got, c.newName)
		}
	}
	if err := renameNode(s, app.BrowserNode{Label: "x"}, "y"); err != nil {
		t.Errorf("renameNode of a non-handle node should be a no-op, got %v", err)
	}
}

// TestInWindowBrowserDrawsIconAndRenameField renders the real browser: first the feature row
// (covering the per-feature icon draw), then the same row with the in-place rename editor open
// (covering the rename input). Skips cleanly without a Vulkan device.
func TestInWindowBrowserDrawsIconAndRenameField(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false // rebuild the default layout for this fresh window/context
	icons = nil         // rebind the icon cache to this fresh window
	browserRename = browserRenameState{}
	s, n := featurePartSession(t)

	frame := func() {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}
	frame()
	frame() // settle the dock layout and draw the browser with feature icons

	beginRename(n) // open the in-place editor; next frames draw the rename input
	frame()
	frame()
	browserRename = browserRenameState{} // leave globals clean for other tests
}
