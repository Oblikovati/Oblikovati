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
	if !isRenameableFeature(n) {
		t.Error("a feature node should be renameable")
	}
	if got := featureNodeName(n); got != "Extrusion1" {
		t.Errorf("featureNodeName = %q, want Extrusion1", got)
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

// TestIsRenameableFeatureRejectsNonFeature: a non-feature node is not renameable.
func TestIsRenameableFeatureRejectsNonFeature(t *testing.T) {
	n := app.BrowserNode{Label: "Solid Bodies", Kind: "bodies"}
	if isRenameableFeature(n) {
		t.Error("a non-feature node must not be renameable")
	}
	if got := featureNodeName(n); got != "Solid Bodies" {
		t.Errorf("featureNodeName of a non-feature node = %q, want its label", got)
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
