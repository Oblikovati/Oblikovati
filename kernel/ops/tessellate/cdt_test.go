// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"
	"testing"
)

// cdtAreaSum sums the (CCW, positive) areas of the triangles. For a correct triangulation of a
// domain this equals the domain's area — the check that catches both gaps (under) and overlaps
// (over), which is exactly what sank the plain-Delaunay attempt on concave/holed trims.
func cdtAreaSum(pts [][2]float64, tris [][3]int) float64 {
	var a float64
	for _, t := range tris {
		a += triTwiceSignedArea(pts[t[0]], pts[t[1]], pts[t[2]]) / 2
	}
	return a
}

func cdtPolyArea(pts [][2]float64, loop []int) float64 {
	var a float64
	for i := range loop {
		p, q := pts[loop[i]], pts[loop[(i+1)%len(loop)]]
		a += p[0]*q[1] - q[0]*p[1]
	}
	return stdmath.Abs(a) / 2
}

func TestOrientAndInCircle(t *testing.T) {
	t.Parallel()
	if orient2d([2]float64{0, 0}, [2]float64{1, 0}, [2]float64{0, 1}) <= 0 {
		t.Error("CCW triangle should be positively oriented")
	}
	// unit square's circumcircle (center .5,.5) contains its center, excludes a far point
	a, b, c := [2]float64{0, 0}, [2]float64{1, 0}, [2]float64{0, 1}
	if inCircle(a, b, c, [2]float64{0.4, 0.4}) <= 0 {
		t.Error("interior point should be inside circumcircle")
	}
	if inCircle(a, b, c, [2]float64{5, 5}) >= 0 {
		t.Error("far point should be outside circumcircle")
	}
}

func TestCDTSquare(t *testing.T) {
	t.Parallel()
	pts := [][2]float64{{0, 0}, {2, 0}, {2, 2}, {0, 2}}
	tris := ConstrainedDelaunay(pts, [][]int{{0, 1, 2, 3}})
	if len(tris) != 2 {
		t.Fatalf("square should triangulate to 2 triangles, got %d", len(tris))
	}
	if got := cdtAreaSum(pts, tris); stdmath.Abs(got-4) > 1e-9 {
		t.Errorf("square area = %g, want 4", got)
	}
}

// TestCDTDuplicateBoundaryPoint guards the #1073 freeze fix: an imported boundary loop with a
// DUPLICATE sample (two points at the same coords, as STEP pcurves produce) must still triangulate to
// the correct domain. The duplicate is skipped by point insertion (it owns no triangle), so a
// constraint referencing it used to make the flip recovery spin its whole 4·len(tris) retry finding
// no crossing — the O(T²) freeze on large faces. constrainedDelaunay now constrains between the
// inserted representatives, and the degenerate (coincident-endpoint) edge is skipped.
func TestCDTDuplicateBoundaryPoint(t *testing.T) {
	t.Parallel()
	// A unit square whose loop repeats the (0,0) corner as a separate index (4) — a duplicate sample.
	pts := [][2]float64{{0, 0}, {2, 0}, {2, 2}, {0, 2}, {0, 0}}
	tris := ConstrainedDelaunay(pts, [][]int{{0, 1, 2, 3, 4}})
	if len(tris) != 2 {
		t.Fatalf("square with a duplicate corner should triangulate to 2 triangles, got %d", len(tris))
	}
	if got := cdtAreaSum(pts, tris); stdmath.Abs(got-4) > 1e-9 {
		t.Errorf("duplicate-corner square area = %g, want 4 (the duplicate constraint cracked the domain)", got)
	}
}

func TestCDTConcaveLShape(t *testing.T) {
	t.Parallel()
	// An L: the triangulation must NOT bridge the notch (that was the plain-Delaunay over-count).
	pts := [][2]float64{{0, 0}, {2, 0}, {2, 1}, {1, 1}, {1, 2}, {0, 2}}
	loop := []int{0, 1, 2, 3, 4, 5}
	tris := ConstrainedDelaunay(pts, [][]int{loop})
	want := cdtPolyArea(pts, loop) // 3
	if got := cdtAreaSum(pts, tris); stdmath.Abs(got-want) > 1e-9 {
		t.Errorf("L-shape triangulated area = %g, want %g (notch bridged or torn)", got, want)
	}
}

func TestCDTSquareWithHole(t *testing.T) {
	t.Parallel()
	pts := [][2]float64{
		{0, 0}, {6, 0}, {6, 6}, {0, 6}, // outer
		{2, 2}, {4, 2}, {4, 4}, {2, 4}, // hole
	}
	tris := ConstrainedDelaunay(pts, [][]int{{0, 1, 2, 3}, {4, 5, 6, 7}})
	want := 36.0 - 4.0
	if got := cdtAreaSum(pts, tris); stdmath.Abs(got-want) > 1e-9 {
		t.Errorf("square-with-hole area = %g, want %g", got, want)
	}
	// no triangle centroid may fall inside the hole
	for _, tr := range tris {
		cx := (pts[tr[0]][0] + pts[tr[1]][0] + pts[tr[2]][0]) / 3
		cy := (pts[tr[0]][1] + pts[tr[1]][1] + pts[tr[2]][1]) / 3
		if cx > 2 && cx < 4 && cy > 2 && cy < 4 {
			t.Errorf("triangle centroid (%g,%g) lies inside the hole", cx, cy)
		}
	}
}

