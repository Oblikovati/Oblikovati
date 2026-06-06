// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	gmath "oblikovati/math"
)

func TestEllipticalArcSamplesEndpoints(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	// Major axis +X, radii 2×1, quarter sweep 0 → π/2.
	arc := s.EllipticalArcs().Add(gmath.P2(0, 0), gmath.V2(1, 0), 2, 1, 0, math.Pi/2)
	poly := naturalPolyline(arc)
	if len(poly) < 2 {
		t.Fatalf("elliptical arc sampled into %d points, want a polyline", len(poly))
	}
	first, last := poly[0], poly[len(poly)-1]
	// θ=0 ⇒ (majorR, 0) = (2,0); θ=π/2 ⇒ (0, minorR) = (0,1).
	if math.Abs(float64(first.X)-2) > 1e-9 || math.Abs(float64(first.Y)) > 1e-9 {
		t.Errorf("start point = %v, want (2,0)", first)
	}
	if math.Abs(float64(last.X)) > 1e-9 || math.Abs(float64(last.Y)-1) > 1e-9 {
		t.Errorf("end point = %v, want (0,1)", last)
	}
}

func TestEllipticalArcRoundTrips(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	s.EllipticalArcs().Add(gmath.P2(1, 1), gmath.V2(0, 1), 3, 2, 0.1, 1.2)

	out := roundTrip(t, sc)
	if got := out.EllipticalArcs().Count(); got != 1 {
		t.Fatalf("elliptical arcs after round trip = %d, want 1", got)
	}
	a := out.EllipticalArcs().Item(0)
	if float64(a.MajorRadius) != 3 || float64(a.MinorRadius) != 2 {
		t.Errorf("radii = %v/%v, want 3/2", a.MajorRadius, a.MinorRadius)
	}
	if math.Abs(float64(a.StartAngle)-0.1) > 1e-9 || math.Abs(float64(a.EndAngle)-1.2) > 1e-9 {
		t.Errorf("angles = %v/%v, want 0.1/1.2", a.StartAngle, a.EndAngle)
	}
}
