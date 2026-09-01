// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// triAreaSum sums the unsigned areas of triangles addressing the combined vertex list.
func triAreaSum(verts []math.Point2, tris [][3]int) float64 {
	var a float64
	for _, t := range tris {
		p, q, r := verts[t[0]], verts[t[1]], verts[t[2]]
		a += stdmath.Abs((q.X-p.X)*(r.Y-p.Y)-(r.X-p.X)*(q.Y-p.Y)) / 2
	}
	return a
}

func ccwSquare(cx, cy, r float64) []math.Point2 {
	return []math.Point2{math.P2(cx-r, cy-r), math.P2(cx+r, cy-r), math.P2(cx+r, cy+r), math.P2(cx-r, cy+r)}
}

// TestEarcutMultiHoleArea pins the multi-hole planar triangulation: a rectangle with several
// holes must triangulate to (rectangle − holes) in area. Regression for the cap-face bug
// where a hole grid lost ~half its area because every hole bridged to one shared corner and
// the ear-clipper stalled (see earcut.go).
func TestEarcutMultiHoleArea(t *testing.T) {
	t.Parallel()
	outer := []math.Point2{math.P2(0, 0), math.P2(4, 0), math.P2(4, 3), math.P2(0, 3)}

	// Axis-aligned holes that share exact X/Y lines — the degenerate case that defeated the
	// single-bridge ear-clipper.
	aligned := [][2]float64{{0.55, 0.55}, {3.45, 0.55}, {0.55, 2.45}, {3.45, 2.45}}
	for nh := 1; nh <= len(aligned); nh++ {
		holes := make([][]math.Point2, 0, nh)
		combined := append([]math.Point2(nil), outer...)
		for i := 0; i < nh; i++ {
			h := ccwSquare(aligned[i][0], aligned[i][1], 0.15)
			holes = append(holes, h)
			combined = append(combined, h...)
		}
		got, want := triAreaSum(combined, earcut(outer, holes)), 12.0-float64(nh)*0.09
		if stdmath.Abs(got-want) > 1e-9 {
			t.Errorf("aligned grid nh=%d: area=%.6f, want %.6f", nh, got, want)
		}
	}

	// A dense 4×4 grid — 16 interacting bridges.
	var holes [][]math.Point2
	combined := append([]math.Point2(nil), outer...)
	for gx := range 4 {
		for gy := range 4 {
			h := ccwSquare(0.5+float64(gx)*1.0, 0.4+float64(gy)*0.7, 0.15)
			holes = append(holes, h)
			combined = append(combined, h...)
		}
	}
	got, want := triAreaSum(combined, earcut(outer, holes)), 12.0-16*0.09
	if stdmath.Abs(got-want) > 1e-9 {
		t.Errorf("4x4 grid: area=%.6f, want %.6f", got, want)
	}
}

// TestEarcutSingleHole covers the common annular case (one hole).
func TestEarcutSingleHole(t *testing.T) {
	t.Parallel()
	outer := []math.Point2{math.P2(0, 0), math.P2(10, 0), math.P2(10, 10), math.P2(0, 10)}
	hole := ccwSquare(5, 5, 2) // 4×4 hole, area 16
	combined := append(append([]math.Point2(nil), outer...), hole...)
	if got, want := triAreaSum(combined, earcut(outer, [][]math.Point2{hole})), 100.0-16.0; stdmath.Abs(got-want) > 1e-9 {
		t.Errorf("single hole: area=%.6f, want %.6f", got, want)
	}
}
