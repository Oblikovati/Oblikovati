// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
)

// TestControlSplineRoundTrip (#150): a control-point (approximating) spline keeps its mode
// across a save/load cycle — it is not silently restored as a fit spline.
func TestControlSplineRoundTrip(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	sp := s.Splines().AddByControlPoints([]math.Point2{math.P2(0, 0), math.P2(2, 1), math.P2(4, 0)}, false)
	if sp.IsFitType() {
		t.Fatal("AddByControlPoints produced a fit spline")
	}

	out := roundTrip(t, sc)
	var restored *Spline
	for _, e := range out.Entities() {
		if v, ok := e.(*Spline); ok {
			restored = v
		}
	}
	if restored == nil {
		t.Fatal("no spline after round trip")
	}
	if restored.IsFitType() {
		t.Error("control-point spline restored as a fit spline (mode lost)")
	}
}

// TestTangentDistanceDimRoundTrip (#152): a tangent-distance dimension keeps its far-side flag
// and resolves its line/circle operands across a save/load cycle.
func TestTangentDistanceDimRoundTrip(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	a, b := s.NewPoint(math.P2(0, 0)), s.NewPoint(math.P2(4, 0))
	line := s.Lines().Add(a, b)
	circle := s.Circles().Add(s.NewPoint(math.P2(0, 5)), 2)
	d, err := s.DimensionConstraints().AddTangentDistance(line, circle, true, "1 cm")
	if err != nil {
		t.Fatalf("AddTangentDistance: %v", err)
	}
	if !d.FarSide() || d.Kind() != TangentDistanceDim {
		t.Fatalf("dimension = farSide %v kind %d, want true tangentDistance", d.FarSide(), d.Kind())
	}

	out := roundTrip(t, sc)
	dims := out.DimensionConstraints()
	if dims.Count() != 1 {
		t.Fatalf("restored dimensions = %d, want 1", dims.Count())
	}
	rd := dims.Item(0)
	if rd.Kind() != TangentDistanceDim || !rd.FarSide() {
		t.Errorf("restored dimension = kind %d farSide %v, want tangentDistance far=true", rd.Kind(), rd.FarSide())
	}
	// Far tangent distance with center 5 above the line and radius 2 is 5+2 = 7.
	if got := rd.Measured(); got < 6.99 || got > 7.01 {
		t.Errorf("restored measured = %v, want ~7", got)
	}
}
