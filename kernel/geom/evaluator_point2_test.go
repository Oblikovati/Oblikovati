// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"math"
	"testing"

	gmath "oblikovati.org/math"
)

// TestCurveParamAtPoint2RoundTrips covers CurveParamAtPoint2 across each 2D curve kind: a point
// sampled at a known parameter must resolve back to a parameter that evaluates to (approximately)
// the same point — exercising the per-kind param solvers (segment/circle/arc/polyline).
func TestCurveParamAtPoint2RoundTrips(t *testing.T) {
	t.Parallel()
	poly, err := NewPolyline2d([]gmath.Point2{gmath.P2(0, 0), gmath.P2(3, 0), gmath.P2(3, 4)})
	if err != nil {
		t.Fatalf("NewPolyline2d: %v", err)
	}
	// Each query point is a clear interior point on the curve (corners are intentionally
	// avoided — a vertex is an ambiguous projection target).
	cases := []struct {
		name  string
		c     Curve2
		query gmath.Point2
	}{
		{"segment", NewLineSegment2d(gmath.P2(0, 0), gmath.P2(10, 0)), gmath.P2(4, 0)},
		{"circle", NewCircle2d(gmath.P2(0, 0), 2), gmath.P2(2, 0)},
		{"arc", NewArc2d(gmath.P2(0, 0), 1, 0, math.Pi/2), gmath.P2(float64(math.Sqrt2)/2, float64(math.Sqrt2)/2)},
		{"polyline", poly, gmath.P2(1.5, 0)},
	}
	for _, tc := range cases {
		got, _ := CurveParamAtPoint2(tc.c, tc.query)
		back := tc.c.PointAt(got)
		if d := math.Hypot(float64(back.X-tc.query.X), float64(back.Y-tc.query.Y)); d > 1e-6 {
			t.Errorf("%s: param %.4f maps to %v, want ~%v (off by %g)", tc.name, got, back, tc.query, d)
		}
	}
}

// TestCurveRangeBox2BoundsCurve covers CurveRangeBox2: the reported box must contain points
// sampled along each curve.
func TestCurveRangeBox2BoundsCurve(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]Curve2{
		"segment": NewLineSegment2d(gmath.P2(-1, -2), gmath.P2(3, 4)),
		"circle":  NewCircle2d(gmath.P2(1, 1), 2),
		"arc":     NewArc2d(gmath.P2(0, 0), 5, 0, math.Pi),
	} {
		box := CurveRangeBox2(c)
		for _, tt := range []float64{0, 0.25, 0.5, 0.75, 1} {
			p := c.PointAt(tt)
			if p.X < box.Min.X-1e-6 || p.X > box.Max.X+1e-6 || p.Y < box.Min.Y-1e-6 || p.Y > box.Max.Y+1e-6 {
				t.Errorf("%s: sampled point %v outside range box %v..%v", name, p, box.Min, box.Max)
			}
		}
	}
}

// TestCurveParamAtPoint2GenericPath covers the sampled multistart solver for curves with no
// closed-form inverse: an ellipse query exercises genericParamAtPoint2 → sampleDistances2 →
// clusterMinima2 → refineClosest2. The recovered parameter must evaluate back to the query point.
func TestCurveParamAtPoint2GenericPath(t *testing.T) {
	t.Parallel()
	el, err := NewEllipseFull2d(gmath.P2(0, 0), gmath.V2(1, 0), 3, 1)
	if err != nil {
		t.Fatalf("NewEllipseFull2d: %v", err)
	}
	want := el.PointAt(0.3) // a point genuinely on the ellipse, away from the axis vertices
	got, _ := CurveParamAtPoint2(el, want)
	back := el.PointAt(got)
	if d := math.Hypot(float64(back.X-want.X), float64(back.Y-want.Y)); d > 1e-4 {
		t.Errorf("ellipse param %.4f maps to %v, want ~%v (off by %g)", got, back, want, d)
	}
}
