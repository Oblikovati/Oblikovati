// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// sqrtHalf is √2/2, the middle weight of a rational quadratic quarter circle.
var sqrtHalf = stdmath.Sqrt2 / 2

func TestInsertThenRemoveRoundTrip(t *testing.T) {
	t.Parallel()
	c := sampleCubicCurve(t)
	refined, err := c.InsertKnot(0.25, 1)
	if err != nil {
		t.Fatalf("InsertKnot: %v", err)
	}
	got, removed, err := refined.RemoveKnot(0.25, 1, 1e-9)
	if err != nil {
		t.Fatalf("RemoveKnot: %v", err)
	}
	if removed != 1 {
		t.Fatalf("removed = %d, want 1 (the just-inserted knot is redundant)", removed)
	}
	if len(got.Ctrl) != len(c.Ctrl) {
		t.Errorf("control count = %d, want %d (back to the original)", len(got.Ctrl), len(c.Ctrl))
	}
	curvesAgree(t, c, got, 1e-9)
}

func TestRemoveKnotRepeatedRoundTrip(t *testing.T) {
	t.Parallel()
	c := sampleCubicCurve(t)
	refined, err := c.InsertKnot(0.25, 3) // multiplicity 3 of a fresh knot
	if err != nil {
		t.Fatalf("InsertKnot x3: %v", err)
	}
	got, removed, err := refined.RemoveKnot(0.25, 3, 1e-9)
	if err != nil {
		t.Fatalf("RemoveKnot x3: %v", err)
	}
	if removed != 3 {
		t.Fatalf("removed = %d, want 3", removed)
	}
	curvesAgree(t, c, got, 1e-9)
}

func TestRemoveKnotKeepsNeededKnot(t *testing.T) {
	t.Parallel()
	// 0.5 is an intrinsic knot of the sample curve; removing it at a tight tolerance
	// must fail (the shape genuinely needs it) and leave the curve untouched.
	c := sampleCubicCurve(t)
	got, removed, err := c.RemoveKnot(0.5, 1, 1e-12)
	if err != nil {
		t.Fatalf("RemoveKnot: %v", err)
	}
	if removed != 0 {
		t.Errorf("removed = %d, want 0 (knot is geometrically required at tight tol)", removed)
	}
	if len(got.Ctrl) != len(c.Ctrl) {
		t.Errorf("control count changed to %d; a failed removal must leave the curve intact", len(got.Ctrl))
	}
}

func TestRemoveKnotRejectsNonInterior(t *testing.T) {
	t.Parallel()
	c := sampleCubicCurve(t)
	if _, _, err := c.RemoveKnot(0.0, 1, 1e-6); err == nil {
		t.Error("removing a boundary knot should error")
	}
	if _, _, err := c.RemoveKnot(0.3, 1, 1e-6); err == nil {
		t.Error("removing a knot that does not appear should error")
	}
	if _, _, err := c.RemoveKnot(0.5, 0, 1e-6); err == nil {
		t.Error("a zero removal count should error")
	}
}

func TestRemoveKnot2dRoundTrip(t *testing.T) {
	t.Parallel()
	c := quarterCircleCurve2d(t)
	refined, err := c.InsertKnot(0.5, 1)
	if err != nil {
		t.Fatalf("InsertKnot: %v", err)
	}
	got, removed, err := refined.RemoveKnot(0.5, 1, 1e-9)
	if err != nil {
		t.Fatalf("RemoveKnot: %v", err)
	}
	if removed != 1 {
		t.Fatalf("removed = %d, want 1", removed)
	}
	lo, hi := c.Domain()
	for i := 0; i <= 20; i++ {
		u := lo + (hi-lo)*float64(i)/20
		if !c.PointAt(u).IsEqualTo(got.PointAt(u), 1e-9) {
			t.Fatalf("2d curve diverges at u=%g after round-trip", u)
		}
	}
}

func TestInsertThenRemoveSurfaceRoundTrip(t *testing.T) {
	t.Parallel()
	s := sampleQuadraticSurface(t)
	refined, err := s.InsertKnotU(0.5, 1)
	if err != nil {
		t.Fatalf("InsertKnotU: %v", err)
	}
	gotU, removed, err := refined.RemoveKnotU(0.5, 1, 1e-9)
	if err != nil {
		t.Fatalf("RemoveKnotU: %v", err)
	}
	if removed != 1 || len(gotU.Ctrl) != len(s.Ctrl) {
		t.Fatalf("U removal: removed=%d net rows=%d, want 1 and %d", removed, len(gotU.Ctrl), len(s.Ctrl))
	}
	refinedV, err := s.InsertKnotV(0.5, 1)
	if err != nil {
		t.Fatalf("InsertKnotV: %v", err)
	}
	gotV, removedV, err := refinedV.RemoveKnotV(0.5, 1, 1e-9)
	if err != nil {
		t.Fatalf("RemoveKnotV: %v", err)
	}
	if removedV != 1 || len(gotV.Ctrl[0]) != len(s.Ctrl[0]) {
		t.Fatalf("V removal: removed=%d net cols=%d, want 1 and %d", removedV, len(gotV.Ctrl[0]), len(s.Ctrl[0]))
	}
	for i := 0; i <= 10; i++ {
		for j := 0; j <= 10; j++ {
			u, v := float64(i)/10, float64(j)/10
			if !s.PointAt(u, v).IsEqualTo(gotV.PointAt(u, v), 1e-9) {
				t.Fatalf("surface diverges at (%g,%g) after V round-trip", u, v)
			}
		}
	}
}

// quarterCircleCurve2d is the rational quadratic quarter circle as a 2D curve, the
// rational case for the 2D removal round-trip.
func quarterCircleCurve2d(t *testing.T) BSplineCurve2d {
	t.Helper()
	c, err := NewBSplineCurve2d(
		2,
		[]math.Point2{math.P2(1, 0), math.P2(1, 1), math.P2(0, 1)},
		[]float64{1, sqrtHalf, 1},
		[]float64{0, 0, 0, 1, 1, 1},
	)
	if err != nil {
		t.Fatalf("quarter-circle 2d: %v", err)
	}
	// A fresh interior knot must be insertable, so widen the domain by inserting first.
	refined, err := c.InsertKnot(0.5, 1)
	if err != nil {
		t.Fatalf("seed insert: %v", err)
	}
	return refined
}
