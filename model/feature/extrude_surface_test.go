// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// TestBuildExtrusionShellSheetIsOpen: with caps=false buildExtrusionShell builds an OPEN,
// non-solid wall sheet — the four side walls of a 2×2 square, no start/end caps — the tool for a
// Surface-operation extrude (Inventor kSurfaceOperation, #1858). The same profile with caps=true
// is the pre-existing closed solid prism (6 faces), proving the refactor left the solid path
// intact.
func TestBuildExtrusionShellSheetIsOpen(t *testing.T) {
	t.Parallel()
	poly := []math.Point2{{X: 0, Y: 0}, {X: 2, Y: 0}, {X: 2, Y: 2}, {X: 0, Y: 2}}

	sheet := buildExtrusionShell(poly, sketch.XYPlane(), span{near: 0, far: 3}, 0, "s", false)
	if sheet.IsSolid() {
		t.Error("caps-off shell is solid, want an open sheet")
	}
	if got := len(sheet.Faces()); got != 4 {
		t.Errorf("sheet has %d faces, want 4 walls (no caps)", got)
	}

	solid := buildExtrusionShell(poly, sketch.XYPlane(), span{near: 0, far: 3}, 0, "s", true)
	if !solid.IsSolid() || len(solid.Faces()) != 6 {
		t.Errorf("caps-on shell = solid?%v with %d faces, want a solid with 6", solid.IsSolid(), len(solid.Faces()))
	}
}

// TestSurfaceExtrudeRingHasInnerAndOuterWalls drives a Surface-operation extrude of an annular
// profile (a square with a square hole) through the feature engine: the open sheet is the ring's
// 4 outer walls plus its 4 inner-loop walls — 8 side faces, no caps, non-solid. Exercises the
// inner-loop tube of the sheet builder (#1858).
func TestSurfaceExtrudeRingHasInnerAndOuterWalls(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(param.NewParameters())
	const side, hole, height = 4.0, 2.0, 3.0
	sk := squareWithHoleSketch(side, hole)
	ring := -1
	for i := 0; i < sk.Profiles().Count(); i++ {
		if len(sk.Profiles().Item(i).InnerLoops()) > 0 {
			ring = i
		}
	}
	if ring < 0 {
		t.Fatal("no annular profile detected")
	}
	NewExtrudeFeatures(fs).AddExtrude(sk, []int{ring}, ops.Surface,
		Extent{Type: DistanceExtent, Distance: func() float64 { return height }}, 0)
	fs.Recompute()

	bodies := fs.Result()
	if len(bodies) != 1 || bodies[0].IsSolid() {
		t.Fatalf("surface ring extrude = %d bodies (solid=%v), want 1 open sheet", len(bodies), len(bodies) == 1 && bodies[0].IsSolid())
	}
	if got := len(bodies[0].Faces()); got != 8 {
		t.Errorf("ring sheet has %d faces, want 8 (4 outer + 4 inner walls, no caps)", got)
	}
}

// TestSurfaceExtrudeMergesMultipleProfiles drives a Surface-operation extrude of TWO disjoint
// square regions in one sketch: the open sheet merges both tubes (4 + 4 = 8 walls, no caps,
// non-solid). Exercises the multi-profile merge of buildProfileSheets (#1858).
func TestSurfaceExtrudeMergesMultipleProfiles(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(param.NewParameters())
	sk := twoSquaresSketch(2, 3) // two 2×2 squares, gap 3
	if sk.Profiles().Count() != 2 {
		t.Fatalf("two-squares sketch has %d regions, want 2", sk.Profiles().Count())
	}
	NewExtrudeFeatures(fs).AddExtrude(sk, []int{0, 1}, ops.Surface,
		Extent{Type: DistanceExtent, Distance: func() float64 { return 3 }}, 0)
	fs.Recompute()

	bodies := fs.Result()
	if len(bodies) != 1 || bodies[0].IsSolid() {
		t.Fatalf("two-profile surface extrude = %d bodies (solid=%v), want 1 open sheet", len(bodies), len(bodies) == 1 && bodies[0].IsSolid())
	}
	if got := len(bodies[0].Faces()); got != 8 {
		t.Errorf("merged sheet has %d faces, want 8 (two 4-wall tubes)", got)
	}
}

// twoSquaresSketch draws two disjoint side×side squares separated by gap along +X, giving two
// independent closed regions in one sketch.
func twoSquaresSketch(side, gap float64) *sketch.Sketch {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	addSquareAt := func(x0 float64) {
		p0 := s.Points().Add(math.P2(x0, 0))
		p1 := s.Points().Add(math.P2(x0+side, 0))
		p2 := s.Points().Add(math.P2(x0+side, side))
		p3 := s.Points().Add(math.P2(x0, side))
		s.Lines().Add(p0, p1)
		s.Lines().Add(p1, p2)
		s.Lines().Add(p2, p3)
		s.Lines().Add(p3, p0)
	}
	addSquareAt(0)
	addSquareAt(side + gap)
	return s
}
