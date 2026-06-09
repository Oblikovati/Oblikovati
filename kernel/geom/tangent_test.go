// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// The analytic TangentAt of every curve must agree with a central finite
// difference of PointAt. This one metamorphic check exercises PointAt,
// TangentAt, and Domain across all curve types — if a derivative formula is
// wrong (a missing chain factor, say), it fails here.

const (
	fdStep = 1e-6
	fdTol  = 1e-5
)

func curve3Samples(t *testing.T) []Curve3 {
	t.Helper()
	line, _ := NewLine(math.P3(1, 2, 3), math.V3(1, 1, 0))
	circle, _ := NewCircle(math.P3(1, 0, 0), math.V3(0, 0, 1), 5)
	arc, _ := NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 3, 0.2, stdmath.Pi)
	ell, _ := NewEllipseFull(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 4, 2)
	earc, _ := NewEllipticalArc(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 4, 2, 0.1, 1.2)
	poly, _ := NewPolyline([]math.Point3{math.P3(0, 0, 0), math.P3(3, 0, 0), math.P3(3, 4, 0)})
	return []Curve3{line, NewLineSegment(math.P3(0, 0, 0), math.P3(2, 5, 9)), circle, arc, ell, earc, poly, quarterCircleNURBS(t)}
}

func curve2Samples(t *testing.T) []Curve2 {
	t.Helper()
	line, _ := NewLine2d(math.P2(1, 2), math.V2(2, 1))
	arc, _ := Arc2dByThreePoints(math.P2(1, 0), math.P2(0, 1), math.P2(-1, 0))
	ell, _ := NewEllipseFull2d(math.P2(0, 0), math.V2(1, 0), 4, 2)
	earc, _ := NewEllipticalArc2d(math.P2(0, 0), math.V2(1, 0), 4, 2, 0.1, 1.2)
	poly, _ := NewPolyline2d([]math.Point2{math.P2(0, 0), math.P2(3, 0), math.P2(3, 4)})
	return []Curve2{line, NewLineSegment2d(math.P2(0, 0), math.P2(2, 5)), NewCircle2d(math.P2(1, 1), 2), arc, ell, earc, poly}
}

func TestAllDomainsAreOrdered(t *testing.T) {
	for i, c := range curve3Samples(t) {
		if lo, hi := c.Domain(); lo >= hi {
			t.Errorf("curve3[%d] Domain = [%v,%v], want lo < hi", i, lo, hi)
		}
	}
	for i, c := range curve2Samples(t) {
		if lo, hi := c.Domain(); lo >= hi {
			t.Errorf("curve2[%d] Domain = [%v,%v], want lo < hi", i, lo, hi)
		}
	}
}

func TestCurve3TangentMatchesFiniteDifference(t *testing.T) {
	for i, c := range curve3Samples(t) {
		for _, tp := range []float64{0.2, 0.45, 0.8} {
			fd := c.PointAt(tp - fdStep).VectorTo(c.PointAt(tp + fdStep)).Scale(1 / (2 * fdStep))
			if !fd.IsEqualTo(c.TangentAt(tp), fdTol) {
				t.Errorf("curve3[%d] tangent at %v: analytic %v, finite-diff %v", i, tp, c.TangentAt(tp), fd)
			}
		}
	}
}

func TestCurve2TangentMatchesFiniteDifference(t *testing.T) {
	for i, c := range curve2Samples(t) {
		for _, tp := range []float64{0.2, 0.45, 0.8} {
			fd := c.PointAt(tp - fdStep).VectorTo(c.PointAt(tp + fdStep)).Scale(1 / (2 * fdStep))
			if !fd.IsEqualTo(c.TangentAt(tp), fdTol) {
				t.Errorf("curve2[%d] tangent at %v: analytic %v, finite-diff %v", i, tp, c.TangentAt(tp), fd)
			}
		}
	}
}
