// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"testing"

	"oblikovati.org/math"
)

// TestWeightedKnotRemovalBoundsGeometricDeviation (audit A15, #1611): on a rational curve with
// weights spanning [0.1, 10], the homogeneous 4-space removal test must bound the TRUE 3D
// deviation (P&T eq. 5.30 weight correction) — the raw 4-space distance under-weighted heavy
// control points and silently deformed the curve beyond tolerance.
func TestWeightedKnotRemovalBoundsGeometricDeviation(t *testing.T) {
	t.Parallel()
	knots := []float64{0, 0, 0, 0.5, 1, 1, 1}
	ctrl := []math.Point3{math.P3(0, 0, 0), math.P3(1, 2, 0), math.P3(3, 2.2, 0), math.P3(4, 0, 0)}
	weights := []float64{1, 20, 0.05, 1} // 400× spread: the raw 4-space distance under-bounds the low-weight point's 3D motion
	c, err := NewBSplineCurve(2, ctrl, weights, knots)
	if err != nil {
		t.Fatalf("NewBSplineCurve: %v", err)
	}
	const tol = 2e-3
	out, removed, err := c.RemoveKnot(0.5, 1, tol)
	if err != nil {
		t.Fatalf("RemoveKnot: %v", err)
	}
	if removed == 0 {
		return // refusing is always within contract; the bound test below only applies to removals
	}
	worst := 0.0
	for i := 0; i <= 64; i++ {
		u := float64(i) / 64
		if d := float64(c.PointAt(u).DistanceTo(out.PointAt(u))); d > worst {
			worst = d
		}
	}
	if worst > tol {
		t.Errorf("knot removal moved the rational curve by %g > tol %g (weight correction missing)", worst, tol)
	}
}

// TestBenignKnotRemovalStillHappens: the weight correction must not be so conservative that a
// plainly redundant knot (inserted then removed on a benign non-rational curve) is refused.
func TestBenignKnotRemovalStillHappens(t *testing.T) {
	t.Parallel()
	knots := []float64{0, 0, 0, 1, 1, 1}
	ctrl := []math.Point3{math.P3(0, 0, 0), math.P3(1, 1, 0), math.P3(2, 0, 0)}
	c, err := NewBSplineCurve(2, ctrl, []float64{1, 1, 1}, knots)
	if err != nil {
		t.Fatalf("NewBSplineCurve: %v", err)
	}
	dense, err := c.InsertKnot(0.5, 1)
	if err != nil {
		t.Fatalf("InsertKnot: %v", err)
	}
	_, removed, err := dense.RemoveKnot(0.5, 1, 1e-9)
	if err != nil {
		t.Fatalf("RemoveKnot: %v", err)
	}
	if removed != 1 {
		t.Errorf("redundant knot not removed (removed=%d) — the 5.30 correction is over-conservative", removed)
	}
}
