// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import (
	"math/big"
	"testing"
)

func poly(pts ...[3]float64) []Point {
	out := make([]Point, len(pts))
	for i, p := range pts {
		out[i] = pt(p)
	}
	return out
}

func TestTriangulateSimplePolygon(t *testing.T) {
	cases := []struct {
		name      string
		poly      []Point
		axis      int
		wantCount int
	}{
		{
			name:      "square CCW",
			poly:      poly([3]float64{0, 0, 0}, [3]float64{4, 0, 0}, [3]float64{4, 4, 0}, [3]float64{0, 4, 0}),
			axis:      2,
			wantCount: 2,
		},
		{
			name:      "square CW (needs reversal)",
			poly:      poly([3]float64{0, 0, 0}, [3]float64{0, 4, 0}, [3]float64{4, 4, 0}, [3]float64{4, 0, 0}),
			axis:      2,
			wantCount: 2,
		},
		{
			name: "L-shape (reflex vertex)",
			poly: poly([3]float64{0, 0, 0}, [3]float64{4, 0, 0}, [3]float64{4, 2, 0},
				[3]float64{2, 2, 0}, [3]float64{2, 4, 0}, [3]float64{0, 4, 0}),
			axis:      2,
			wantCount: 4,
		},
		{
			name: "collinear vertex preserved",
			poly: poly([3]float64{0, 0, 0}, [3]float64{2, 0, 0}, [3]float64{4, 0, 0},
				[3]float64{4, 4, 0}, [3]float64{0, 4, 0}),
			axis:      2,
			wantCount: 3, // (2,0,0) is kept as a vertex → one extra triangle
		},
		{
			name:      "square in x=0 plane (drop x)",
			poly:      poly([3]float64{0, 0, 0}, [3]float64{0, 4, 0}, [3]float64{0, 4, 4}, [3]float64{0, 0, 4}),
			axis:      0,
			wantCount: 2,
		},
	}
	for _, tc := range cases {
		tris := triangulateSimplePolygon(tc.poly, tc.axis)
		if len(tris) != tc.wantCount {
			t.Fatalf("%s: got %d triangles, want %d", tc.name, len(tris), tc.wantCount)
		}
		assertPolygonTiling(t, tc.name, tc.poly, tris, tc.axis)
	}
}

// assertPolygonTiling checks exact area conservation, non-degeneracy, consistent
// (CCW) orientation, and that every triangle vertex is a polygon vertex.
func assertPolygonTiling(t *testing.T, name string, poly []Point, tris [][3]Point, axis int) {
	t.Helper()
	want := new(big.Rat).Abs(signedArea2(poly, axis))
	sum := new(big.Rat)
	for _, tt := range tris {
		d := orient2Val(tt[0], tt[1], tt[2], axis)
		if d.Sign() <= 0 {
			t.Fatalf("%s: triangle is degenerate or clockwise (det sign %d)", name, d.Sign())
		}
		sum.Add(sum, d)
		for _, v := range tt {
			if !isPolygonVertex(poly, v) {
				t.Fatalf("%s: triangle uses a non-polygon vertex %v", name, v.Round())
			}
		}
	}
	if sum.Cmp(want) != 0 {
		t.Fatalf("%s: area not conserved: sum=%s want=%s", name, sum.RatString(), want.RatString())
	}
}

// TestTriangulateDegenerateInput documents the defensive contract: a fully
// collinear "polygon" (zero area, no ear) yields no triangles and, crucially, does
// not hang — clipOneEar reports no progress and triangulation bails.
func TestTriangulateDegenerateInput(t *testing.T) {
	collinear := poly([3]float64{0, 0, 0}, [3]float64{1, 0, 0}, [3]float64{2, 0, 0}, [3]float64{3, 0, 0})
	if got := triangulateSimplePolygon(collinear, 2); len(got) != 0 {
		t.Fatalf("degenerate polygon: got %d triangles, want 0", len(got))
	}
}

func isPolygonVertex(poly []Point, v Point) bool {
	for _, p := range poly {
		if p.Equal(v) {
			return true
		}
	}
	return false
}
