// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/math"
)

// triArea2 returns twice the absolute area of triangle (a,b,c).
func triArea2(a, b, c math.Point2) float64 {
	v := (b.X-a.X)*(c.Y-a.Y) - (c.X-a.X)*(b.Y-a.Y)
	if v < 0 {
		return -v
	}
	return v
}

// sumFillArea sums the areas of the triangles FillTriangles returns for verts.
func sumFillArea(verts []math.Point2, tris [][3]int) float64 {
	var area float64
	for _, t := range tris {
		area += triArea2(verts[t[0]], verts[t[1]], verts[t[2]]) / 2
	}
	return area
}

// TestFillTrianglesSquare: a unit square triangulates to its full area (2 triangles).
func TestFillTrianglesSquare(t *testing.T) {
	outer := []math.Point2{math.P2(0, 0), math.P2(4, 0), math.P2(4, 4), math.P2(0, 4)}
	tris := FillTriangles(outer, nil)
	if len(tris) != 2 {
		t.Fatalf("square gave %d triangles, want 2", len(tris))
	}
	if got := sumFillArea(outer, tris); got != 16 {
		t.Errorf("filled area = %.3f, want 16", got)
	}
}

// TestFillTrianglesWithHole: a 10×10 square with a 2×2 hole fills to area 100−4 = 96, and every
// triangle index addresses outer++hole in order — the contract the region-fill overlay relies on.
func TestFillTrianglesWithHole(t *testing.T) {
	outer := []math.Point2{math.P2(0, 0), math.P2(10, 0), math.P2(10, 10), math.P2(0, 10)}
	hole := []math.Point2{math.P2(4, 4), math.P2(6, 4), math.P2(6, 6), math.P2(4, 6)}
	verts := append(append([]math.Point2(nil), outer...), hole...)
	tris := FillTriangles(outer, [][]math.Point2{hole})
	if len(tris) == 0 {
		t.Fatal("no triangles for a square with a hole")
	}
	for _, tr := range tris {
		for _, idx := range tr {
			if idx < 0 || idx >= len(verts) {
				t.Fatalf("triangle index %d out of range [0,%d)", idx, len(verts))
			}
		}
	}
	if got := sumFillArea(verts, tris); got < 95.9 || got > 96.1 {
		t.Errorf("filled area = %.3f, want ~96 (100 − 4 hole)", got)
	}
}

// TestFillTrianglesDegenerate: fewer than 3 points bounds no area, so no triangles.
func TestFillTrianglesDegenerate(t *testing.T) {
	if tris := FillTriangles([]math.Point2{math.P2(0, 0), math.P2(1, 0)}, nil); len(tris) != 0 {
		t.Errorf("a 2-point loop gave %d triangles, want 0", len(tris))
	}
}
