// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// trueAngleOf returns the geometric angle of point p about the origin in the +X/+Y frame.
func trueAngleOf(p math.Point2) float64 { return stdmath.Atan2(float64(p.Y), float64(p.X)) }

// TestEllipticalArcAddUsesTrueAngle is the #1829 regression: EllipticalArcs.Add interprets its
// start/end as Inventor's TRUE geometric angle θ, not the parametric (eccentric-anomaly) angle. On a
// 2:1 ellipse (major axis +X) an arc authored from θ=45° must actually START on the 45° ray — the
// parametric interpretation would place it at ~26.57°.
func TestEllipticalArcAddUsesTrueAngle(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	e := s.EllipticalArcs().Add(math.P2(0, 0), math.V2(1, 0), 2, 1, stdmath.Pi/4, stdmath.Pi/2)

	pts := sampleEllipticalArcEntityN(e, 64)
	if got := trueAngleOf(pts[0]); stdmath.Abs(got-stdmath.Pi/4) > 1e-9 {
		t.Errorf("arc start true angle = %.6f rad (%.3f°), want π/4 (45°) — Add must interpret θ as the true angle (#1829)",
			got, got*180/stdmath.Pi)
	}
	// θ=90° is on the minor axis where θ=a, so the end is exactly (0, minorR)=(0,1).
	end := pts[len(pts)-1]
	if stdmath.Abs(float64(end.X)) > 1e-9 || stdmath.Abs(float64(end.Y)-1) > 1e-9 {
		t.Errorf("arc end = %v, want (0,1)", end)
	}
}

// TestEllipticalArcAddParametricIsVerbatim: AddParametric stores the eccentric-anomaly angle directly
// (the DXF/restore path), so its start point is the parametric point P(a), not the true-angle point.
func TestEllipticalArcAddParametricIsVerbatim(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	const a = stdmath.Pi / 4 // parametric
	e := s.EllipticalArcs().AddParametric(math.P2(0, 0), math.V2(1, 0), 2, 1, a, stdmath.Pi/2)

	pts := sampleEllipticalArcEntityN(e, 64)
	// P(a) = (Rmaj·cos a, Rmin·sin a) = (2·√2/2, 1·√2/2) = (√2, √2/2).
	wantX, wantY := stdmath.Sqrt2, stdmath.Sqrt2/2
	if stdmath.Abs(float64(pts[0].X)-wantX) > 1e-9 || stdmath.Abs(float64(pts[0].Y)-wantY) > 1e-9 {
		t.Errorf("AddParametric start = %v, want (%.4f,%.4f) — must NOT convert", pts[0], wantX, wantY)
	}
}

// TestParamArcFromTruePreservesSweep: the θ→a conversion keeps the sweep direction and keeps both
// endpoints on their true rays, including a CW arc and one crossing the atan2 branch.
func TestParamArcFromTrue(t *testing.T) {
	const rMaj, rMin = 3.0, 1.0
	cases := []struct{ startTrue, endTrue float64 }{
		{stdmath.Pi / 6, 2 * stdmath.Pi / 3},     // CCW, off-axis
		{2 * stdmath.Pi / 3, stdmath.Pi / 6},     // CW
		{5 * stdmath.Pi / 6, 7 * stdmath.Pi / 6}, // crosses θ=π
	}
	for _, c := range cases {
		aS, aE := paramArcFromTrue(c.startTrue, c.endTrue, rMaj, rMin)
		// Each parametric endpoint must sit on its true-angle ray.
		if got := paramPointTrueAngle(aS, rMaj, rMin); angleDiff(got, c.startTrue) > 1e-9 {
			t.Errorf("start: param a=%.4f lands at true %.4f, want %.4f", aS, got, c.startTrue)
		}
		if got := paramPointTrueAngle(aE, rMaj, rMin); angleDiff(got, c.endTrue) > 1e-9 {
			t.Errorf("end: param a=%.4f lands at true %.4f, want %.4f", aE, got, c.endTrue)
		}
		// Sweep sign preserved.
		if (c.endTrue-c.startTrue > 0) != (aE-aS > 0) {
			t.Errorf("sweep sign flipped: true %.4f→%.4f gave param %.4f→%.4f", c.startTrue, c.endTrue, aS, aE)
		}
	}
}

// paramPointTrueAngle is the true geometric angle of the parametric point P(a).
func paramPointTrueAngle(a, rMaj, rMin float64) float64 {
	return stdmath.Atan2(rMin*stdmath.Sin(a), rMaj*stdmath.Cos(a))
}

// angleDiff is the absolute smallest difference between two angles (mod 2π).
func angleDiff(x, y float64) float64 {
	d := stdmath.Mod(x-y, 2*stdmath.Pi)
	if d > stdmath.Pi {
		d -= 2 * stdmath.Pi
	}
	if d < -stdmath.Pi {
		d += 2 * stdmath.Pi
	}
	return stdmath.Abs(d)
}
