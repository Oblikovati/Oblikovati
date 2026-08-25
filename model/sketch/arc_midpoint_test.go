// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// TestArcMidpointRespectsWinding: for the same three points, a CCW and a CW arc take OPPOSITE
// midpoints — the disambiguation an analytic rebuild needs (a semicircle's midpoint is otherwise
// ambiguous).
func TestArcMidpointRespectsWinding(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	// Semicircle radius 1 about the origin, from (1,0) to (-1,0).
	ccw := s.Arcs().AddByCenterStartEnd(math.P2(0, 0), math.P2(1, 0), math.P2(-1, 0), true)
	cw := s.Arcs().AddByCenterStartEnd(math.P2(0, 0), math.P2(1, 0), math.P2(-1, 0), false)

	mc := ccw.ArcMidpoint()
	if stdmath.Abs(float64(mc.X)) > 1e-9 || stdmath.Abs(float64(mc.Y)-1) > 1e-9 {
		t.Errorf("CCW semicircle midpoint = %v, want (0,1) (upper half)", mc)
	}
	mw := cw.ArcMidpoint()
	if stdmath.Abs(float64(mw.X)) > 1e-9 || stdmath.Abs(float64(mw.Y)+1) > 1e-9 {
		t.Errorf("CW semicircle midpoint = %v, want (0,-1) (lower half)", mw)
	}
}

// TestArcMidpointOnCircle: the midpoint lies on the arc's circle (radius from center).
func TestArcMidpointOnCircle(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	a := s.Arcs().AddByCenterStartEnd(math.P2(2, 1), math.P2(5, 1), math.P2(2, 4), true) // quarter, r=3
	m := a.ArcMidpoint()
	if r := float64(math.P2(2, 1).DistanceTo(m)); stdmath.Abs(r-3) > 1e-9 {
		t.Errorf("midpoint radius = %.6f, want 3", r)
	}
	// Quarter arc CCW from +X to +Y about (2,1): midpoint at 45° → (2+3/√2, 1+3/√2).
	want := 3 / stdmath.Sqrt2
	if stdmath.Abs(float64(m.X-2)-want) > 1e-9 || stdmath.Abs(float64(m.Y-1)-want) > 1e-9 {
		t.Errorf("quarter-arc midpoint = %v, want (%.4f,%.4f) offset", m, want, want)
	}
}
