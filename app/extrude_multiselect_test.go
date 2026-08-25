// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/sketch"
)

// newPartWithTwoSquares sets up a part whose sketch holds two disjoint square regions
// (a gap apart), returning the session and a profile handle for each region.
func newPartWithTwoSquares(t *testing.T, side, gap float64) (*Session, ProfileHandle, ProfileHandle) {
	t.Helper()
	s := NewSession()
	def := compdef.NewPartComponentDefinition()
	pd, err := s.Workspace().Add(doc.Part, "part.obk", true)
	if err != nil {
		t.Fatalf("Add part: %v", err)
	}
	pd.SetContent(def)
	sk := def.Sketches().Add(sketch.XYPlane())
	addToolSquare(sk, 0, side)
	addToolSquare(sk, gap, side)
	if n := sk.Profiles().Count(); n != 2 {
		t.Fatalf("two-square sketch has %d regions, want 2", n)
	}
	return s, ProfileHandle{Sketch: sk, ProfileIndex: 0}, ProfileHandle{Sketch: sk, ProfileIndex: 1}
}

// addToolSquare adds a side×side square with lower-left corner at (dx,0).
func addToolSquare(sk *sketch.Sketch, dx, side float64) {
	c0 := sk.Points().Add(math.P2(dx, 0))
	c1 := sk.Points().Add(math.P2(dx+side, 0))
	c2 := sk.Points().Add(math.P2(dx+side, side))
	c3 := sk.Points().Add(math.P2(dx, side))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
}

func TestExtrudeToolCtrlAccumulatesRegions(t *testing.T) {
	_, r0, r1 := newPartWithTwoSquares(t, 2, 10)
	ext := NewExtrudeTool()
	ext.Pick(nil, r0)                  // plain click: one region
	ext.PickWithMods(nil, r1, CtrlMod) // Ctrl+click: add the second
	if got := ext.PickedProfiles(); len(got) != 2 || got[0] != r0 || got[1] != r1 {
		t.Fatalf("picked = %v, want [r0 r1]", got)
	}
	// Ctrl+click an already-picked region toggles it off.
	ext.PickWithMods(nil, r0, CtrlMod)
	if got := ext.PickedProfiles(); len(got) != 1 || got[0] != r1 {
		t.Fatalf("after toggle = %v, want [r1]", got)
	}
	// A plain click (no modifier) replaces the whole selection.
	ext.PickWithMods(nil, r0, 0)
	if got := ext.PickedProfiles(); len(got) != 1 || got[0] != r0 {
		t.Fatalf("after plain pick = %v, want [r0]", got)
	}
}

func TestExtrudeToolCtrlMultiSelectMergesIntoOneBody(t *testing.T) {
	s, r0, r1 := newPartWithTwoSquares(t, 2, 10)
	ext := NewExtrudeTool()
	s.StartTool(ext)
	ext.Pick(s, r0)
	ext.PickWithMods(s, r1, CtrlMod)
	ext.SetDistance(4)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("two-region extrude produced %d bodies, want 1 merged", def.SurfaceBodies().Count())
	}
	if got := len(def.SurfaceBodies().Item(0).Faces()); got != 12 {
		t.Errorf("merged body has %d faces, want 12 (two prisms)", got)
	}
}

// coordPicker resolves clicks to different selectables by x coordinate, so a test can
// click two different regions through the real Pointer path (modifier plumbing).
type coordPicker struct {
	left, right Selectable
	split       float64
}

func (p coordPicker) Pick(x, _ float64, filter *SelectionFilter) (Selectable, bool) {
	sel := p.left
	if x >= p.split {
		sel = p.right
	}
	if sel == nil || !filter.Accepts(sel.SelectionKind()) {
		return nil, false
	}
	return sel, true
}

// TestRegionModifierHintAddRemove: with a region tool active and Ctrl held, the cursor hint is ADD
// over an unpicked region and REMOVE over an already-picked one — the +/− the head badges the cursor
// with — and nothing without a modifier or without an active region tool.
func TestRegionModifierHintAddRemove(t *testing.T) {
	s, r0, r1 := newPartWithTwoSquares(t, 2, 10)
	s.SetPicker(coordPicker{left: r0, right: r1, split: 100})

	// No active tool → no hint even with Ctrl over a region.
	if show, _ := s.RegionModifierHint(10, 10, CtrlMod); show {
		t.Error("no tool active: expected no cursor hint")
	}
	ext := NewExtrudeTool()
	s.StartTool(ext)
	ext.Pick(s, r0) // region 0 is now picked

	if show, _ := s.RegionModifierHint(10, 10, 0); show {
		t.Error("no modifier held: expected no cursor hint")
	}
	show, add := s.RegionModifierHint(10, 10, CtrlMod) // over the already-picked region 0
	if !show || add {
		t.Errorf("over a picked region: show=%v add=%v, want show=true add=false (REMOVE)", show, add)
	}
	show, add = s.RegionModifierHint(200, 10, CtrlMod) // over the unpicked region 1
	if !show || !add {
		t.Errorf("over an unpicked region: show=%v add=%v, want show=true add=true (ADD)", show, add)
	}
}

func TestExtrudeCtrlClickThroughPointerCapturesBothRegions(t *testing.T) {
	s, r0, r1 := newPartWithTwoSquares(t, 2, 10)
	s.SetPicker(coordPicker{left: r0, right: r1, split: 100})
	ext := NewExtrudeTool()
	s.StartTool(ext)
	s.Pointer(PointerEvent{X: 10, Y: 10, Button: LeftButton})                 // pick region 0
	s.Pointer(PointerEvent{X: 200, Y: 10, Button: LeftButton, Mods: CtrlMod}) // Ctrl+click region 1
	if got := ext.PickedProfiles(); len(got) != 2 {
		t.Fatalf("Ctrl+click through Pointer captured %d regions, want 2 (%v)", len(got), got)
	}
}
