// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// regionAreas returns the (absolute) outer-loop area of every detected profile, so a
// test can assert how a sketch was carved into regions independent of their order.
func regionAreas(s *Sketch) []float64 {
	ps := s.Profiles()
	var out []float64
	for i := 0; i < ps.Count(); i++ {
		p := ps.Item(i)
		if p.IsClosed() {
			out = append(out, stdmath.Abs(signedArea2d(p.OuterLoop().Polygon())))
		}
	}
	return out
}

func TestRectangleSplitByLineYieldsTwoRegions(t *testing.T) {
	// The reported bug: a rectangle with a divider must become two regions, each
	// bounded by part of the rectangle and the divider — not one region ignoring it.
	s := NewSketches().Add(XYPlane())
	addRectangle(s, 0, 0, 10, 6)
	mid := s.Lines() // a horizontal divider whose ends land on the left/right edges (T-junctions)
	mid.Add(s.Points().Add(math.P2(0, 3)), s.Points().Add(math.P2(10, 3)))
	if got := regionAreas(s); len(got) != 2 || !approxArea(got, 30) {
		t.Fatalf("split rectangle regions = %v, want two of area 30", got)
	}
}

func TestRectangleSplitByArcYieldsTwoRegions(t *testing.T) {
	// A shallow arc dividing the rectangle (its endpoints sit on the side edges). The
	// two regions' areas sum to the rectangle's 100 regardless of the arc's bulge.
	s := NewSketches().Add(XYPlane())
	addRectangle(s, 0, 0, 10, 10)
	left := s.Points().Add(math.P2(0, 5))
	right := s.Points().Add(math.P2(10, 5))
	s.Arcs().Add(s.Points().Add(math.P2(5, -10)), left, right, false) // shallow bulge up to y≈5.8, inside
	areas := regionAreas(s)
	if len(areas) != 2 {
		t.Fatalf("arc-split rectangle regions = %d, want 2 (%v)", len(areas), areas)
	}
	if sum := areas[0] + areas[1]; !math.IsNearZero(sum-100, 1e-6) {
		t.Errorf("region areas %v sum to %.4f, want 100", areas, sum)
	}
}

func TestSquareWithBothDiagonalsYieldsFourTriangles(t *testing.T) {
	// Two diagonals that cross mid-span (no shared point at the crossing) → 4 cells.
	s := NewSketches().Add(XYPlane())
	bl := s.Points().Add(math.P2(0, 0))
	br := s.Points().Add(math.P2(10, 0))
	tr := s.Points().Add(math.P2(10, 10))
	tl := s.Points().Add(math.P2(0, 10))
	s.Lines().Add(bl, br)
	s.Lines().Add(br, tr)
	s.Lines().Add(tr, tl)
	s.Lines().Add(tl, bl)
	s.Lines().Add(bl, tr) // diagonal /
	s.Lines().Add(br, tl) // diagonal \ — crosses the first at (5,5)
	areas := regionAreas(s)
	if len(areas) != 4 {
		t.Fatalf("square+diagonals regions = %d, want 4 triangles (%v)", len(areas), areas)
	}
	for _, a := range areas {
		if !math.IsNearZero(a-25, 1e-6) {
			t.Errorf("triangle area = %.4f, want 25 (%v)", a, areas)
		}
	}
}

func TestOverlappingCirclesYieldThreeRegions(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	s.Circles().AddByCenterRadius(math.P2(0, 0), 3)
	s.Circles().AddByCenterRadius(math.P2(4, 0), 3)
	if got := regionAreas(s); len(got) != 3 {
		t.Fatalf("two overlapping circles → %d regions, want 3 (lens + two crescents): %v", len(got), got)
	}
}

// TestCrossingBarsInsideBoundaryMergeToOneHole: a boundary with a grid of crossing bars (a
// disconnected island whose interior is subdivided into many abutting cells) yields one profile
// whose island is a single merged hole — not a tiling of overlapping cell-loops (#863).
func TestCrossingBarsInsideBoundaryMergeToOneHole(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	addRectangle(s, 1.5, 1.5, 8.5, 8.5) // boundary
	for _, x := range []float64{3, 5, 7} {
		addRectangle(s, x-0.2, 1.8, x+0.2, 8.2) // crossing vertical bars
	}
	for _, y := range []float64{3, 5, 7} {
		addRectangle(s, 1.8, y-0.2, 8.2, y+0.2) // crossing horizontal bars
	}
	ps := s.Profiles()
	if ps.Count() != 1 {
		t.Fatalf("crossing-bar boundary profiles = %d, want 1", ps.Count())
	}
	p := ps.Item(0)
	if len(p.InnerLoops()) != 1 {
		t.Fatalf("inner loops = %d, want 1 (the island merged to one outline)", len(p.InnerLoops()))
	}
	// Area-consistent: the boundary (49) minus the island's footprint, with no double-counting.
	if got := p.Area(); !math.IsNearZero(got-24.84, 1e-6) {
		t.Errorf("frame area = %v, want 24.84 (49 − island footprint 24.16)", got)
	}
}

func TestPlainRectangleStillOneRegion(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	addRectangle(s, 0, 0, 4, 3)
	if got := regionAreas(s); len(got) != 1 || !math.IsNearZero(got[0]-12, 1e-6) {
		t.Fatalf("plain rectangle regions = %v, want one of area 12", got)
	}
}

// approxArea reports whether every area in got is within tol of want.
func approxArea(got []float64, want float64) bool {
	for _, a := range got {
		if !math.IsNearZero(a-want, 1e-6) {
			return false
		}
	}
	return true
}