func TestCDTInteriorSteinerPoints(t *testing.T) {
	t.Parallel()
	// A square boundary plus interior grid points: area is preserved and every point is used.
	pts := [][2]float64{{0, 0}, {3, 0}, {3, 3}, {0, 3}}
	for u := 0.5; u < 3; u += 0.5 {
		for v := 0.5; v < 3; v += 0.5 {
			pts = append(pts, [2]float64{u, v})
		}
	}
	tris := ConstrainedDelaunay(pts, [][]int{{0, 1, 2, 3}})
	if got := cdtAreaSum(pts, tris); stdmath.Abs(got-9) > 1e-9 {
		t.Errorf("refined square area = %g, want 9", got)
	}
	if len(tris) < 8 {
		t.Errorf("interior points should refine the mesh, got only %d triangles", len(tris))
	}
}

// TestCDTCocircularNgon pins the exact-predicate robustness fix: a convex N-gon whose vertices lie
// (near-)exactly on a circle is the worst case for the float in-circle determinant — the radius²
// terms swamp their tiny difference, the naive sign flips, and the Bowyer–Watson cavity grabs the
// wrong triangles, emitting inverted/overlapping triangles (area far exceeding the polygon). With the
// adaptive-exact predicate the triangulation is exactly N-2 triangles whose areas sum to the polygon
// area, with none inverted. This is the degeneracy a planar face's circular bore hole hits.
func TestCDTCocircularNgon(t *testing.T) {
	t.Parallel()
	for _, n := range []int{6, 8, 12, 20, 32, 64} {
		var pts [][2]float64
		var loop []int
		for i := range n {
			a := 2 * stdmath.Pi * float64(i) / float64(n)
			pts = append(pts, [2]float64{25 * stdmath.Cos(a), 25 * stdmath.Sin(a)})
			loop = append(loop, i)
		}
		tris := ConstrainedDelaunay(pts, [][]int{loop})
		if len(tris) != n-2 {
			t.Errorf("n=%d: %d triangles, want %d (cocircular degeneracy tangled the mesh)", n, len(tris), n-2)
		}
		neg := 0
		for _, tr := range tris {
			if orient2d(pts[tr[0]], pts[tr[1]], pts[tr[2]]) <= 0 {
				neg++
			}
		}
		if neg != 0 {
			t.Errorf("n=%d: %d inverted/degenerate triangles, want 0", n, neg)
		}
		want := cdtPolyArea(pts, loop)
		if got := cdtAreaSum(pts, tris); stdmath.Abs(got-want) > 1e-6*want {
			t.Errorf("n=%d: area %g, want %g", n, got, want)
		}
	}
}

// TestCDTCocircularNgonWithHoles guards the full planar-face shape that broke before: a cocircular
// outer ring plus two cocircular bore holes. The holes must be excluded (domain area = outer − holes)
// and no triangle may fall inside a hole.
func TestCDTCocircularNgonWithHoles(t *testing.T) {
	t.Parallel()
	circle := func(cx, cy, r float64, n int, start int) ([][2]float64, []int) {
		var pts [][2]float64
		var idx []int
		for i := range n {
			a := 2 * stdmath.Pi * float64(i) / float64(n)
			pts = append(pts, [2]float64{cx + r*stdmath.Cos(a), cy + r*stdmath.Sin(a)})
			idx = append(idx, start+i)
		}
		return pts, idx
	}
	outerPts, outer := circle(0, 0, 25, 30, 0)
	pts := outerPts
	loops := [][]int{outer}
	holeArea := 0.0
	for _, c := range [][2]float64{{8, 5}, {-10, -6}} {
		hp, hi := circle(c[0], c[1], 1.75, 32, len(pts))
		pts = append(pts, hp...)
		loops = append(loops, hi)
		holeArea += cdtPolyArea(pts, hi)
	}
	tris := ConstrainedDelaunay(pts, loops)
	want := cdtPolyArea(pts, outer) - holeArea
	if got := cdtAreaSum(pts, tris); stdmath.Abs(got-want) > 1e-6*want {
		t.Errorf("area %g, want %g (holes filled or mesh tangled)", got, want)
	}
	for _, c := range [][2]float64{{8, 5}, {-10, -6}} {
		for _, tr := range tris {
			cx := (pts[tr[0]][0] + pts[tr[1]][0] + pts[tr[2]][0]) / 3
			cy := (pts[tr[0]][1] + pts[tr[1]][1] + pts[tr[2]][1]) / 3
			if (cx-c[0])*(cx-c[0])+(cy-c[1])*(cy-c[1]) < 1.0 {
				t.Errorf("triangle centroid (%.2f,%.2f) lies inside hole at %v", cx, cy, c)
			}
		}
	}
}
