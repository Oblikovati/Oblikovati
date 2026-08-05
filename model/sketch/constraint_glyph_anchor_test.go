// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
)

// A constraint marker used to sit at the centroid of everything the relation touched, which for a
// relation spanning two curves is a point on neither of them: a perpendicular between a line at
// y=60 and one at x=100 was drawn at (55,37.5), and two circles held to an equal radius were
// marked ~90 units from either circumference. These pin the rule that replaced it — one marker per
// operand, each ON that operand — and the contact cases that collapse back to a single marker.

// distanceToEntity is how far a marker sits from the geometry it annotates: 0 when it lies on it.
func distanceToEntity(t *testing.T, e Entity, at math.Point2) float64 {
	t.Helper()
	on, ok := ClosestPointOnEntity(e, at)
	if !ok {
		t.Fatalf("entity %T has no outline to measure against", e)
	}
	return float64(on.DistanceTo(at))
}

// TestDistantCurvesAreEachMarkedOnThemselves is the reported bug: markers for a relation between
// two separated curves floated in the space between them instead of sitting on either.
func TestDistantCurvesAreEachMarkedOnThemselves(t *testing.T) {
	sk := NewSketches().Add(XYPlane())
	l1 := sk.Lines().AddByTwoPoints(math.P2(0, 60), math.P2(20, 60))
	l2 := sk.Lines().AddByTwoPoints(math.P2(100, 0), math.P2(100, 30))
	sk.GeometricConstraints().AddPerpendicular(l1, l2)

	ats := anchorsOfKind(sk, PerpendicularKind)
	if len(ats) != 2 {
		t.Fatalf("got %d markers for a relation spanning 2 curves, want one on each", len(ats))
	}
	if d := distanceToEntity(t, l1, ats[0]); d > 1e-9 {
		t.Errorf("first marker at %v is %.3f from the line it annotates, want 0", ats[0], d)
	}
	if d := distanceToEntity(t, l2, ats[1]); d > 1e-9 {
		t.Errorf("second marker at %v is %.3f from the line it annotates, want 0", ats[1], d)
	}
	// The old centroid, the position this test exists to rule out.
	for _, at := range ats {
		if closeTo(at, math.P2(55, 37.5)) {
			t.Error("a marker is still at the centroid of both operands — empty space")
		}
	}
}

// TestTangencyIsMarkedOnceAtTheTouchPoint: the two operands of a tangency meet at one point, and
// that point is the whole subject of the relation, so both markers settle there and collapse into
// one. A tangency drawn twice, or drawn at the circle's centre, would both be wrong.
func TestTangencyIsMarkedOnceAtTheTouchPoint(t *testing.T) {
	sk := NewSketches().Add(XYPlane())
	ln := sk.Lines().AddByTwoPoints(math.P2(-30, 0), math.P2(60, 0))
	ci := sk.Circles().AddByCenterRadius(math.P2(10, 7), 7)
	sk.GeometricConstraints().AddTangent(ln, ci)

	g := glyphOfKind(t, sk, TangentKind)
	if want := math.P2(10, 0); !closeTo(g.At, want) {
		t.Errorf("tangency marked at %v, want the touch point %v", g.At, want)
	}
}

// TestConcentricityIsMarkedAtTheSharedCentre: concentricity holds two curves by their CENTRES, so
// its marker belongs at the centre. Pushing it out onto a circumference — right for a tangency or
// an equal radius — would put it where the relation constrains nothing.
func TestConcentricityIsMarkedAtTheSharedCentre(t *testing.T) {
	sk := NewSketches().Add(XYPlane())
	centre := math.P2(-40, 30)
	c1 := sk.Circles().AddByCenterRadius(centre, 6)
	c2 := sk.Circles().AddByCenterRadius(centre, 12)
	sk.GeometricConstraints().AddConcentric(c1, c2)

	g := glyphOfKind(t, sk, ConcentricKind)
	if !closeTo(g.At, centre) {
		t.Errorf("concentricity marked at %v, want the shared centre %v", g.At, centre)
	}
}

