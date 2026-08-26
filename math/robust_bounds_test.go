// SPDX-License-Identifier: GPL-2.0-only

package math

import "testing"

// gridPoints returns an n×n grid of points spanning [0,size] in X and Y at z=0 — a stand-in
// for a normal drawing's bulk.
func gridPoints(n int, size float64) []Point3 {
	pts := make([]Point3, 0, n*n)
	for i := range n {
		for j := range n {
			x := size * float64(i) / float64(n-1)
			y := size * float64(j) / float64(n-1)
			pts = append(pts, P3(x, y, 0))
		}
	}
	return pts
}

// TestRobustBoxFramesExactlyWithoutStrays: a normal drawing (no outliers) is framed by its
// exact bounding box — no clipping.
func TestRobustBoxFramesExactlyWithoutStrays(t *testing.T) {
	pts := gridPoints(10, 1000)
	b := RobustPointBox(pts)
	if float64(b.Min.X) != 0 || float64(b.Min.Y) != 0 || float64(b.Max.X) != 1000 || float64(b.Max.Y) != 1000 {
		t.Errorf("box = %+v, want exact [0,1000]^2 (no strays should mean no clipping)", b)
	}
}

// TestRobustBoxExcludesFarStray: one entity kilometres away from a 1000-unit drawing must not
// drag the framed box out — the box stays on the bulk so Fit shows the drawing, not empty space.
func TestRobustBoxExcludesFarStray(t *testing.T) {
	pts := gridPoints(10, 1000)
	pts = append(pts, P3(-1e8, 5e7, 0)) // a stray 100,000 km away (tf-1 style)
	b := RobustPointBox(pts)
	if float64(b.Min.X) < -1e6 {
		t.Errorf("box Min.X = %v, the far stray was not excluded", b.Min.X)
	}
	if float64(b.Max.X) != 1000 || float64(b.Max.Y) != 1000 {
		t.Errorf("box = %+v, want the bulk's [0,1000] extent", b)
	}
}

// TestRobustBoxKeepsLegitimateSpread: a long, thin but legitimate drawing (every point part of
// the geometry, spread evenly) keeps its full extent — the margin scales with the largest
// dimension, so nothing is clipped.
func TestRobustBoxKeepsLegitimateSpread(t *testing.T) {
	pts := make([]Point3, 0, 200)
	for i := range 200 {
		pts = append(pts, P3(float64(i)*1000, float64(i%3), 0)) // X spans 0..199000, Y tiny
	}
	b := RobustPointBox(pts)
	if float64(b.Max.X) != 199000 {
		t.Errorf("box Max.X = %v, want 199000 (legitimate spread must not be clipped)", b.Max.X)
	}
}

// TestRobustBoxFewPointsExact: below the minimum-point threshold the exact box is used (too few
// points to distinguish a stray from real spread).
func TestRobustBoxFewPointsExact(t *testing.T) {
	pts := []Point3{P3(0, 0, 0), P3(10, 10, 10), P3(1e8, 0, 0)}
	b := RobustPointBox(pts)
	if float64(b.Max.X) != 1e8 {
		t.Errorf("box Max.X = %v, want exact 1e8 for a tiny point set", b.Max.X)
	}
}

// TestRobustBoxCoincidentPoints: identical points (zero spread) frame to that point rather than
// collapsing to nothing.
func TestRobustBoxCoincidentPoints(t *testing.T) {
	pts := make([]Point3, 50)
	for i := range pts {
		pts[i] = P3(7, 8, 9)
	}
	b := RobustPointBox(pts)
	if float64(b.Min.X) != 7 || float64(b.Max.Z) != 9 {
		t.Errorf("box = %+v, want the single point (7,8,9)", b)
	}
}

// TestRobustBoxEmpty: no points yields the empty box.
func TestRobustBoxEmpty(t *testing.T) {
	if !RobustPointBox(nil).IsEmpty() {
		t.Error("empty input should yield the empty box")
	}
}
