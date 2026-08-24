// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import "testing"

func TestTrianglePolygonOverlapPartial(t *testing.T) {
	a := tri([3]float64{0, 0, 0}, [3]float64{4, 0, 0}, [3]float64{0, 4, 0})
	clip := tri([3]float64{1, 1, 0}, [3]float64{5, 1, 0}, [3]float64{1, 5, 0})
	got := trianglePolygonOverlap(a, clip, 2)
	want := []Point{pt([3]float64{1, 1, 0}), pt([3]float64{3, 1, 0}), pt([3]float64{1, 3, 0})}
	assertSamePointSet(t, got, want)
}

func TestTrianglePolygonOverlapCWClipSameResult(t *testing.T) {
	a := tri([3]float64{0, 0, 0}, [3]float64{4, 0, 0}, [3]float64{0, 4, 0})
	// Same clip triangle wound clockwise: the CCW-normalization branch must yield
	// the identical overlap.
	clipCW := tri([3]float64{1, 1, 0}, [3]float64{1, 5, 0}, [3]float64{5, 1, 0})
	got := trianglePolygonOverlap(a, clipCW, 2)
	want := []Point{pt([3]float64{1, 1, 0}), pt([3]float64{3, 1, 0}), pt([3]float64{1, 3, 0})}
	assertSamePointSet(t, got, want)
}

func TestTrianglePolygonOverlapDisjoint(t *testing.T) {
	a := tri([3]float64{0, 0, 0}, [3]float64{4, 0, 0}, [3]float64{0, 4, 0})
	far := tri([3]float64{10, 10, 0}, [3]float64{14, 10, 0}, [3]float64{10, 14, 0})
	if got := trianglePolygonOverlap(a, far, 2); len(got) != 0 {
		t.Fatalf("disjoint triangles overlap to %d points, want 0", len(got))
	}
}

func TestCoplanarConstraintsEdgeAndDisjoint(t *testing.T) {
	a := tri([3]float64{0, 0, 0}, [3]float64{4, 0, 0}, [3]float64{0, 4, 0})
	// Shares only the hypotenuse (4,0)-(0,4): overlap is that segment → one edge.
	acrossHyp := tri([3]float64{4, 0, 0}, [3]float64{0, 4, 0}, [3]float64{4, 4, 0})
	if segs := coplanarConstraints(a, acrossHyp); len(segs) != 1 {
		t.Fatalf("edge-sharing overlap gave %d constraints, want 1", len(segs))
	}
	far := tri([3]float64{10, 10, 0}, [3]float64{14, 10, 0}, [3]float64{10, 14, 0})
	if segs := coplanarConstraints(a, far); segs != nil {
		t.Fatalf("disjoint overlap gave %d constraints, want 0", len(segs))
	}
}

func assertSamePointSet(t *testing.T, got, want []Point) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("overlap has %d vertices, want %d", len(got), len(want))
	}
	for _, w := range want {
		if !polyHasVertex(got, w) {
			t.Fatalf("overlap is missing vertex %v", w.Round())
		}
	}
}

func polyHasVertex(poly []Point, v Point) bool {
	for _, p := range poly {
		if p.Equal(v) {
			return true
		}
	}
	return false
}
