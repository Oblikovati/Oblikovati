// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	gmath "github.com/Oblikovati/oblikovati/math"
)

// TestMoveEntities3D checks a line's endpoints translate together.
func TestMoveEntities3D(t *testing.T) {
	s := NewSketches3D().Add()
	l := s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(1, 0, 0))
	s.MoveEntities3D([]Entity{l}, gmath.V3(2, 3, 4))
	if l.A.Position() != gmath.P3(2, 3, 4) || l.B.Position() != gmath.P3(3, 3, 4) {
		t.Errorf("moved line = %v..%v, want (2,3,4)..(3,3,4)", l.A.Position(), l.B.Position())
	}
}

// TestRotateEntities3D checks a point rotates 90° about +Z through the origin.
func TestRotateEntities3D(t *testing.T) {
	s := NewSketches3D().Add()
	p := s.AddPoint3D(gmath.P3(1, 0, 0))
	z, _ := gmath.NewUnitVector3(0, 0, 1)
	s.RotateEntities3D([]Entity{p}, gmath.P3(0, 0, 0), z, math.Pi/2)
	if p.Position().DistanceTo(gmath.P3(0, 1, 0)) > 1e-9 {
		t.Errorf("rotated point = %v, want (0,1,0)", p.Position())
	}
}

// TestCopyEntities3D checks a copy creates new entities at translated positions, leaving
// the original in place.
func TestCopyEntities3D(t *testing.T) {
	s := NewSketches3D().Add()
	l := s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(1, 0, 0))
	created := s.CopyEntities3D([]Entity{l}, gmath.V3(0, 5, 0))
	if len(created) != 1 || s.EntityCount() != 2 {
		t.Fatalf("copy: created %d, entityCount %d, want 1/2", len(created), s.EntityCount())
	}
	if l.A.Position() != gmath.P3(0, 0, 0) {
		t.Error("copy must not move the original")
	}
	nl := created[0].(*Line3D)
	if nl.A.Position() != gmath.P3(0, 5, 0) || nl.B.Position() != gmath.P3(1, 5, 0) {
		t.Errorf("copied line = %v..%v, want (0,5,0)..(1,5,0)", nl.A.Position(), nl.B.Position())
	}
}

// TestCopyEntities3DSharedPoints checks a connected chain copies as a connected whole
// (shared endpoints stay shared in the copy).
func TestCopyEntities3DSharedPoints(t *testing.T) {
	s := NewSketches3D().Add()
	a := s.newPoint3D(gmath.P3(0, 0, 0))
	b := s.newPoint3D(gmath.P3(1, 0, 0))
	c := s.newPoint3D(gmath.P3(2, 0, 0))
	l1 := s.addLine3DPts(a, b)
	l2 := s.addLine3DPts(b, c)
	created := s.CopyEntities3D([]Entity{l1, l2}, gmath.V3(0, 0, 1))
	if created[0].(*Line3D).B != created[1].(*Line3D).A {
		t.Error("copied chain should share the middle point")
	}
}

// TestCopyEntities3DAllKinds copies one of every entity kind, exercising every clone +
// point-collection path, and checks each kind is duplicated.
func TestCopyEntities3DAllKinds(t *testing.T) {
	s := NewSketches3D().Add()
	z, _ := gmath.NewUnitVector3(0, 0, 1)
	x, _ := gmath.NewUnitVector3(1, 0, 0)
	originals := []Entity{
		s.AddPoint3D(gmath.P3(0, 0, 0)),
		s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(1, 0, 0)),
		s.AddCircle3D(gmath.P3(0, 0, 0), z, 2),
		s.AddArc3D(gmath.P3(0, 0, 0), gmath.P3(1, 0, 0), gmath.P3(0, 1, 0), true),
		s.AddEllipse3D(gmath.P3(0, 0, 0), z, x, 4, 2),
		s.AddEllipticalArc3D(gmath.P3(0, 0, 0), z, x, 4, 2, 0, math.Pi/2),
		s.AddHelix3D(gmath.P3(0, 0, 0), z, 3, 5, 0, 2, false),
		s.AddSpline3D([]gmath.Point3{{X: 0}, {X: 1, Y: 1}, {X: 2}}, false, true),
		s.AddFixedSpline3D([]gmath.Point3{{X: 0}, {X: 1, Y: 1, Z: 1}}, false),
	}
	before := s.EntityCount()
	created := s.CopyEntities3D(originals, gmath.V3(0, 0, 10))
	if len(created) != len(originals) {
		t.Fatalf("copied %d of %d entities", len(created), len(originals))
	}
	if s.EntityCount() != before+len(originals) {
		t.Errorf("entityCount = %d, want %d", s.EntityCount(), before+len(originals))
	}
	// The copied fixed spline's coordinates are translated +10 in Z.
	for _, e := range created {
		if fs, ok := e.(*FixedSpline3D); ok && fs.Pts[0].Z != 10 {
			t.Errorf("copied fixed spline not translated: %v", fs.Pts[0])
		}
	}

	// Moving the whole selection exercises entityPoints3D for every kind.
	s.MoveEntities3D(originals, gmath.V3(1, 0, 0))

	// An equation curve has no spatial copy (parametric definition) — it is skipped.
	eq, err := s.AddEquationCurve3D("t", "t", "t", 0, 1)
	if err != nil {
		t.Fatalf("equation curve: %v", err)
	}
	if got := s.CopyEntities3D([]Entity{eq}, gmath.V3(0, 0, 1)); len(got) != 0 {
		t.Errorf("copying an equation curve should yield nothing, got %d", len(got))
	}
}

// TestDeleteEntity3D checks delete removes the entity, its orphaned points, and the
// constraints that referenced them.
func TestDeleteEntity3D(t *testing.T) {
	s := NewSketches3D().Add()
	l := s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(1, 0, 0))
	keep := s.AddPoint3D(gmath.P3(5, 5, 5))
	s.GeometricConstraints3D().add(NewParallelToXAxis3D(l))
	s.GeometricConstraints3D().add(NewGround3D(keep))
	if _, err := s.DimensionConstraints3D().AddLineLength(l, "10 cm"); err != nil {
		t.Fatalf("AddLineLength: %v", err)
	}

	if !s.DeleteEntity3D(l) {
		t.Fatal("DeleteEntity3D should report the line was present")
	}
	if s.EntityCount() != 1 {
		t.Errorf("after delete, entityCount = %d, want 1 (the kept point)", s.EntityCount())
	}
	if len(s.AllPoints3D()) != 1 {
		t.Errorf("after delete, points = %d, want 1 (line endpoints pruned)", len(s.AllPoints3D()))
	}
	if s.GeometricConstraints3D().Count() != 1 {
		t.Errorf("the line's constraint should be detached, leaving 1 (ground), got %d", s.GeometricConstraints3D().Count())
	}
	if s.DimensionConstraints3D().Count() != 0 {
		t.Errorf("the line's dimension should be detached, got %d", s.DimensionConstraints3D().Count())
	}
}

// TestDeleteEntity3DMissing checks deleting an absent entity reports false.
func TestDeleteEntity3DMissing(t *testing.T) {
	s := NewSketches3D().Add()
	orphan := NewPoint3D(gmath.P3(0, 0, 0))
	if s.DeleteEntity3D(orphan) {
		t.Error("deleting an entity not in the sketch should report false")
	}
}
