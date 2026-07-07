// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
)

// TestAllDimensionKindsSurviveRoundTrip exercises restoreDimension and restoreAdvancedDimension
// across every dimension kind, plus the ellipse/elliptical-arc/spline entity restorers: a sketch
// carrying one of each is serialized and restored, and the restored sketch must hold the same
// number of dimension constraints and entities.
func TestAllDimensionKindsSurviveRoundTrip(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	dc := s.DimensionConstraints()

	pt := func(x, y float64) *Point { return s.NewPoint(math.P2(x, y)) }
	a, b, c := pt(0, 0), pt(4, 0), pt(0, 3)
	l1 := s.Lines().Add(a, b)
	l2 := s.Lines().Add(a, c)
	circle := s.Circles().Add(pt(2, 2), 1.0)
	arc := s.Arcs().Add(pt(6, 0), pt(8, 0), pt(6, 2), true)
	ell := s.Ellipses().Add(math.P2(10, 0), math.V2(1, 0), 2, 1)
	s.EllipticalArcs().Add(math.P2(14, 0), math.V2(1, 0), 2, 1, 0, 1.5)
	s.Splines().AddByPoints([]math.Point2{math.P2(0, 5), math.P2(1, 6), math.P2(2, 5)}, false)

	chk := func(name string, err error) {
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	_, err := dc.AddDistance(a, b, "4 cm")
	chk("distance", err)
	_, err = dc.AddRadius(circle, "1 cm")
	chk("radius", err)
	_, err = dc.AddDiameter(circle, "2 cm")
	chk("diameter", err)
	_, err = dc.AddAngle(l1, l2, "90 deg")
	chk("angle", err)
	_, err = dc.AddArcLength(arc, "3 cm")
	chk("arcLength", err)
	_, err = dc.AddEllipseRadius(ell, "2 cm")
	chk("ellipseRadius", err)
	_, err = dc.AddOffsetDim(c, l1, "3 cm")
	chk("offset", err)
	_, err = dc.AddThreePointAngle(a, b, c, "45 deg")
	chk("threePointAngle", err)
	_, err = dc.AddTangentDistance(l1, circle, false, "1 cm")
	chk("tangentDistance", err)

	wantDims := s.DimensionConstraints().Count()
	wantEnts := len(s.Entities())

	out := roundTrip(t, sc)
	if got := out.DimensionConstraints().Count(); got != wantDims {
		t.Errorf("dimension count after round trip = %d, want %d", got, wantDims)
	}
	if got := len(out.Entities()); got != wantEnts {
		t.Errorf("entity count after round trip = %d, want %d", got, wantEnts)
	}
}

// TestArcRadiusDiameterSurviveRoundTrip guards the reader fix for radius/diameter
// dimensions whose target is an *arc* (not a circle). AddRadius/AddDiameter accept any
// CircularCurve, and Inventor commonly radius-dimensions arcs, but restoreDimension used
// to resolve the operand via r.circle and reject arcs with "entity N is *sketch.Arc,
// want a circle" — which blocked ~40% of real Inventor-exported parts on open. The
// dimensions must restore and still target the arc.
func TestArcRadiusDiameterSurviveRoundTrip(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	dc := s.DimensionConstraints()

	pt := func(x, y float64) *Point { return s.NewPoint(math.P2(x, y)) }
	rArc := s.Arcs().Add(pt(0, 0), pt(3, 0), pt(0, 3), true)
	dArc := s.Arcs().Add(pt(10, 0), pt(12, 0), pt(10, 2), true)

	if _, err := dc.AddRadius(rArc, "5 cm"); err != nil {
		t.Fatalf("AddRadius on arc: %v", err)
	}
	if _, err := dc.AddDiameter(dArc, "8 cm"); err != nil {
		t.Fatalf("AddDiameter on arc: %v", err)
	}

	// ApplyRecipe inside roundTrip is where the old r.circle restore path failed.
	out := roundTrip(t, sc)
	dims := out.DimensionConstraints().All()
	if len(dims) != 2 {
		t.Fatalf("restored dimension count = %d, want 2", len(dims))
	}
	for _, d := range dims {
		if d.Kind() != RadiusDim && d.Kind() != DiameterDim {
			t.Errorf("restored dim kind = %v, want radius or diameter", d.Kind())
		}
		if len(d.Refs()) != 1 {
			t.Fatalf("restored dim refs = %d, want 1", len(d.Refs()))
		}
		if _, ok := d.Refs()[0].(*Arc); !ok {
			t.Errorf("restored dim target = %T, want *Arc", d.Refs()[0])
		}
	}
}
