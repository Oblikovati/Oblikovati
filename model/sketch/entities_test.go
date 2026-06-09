// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
)

func TestCreateCurvesAndCounts(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	line := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(3, 4))
	circ := s.Circles().AddByCenterRadius(math.P2(1, 1), 2)
	arc := s.Arcs().AddByCenterStartEnd(math.P2(0, 0), math.P2(1, 0), math.P2(0, 1), true)
	ell := s.Ellipses().Add(math.P2(0, 0), math.V2(1, 0), 4, 2)

	if s.Lines().Count() != 1 || s.Circles().Count() != 1 || s.Arcs().Count() != 1 || s.Ellipses().Count() != 1 {
		t.Fatal("typed collection counts wrong")
	}
	// 4 curves are 4 drawable entities.
	if s.EntityCount() != 4 {
		t.Errorf("EntityCount = %d, want 4", s.EntityCount())
	}
	if !approx(line.Length(), 5) { // 3-4-5
		t.Errorf("line length = %v, want 5", line.Length())
	}
	if circ.Radius != 2 {
		t.Errorf("circle radius = %v", circ.Radius)
	}
	if !approx(arc.Radius(), 1) {
		t.Errorf("arc radius = %v, want 1", arc.Radius())
	}
	if ell.MajorRadius != 4 || ell.MinorRadius != 2 {
		t.Errorf("ellipse radii = %v,%v", ell.MajorRadius, ell.MinorRadius)
	}
}

func TestSharedPointIsStructuralCoincidence(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l1 := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0))
	// Second line starts at the first line's endpoint — sharing the *Point makes
	// the two coincident with no explicit constraint.
	l2 := s.Lines().Add(l1.EndPoint(), s.Points().Add(math.P2(1, 2)))
	if l1.EndPoint() != l2.StartPoint() {
		t.Fatal("shared endpoint not shared by pointer")
	}
	// Moving the shared point moves both lines' touching ends together.
	l1.EndPoint().SetPosition(math.P2(5, 5))
	if !l2.StartPoint().Position().IsEqualTo(math.P2(5, 5), tol) {
		t.Error("moving the shared point did not move the other line's end")
	}
}

func TestAllPointsCollectsEveryVariable(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0)) // 2 points
	s.Circles().AddByCenterRadius(math.P2(2, 2), 1)        // 1 point
	s.Points().Add(math.P2(9, 9))                          // 1 point
	if got := len(s.AllPoints()); got != 4 {
		t.Errorf("AllPoints = %d, want 4 (2 line + 1 center + 1 standalone)", got)
	}
}

func TestConstructionFlag(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 1))
	if l.IsConstruction() {
		t.Error("new line should be normal geometry")
	}
	l.SetConstruction(true)
	if !l.IsConstruction() {
		t.Error("SetConstruction ignored")
	}
}

func TestStandalonePointInEntitiesButEndpointsNot(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 1)) // endpoints NOT in Entities
	p := s.Points().Add(math.P2(2, 2))                     // standalone IS in Entities
	if s.Points().Count() != 1 || s.Points().Item(0) != p {
		t.Fatal("standalone point not tracked")
	}
	// Entities = 1 line + 1 standalone point = 2 (endpoints are not separate entities).
	if s.EntityCount() != 2 {
		t.Errorf("EntityCount = %d, want 2", s.EntityCount())
	}
}

func TestEntityAccessorsAndItems(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	line := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(2, 0))
	if !line.Direction().IsEqualTo(math.V2(2, 0), tol) {
		t.Errorf("Direction = %v, want (2,0)", line.Direction())
	}
	if line.EntityID() == 0 || line.StartPoint().EntityID() == 0 {
		t.Error("entity/point ids should be nonzero")
	}
	circ := s.Circles().AddByCenterRadius(math.P2(0, 0), 1)
	arc := s.Arcs().AddByCenterStartEnd(math.P2(0, 0), math.P2(1, 0), math.P2(0, 1), false)
	ell := s.Ellipses().Add(math.P2(0, 0), math.V2(1, 0), 3, 1)
	if s.Lines().Item(0) != line || s.Circles().Item(0) != circ ||
		s.Arcs().Item(0) != arc || s.Ellipses().Item(0) != ell {
		t.Error("typed Item accessors returned the wrong entity")
	}
}

func TestBlockDefinitionEntities(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	def := s.Blocks().DefineBlock("b")
	scratch := NewSketches().Add(XYPlane())
	def.Add(scratch.Points().Add(math.P2(0, 0)))
	if len(def.Entities()) != 1 {
		t.Errorf("definition Entities = %d, want 1", len(def.Entities()))
	}
}

func approx(a, b math.Scalar) bool {
	d := a - b
	return d < 1e-9 && d > -1e-9
}
