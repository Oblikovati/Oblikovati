// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// #2028: the two three-point arc tools took their points in OPPOSITE orders — ArcTool was
// start/end/through, ThreePointArcSlotTool was start/through/end — so the same gesture meant
// different things depending on which tool was armed.
//
// These assert the arc's ENDPOINTS, not its centre. The centre cannot tell the orders apart:
// it is the circumcentre of the same three points either way, and a circumcentre does not
// depend on the order of its arguments. Only which two of the three become the endpoints (and
// hence the sweep) differs — an earlier version of this test checked the centre, passed under
// a deliberately reverted fix, and proved nothing.
var (
	arcStart   = math.P2(0, 0)
	arcEnd     = math.P2(4, 0)
	arcThrough = math.P2(2, 2)
)

// arcEndpoints returns an arc's start and end positions.
func arcEndpoints(a *sketch.Arc) (math.Point2, math.Point2) {
	return a.Start.Position(), a.End.Position()
}

// endpointsMatch reports whether the arc spans want1..want2, in either direction (the sweep
// sense is a separate property).
func endpointsMatch(a *sketch.Arc, want1, want2 math.Point2) bool {
	s, e := arcEndpoints(a)
	const tol = 1e-6
	if s.DistanceTo(want1) <= tol && e.DistanceTo(want2) <= tol {
		return true
	}
	return s.DistanceTo(want2) <= tol && e.DistanceTo(want1) <= tol
}

// TestThreePointArcInterpretsItsClicksAsStartEndThrough pins ArcTool's documented order: the
// first two clicks are the endpoints and the THIRD is the point the arc passes through.
func TestThreePointArcInterpretsItsClicksAsStartEndThrough(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	s.StartTool(NewArcTool())
	for _, p := range []math.Point2{arcStart, arcEnd, arcThrough} {
		clickSketch(t, s, p)
	}
	if sk.Arcs().Count() != 1 {
		t.Fatalf("got %d arcs, want 1", sk.Arcs().Count())
	}
	a := sk.Arcs().Item(0)
	if !endpointsMatch(a, arcStart, arcEnd) {
		s0, e0 := arcEndpoints(a)
		t.Errorf("arc spans %v..%v, want %v..%v — the third click is not being read as the point ON the arc",
			s0, e0, arcStart, arcEnd)
	}
	// And it really does pass through the third click: that point is on the circle.
	r := a.Center.Position().DistanceTo(arcStart)
	if got := a.Center.Position().DistanceTo(arcThrough); stdmath.Abs(got-r) > 1e-6 {
		t.Errorf("the through point is %v from the centre but the radius is %v", got, r)
	}
}

// TestBothThreePointArcToolsAgreeOnPickOrder is the regression: the identical three clicks must
// give both tools the same arc span. Before the fix the slot read the SECOND click as the point
// on the arc, so its arc ran (0,0)..(2,2) while the plain arc ran (0,0)..(4,0).
func TestBothThreePointArcToolsAgreeOnPickOrder(t *testing.T) {
	t.Parallel()
	sPlain, skPlain := sketchSession(t)
	sPlain.StartTool(NewArcTool())
	for _, p := range []math.Point2{arcStart, arcEnd, arcThrough} {
		clickSketch(t, sPlain, p)
	}
	if skPlain.Arcs().Count() != 1 {
		t.Fatalf("plain arc: got %d arcs, want 1", skPlain.Arcs().Count())
	}
	wantS, wantE := arcEndpoints(skPlain.Arcs().Item(0))

	sSlot, skSlot := sketchSession(t)
	sSlot.StartTool(NewThreePointArcSlotTool(1))
	for _, p := range []math.Point2{arcStart, arcEnd, arcThrough} {
		clickSketch(t, sSlot, p)
	}
	if skSlot.Arcs().Count() == 0 {
		t.Fatal("arc slot produced no arcs")
	}
	// A slot's rails are offset from the centre arc, so compare the ANGULAR span about the
	// shared centre rather than the raw endpoints.
	if !sameAngularSpan(skSlot.Arcs().Item(0), wantS, wantE) {
		s0, e0 := arcEndpoints(skSlot.Arcs().Item(0))
		t.Errorf("arc slot spans %v..%v (centre %v) but the plain arc spans %v..%v — the two tools disagree on pick order",
			s0, e0, skSlot.Arcs().Item(0).Center.Position(), wantS, wantE)
	}
}

// sameAngularSpan reports whether arc a starts and ends at the same angles about its own centre
// as want1..want2 do — the rail radius differs from the centre arc's, so only the angles match.
func sameAngularSpan(a *sketch.Arc, want1, want2 math.Point2) bool {
	c := a.Center.Position()
	s, e := arcEndpoints(a)
	const tol = 1e-6
	if angleClose(c, s, want1, tol) && angleClose(c, e, want2, tol) {
		return true
	}
	return angleClose(c, s, want2, tol) && angleClose(c, e, want1, tol)
}

// angleClose reports whether p and q lie at the same angle about centre c.
func angleClose(c, p, q math.Point2, tol float64) bool {
	dp, dq := c.VectorTo(p), c.VectorTo(q)
	lp, lq := dp.Length(), dq.Length()
	if lp < tol || lq < tol {
		return false
	}
	return stdmath.Abs(dp.Cross(dq)/(lp*lq)) <= tol && dp.Dot(dq) > 0
}
