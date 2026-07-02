// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/param"
)

// TestSketch3DAddPointEntities checks standalone points are both solver variables and
// enumerable entities, and are reachable by id.
func TestSketch3DAddPointEntities(t *testing.T) {
	s := NewSketches3D().Add()
	a := s.AddPoint3D(math.P3(1, 2, 3))
	b := s.AddPoint3D(math.P3(4, 5, 6))

	if s.EntityCount() != 2 {
		t.Fatalf("EntityCount = %d, want 2", s.EntityCount())
	}
	if len(s.AllPoints3D()) != 2 {
		t.Fatalf("AllPoints3D = %d, want 2", len(s.AllPoints3D()))
	}
	if got, ok := s.EntityByID(a.EntityID()); !ok || got != a {
		t.Error("EntityByID(a) failed")
	}
	if got, ok := s.PointByID(b.id); !ok || got != b {
		t.Error("PointByID(b) failed")
	}
	if _, ok := s.EntityByID(99999); ok {
		t.Error("EntityByID of a missing id should fail")
	}
}

// TestSketch3DDOFAndSolve checks the dimension-agnostic solver counts 3 DOF per free 3D
// point and that a coincident constraint removes 3 of them, solving b onto a.
func TestSketch3DDOFAndSolve(t *testing.T) {
	s := NewSketches3D().Add()
	a := s.AddPoint3D(math.P3(0, 0, 0))
	b := s.AddPoint3D(math.P3(5, 5, 5))

	if dof := s.DegreesOfFreedom(); dof != 6 {
		t.Fatalf("DOF = %d, want 6 (two free 3D points)", dof)
	}

	s.GeometricConstraints3D().add(NewCoincident3D(a, b))
	if dof := s.DegreesOfFreedom(); dof != 3 {
		t.Fatalf("DOF after coincident = %d, want 3", dof)
	}

	res := s.Solve()
	if !res.Converged {
		t.Fatalf("solve did not converge: %+v", res)
	}
	if b.Position().DistanceTo(a.Position()) > 1e-7 {
		t.Errorf("after solve b=%v should coincide with a=%v", b.Position(), a.Position())
	}
	if !s.Health().OK() {
		t.Errorf("a solvable sketch should be healthy, got %+v", s.Health())
	}
}

// TestSketch3DProperties checks the display/solve properties round-trip through their
// getters/setters.
func TestSketch3DProperties(t *testing.T) {
	s := NewSketches3D().Add()
	if !s.Visible() || !s.DimensionsVisible() {
		t.Fatal("a new 3D sketch should be visible with visible dimensions")
	}
	s.SetVisible(false)
	s.SetDimensionsVisible(false)
	s.SetColor("#ff0000")
	s.SetDeferUpdates(true)
	if s.Visible() || s.DimensionsVisible() || s.Color() != "#ff0000" || !s.DeferUpdates() {
		t.Errorf("property setters did not stick: %+v", s)
	}
}

// TestSketch3DAutoName checks the collection mints unique "3D Sketch{N}" names.
func TestSketch3DAutoName(t *testing.T) {
	c := NewSketches3D()
	if got := c.Add().Name(); got != "3D Sketch1" {
		t.Errorf("first auto-name = %q, want %q", got, "3D Sketch1")
	}
	if got := c.Add().Name(); got != "3D Sketch2" {
		t.Errorf("second auto-name = %q, want %q", got, "3D Sketch2")
	}
}

// TestSketch3DRemove checks removing a sketch by id.
func TestSketch3DRemove(t *testing.T) {
	c := NewSketches3D()
	s := c.Add()
	if !c.Remove(s.ID()) || c.Count() != 0 {
		t.Fatalf("Remove failed, count=%d", c.Count())
	}
	if c.Remove(s.ID()) {
		t.Error("removing a missing sketch should report false")
	}
}

// TestSketch3DParameterStore checks the dimension collection and parameter store
// accessors, and that SetParameters repoints both at a shared store.
func TestSketch3DParameterStore(t *testing.T) {
	s := NewSketches3D().Add()
	if s.DimensionConstraints3D() == nil || s.Parameters() == nil {
		t.Fatal("a new 3D sketch should have a dimension collection and a parameter store")
	}
	shared := param.NewParameters()
	s.SetParameters(shared)
	if s.Parameters() != shared || s.DimensionConstraints3D().params != shared {
		t.Error("SetParameters should repoint both the sketch and its dimension collection")
	}
}

// TestSketch3DSerializeErrors covers the missing-codec and unknown-id error paths of the
// 3D sketch codec.
func TestSketch3DSerializeErrors(t *testing.T) {
	// CustomConstraint3D is a solver adapter over an opaque residual closure —
	// deliberately not serializable.
	if _, err := serializeConstraint3D(NewCustomConstraint3D(func() []float64 { return nil }, nil)); err == nil {
		t.Error("serializeConstraint3D should error for an unsupported constraint")
	}
	// Restoring a constraint that references an unknown point id fails honestly.
	idmap := map[int]*Point3D{}
	if err := restoreConstraint3D(NewSketches3D().Add(), Constraint3DRow{Kind: "coincident", Points: []int{7, 8}}, idmap, map[int]Entity{}); err == nil {
		t.Error("restoreConstraint3D should error on an unknown point id")
	}
	// An unknown constraint kind is a corrupt-recipe error.
	a := NewPoint3D(math.P3(0, 0, 0))
	if err := restoreConstraint3D(NewSketches3D().Add(), Constraint3DRow{Kind: "bogus", Points: []int{1}}, map[int]*Point3D{1: a}, map[int]Entity{}); err == nil {
		t.Error("restoreConstraint3D should error on an unknown kind")
	}
}

