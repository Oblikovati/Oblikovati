// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// entList collects sketch entities into the slice the apply* functions take.
func entList(ents ...sketch.Entity) []sketch.Entity { return ents }

// selectEntities replaces the selection with the given sketch entities (used by the
// selection/highlight tests).
func selectEntities(s *Session, ents ...sketch.Entity) {
	s.Selection().Clear()
	for _, e := range ents {
		s.Selection().Add(SketchEntityHandle{Entity: e})
	}
}

func lineDir(l *sketch.Line) math.Vector2 { return l.A.Position().VectorTo(l.B.Position()) }

func cosBetween(l1, l2 *sketch.Line) float64 {
	d1, d2 := lineDir(l1), lineDir(l2)
	return stdmath.Abs(d1.Dot(d2)) / (d1.Length() * d2.Length())
}

func TestPerpendicularConstraintSolves(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	l1 := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	l2 := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 4)) // 45°, not perpendicular
	if err := applyPerpendicular(s, entList(l1, l2)); err != nil {
		t.Fatalf("applyPerpendicular: %v", err)
	}
	if c := cosBetween(l1, l2); c > 1e-3 {
		t.Errorf("lines not perpendicular after solve: |cos| = %v", c)
	}
}

func TestParallelConstraintSolves(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	l1 := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	l2 := sk.Lines().AddByTwoPoints(math.P2(0, 2), math.P2(4, 3))
	if err := applyParallel(s, entList(l1, l2)); err != nil {
		t.Fatalf("applyParallel: %v", err)
	}
	if c := cosBetween(l1, l2); c < 1-1e-3 {
		t.Errorf("lines not parallel after solve: |cos| = %v", c)
	}
}

func TestHorizontalConstraintOnLine(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 2))
	if err := applyHorizontal(s, entList(l)); err != nil {
		t.Fatalf("applyHorizontal: %v", err)
	}
	if dy := stdmath.Abs(float64(l.A.Y - l.B.Y)); dy > 1e-6 {
		t.Errorf("line not horizontal after solve: Δy = %v", dy)
	}
}

func TestVerticalConstraintOnLine(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(2, 4))
	if err := applyVertical(s, entList(l)); err != nil {
		t.Fatalf("applyVertical: %v", err)
	}
	if dx := stdmath.Abs(float64(l.A.X - l.B.X)); dx > 1e-6 {
		t.Errorf("line not vertical after solve: Δx = %v", dx)
	}
}

func TestCollinearConstraint(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	l1 := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	l2 := sk.Lines().AddByTwoPoints(math.P2(0, 1), math.P2(4, 1))
	if err := applyCollinear(s, entList(l1, l2)); err != nil {
		t.Fatalf("applyCollinear: %v", err)
	}
	if c := cosBetween(l1, l2); c < 1-1e-3 {
		t.Errorf("collinear lines not parallel: |cos| = %v", c)
	}
}

func TestCoincidentConstraintOnPoints(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	p1 := sk.Points().Add(math.P2(0, 0))
	p2 := sk.Points().Add(math.P2(2, 1))
	if err := applyCoincident(s, entList(p1, p2)); err != nil {
		t.Fatalf("applyCoincident: %v", err)
	}
	if d := p1.Position().DistanceTo(p2.Position()); d > 1e-6 {
		t.Errorf("points not coincident after solve: distance = %v", d)
	}
}

func TestEqualRadiusAndLength(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	c1 := sk.Circles().AddByCenterRadius(math.P2(0, 0), 2)
	c2 := sk.Circles().AddByCenterRadius(math.P2(8, 0), 5)
	if err := applyEqual(s, entList(c1, c2)); err != nil {
		t.Fatalf("applyEqual circles: %v", err)
	}
	if d := stdmath.Abs(c1.Radius - c2.Radius); d > 1e-6 {
		t.Errorf("radii not equal after solve: Δr = %v", d)
	}
	l1 := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(3, 0))
	l2 := sk.Lines().AddByTwoPoints(math.P2(0, 5), math.P2(7, 5))
	if err := applyEqual(s, entList(l1, l2)); err != nil {
		t.Fatalf("applyEqual lines: %v", err)
	}
	if d := lineDir(l1).Length() - lineDir(l2).Length(); d > 1e-6 || d < -1e-6 {
		t.Errorf("lines not equal length: Δ = %v", d)
	}
}

