// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// CurveIsClosed answers by measurement — do the curve's domain ends meet, against its own scale — not
// by curve kind. The type switch it replaces classed every kind it did not list as open, and the exact
// ruled∩quadric section then skipped the closed-loop re-emission both boolean walls rely on to weld
// (Oblikovati/Oblikovati#3489). These pin the measurement over every family the boolean feeds it.

func TestCurveIsClosedByMeasurementNotByKind(t *testing.T) {
	circle, err := NewCircle(math.P3(1, 2, 3), math.V3(0, 0, 1), 5)
	if err != nil {
		t.Fatalf("NewCircle: %v", err)
	}
	cyl, err := NewCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	rod, err := NewCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	res := ResolutionForBox(math.NewBox(math.P3(-6, -6, -6), math.P3(6, 6, 6)))
	sections, ok := RuledQuadricSection(rod, cyl, res)
	if !ok || len(sections) != 2 {
		t.Fatalf("RuledQuadricSection: ok=%v n=%d, want two full-azimuth loops", ok, len(sections))
	}
	closedPoly, err := NewMarchedPolyline(regularRing(24, 4), 1e-4)
	if err != nil {
		t.Fatalf("NewMarchedPolyline: %v", err)
	}
	openPoly, err := NewPolyline([]math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(1, 1, 0)})
	if err != nil {
		t.Fatalf("NewPolyline: %v", err)
	}

	cases := []struct {
		name string
		c    Curve3
		want bool
	}{
		{"a circle closes", circle, true},
		// The kind the retired type switch did not list — the regression this predicate exists for.
		{"an exact ruled∩quadric section loop closes", sections[0], true},
		{"its second branch closes too", &closedFirst{sections[1]}, true},
		{"a ring polyline whose ends meet closes", &closedPoly, true},
		{"an open polyline does not", &openPoly, false},
		{"a line segment does not", NewLineSegment(math.P3(0, 0, 0), math.P3(1, 1, 1)), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := CurveIsClosed(c.c); got != c.want {
				t.Errorf("CurveIsClosed(%T) = %v, want %v", c.c, got, c.want)
			}
		})
	}
}

// closedFirst wraps a curve unchanged; it exists so the table holds sections[1] by interface without
// the test caring what concrete type the intersector returned.
type closedFirst struct{ Curve3 }

// TestCurveIsClosedRefusesAnUnboundedCurve: an infinite line cannot close, and the predicate must not
// evaluate a point at an infinite parameter to find that out.
func TestCurveIsClosedRefusesAnUnboundedCurve(t *testing.T) {
	line, err := NewLine(math.P3(0, 0, 0), math.V3(1, 0, 0))
	if err != nil {
		t.Fatalf("NewLine: %v", err)
	}
	if CurveIsClosed(line) {
		t.Error("an unbounded line reported closed")
	}
}

// regularRing is a closed n-gon of the given radius in the z=0 plane, last vertex repeating the first.
func regularRing(n int, r float64) []math.Point3 {
	pts := make([]math.Point3, 0, n+1)
	for i := 0; i <= n; i++ {
		a := 2 * stdmath.Pi * float64(i%n) / float64(n)
		pts = append(pts, math.P3(math.Scalar(r*stdmath.Cos(a)), math.Scalar(r*stdmath.Sin(a)), 0))
	}
	return pts
}
