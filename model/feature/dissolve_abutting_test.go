// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// slotWithCornerReliefSketch reproduces Inventor's dog-bone: a rounded-rect slot whose four corners
// are relieved by small discs. Drawn as the slot outline plus four corner circles, the arrangement
// splits it into a slot cell abutting four disc cells — five profiles sharing arcs, the exact shape
// that cracked BigChunkyPlate's top face when cut one prism at a time (#38).
func slotWithCornerReliefSketch(t *testing.T) *sketch.Sketch {
	t.Helper()
	s := sketch.NewSketches().Add(sketch.XYPlane())
	// A slot from (-4,0) to (4,0), half-height 2, corner radius 0.5 — as a rounded rectangle.
	roundedRect(s, -4, 4, -2, 2, 0.5)
	for _, c := range [][2]float64{{-3.5, 1.5}, {3.5, 1.5}, {-3.5, -1.5}, {3.5, -1.5}} {
		s.Circles().AddByCenterRadius(math.P2(math.Scalar(c[0]), math.Scalar(c[1])), 0.5)
	}
	return s
}

// roundedRect adds a rounded rectangle [x0,x1]x[y0,y1] with corner radius r as lines + arcs.
func roundedRect(s *sketch.Sketch, x0, x1, y0, y1, r float64) {
	// Four straight edges inset by r, joined by quarter-circle arcs — enough for the arrangement to
	// build a single rounded-rect cell that the corner discs then abut.
	pts := [][2]float64{{x0 + r, y0}, {x1 - r, y0}, {x1, y0 + r}, {x1, y1 - r}, {x1 - r, y1}, {x0 + r, y1}, {x0, y1 - r}, {x0, y0 + r}}
	p := make([]*sketch.Point, len(pts))
	for i, q := range pts {
		p[i] = s.Points().Add(math.P2(math.Scalar(q[0]), math.Scalar(q[1])))
	}
	s.Lines().Add(p[0], p[1])
	s.Lines().Add(p[2], p[3])
	s.Lines().Add(p[4], p[5])
	s.Lines().Add(p[6], p[7])
	centres := [][2]float64{{x1 - r, y0 + r}, {x1 - r, y1 - r}, {x0 + r, y1 - r}, {x0 + r, y0 + r}}
	ends := [][2]int{{1, 2}, {3, 4}, {5, 6}, {7, 0}}
	for i, c := range centres {
		s.Arcs().AddByCenterStartEnd(math.P2(math.Scalar(c[0]), math.Scalar(c[1])), p[ends[i][0]].Position(), p[ends[i][1]].Position(), true)
	}
}

// TestDissolveFusesSlotAndCornerDiscs: the five abutting cells of a dog-bone slot dissolve into ONE
// outer loop, conserving area, so the region extrudes as a single clean prism (#38). Disjoint cells
// must NOT be fused — see TestDissolveKeepsDisjointCellsSeparate.
func TestDissolveFusesSlotAndCornerDiscs(t *testing.T) {
	t.Parallel()
	sk := slotWithCornerReliefSketch(t)
	profs := sk.Profiles().All()
	if len(profs) < 5 {
		t.Fatalf("expected ≥5 abutting profiles for the dog-bone, got %d", len(profs))
	}
	groups := abuttingProfileGroups(profs)
	if len(groups) != 1 {
		t.Fatalf("the slot and its corner discs abut, so they are ONE connected group; got %d groups", len(groups))
	}
	merged, ok := dissolveGroup(profs, groups[0])
	if !ok {
		t.Fatal("dissolveGroup declined a clean hole-free abutting group")
	}
	if len(merged) != 1 {
		t.Fatalf("the dog-bone union is one simply-connected region; got %d regions", len(merged))
	}
	if len(merged[0].inners) != 0 {
		t.Errorf("the hole-free dog-bone has no inner loops; got %d", len(merged[0].inners))
	}
	var want float64
	for _, p := range profs {
		want += stdmath.Abs(p.Area())
	}
	got := polygonArea2D(merged[0].outer)
	if stdmath.Abs(got-want) > 1e-6*want {
		t.Errorf("merged area %g, want the union %g (abutting cells don't overlap, so union = sum)", got, want)
	}
}

// TestDissolveCarriesInnerLoops: a relieved slot with a BORE through it (an inner loop) plus its four
// corner discs still dissolves — the merged region carries the bore forward as an inner loop, so a
// hole-carrying dog-bone fuses instead of falling back to the coincident-wall path. This is the
// blind-pocket-bottom crack that kept BigChunkyPlate open at z=1.8 (#38 follow-up).
func TestDissolveCarriesInnerLoops(t *testing.T) {
	t.Parallel()
	s := sketch.NewSketches().Add(sketch.XYPlane())
	roundedRect(s, -4, 4, -2, 2, 0.5)
	for _, c := range [][2]float64{{-3.5, 1.5}, {3.5, 1.5}, {-3.5, -1.5}, {3.5, -1.5}} {
		s.Circles().AddByCenterRadius(math.P2(math.Scalar(c[0]), math.Scalar(c[1])), 0.5)
	}
	s.Circles().AddByCenterRadius(math.P2(0, 0), 0.75) // a bore through the middle of the slot
	profs := s.Profiles().All()
	// The slot-with-bore cell abuts the four corner discs; the bore is its own (inner) region.
	groups := abuttingProfileGroups(profs)
	var big []int
	for _, g := range groups {
		if len(g) > len(big) {
			big = g
		}
	}
	regions, ok := dissolveGroup(profs, big)
	if !ok {
		t.Fatal("dissolveGroup declined a hole-carrying abutting group")
	}
	if len(regions) != 1 {
		t.Fatalf("the dog-bone union is one region; got %d", len(regions))
	}
	if len(regions[0].inners) == 0 {
		t.Error("the merged region must carry the bore forward as an inner loop; got none")
	}
}

// TestDissolveKeepsDisjointCellsSeparate: three separated circles share no edge, so they stay three
// groups and are NOT fused — a lone bore must keep its own prism (and its analytic cylinder), the
// #33 per-region behaviour the dissolve must not disturb.
func TestDissolveKeepsDisjointCellsSeparate(t *testing.T) {
	t.Parallel()
	s := sketch.NewSketches().Add(sketch.XYPlane())
	for _, cx := range []float64{-5, 0, 5} {
		s.Circles().AddByCenterRadius(math.P2(math.Scalar(cx), 0), 1)
	}
	profs := s.Profiles().All()
	groups := abuttingProfileGroups(profs)
	if len(groups) != len(profs) {
		t.Errorf("disjoint circles must each be their own group; got %d groups for %d profiles", len(groups), len(profs))
	}
	for _, g := range groups {
		if _, ok := dissolveGroup(profs, g); ok {
			t.Error("a singleton group must not dissolve (it would facet a lone bore's analytic cylinder)")
		}
	}
}