func TestConcentricOnCircles(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	c1 := sk.Circles().AddByCenterRadius(math.P2(0, 0), 2)
	c2 := sk.Circles().AddByCenterRadius(math.P2(5, 3), 4)
	if err := applyConcentric(s, entList(c1, c2)); err != nil {
		t.Fatalf("applyConcentric: %v", err)
	}
	if d := c1.Center.Position().DistanceTo(c2.Center.Position()); d > 1e-6 {
		t.Errorf("circles not concentric after solve: centre distance = %v", d)
	}
}

func TestTangentLineToCircle(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	c := sk.Circles().AddByCenterRadius(math.P2(0, 0), 2)
	l := sk.Lines().AddByTwoPoints(math.P2(-4, 3), math.P2(4, 3)) // above the circle
	if err := applyTangent(s, entList(l, c)); err != nil {
		t.Fatalf("applyTangent: %v", err)
	}
	d := segmentDistance(c.Center.Position(), l.A.Position(), l.B.Position())
	if stdmath.Abs(d-c.Radius) > 1e-3 {
		t.Errorf("line not tangent after solve: |dist-r| = %v", stdmath.Abs(d-c.Radius))
	}
}

// TestCircularConstraintsAcceptArcs checks the apply* layer treats an arc like a circle:
// concentric, equal-radius, point-on-curve coincidence, and tangent all accept arcs.
func TestCircularConstraintsAcceptArcs(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	arc := sk.Arcs().AddByCenterStartEnd(math.P2(0, 0), math.P2(3, 0), math.P2(0, 3), true)
	circ := sk.Circles().AddByCenterRadius(math.P2(6, 4), 5)

	if err := applyConcentric(s, entList(arc, circ)); err != nil {
		t.Fatalf("applyConcentric arc+circle: %v", err)
	}
	if d := arc.Center.Position().DistanceTo(circ.Center.Position()); d > 1e-6 {
		t.Errorf("arc and circle not concentric after solve: centre distance = %v", d)
	}

	if err := applyEqual(s, entList(arc, circ)); err != nil {
		t.Fatalf("applyEqual arc+circle: %v", err)
	}
	if d := stdmath.Abs(arc.Radius() - circ.Radius); d > 1e-6 {
		t.Errorf("arc and circle radii not equal after solve: Δr = %v", d)
	}

	p := sk.Points().Add(math.P2(1, 0))
	if err := applyCoincident(s, entList(p, arc)); err != nil {
		t.Fatalf("applyCoincident point+arc: %v", err)
	}
	if d := stdmath.Abs(p.Position().DistanceTo(arc.Center.Position()) - arc.Radius()); d > 1e-3 {
		t.Errorf("point not driven onto arc ring: |dist-r| = %v", d)
	}
}

// TestTangentTwoCircles checks the polymorphic Tangent applies a curve-to-curve tangent
// when two circles (no line) are picked.
func TestTangentTwoCircles(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	a := sk.Circles().AddByCenterRadius(math.P2(0, 0), 2)
	b := sk.Circles().AddByCenterRadius(math.P2(6, 0), 3) // separated ⇒ external tangency
	if err := applyTangent(s, entList(a, b)); err != nil {
		t.Fatalf("applyTangent two circles: %v", err)
	}
	d := a.Center.Position().DistanceTo(b.Center.Position())
	if stdmath.Abs(d-(a.Radius+b.Radius)) > 1e-3 {
		t.Errorf("circles not externally tangent after solve: |dist-(r1+r2)| = %v", stdmath.Abs(d-(a.Radius+b.Radius)))
	}
}

func TestFixAddsConstraint(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	p := sk.Points().Add(math.P2(1, 1))
	before := len(sk.GeometricConstraints().All())
	if err := applyFix(s, entList(p)); err != nil {
		t.Fatalf("applyFix: %v", err)
	}
	if len(sk.GeometricConstraints().All()) <= before {
		t.Error("Fix should add a constraint")
	}
}

