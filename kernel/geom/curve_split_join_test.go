// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"testing"

	"oblikovati.org/math"
)

// arcLikeCurve is a degree-3 clamped curve through 5 control points (a gentle S).
func arcLikeCurve(t *testing.T) BSplineCurve {
	t.Helper()
	ctrl := []math.Point3{
		math.P3(0, 0, 0), math.P3(1, 2, 0), math.P3(2, -1, 1), math.P3(3, 1, 0), math.P3(4, 0, 0),
	}
	c, err := NewBSplineCurve(3, ctrl, []float64{1, 1, 1, 1, 1}, []float64{0, 0, 0, 0, 0.5, 1, 1, 1, 1})
	if err != nil {
		t.Fatalf("arcLikeCurve: %v", err)
	}
	return c
}

// TestSplitCurveReproducesOriginal: the two halves, sampled over their own [0,1], trace the same
// points as the original curve's two parameter halves.
func TestSplitCurveReproducesOriginal(t *testing.T) {
	t.Parallel()
	c := arcLikeCurve(t)
	a, b, err := SplitCurve(c, 0.5)
	if err != nil {
		t.Fatalf("SplitCurve: %v", err)
	}
	for i := 0; i <= 10; i++ {
		s := float64(i) / 10
		if d := a.PointAt(s).DistanceTo(c.PointAt(0.5 * s)); d > 1e-7 {
			t.Errorf("left half off by %.3g at s=%g", d, s)
		}
		if d := b.PointAt(s).DistanceTo(c.PointAt(0.5 + 0.5*s)); d > 1e-7 {
			t.Errorf("right half off by %.3g at s=%g", d, s)
		}
	}
}

func TestSplitCurveRejectsOutOfDomain(t *testing.T) {
	t.Parallel()
	c := arcLikeCurve(t)
	if _, _, err := SplitCurve(c, 0); err == nil {
		t.Error("split at the domain start should error")
	}
	if _, _, err := SplitCurve(c, 1.5); err == nil {
		t.Error("split past the domain should error")
	}
}

// TestJoinCurvesRoundTripsSplit: splitting then re-joining yields a curve tracing the SAME image as
// the original (the seam reparametrizes, so identity is checked as path containment, not per-param).
func TestJoinCurvesRoundTripsSplit(t *testing.T) {
	t.Parallel()
	c := arcLikeCurve(t)
	a, b, err := SplitCurve(c, 0.4)
	if err != nil {
		t.Fatalf("SplitCurve: %v", err)
	}
	joined, err := JoinCurves([]BSplineCurve{a, b})
	if err != nil {
		t.Fatalf("JoinCurves: %v", err)
	}
	if d := joined.PointAt(0).DistanceTo(c.PointAt(0)); d > 1e-7 {
		t.Errorf("rejoined start off by %.3g", d)
	}
	if d := joined.PointAt(1).DistanceTo(c.PointAt(1)); d > 1e-7 {
		t.Errorf("rejoined end off by %.3g", d)
	}
	// Every rejoined sample lies on the original curve's image (min distance over a dense sampling).
	orig := make([]math.Point3, 201)
	for i := range orig {
		orig[i] = c.PointAt(float64(i) / 200)
	}
	for i := 0; i <= 40; i++ {
		p := joined.PointAt(float64(i) / 40)
		best := math.Scalar(1e18)
		for _, q := range orig {
			if d := p.DistanceTo(q); d < best {
				best = d
			}
		}
		if best > 1e-3 {
			t.Errorf("rejoined sample %d strayed %.3g off the original image", i, best)
		}
	}
}

// TestJoinCurvesElevatesMixedDegree: a degree-1 segment and a degree-3 curve join into one curve
// passing through both, interpolating the seam corner.
func TestJoinCurvesElevatesMixedDegree(t *testing.T) {
	t.Parallel()
	seg, _ := NewBSplineCurve(1, []math.Point3{math.P3(4, 0, 0), math.P3(6, 0, 0)}, []float64{1, 1}, []float64{0, 0, 1, 1})
	c := arcLikeCurve(t)
	joined, err := JoinCurves([]BSplineCurve{c, seg})
	if err != nil {
		t.Fatalf("JoinCurves: %v", err)
	}
	if d := joined.PointAt(0).DistanceTo(math.P3(0, 0, 0)); d > 1e-7 {
		t.Errorf("joined start off by %.3g", d)
	}
	if d := joined.PointAt(1).DistanceTo(math.P3(6, 0, 0)); d > 1e-7 {
		t.Errorf("joined end off by %.3g", d)
	}
}

func TestJoinCurvesRejectsGap(t *testing.T) {
	t.Parallel()
	a, _ := NewBSplineCurve(1, []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0)}, []float64{1, 1}, []float64{0, 0, 1, 1})
	b, _ := NewBSplineCurve(1, []math.Point3{math.P3(5, 5, 5), math.P3(6, 6, 6)}, []float64{1, 1}, []float64{0, 0, 1, 1})
	if _, err := JoinCurves([]BSplineCurve{a, b}); err == nil {
		t.Error("joining curves with a gap should error")
	}
}
