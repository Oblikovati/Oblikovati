// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// rectHole returns an axis-aligned rectangle [x0,x1]×[y0,y1] as a hole loop.
func rectHole(x0, y0, x1, y1 float64) []math.Point2 {
	return []math.Point2{{X: x0, Y: y0}, {X: x1, Y: y0}, {X: x1, Y: y1}, {X: x0, Y: y1}}
}

// triSetArea sums the absolute area of earcut's triangles over a combined vertex list (outer
// then holes), so a result can be area-checked.
func triSetArea(tris [][3]int, outer []math.Point2, holes [][]math.Point2) float64 {
	verts := append([]math.Point2(nil), outer...)
	for _, h := range holes {
		verts = append(verts, h...)
	}
	var a float64
	for _, t := range tris {
		a += stdmath.Abs(triArea(verts[t[0]], verts[t[1]], verts[t[2]]))
	}
	return a
}

// TestEarcutOverlappingGridHolesNoPanic regresses #873: a boundary with a grid of crossing
// (overlapping) bar holes used to collapse a ring and nil-deref in cureLocalIntersections /
// splitEarcut. It must now triangulate without panicking (best-effort on this degenerate input).
func TestEarcutOverlappingGridHolesNoPanic(t *testing.T) {
	t.Parallel()
	outer := rectHole(0, 0, 10, 10)
	var holes [][]math.Point2
	for _, x := range []float64{3, 5, 7} {
		holes = append(holes, rectHole(x-0.2, 1, x+0.2, 9)) // vertical bars
	}
	for _, y := range []float64{3, 5, 7} {
		holes = append(holes, rectHole(1, y-0.2, 9, y+0.2)) // horizontal bars (cross the verticals)
	}
	tris := earcut(outer, holes) // must not panic
	if len(tris) == 0 {
		t.Error("earcut returned no triangles for the grid-holes face")
	}
}

// TestEarcutPairOverlapNoPanic: a pair of overlapping holes also triangulates without panic.
func TestEarcutPairOverlapNoPanic(t *testing.T) {
	t.Parallel()
	outer := rectHole(0, 0, 10, 10)
	holes := [][]math.Point2{rectHole(3, 3, 6, 6), rectHole(5, 5, 8, 8)}
	_ = earcut(outer, holes) // must not panic
}

// TestEarcutDisjointHolesExactArea guards against regression: with valid, non-overlapping
// holes the triangulation still covers exactly outer − holes.
func TestEarcutDisjointHolesExactArea(t *testing.T) {
	t.Parallel()
	outer := rectHole(0, 0, 10, 10)
	holes := [][]math.Point2{rectHole(1, 1, 3, 9), rectHole(4, 1, 6, 9), rectHole(7, 1, 9, 9)}
	want := 100.0
	for _, h := range holes {
		want -= stdmath.Abs(signedArea(h))
	}
	if got := triSetArea(earcut(outer, holes), outer, holes); stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("earcut area = %g, want %g (outer − 3 disjoint holes)", got, want)
	}
}
