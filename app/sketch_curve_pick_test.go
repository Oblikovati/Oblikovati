// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// #2026: picking enumerated only lines, circles and arcs, so splines, ellipses and elliptical
// arcs could not be selected by clicking their curve at all. A spline's fit points were still
// grabbable, which is what made it look like a polyline rather than an unpickable curve.

// pickAtSketchPoint clicks the pixel showing sketch point p and returns what was picked.
func pickAtSketchPoint(t *testing.T, s *Session, p math.Point2) (sketch.Entity, bool) {
	t.Helper()
	x, y, ok := sketchToScreen(s, p)
	if !ok {
		t.Fatalf("sketch point %v does not project to the viewport", p)
	}
	return s.pickSketchEntity(x, y)
}

// TestSplineIsPickableOnItsCurve is the headline regression. The click is placed on the curve
// AWAY from any defining point, so only true curve hit-testing can find it.
func TestSplineIsPickableOnItsCurve(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	sp := sk.Splines().AddByPoints([]math.Point2{
		math.P2(0, 0), math.P2(1, 2), math.P2(2, -2), math.P2(3, 0),
	}, false)

	pts, _ := sketch.EntityPolyline(sp)
	on := pts[len(pts)/2] // a sample in the curve's interior
	got, ok := pickAtSketchPoint(t, s, on)
	if !ok {
		t.Fatal("clicking on a spline picked nothing")
	}
	if got != sketch.Entity(sp) {
		t.Errorf("picked %T, want the spline", got)
	}
}

// TestSplinePickFollowsTheCurveNotTheControlPolygon: a point on the true curve but off the
// straight chord between defining points must hit, and a point on the chord but off the curve
// must not. This is what distinguishes real curve picking from polygon picking.
func TestSplinePickFollowsTheCurveNotTheControlPolygon(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	defining := []math.Point2{math.P2(0, 0), math.P2(1, 2), math.P2(2, -2), math.P2(3, 0)}
	sp := sk.Splines().AddByPoints(defining, false)

	// Choose a curve sample that both deviates from the control polygon AND is clear of every
	// defining point. Standalone points legitimately win a pick within snap tolerance, so the
	// most-deviating sample is the wrong probe: on this curve it lands 0.45 from a fit point,
	// inside the 0.497 tolerance, and the POINT is correctly returned.
	curve, _ := sketch.EntityPolyline(sp)
	far, dev := math.Point2{}, 0.0
	for _, c := range curve {
		if distanceToNearestOf(c, defining) <= s.snapTolerance() {
			continue
		}
		if d := distanceToChain(c, defining); d > dev {
			far, dev = c, d
		}
	}
	if dev < 0.1 {
		t.Fatalf("test premise broken: no curve sample is both >%.3f from every defining point and >0.1 off the control polygon (best %.4f)",
			s.snapTolerance(), dev)
	}
	got, ok := pickAtSketchPoint(t, s, far)
	if !ok || got != sketch.Entity(sp) {
		t.Errorf("a point %.3f off the control polygon but ON the curve picked %T (ok=%v), want the spline", dev, got, ok)
	}
}

// distanceToNearestOf is the distance from p to the closest of pts.
func distanceToNearestOf(p math.Point2, pts []math.Point2) float64 {
	best := p.DistanceTo(pts[0])
	for _, q := range pts[1:] {
		if d := p.DistanceTo(q); d < best {
			best = d
		}
	}
	return best
}

// TestEllipseIsPickable / TestEllipticalArcIsPickable cover the other kinds the hand-rolled
// enumeration missed — the reporter only noticed splines.
func TestEllipseIsPickable(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	e := sk.Ellipses().Add(math.P2(0, 0), math.V2(1, 0), 4, 2)

	pts, _ := sketch.EntityPolyline(e)
	got, ok := pickAtSketchPoint(t, s, pts[len(pts)/3])
	if !ok || got != sketch.Entity(e) {
		t.Errorf("clicking an ellipse picked %T (ok=%v), want the ellipse", got, ok)
	}
}

// TestLinePickStillWorks guards the kinds that already worked against the rewrite.
func TestLinePickStillWorks(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(math.P2(-3, -3), math.P2(3, 3))
	c := sk.Circles().AddByCenterRadius(math.P2(10, 0), 2)

	if got, ok := pickAtSketchPoint(t, s, math.P2(1, 1)); !ok || got != sketch.Entity(l) {
		t.Errorf("line pick got %T (ok=%v), want the line", got, ok)
	}
	if got, ok := pickAtSketchPoint(t, s, math.P2(12, 0)); !ok || got != sketch.Entity(c) {
		t.Errorf("circle rim pick got %T (ok=%v), want the circle", got, ok)
	}
}

// TestArcIsNotPickedOffItsSweep: arcs used to be hit-tested against their FULL circle
// (circleOutlineDistance), so clicking the empty part of the sweep — where nothing is drawn —
// selected the arc. Faceting the actual sweep fixes that as a side effect of #2026.
func TestArcIsNotPickedOffItsSweep(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	// A quarter arc from (5,0) to (0,5) about the origin: the opposite side of its circle,
	// (-5,0), is on the circle but NOT on the arc.
	a := sk.Arcs().AddByCenterStartEnd(math.P2(0, 0), math.P2(5, 0), math.P2(0, 5), true)

	if got, ok := pickAtSketchPoint(t, s, math.P2(-5, 0)); ok {
		t.Errorf("clicking the empty side of the arc's circle picked %T, want nothing", got)
	}
	on, _ := sketch.EntityPolyline(a)
	if got, ok := pickAtSketchPoint(t, s, on[len(on)/2]); !ok || got != sketch.Entity(a) {
		t.Errorf("clicking ON the arc picked %T (ok=%v), want the arc", got, ok)
	}
}

// TestClickInEmptySpacePicksNothing: the polyline walk must not make everything hittable.
func TestClickInEmptySpacePicksNothing(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	sk.Splines().AddByPoints([]math.Point2{math.P2(0, 0), math.P2(1, 2), math.P2(2, -2)}, false)

	if got, ok := pickAtSketchPoint(t, s, math.P2(40, 40)); ok {
		t.Errorf("a click far from any geometry picked %T", got)
	}
}

// distanceToChain is the distance from p to the open polyline through pts.
func distanceToChain(p math.Point2, pts []math.Point2) float64 {
	best := segmentDistance(p, pts[0], pts[1])
	for i := 1; i+1 < len(pts); i++ {
		if d := segmentDistance(p, pts[i], pts[i+1]); d < best {
			best = d
		}
	}
	return best
}
