// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
)

func TestSketchesAddAndLookup(t *testing.T) {
	c := NewSketches()
	s := c.AddNamed("Sketch1", XYPlane())
	if c.Count() != 1 || c.Item(0) != s {
		t.Fatal("Add did not register the sketch")
	}
	if got, ok := c.ByID(s.ID()); !ok || got != s {
		t.Error("ByID did not find the sketch")
	}
	if s.Name() != "Sketch1" {
		t.Errorf("Name = %q", s.Name())
	}
	if !s.Health().OK() {
		t.Error("new sketch is not healthy")
	}
	if !s.Visible() {
		t.Error("new sketch is not visible by default")
	}
}

func TestSketchEditStateAndVisibility(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	if s.IsEditing() {
		t.Error("new sketch is already editing")
	}
	s.Edit()
	if !s.IsEditing() {
		t.Error("Edit did not enter edit mode")
	}
	s.ExitEdit()
	if s.IsEditing() {
		t.Error("ExitEdit did not leave edit mode")
	}
	s.SetVisible(false)
	if s.Visible() {
		t.Error("SetVisible(false) ignored")
	}
	s.SetName("Renamed")
	if s.Name() != "Renamed" {
		t.Errorf("SetName ignored: %q", s.Name())
	}
}

func TestSketchCoordinateMapping(t *testing.T) {
	s := NewSketches().Add(XZPlane())
	p := s.ToModel(math.P2(1, 2))
	if !p.IsEqualTo(math.P3(1, 0, 2), tol) {
		t.Errorf("ToModel = %v, want (1,0,2)", p)
	}
	if b := s.ToSketch(p); !b.IsEqualTo(math.P2(1, 2), tol) {
		t.Errorf("ToSketch round trip = %v, want (1,2)", b)
	}
}

func TestSketchPlaneAndEntitiesAccessors(t *testing.T) {
	s := NewSketches().Add(XZPlane())
	if s.Plane().Normal().AsVector().IsEqualTo(math.V3(0, 0, 1), tol) {
		t.Error("XZ plane normal should not be +Z")
	}
	if len(s.Entities()) != 0 {
		t.Error("new sketch should have no entities")
	}
}

func TestSketch3DAndDrawingSketchCollections(t *testing.T) {
	c3 := NewSketches3D()
	c3.Add() // unnamed Add → Item(0)
	s3 := c3.AddNamed("Path")
	if c3.Count() != 2 || c3.Item(1) != s3 {
		t.Fatal("Sketches3D.Add failed")
	}
	if got, ok := c3.ByID(s3.ID()); !ok || got != s3 {
		t.Error("Sketches3D.ByID failed")
	}
	if s3.EntityCount() != 0 || len(s3.Entities()) != 0 {
		t.Error("new 3D sketch should have no entities")
	}

	cd := NewDrawingSketches()
	sd := cd.Add()
	if cd.Count() != 1 || cd.Item(0) != sd {
		t.Fatal("DrawingSketches.Add failed")
	}
	if _, ok := cd.ByID(sd.ID()); !ok {
		t.Error("DrawingSketches.ByID failed")
	}
	if len(sd.Entities()) != 0 {
		t.Error("new drawing sketch should have no entities")
	}
	if sd.EntityCount() != 0 {
		t.Error("new drawing sketch should have no entities")
	}
}

func TestSketchIDsAreUnique(t *testing.T) {
	c := NewSketches()
	a := c.Add(XYPlane())
	b := c.Add(XYPlane())
	if a.ID() == b.ID() {
		t.Error("sketches share an id")
	}
}

func TestAddAutoNamesSequentially(t *testing.T) {
	c := NewSketches()
	if got := c.Add(XYPlane()).Name(); got != "Sketch1" {
		t.Errorf("first Add name = %q, want Sketch1", got)
	}
	if got := c.Add(XYPlane()).Name(); got != "Sketch2" {
		t.Errorf("second Add name = %q, want Sketch2", got)
	}
}

func TestAddSkipsNameTakenByAddNamed(t *testing.T) {
	c := NewSketches()
	c.AddNamed("Sketch1", XYPlane()) // a restored sketch already holds Sketch1
	if got := c.Add(XYPlane()).Name(); got != "Sketch2" {
		t.Errorf("Add after explicit Sketch1 = %q, want Sketch2 (no collision)", got)
	}
}

func TestRemoveSketch(t *testing.T) {
	c := NewSketches()
	a := c.Add(XYPlane())
	b := c.Add(XZPlane())
	if !c.Remove(a.ID()) {
		t.Fatal("Remove reported the sketch missing")
	}
	if c.Count() != 1 || c.Item(0) != b {
		t.Errorf("after Remove: count=%d, item0==b: %v", c.Count(), c.Item(0) == b)
	}
	if _, ok := c.ByID(a.ID()); ok {
		t.Error("ByID still finds the removed sketch")
	}
	if c.Remove(a.ID()) {
		t.Error("second Remove of the same id reported success")
	}
}