// TestSketch3DDistanceDimension checks a distance dimension drives the sketch (a
// driving dimension counts toward Constraints; a driven one only reports) and that the
// dimension reports its kind name.
func TestSketch3DDistanceDimension(t *testing.T) {
	s := NewSketches3D().Add()
	a := s.AddPoint3D(math.P3(0, 0, 0))
	b := s.AddPoint3D(math.P3(10, 0, 0))
	d, err := s.DimensionConstraints3D().AddDistance(a, b, "5 cm")
	if err != nil {
		t.Fatalf("AddDistance: %v", err)
	}
	if d.KindName() != "distance" {
		t.Errorf("KindName = %q, want distance", d.KindName())
	}
	if got := len(s.Constraints()); got != 1 {
		t.Errorf("a driving dimension should be in Constraints, got %d", got)
	}
	d.SetDriven(true)
	if got := len(s.Constraints()); got != 0 {
		t.Errorf("a driven dimension should be excluded from Constraints, got %d", got)
	}
}

// TestSketch3DPointByIDMiss covers the not-found path of PointByID.
func TestSketch3DPointByIDMiss(t *testing.T) {
	s := NewSketches3D().Add()
	s.AddPoint3D(math.P3(0, 0, 0))
	if _, ok := s.PointByID(424242); ok {
		t.Error("PointByID of a missing id should fail")
	}
}

// TestSketch3DNameCollision checks the auto-namer skips a number already taken by an
// explicitly-named sketch.
func TestSketch3DNameCollision(t *testing.T) {
	c := NewSketches3D()
	c.AddNamed("3D Sketch1")
	if got := c.Add().Name(); got != "3D Sketch2" {
		t.Errorf("auto-name after a taken '3D Sketch1' = %q, want '3D Sketch2'", got)
	}
}

// TestSketch3DRecipeErrorsPropagate checks a failing constraint codec surfaces through
// MarshalRecipe3D, and a malformed recipe surfaces through ApplyRecipe3D (with context).
func TestSketch3DRecipeErrorsPropagate(t *testing.T) {
	src := NewSketches3D()
	s := src.Add()
	s.GeometricConstraints3D().add(NewCustomConstraint3D(func() []float64 { return nil }, nil)) // no codec ⇒ marshal fails
	if _, err := src.MarshalRecipe3D(); err == nil {
		t.Error("MarshalRecipe3D should propagate a constraint-codec error")
	}

	bad := []SketchData3D{{
		Name:        "broken",
		Constraints: []Constraint3DRow{{Kind: "coincident", Points: []int{1, 2}}}, // points never declared
	}}
	if err := NewSketches3D().ApplyRecipe3D(bad); err == nil {
		t.Error("ApplyRecipe3D should propagate a malformed-recipe error")
	}
}

// TestSketch3DSerializeRoundTrip checks points, standalone flags, properties and a
// coincident constraint survive marshal→apply with equal geometry.
func TestSketch3DSerializeRoundTrip(t *testing.T) {
	src := NewSketches3D()
	s := src.Add()
	s.SetVisible(false)
	s.SetDimensionsVisible(false)
	s.SetColor("#00ff00")
	a := s.AddPoint3D(math.P3(1, 2, 3))
	b := s.AddPoint3D(math.P3(-4, 5, -6))
	c := s.AddPoint3D(math.P3(7, 8, 9))
	s.GeometricConstraints3D().add(NewCoincident3D(a, b))
	s.GeometricConstraints3D().add(NewCollinear3D(a, b, c))
	s.GeometricConstraints3D().add(NewConcentric3D(a, c))

	data, err := src.MarshalRecipe3D()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	dst := NewSketches3D()
	if err := dst.ApplyRecipe3D(data); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if dst.Count() != 1 {
		t.Fatalf("restored %d sketches, want 1", dst.Count())
	}
	got := dst.Item(0)
	if got.Name() != s.Name() || got.Visible() || got.DimensionsVisible() || got.Color() != "#00ff00" {
		t.Errorf("restored properties mismatch: %+v", got)
	}
	if got.EntityCount() != 3 {
		t.Fatalf("restored EntityCount = %d, want 3", got.EntityCount())
	}
	pts := got.AllPoints3D()
	if pts[0].Position() != math.P3(1, 2, 3) || pts[1].Position() != math.P3(-4, 5, -6) {
		t.Errorf("restored point positions mismatch: %v, %v", pts[0].Position(), pts[1].Position())
	}
	if got.GeometricConstraints3D().Count() != 3 {
		t.Errorf("restored constraints = %d, want 3 (coincident+collinear+concentric)", got.GeometricConstraints3D().Count())
	}
}
