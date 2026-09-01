// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

var fitTestPoints = []math.Point2{
	math.P2(0, 0), math.P2(1, 2), math.P2(3, 2.5), math.P2(4, 0), math.P2(5, -1),
}

// TestFittedParamCurvePassesThroughPoints: every parameterization must
// interpolate — curve(ūₖ) = pₖ exactly (to solver tolerance).
func TestFittedParamCurvePassesThroughPoints(t *testing.T) {
	t.Parallel()
	for _, param := range []FitParameterization{FitCentripetal, FitChordLength, FitUniform} {
		curve, ubar, err := NewFittedBSplineCurve2dParam(fitTestPoints, param)
		if err != nil {
			t.Fatalf("param %d: %v", param, err)
		}
		if len(ubar) != len(fitTestPoints) {
			t.Fatalf("param %d: %d parameters for %d points", param, len(ubar), len(fitTestPoints))
		}
		for k, p := range fitTestPoints {
			got := curve.PointAt(ubar[k])
			if float64(got.DistanceTo(p)) > 1e-9 {
				t.Errorf("param %d: curve(ū[%d]) = %v, want %v", param, k, got, p)
			}
		}
	}
}

// TestFitParameterizationsDiffer: with uneven spacing the three
// parameterizations must produce genuinely different curves between points —
// otherwise the public fit methods would be dead settings.
func TestFitParameterizationsDiffer(t *testing.T) {
	t.Parallel()
	centr, _, err := NewFittedBSplineCurve2dParam(fitTestPoints, FitCentripetal)
	if err != nil {
		t.Fatalf("centripetal: %v", err)
	}
	chord, _, err := NewFittedBSplineCurve2dParam(fitTestPoints, FitChordLength)
	if err != nil {
		t.Fatalf("chordal: %v", err)
	}
	maxGap := 0.0
	for i := 0; i <= 100; i++ {
		u := float64(i) / 100
		if d := float64(centr.PointAt(u).DistanceTo(chord.PointAt(u))); d > maxGap {
			maxGap = d
		}
	}
	if maxGap < 1e-6 {
		t.Errorf("centripetal and chordal curves coincide (max gap %g); the fit method would be a dead setting", maxGap)
	}
}

// TestClosedFittedCurveInterpolatesAndCloses: the closed fit passes through
// every loop point at its returned parameter and returns to points[0] at the
// final (wrap) parameter, with a tangent-continuous seam.
func TestClosedFittedCurveInterpolatesAndCloses(t *testing.T) {
	t.Parallel()
	loop := []math.Point2{math.P2(2, 0), math.P2(0, 1.5), math.P2(-2, 0), math.P2(0, -1.5)}
	curve, ubar, err := NewClosedFittedBSplineCurve2d(loop, FitCentripetal)
	if err != nil {
		t.Fatalf("closed fit: %v", err)
	}
	if len(ubar) != len(loop)+1 {
		t.Fatalf("parameters = %d, want %d (loop + closing wrap)", len(ubar), len(loop)+1)
	}
	for k, p := range loop {
		if got := curve.PointAt(ubar[k]); float64(got.DistanceTo(p)) > 1e-9 {
			t.Errorf("curve(ū[%d]) = %v, want %v", k, got, p)
		}
	}
	if got := curve.PointAt(ubar[len(loop)]); float64(got.DistanceTo(loop[0])) > 1e-9 {
		t.Errorf("wrap point = %v, want the seam back at %v", got, loop[0])
	}
	seamIn := curve.TangentAt(ubar[len(loop)] - 1e-9)
	seamOut := curve.TangentAt(ubar[0] + 1e-9)
	angle := stdmath.Atan2(float64(seamIn.Cross(seamOut)), float64(seamIn.Dot(seamOut)))
	if stdmath.Abs(angle) > 1e-3 {
		t.Errorf("seam tangent kink = %g rad, want continuous (< 1e-3)", angle)
	}
}

// TestClosedFitRejectsTinyLoops: fewer than 3 points cannot enclose a loop.
func TestClosedFitRejectsTinyLoops(t *testing.T) {
	t.Parallel()
	if _, _, err := NewClosedFittedBSplineCurve2d(fitTestPoints[:2], FitCentripetal); err == nil {
		t.Error("a 2-point closed fit must be rejected")
	}
}

// TestFittedBSplineCurve3dParamInterpolates: the 3D variant interpolates too.
func TestFittedBSplineCurve3dParamInterpolates(t *testing.T) {
	t.Parallel()
	pts := []math.Point3{
		math.P3(0, 0, 0), math.P3(1, 2, 1), math.P3(3, 2, -1), math.P3(4, 0, 0),
	}
	curve, ubar, err := NewFittedBSplineCurveParam(pts, FitCentripetal)
	if err != nil {
		t.Fatalf("fit 3D: %v", err)
	}
	for k, p := range pts {
		if got := curve.PointAt(ubar[k]); float64(got.DistanceTo(p)) > 1e-9 {
			t.Errorf("curve(ū[%d]) = %v, want %v", k, got, p)
		}
	}
}

// TestClosedFittedBSplineCurve3dCloses: 3D closed loops seam back to start.
func TestClosedFittedBSplineCurve3dCloses(t *testing.T) {
	t.Parallel()
	loop := []math.Point3{math.P3(1, 0, 0), math.P3(0, 1, 0.5), math.P3(-1, 0, 0), math.P3(0, -1, -0.5)}
	curve, ubar, err := NewClosedFittedBSplineCurve(loop, FitCentripetal)
	if err != nil {
		t.Fatalf("closed fit 3D: %v", err)
	}
	if got := curve.PointAt(ubar[len(loop)]); float64(got.DistanceTo(loop[0])) > 1e-9 {
		t.Errorf("wrap point = %v, want %v", got, loop[0])
	}
}
