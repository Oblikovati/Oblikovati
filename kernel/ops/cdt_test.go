// SPDX-License-Identifier: GPL-2.0-only

package ops

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
		a += orient2d(pts[t[0]], pts[t[1]], pts[t[2]]) / 2
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
	pts := [][2]float64{{0, 0}, {2, 0}, {2, 2}, {0, 2}}
	tris := constrainedDelaunay(pts, [][]int{{0, 1, 2, 3}})
	if len(tris) != 2 {
		t.Fatalf("square should triangulate to 2 triangles, got %d", len(tris))
	}
	if got := cdtAreaSum(pts, tris); stdmath.Abs(got-4) > 1e-9 {
		t.Errorf("square area = %g, want 4", got)
	}
}

func TestCDTConcaveLShape(t *testing.T) {
	// An L: the triangulation must NOT bridge the notch (that was the plain-Delaunay over-count).
	pts := [][2]float64{{0, 0}, {2, 0}, {2, 1}, {1, 1}, {1, 2}, {0, 2}}
	loop := []int{0, 1, 2, 3, 4, 5}
	tris := constrainedDelaunay(pts, [][]int{loop})
	want := cdtPolyArea(pts, loop) // 3
	if got := cdtAreaSum(pts, tris); stdmath.Abs(got-want) > 1e-9 {
		t.Errorf("L-shape triangulated area = %g, want %g (notch bridged or torn)", got, want)
	}
}

func TestCDTSquareWithHole(t *testing.T) {
	pts := [][2]float64{
		{0, 0}, {6, 0}, {6, 6}, {0, 6}, // outer
		{2, 2}, {4, 2}, {4, 4}, {2, 4}, // hole
	}
	tris := constrainedDelaunay(pts, [][]int{{0, 1, 2, 3}, {4, 5, 6, 7}})
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
	// A square boundary plus interior grid points: area is preserved and every point is used.
	pts := [][2]float64{{0, 0}, {3, 0}, {3, 3}, {0, 3}}
	for u := 0.5; u < 3; u += 0.5 {
		for v := 0.5; v < 3; v += 0.5 {
			pts = append(pts, [2]float64{u, v})
		}
	}
	tris := constrainedDelaunay(pts, [][]int{{0, 1, 2, 3}})
	if got := cdtAreaSum(pts, tris); stdmath.Abs(got-9) > 1e-9 {
		t.Errorf("refined square area = %g, want 9", got)
	}
	if len(tris) < 8 {
		t.Errorf("interior points should refine the mesh, got only %d triangles", len(tris))
	}
}