// TestSharedCornerDoesNotRecruitTheWholeLoop: a rectangle's corner belongs to two edges, so a
// point-per-entity rule that accepted a single shared point would treat every edge meeting there
// as an operand and scatter markers around the loop. An entity is an operand only when the
// relation moves two of its points.
func TestSharedCornerDoesNotRecruitTheWholeLoop(t *testing.T) {
	sk := NewSketches().Add(XYPlane())
	a := sk.Points().Add(math.P2(0, 0))
	b := sk.Points().Add(math.P2(10, 0))
	c := sk.Points().Add(math.P2(10, 6))
	d := sk.Points().Add(math.P2(0, 6))
	bottom, right := sk.Lines().Add(a, b), sk.Lines().Add(b, c)
	sk.Lines().Add(c, d)
	sk.Lines().Add(d, a)
	sk.GeometricConstraints().AddPerpendicular(bottom, right)

	ats := anchorsOfKind(sk, PerpendicularKind)
	if len(ats) != 1 {
		t.Fatalf("got %d markers %v, want 1 at the corner the two edges share", len(ats), ats)
	}
	if want := math.P2(10, 0); !closeTo(ats[0], want) {
		t.Errorf("marker at %v, want the shared corner %v", ats[0], want)
	}
}

// TestNoMarkerLandsInEmptySpace is the general form of the bug, swept over the relation kinds that
// span two operands. Whatever the kind, every marker must lie on one of the sketch's curves —
// which is the one property the old centroid could not promise.
func TestNoMarkerLandsInEmptySpace(t *testing.T) {
	sk := NewSketches().Add(XYPlane())
	l1 := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(30, 0))
	l2 := sk.Lines().AddByTwoPoints(math.P2(-20, 50), math.P2(-20, 90))
	c1 := sk.Circles().AddByCenterRadius(math.P2(70, -40), 4)
	c2 := sk.Circles().AddByCenterRadius(math.P2(120, 80), 9)
	g := sk.GeometricConstraints()
	g.AddPerpendicular(l1, l2)
	g.AddParallel(l1, l2)
	g.AddEqualLength(l1, l2)
	g.AddEqualRadius(c1, c2)
	g.AddConcentric(c1, c2)

	curves := []Entity{l1, l2, c1, c2}
	for _, gl := range sk.ConstraintGlyphs() {
		if !onAnyCurve(t, curves, gl.At) {
			t.Errorf("%v marker at %v lies on no curve — it is floating in empty space", gl.Kind, gl.At)
		}
	}
}

// onAnyCurve reports whether a marker lies on one of the curves, or at a shared centre — the one
// off-curve position a marker is allowed to take (concentricity).
func onAnyCurve(t *testing.T, curves []Entity, at math.Point2) bool {
	t.Helper()
	for _, e := range curves {
		if distanceToEntity(t, e, at) <= 1e-9 {
			return true
		}
		if cc, ok := e.(CircularCurve); ok && closeTo(at, math.P2(cc.CenterPoint().X, cc.CenterPoint().Y)) {
			return true
		}
	}
	return false
}

// TestSingleCurveRelationKeepsItsMidpoint: a relation on ONE line was already anchored correctly,
// at that line's midpoint, and the per-operand rule must not disturb it.
func TestSingleCurveRelationKeepsItsMidpoint(t *testing.T) {
	sk := NewSketches().Add(XYPlane())
	ln := sk.Lines().AddByTwoPoints(math.P2(2, 5), math.P2(12, 5))
	sk.GeometricConstraints().AddLineHorizontal(ln)

	g := glyphOfKind(t, sk, SingleLineHorizontalKind)
	if want := math.P2(7, 5); !closeTo(g.At, want) {
		t.Errorf("horizontal marker at %v, want the line's midpoint %v", g.At, want)
	}
}
