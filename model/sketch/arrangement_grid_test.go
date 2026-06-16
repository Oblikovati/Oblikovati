// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
)

// TestGridArrangementDetectsManyRegions builds many separate closed squares — enough
// segments to take the grid broad-phase (arrBruteMax) — and checks region detection
// still finds each one. This guards the grid path: it must agree with the brute-force
// arrangement, just without the O(n²) blowup that froze dense imported sketches.
func TestGridArrangementDetectsManyRegions(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	const squares = 400 // 400 × 4 lines = 1600 segments > arrBruteMax (1024)
	for k := 0; k < squares; k++ {
		x := float64(k%20) * 10 // spread out on a 20-wide grid so squares don't touch
		y := float64(k/20) * 10
		corners := []math.Point2{
			math.P2(x, y), math.P2(x+1, y), math.P2(x+1, y+1), math.P2(x, y+1),
		}
		pts := make([]*Point, 4)
		for i, c := range corners {
			pts[i] = s.NewPoint(c)
		}
		for i := range pts {
			s.Lines().Add(pts[i], pts[(i+1)%4]) // close the loop
		}
	}
	if s.Lines().Count() <= arrBruteMax {
		t.Fatalf("test needs > %d segments to exercise the grid path, has %d", arrBruteMax, s.Lines().Count())
	}
	if got := s.Profiles().Count(); got != squares {
		t.Fatalf("grid arrangement found %d profiles, want %d closed squares", got, squares)
	}
}

// TestGridAndBruteAgreeOnCrossing checks the grid path resolves crossings the same as
// brute force: two crossing squares (an overlap) split into the expected regions, with
// segment counts on both sides of the threshold.
func TestGridAndBruteAgreeOnCrossing(t *testing.T) {
	build := func(extra int) int {
		s := NewSketches().Add(XYPlane())
		// two overlapping squares → 3 regions (left-only, overlap, right-only)
		addClosed(s, []math.Point2{{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 4}, {X: 0, Y: 4}})
		addClosed(s, []math.Point2{{X: 2, Y: 1}, {X: 6, Y: 1}, {X: 6, Y: 5}, {X: 2, Y: 5}})
		for k := 0; k < extra; k++ { // padding squares far away to cross the threshold
			x := 100 + float64(k%30)*10
			y := float64(k/30) * 10
			addClosed(s, []math.Point2{{X: x, Y: y}, {X: x + 1, Y: y}, {X: x + 1, Y: y + 1}, {X: x, Y: y + 1}})
		}
		return s.Profiles().Count() - extra // regions from the two overlapping squares
	}
	brute := build(0)  // few segments → brute force
	grid := build(400) // > arrBruteMax → grid path
	if brute != grid {
		t.Fatalf("grid path found %d overlap regions, brute found %d", grid, brute)
	}
	if brute < 3 {
		t.Fatalf("overlapping squares gave %d regions, want >= 3", brute)
	}
}

func addClosed(s *Sketch, corners []math.Point2) {
	pts := make([]*Point, len(corners))
	for i, c := range corners {
		pts[i] = s.NewPoint(c)
	}
	for i := range pts {
		s.Lines().Add(pts[i], pts[(i+1)%len(pts)])
	}
}