func TestSymmetryConstraintSolves(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	axis := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0)) // the X axis
	sk.GeometricConstraints().AddFix(axis.A)                        // pin the mirror axis so
	sk.GeometricConstraints().AddFix(axis.B)                        // the points are what move
	a := sk.Points().Add(math.P2(3, 2))
	b := sk.Points().Add(math.P2(4, 3)) // not yet the mirror of a
	if err := applySymmetry(s, entList(a, b, axis)); err != nil {
		t.Fatalf("applySymmetry: %v", err)
	}
	// Symmetric about the X axis ⇒ equal X, opposite Y, and the midpoint on the axis.
	if dx := stdmath.Abs(a.X - b.X); dx > 1e-6 {
		t.Errorf("symmetric points should share X: Δx = %v", dx)
	}
	if sy := stdmath.Abs(a.Y + b.Y); sy > 1e-6 {
		t.Errorf("symmetric points should have opposite Y: Ay+By = %v", sy)
	}
}

func TestSymmetryWrongInputErrors(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	p1, p2 := sk.Points().Add(math.P2(0, 0)), sk.Points().Add(math.P2(1, 1))
	if err := applySymmetry(s, entList(p1, p2)); err == nil {
		t.Error("symmetry without an axis line should error")
	}
}

func TestSmoothConstraintJoinsSplineToLine(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	line := sk.Lines().AddByTwoPoints(math.P2(-2, 0), math.P2(0, 0))
	sk.GeometricConstraints().AddFix(line.A)
	sk.GeometricConstraints().AddFix(line.B) // pin the line so the spline moves
	sp := sk.Splines().AddByControlPoints([]math.Point2{{X: 0.5, Y: 0.5}, {X: 1, Y: 1}, {X: 2, Y: 0}}, false)
	before := len(sk.GeometricConstraints().All())
	if err := applySmooth(s, entList(line, sp)); err != nil {
		t.Fatalf("applySmooth: %v", err)
	}
	if len(sk.GeometricConstraints().All()) != before+1 {
		t.Fatalf("smooth should add exactly one constraint")
	}
	// The solve in applySmooth should join the spline start to the line end.
	if d := sp.Points[0].Position().DistanceTo(line.B.Position()); d > 1e-6 {
		t.Errorf("spline not joined to the line end after smooth: d = %v", d)
	}
}

func TestSmoothWithoutSplineErrors(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	l1 := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0))
	l2 := sk.Lines().AddByTwoPoints(math.P2(1, 0), math.P2(2, 1))
	if err := applySmooth(s, entList(l1, l2)); err == nil {
		t.Error("smooth without a spline should error (matches Inventor)")
	}
}

func TestConstraintWrongInputErrors(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0))
	if err := applyParallel(s, entList(l)); err == nil {
		t.Error("parallel with one line should error")
	}
	if err := applyCoincident(s, nil); err == nil {
		t.Error("coincident with nothing should error")
	}
}

func TestDimensionFromEntities(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	p1 := sk.Points().Add(math.P2(0, 0))
	p2 := sk.Points().Add(math.P2(3, 0))
	if err := applyDimension(s, entList(p1, p2)); err != nil {
		t.Fatalf("dimension distance: %v", err)
	}
	c := sk.Circles().AddByCenterRadius(math.P2(10, 0), 2)
	if err := applyDimension(s, entList(c)); err != nil {
		t.Fatalf("dimension radius: %v", err)
	}
	l1 := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	l2 := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 4))
	if err := applyDimension(s, entList(l1, l2)); err != nil {
		t.Fatalf("dimension angle: %v", err)
	}
	if sk.DimensionConstraints().Count() != 3 {
		t.Errorf("total dimensions = %d, want 3", sk.DimensionConstraints().Count())
	}
}

func TestDimensionNothingErrors(t *testing.T) {
	t.Parallel()
	s, _ := sketchSession(t)
	if err := applyDimension(s, nil); err == nil {
		t.Error("dimension with no entities should error")
	}
}

func TestSketchClickSelectsEntity(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	// A line through the origin; clicking the viewport centre (maps to sketch 0,0) selects it.
	sk.Lines().AddByTwoPoints(math.P2(-2, -2), math.P2(2, 2))
	s.Click(100, 100)
	if s.Selection().Count() != 1 {
		t.Fatalf("sketch click selected %d entities, want 1", s.Selection().Count())
	}
	if _, ok := s.Selection().First().(SketchEntityHandle); !ok {
		t.Errorf("sketch click selected %T, want a SketchEntityHandle", s.Selection().First())
	}
}
