// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
)

// satisfied reports whether all of a constraint's residuals are within tol of zero.
func satisfied(c Constraint) bool {
	for _, r := range c.Residuals() {
		if r > 1e-9 || r < -1e-9 {
			return false
		}
	}
	return true
}

func TestGeometricConstraintResiduals(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()

	// Coincident: zero only when the points are equal.
	a := s.Points().Add(math.P2(1, 1))
	b := s.Points().Add(math.P2(2, 1))
	co := g.AddCoincident(a, b)
	if satisfied(co) {
		t.Error("coincident satisfied for distinct points")
	}
	b.SetPosition(math.P2(1, 1))
	if !satisfied(co) {
		t.Error("coincident not satisfied for equal points")
	}

	// Horizontal / Vertical.
	h := g.AddHorizontal(s.Points().Add(math.P2(0, 5)), s.Points().Add(math.P2(3, 5)))
	if !satisfied(h) {
		t.Error("horizontal not satisfied for same-Y points")
	}
	v := g.AddVertical(s.Points().Add(math.P2(7, 0)), s.Points().Add(math.P2(7, 9)))
	if !satisfied(v) {
		t.Error("vertical not satisfied for same-X points")
	}

	// Parallel / Perpendicular.
	l1 := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(2, 0))
	l2 := s.Lines().AddByTwoPoints(math.P2(0, 1), math.P2(4, 1))
	if !satisfied(g.AddParallel(l1, l2)) {
		t.Error("parallel not satisfied for parallel lines")
	}
	l3 := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(0, 2))
	if !satisfied(g.AddPerpendicular(l1, l3)) {
		t.Error("perpendicular not satisfied for orthogonal lines")
	}

	// Collinear, Concentric, Equal.
	l4 := s.Lines().AddByTwoPoints(math.P2(5, 0), math.P2(9, 0))
	if !satisfied(g.AddCollinear(l1, l4)) {
		t.Error("collinear not satisfied for lines on the X axis")
	}
	c1 := s.Circles().AddByCenterRadius(math.P2(1, 1), 2)
	c2 := s.Circles().AddByCenterRadius(math.P2(1, 1), 3)
	if !satisfied(g.AddConcentric(c1, c2)) {
		t.Error("concentric not satisfied for same-center circles")
	}
	if !satisfied(g.AddEqualRadius(c1, c1)) {
		t.Error("equal-radius not satisfied for the same circle")
	}
	if !satisfied(g.AddEqualLength(l1, l3)) { // both length 2
		t.Error("equal-length not satisfied for equal lines")
	}
}

func TestTangentAndSymmetryAndFix(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()

	// A horizontal line y=2 is tangent to a circle of radius 2 centered at origin.
	line := s.Lines().AddByTwoPoints(math.P2(-5, 2), math.P2(5, 2))
	circ := s.Circles().AddByCenterRadius(math.P2(0, 0), 2)
	if !satisfied(g.AddTangent(line, circ)) {
		t.Error("tangent not satisfied at distance == radius")
	}
	circ.Radius = 1
	if satisfied(g.Item(g.Count() - 1)) {
		t.Error("tangent still satisfied after radius change")
	}

	// Symmetry about the X axis: (3,2) and (3,-2).
	axis := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0))
	pa := s.Points().Add(math.P2(3, 2))
	pb := s.Points().Add(math.P2(3, -2))
	if !satisfied(g.AddSymmetry(pa, pb, axis)) {
		t.Error("symmetry not satisfied for mirror points about the X axis")
	}

	// Fix pins a point; moving it breaks the constraint.
	p := s.Points().Add(math.P2(4, 4))
	fix := g.AddFix(p)
	if !satisfied(fix) {
		t.Error("fix not satisfied at its captured location")
	}
	p.SetPosition(math.P2(4, 5))
	if satisfied(fix) {
		t.Error("fix satisfied after the point moved")
	}
}

func TestAllConstraintsExposeVariables(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()
	p1 := s.Points().Add(math.P2(0, 0))
	p2 := s.Points().Add(math.P2(1, 0))
	l1 := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0))
	l2 := s.Lines().AddByTwoPoints(math.P2(0, 1), math.P2(1, 1))
	c1 := s.Circles().AddByCenterRadius(math.P2(0, 0), 1)
	c2 := s.Circles().AddByCenterRadius(math.P2(3, 0), 1)
	arc := s.Arcs().AddByCenterStartEnd(math.P2(0, 5), math.P2(1, 5), math.P2(0, 6), true)
	cons := []Constraint{
		g.AddCoincident(p1, p2), g.AddHorizontal(p1, p2), g.AddVertical(p1, p2),
		g.AddParallel(l1, l2), g.AddPerpendicular(l1, l2), g.AddCollinear(l1, l2),
		g.AddConcentric(c1, c2), g.AddEqualLength(l1, l2), g.AddEqualRadius(c1, c2),
		g.AddTangent(l1, c1), g.AddSymmetry(p1, p2, l1), g.AddFix(p1),
		g.AddLineSymmetry(l1, l2, l1), g.AddCircularSymmetry(c1, c2, l1),
		g.AddMidpointToArc(s.Points().Add(math.P2(0, 0)), arc),
	}
	for _, c := range cons {
		if len(c.Variables()) == 0 {
			t.Errorf("%T exposed no variables", c)
		}
		_ = c.Residuals() // must not panic
	}
}

func TestConstraintCollectionDeleteAndVariables(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()
	a := s.Points().Add(math.P2(0, 0))
	b := s.Points().Add(math.P2(1, 0))
	c := g.AddCoincident(a, b)
	if g.Count() != 1 || len(g.All()) != 1 {
		t.Fatal("constraint not registered")
	}
	if len(c.Variables()) != 4 {
		t.Errorf("coincident has %d variables, want 4", len(c.Variables()))
	}
	if !g.Delete(c) || g.Count() != 0 || g.Delete(c) {
		t.Error("Delete behavior wrong")
	}
}

