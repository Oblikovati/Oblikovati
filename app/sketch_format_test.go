// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// squareWithMidline draws a 4×4 square plus a midline (which, as normal geometry, splits it into
// two regions), returning the active sketch and the midline.
func squareWithMidline(t *testing.T) (*Session, *sketch.Sketch, *sketch.Line) {
	t.Helper()
	s, def := emptyPartSession(t)
	sk := def.Sketches().Add(sketch.XYPlane())
	s.EnterSketch(sk)
	sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	sk.Lines().AddByTwoPoints(math.P2(4, 0), math.P2(4, 4))
	sk.Lines().AddByTwoPoints(math.P2(4, 4), math.P2(0, 4))
	sk.Lines().AddByTwoPoints(math.P2(0, 4), math.P2(0, 0))
	mid := sk.Lines().AddByTwoPoints(math.P2(0, 2), math.P2(4, 2))
	if sk.Profiles().Count() != 2 {
		t.Fatalf("setup: midline should split the square into 2 regions, got %d", sk.Profiles().Count())
	}
	return s, sk, mid
}

// Toggling Centerline on the selected midline removes it from profiles (it no longer closes a
// region) and marks it as an axis.
func TestSketchCenterlineToggle(t *testing.T) {
	s, sk, mid := squareWithMidline(t)
	s.Selection().Add(SketchEntityHandle{Entity: mid})
	if n := s.ToggleCenterline(); n != 1 {
		t.Fatalf("ToggleCenterline changed %d entities, want 1", n)
	}
	if !mid.IsCenterline() || !mid.IsConstruction() {
		t.Error("midline should now be a centerline (and construction)")
	}
	if sk.Profiles().Count() != 1 {
		t.Errorf("after centerline toggle the square has %d regions, want 1", sk.Profiles().Count())
	}
	if len(sk.Centerlines()) != 1 {
		t.Errorf("Centerlines() = %d, want 1", len(sk.Centerlines()))
	}
}

// Toggling Construction on the selected midline likewise drops it from profiles.
func TestSketchConstructionToggle(t *testing.T) {
	s, sk, mid := squareWithMidline(t)
	s.Selection().Add(SketchEntityHandle{Entity: mid})
	if n := s.ToggleConstruction(); n != 1 {
		t.Fatalf("ToggleConstruction changed %d entities, want 1", n)
	}
	if !mid.IsConstruction() {
		t.Error("midline should now be construction")
	}
	if sk.Profiles().Count() != 1 {
		t.Errorf("after construction toggle the square has %d regions, want 1", sk.Profiles().Count())
	}
}

func TestSketchFormatCommandsRegistered(t *testing.T) {
	s, _, mid := squareWithMidline(t)
	s.Selection().Add(SketchEntityHandle{Entity: mid})
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Sketch.Centerline"); err != nil {
		t.Fatalf("execute Sketch.Centerline: %v", err)
	}
	if !mid.IsCenterline() {
		t.Error("Sketch.Centerline command did not mark the line")
	}
}
