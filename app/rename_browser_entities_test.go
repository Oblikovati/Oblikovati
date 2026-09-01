// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// TestRenameSketchUniqueAndNonEmpty: a 2D sketch renames, but an empty or duplicate name is
// rejected and the original kept.
func TestRenameSketchUniqueAndNonEmpty(t *testing.T) {
	t.Parallel()
	s, def := emptyPartSession(t)
	a := def.Sketches().Add(sketch.XYPlane())
	b := def.Sketches().Add(sketch.XYPlane())
	if err := s.RenameSketch(a, "Profile"); err != nil {
		t.Fatalf("rename: %v", err)
	}
	if a.Name() != "Profile" {
		t.Errorf("sketch name = %q, want Profile", a.Name())
	}
	if err := s.RenameSketch(b, "Profile"); err == nil {
		t.Error("a duplicate sketch name should be rejected")
	}
	if err := s.RenameSketch(a, ""); err == nil {
		t.Error("an empty sketch name should be rejected")
	}
}

// TestRenameSketch3D: a 3D sketch renames through its own method.
func TestRenameSketch3D(t *testing.T) {
	t.Parallel()
	s, def := emptyPartSession(t)
	sk := def.Sketches3D().Add()
	other := def.Sketches3D().Add()
	if err := s.RenameSketch3D(sk, "Wire Path"); err != nil {
		t.Fatalf("rename 3D: %v", err)
	}
	if sk.Name() != "Wire Path" {
		t.Errorf("3D sketch name = %q, want Wire Path", sk.Name())
	}
	if err := s.RenameSketch3D(other, "Wire Path"); err == nil {
		t.Error("a duplicate 3D-sketch name should be rejected")
	}
	if err := s.RenameSketch3D(sk, ""); err == nil {
		t.Error("an empty 3D-sketch name should be rejected")
	}
}

// TestRenameWorkPlaneUserAndOrigin: a user work plane renames (unique), but the grounded origin
// planes cannot be renamed.
func TestRenameWorkPlaneUserAndOrigin(t *testing.T) {
	t.Parallel()
	s, def := emptyPartSession(t)
	p1 := def.WorkPlanes().AddByPlaneAndOffset(feature.OriginXYPlane, func() float64 { return 2 })
	p2 := def.WorkPlanes().AddByPlaneAndOffset(feature.OriginXYPlane, func() float64 { return 4 })
	def.Recompute()

	if err := s.RenameWorkPlane(p1, "Mount Face"); err != nil {
		t.Fatalf("rename work plane: %v", err)
	}
	if p1.Name() != "Mount Face" {
		t.Errorf("work plane name = %q, want Mount Face", p1.Name())
	}
	if err := s.RenameWorkPlane(p2, "Mount Face"); err == nil {
		t.Error("a duplicate work-plane name should be rejected")
	}
	origin, _ := def.WorkPlaneByName("XY Plane")
	if err := s.RenameWorkPlane(origin, "Base"); err == nil {
		t.Error("an origin coordinate-system plane must not be renameable")
	}
}

// TestRenameWorkAxisAndPoint: a user work axis and work point rename, and their origin
// counterparts are rejected.
func TestRenameWorkAxisAndPoint(t *testing.T) {
	t.Parallel()
	s, def := emptyPartSession(t)
	axis := def.WorkAxes().AddByPlaneIntersection(feature.OriginXYPlane, feature.OriginXZPlane)
	axis2 := def.WorkAxes().AddByPlaneIntersection(feature.OriginXYPlane, feature.OriginYZPlane)
	point := def.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(1, 2, 3) })
	point2 := def.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(4, 5, 6) })
	def.Recompute()

	if err := s.RenameWorkAxis(axis, "Spin Axis"); err != nil {
		t.Fatalf("rename axis: %v", err)
	}
	if axis.Name() != "Spin Axis" {
		t.Errorf("axis name = %q, want Spin Axis", axis.Name())
	}
	if err := s.RenameWorkAxis(axis2, "Spin Axis"); err == nil {
		t.Error("a duplicate work-axis name should be rejected")
	}
	if err := s.RenameWorkPoint(point, "Pivot"); err != nil {
		t.Fatalf("rename point: %v", err)
	}
	if point.Name() != "Pivot" {
		t.Errorf("point name = %q, want Pivot", point.Name())
	}
	if err := s.RenameWorkPoint(point2, "Pivot"); err == nil {
		t.Error("a duplicate work-point name should be rejected")
	}

	xAxis, _ := def.WorkGeometry().AxisByRef(feature.OriginXAxis)
	if err := s.RenameWorkAxis(xAxis, "Roll"); err == nil {
		t.Error("an origin coordinate-system axis must not be renameable")
	}
	center, _ := def.WorkGeometry().WorkPointByRef(feature.OriginCenter)
	if err := s.RenameWorkPoint(center, "Hub"); err == nil {
		t.Error("the origin centre point must not be renameable")
	}
}