func TestPointOnLineMidpointAndCircle(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()
	line := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))

	// Point on line: satisfied only when the point lies on y=0.
	p := s.Points().Add(math.P2(2, 1))
	on := g.AddPointOnLine(p, line)
	if satisfied(on) {
		t.Error("point-on-line satisfied for an off-line point")
	}
	p.SetPosition(math.P2(2, 0))
	if !satisfied(on) {
		t.Error("point-on-line not satisfied for a point on the line")
	}

	// Midpoint: satisfied only at the line midpoint (2,0).
	mp := s.Points().Add(math.P2(0, 0))
	mid := g.AddMidpoint(mp, line)
	if satisfied(mid) {
		t.Error("midpoint satisfied away from the midpoint")
	}
	mp.SetPosition(math.P2(2, 0))
	if !satisfied(mid) {
		t.Error("midpoint not satisfied at the midpoint")
	}

	// Point on circle: satisfied only on the radius-5 ring.
	circle := s.Circles().AddByCenterRadius(math.P2(0, 0), 5)
	cp := s.Points().Add(math.P2(3, 0))
	onc := g.AddPointOnCircle(cp, circle)
	if satisfied(onc) {
		t.Error("point-on-circle satisfied inside the ring")
	}
	cp.SetPosition(math.P2(5, 0))
	if !satisfied(onc) {
		t.Error("point-on-circle not satisfied on the ring")
	}
}

func TestPointOnLineSolves(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	line := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	g := s.GeometricConstraints()
	g.AddFix(line.A) // pin the line so the point is what moves
	g.AddFix(line.B)
	p := s.Points().Add(math.P2(1, 3)) // off the line
	g.AddPointOnLine(p, line)
	s.Solve()
	if y := float64(p.Position().Y); y > 1e-6 || y < -1e-6 {
		t.Errorf("point not driven onto the line: y = %v", y)
	}
}

// TestCircularConstraintsAcceptArcs verifies the circular constraints are polymorphic
// over Circle and Arc (like Inventor): an arc is accepted anywhere a circle is. The arc
// here has center (0,0), radius 3 (start at (3,0), 90° sweep to (0,3)).
func TestCircularConstraintsAcceptArcs(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()
	arc := s.Arcs().AddByCenterStartEnd(math.P2(0, 0), math.P2(3, 0), math.P2(0, 3), true)

	// Concentric arc↔circle: satisfied only when the centers coincide.
	circ := s.Circles().AddByCenterRadius(math.P2(2, 1), 5)
	con := g.AddConcentric(arc, circ)
	if satisfied(con) {
		t.Error("concentric satisfied with offset centers")
	}
	circ.Center.SetPosition(math.P2(0, 0))
	if !satisfied(con) {
		t.Error("concentric not satisfied once centers coincide")
	}

	// Equal radius arc↔circle: arc radius is 3, so equal only when the circle is too.
	eq := g.AddEqualRadius(arc, circ) // circ.Radius == 5 here
	if satisfied(eq) {
		t.Error("equal-radius satisfied with unequal radii (3 vs 5)")
	}
	circ.Radius = 3
	if !satisfied(eq) {
		t.Error("equal-radius not satisfied once radii match (3 == 3)")
	}

	// Point on arc: satisfied only on the radius-3 ring of the arc's circle.
	p := s.Points().Add(math.P2(1, 0))
	onArc := g.AddPointOnCircle(p, arc)
	if satisfied(onArc) {
		t.Error("point-on-arc satisfied inside the arc's ring")
	}
	p.SetPosition(math.P2(0, 3))
	if !satisfied(onArc) {
		t.Error("point-on-arc not satisfied on the ring")
	}

	// Tangent line↔arc: the line y=3 grazes the radius-3 ring.
	line := s.Lines().AddByTwoPoints(math.P2(-5, 3), math.P2(5, 3))
	if !satisfied(g.AddTangent(line, arc)) {
		t.Error("line not tangent to arc at distance == radius")
	}

	for _, c := range []Constraint{con, eq, onArc} {
		if len(c.Variables()) == 0 {
			t.Errorf("%T exposed no variables", c)
		}
	}
}

// TestCircularTangentExternalAndInternal covers tangency between two circular curves:
// external (centers r1+r2 apart) when the circles start separated, internal (|r1−r2|
// apart) when one starts inside the other.
func TestCircularTangentExternalAndInternal(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()

	// External: radius-2 and radius-3 circles, centers 5 apart ⇒ already tangent.
	a := s.Circles().AddByCenterRadius(math.P2(0, 0), 2)
	b := s.Circles().AddByCenterRadius(math.P2(5, 0), 3)
	ext := g.AddCircularTangent(a, b)
	if ext.internal {
		t.Error("expected external tangency for separated circles")
	}
	if !satisfied(ext) {
		t.Error("external tangent not satisfied at distance == r1+r2")
	}

	// Internal: a small circle near the center of a big one ⇒ |R−r| apart at tangency.
	big := s.Circles().AddByCenterRadius(math.P2(0, 0), 5)
	small := s.Circles().AddByCenterRadius(math.P2(1, 0), 2) // center distance 1 ≈ |5−2|=3? closer to internal
	in := g.AddCircularTangent(big, small)
	if !in.internal {
		t.Error("expected internal tangency for a circle inside another")
	}
	small.Center.SetPosition(math.P2(3, 0)) // |R−r| = 3 ⇒ internally tangent
	if !satisfied(in) {
		t.Error("internal tangent not satisfied at distance == |r1−r2|")
	}
}
