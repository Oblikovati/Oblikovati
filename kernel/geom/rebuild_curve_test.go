// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// wavyMultiSpanCurve is a smooth degree-3 sinusoid-like curve carried on many spans (one
// interior knot per hump). A clean rebuild should approximate it with far fewer CVs.
func wavyMultiSpanCurve(t *testing.T) BSplineCurve {
	t.Helper()
	var pts []math.Point3
	for i := 0; i <= 24; i++ {
		x := float64(i) / 24 * 4
		pts = append(pts, math.P3(x, stdmath.Sin(x), 0))
	}
	c, err := NewFittedBSplineCurve(pts)
	if err != nil {
		t.Fatalf("wavy curve: %v", err)
	}
	return c
}

// brueteForceDeviation is an INDEPENDENT oracle: for each src sample it scans a dense grid of
// the rebuilt curve for the minimum distance (no reliance on CurveParamAtPoint3).
func bruteForceDeviation(t *testing.T, src, rebuilt BSplineCurve) float64 {
	t.Helper()
	slo, shi := src.Domain()
	rlo, rhi := rebuilt.Domain()
	maxDev := 0.0
	for i := 0; i <= 300; i++ {
		p := src.PointAt(slo + (shi-slo)*float64(i)/300)
		best := stdmath.Inf(1)
		for j := 0; j <= 600; j++ {
			d := float64(p.DistanceTo(rebuilt.PointAt(rlo + (rhi-rlo)*float64(j)/600)))
			if d < best {
				best = d
			}
		}
		if best > maxDev {
			maxDev = best
		}
	}
	return maxDev
}

func TestRebuildCurveApproximatesWithinTolerance(t *testing.T) {
	src := wavyMultiSpanCurve(t)
	rebuilt, dev, err := RebuildCurve(src, 3, 8, 0)
	if err != nil {
		t.Fatalf("RebuildCurve: %v", err)
	}
	if rebuilt.Degree != 3 {
		t.Errorf("degree = %d, want 3", rebuilt.Degree)
	}
	if len(rebuilt.Ctrl) != 8 {
		t.Errorf("control count = %d, want 8", len(rebuilt.Ctrl))
	}
	if len(rebuilt.Ctrl) >= len(src.Ctrl) {
		t.Errorf("rebuild should use fewer CVs than the source (%d), got %d", len(src.Ctrl), len(rebuilt.Ctrl))
	}
	if dev > 1e-2 {
		t.Errorf("deviation %g too large for an 8-CV cubic over a gentle sine", dev)
	}
}

func TestRebuildCurveDeviationMatchesOracle(t *testing.T) {
	src := wavyMultiSpanCurve(t)
	rebuilt, dev, err := RebuildCurve(src, 3, 6, 0)
	if err != nil {
		t.Fatalf("RebuildCurve: %v", err)
	}
	oracle := bruteForceDeviation(t, src, rebuilt)
	// The reported deviation must agree with the independent oracle to sampling resolution.
	if stdmath.Abs(dev-oracle) > 1e-3 {
		t.Errorf("reported deviation %g disagrees with oracle %g", dev, oracle)
	}
}

func TestRebuildCurveSingleSpanIsBezier(t *testing.T) {
	// A genuine cubic Bézier rebuilt to a single span (4 CVs) must reproduce it nearly exactly.
	src, err := NewBSplineCurveUniformWeights(
		3,
		[]math.Point3{math.P3(0, 0, 0), math.P3(1, 2, 0), math.P3(3, 2, 0), math.P3(4, 0, 0)},
		[]float64{0, 0, 0, 0, 1, 1, 1, 1},
	)
	if err != nil {
		t.Fatalf("bezier: %v", err)
	}
	rebuilt, dev, err := RebuildCurve(src, 3, 4, 0)
	if err != nil {
		t.Fatalf("RebuildCurve: %v", err)
	}
	if len(rebuilt.Knots) != 8 || len(rebuilt.Ctrl) != 4 {
		t.Errorf("single-span rebuild: %d ctrl, %d knots; want 4 and 8", len(rebuilt.Ctrl), len(rebuilt.Knots))
	}
	if dev > 1e-9 {
		t.Errorf("rebuilding a cubic Bézier to a single span should be near-exact, dev=%g", dev)
	}
}

func TestRebuildCurveIdempotentOnClean(t *testing.T) {
	// Rebuilding an already-clean single-span cubic to the same target changes nothing material.
	src, err := NewBSplineCurveUniformWeights(
		3,
		[]math.Point3{math.P3(0, 0, 0), math.P3(1, 1, 0), math.P3(2, -1, 0), math.P3(3, 0, 0)},
		[]float64{0, 0, 0, 0, 1, 1, 1, 1},
	)
	if err != nil {
		t.Fatalf("clean curve: %v", err)
	}
	_, dev, err := RebuildCurve(src, 3, 4, 0)
	if err != nil {
		t.Fatalf("RebuildCurve: %v", err)
	}
	if dev > 1e-9 {
		t.Errorf("rebuild of an already-clean curve should be idempotent, dev=%g", dev)
	}
}

func TestApproximateBSplineCurveValidates(t *testing.T) {
	pts := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(2, 0, 0)}
	if _, err := NewApproximatedBSplineCurve(pts, 3, 4, FitChordLength); err == nil {
		t.Error("nctrl > number of points should error")
	}
	if _, err := NewApproximatedBSplineCurve(pts, 2, 2, FitChordLength); err == nil {
		t.Error("nctrl < degree+1 should error")
	}
}

func TestApproximateBSplineCurveRejectsDegenerate(t *testing.T) {
	coincident3 := []math.Point3{math.P3(1, 1, 1), math.P3(1, 1, 1), math.P3(1, 1, 1), math.P3(1, 1, 1)}
	if _, err := NewApproximatedBSplineCurve(coincident3, 3, 4, FitChordLength); err == nil {
		t.Error("coincident 3D points should error (no chord length to parameterize by)")
	}
	coincident2 := []math.Point2{math.P2(2, 2), math.P2(2, 2), math.P2(2, 2), math.P2(2, 2)}
	if _, err := NewApproximatedBSplineCurve2d(coincident2, 3, 4, FitChordLength); err == nil {
		t.Error("coincident 2D points should error (no chord length to parameterize by)")
	}
}

func TestApproximateBSplineCurve2d(t *testing.T) {
	var pts []math.Point2
	for i := 0; i <= 20; i++ {
		x := float64(i) / 20 * 3
		pts = append(pts, math.P2(x, stdmath.Sin(x)))
	}
	c, err := NewApproximatedBSplineCurve2d(pts, 3, 6, FitChordLength)
	if err != nil {
		t.Fatalf("NewApproximatedBSplineCurve2d: %v", err)
	}
	if c.Degree != 3 || len(c.Ctrl) != 6 {
		t.Errorf("got degree %d, %d ctrl; want 3 and 6", c.Degree, len(c.Ctrl))
	}
	// Endpoints are interpolated.
	if !c.PointAt(0).IsEqualTo(pts[0], 1e-9) || !c.PointAt(1).IsEqualTo(pts[len(pts)-1], 1e-9) {
		t.Error("approximation must interpolate the endpoints")
	}
}
