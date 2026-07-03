// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// TestFeatureHandleRenameCapability proves a feature handle self-describes its name, reports
// itself renameable, and renames through the capability — the behavior head/ui's old
// per-handle-type switch performed, now driven by NodeRenameable (#1630).
func TestFeatureHandleRenameCapability(t *testing.T) {
	s, block := newPartWithBlock(t, 2)
	f := activePartDef(t, s).Features().Item(0)
	h := FeatureHandle{Feature: f}
	if h.NodeName() != f.Name() {
		t.Errorf("NodeName = %q, want %q", h.NodeName(), f.Name())
	}
	if !h.Renameable() {
		t.Error("a feature node should be renameable")
	}
	if err := h.Rename(s, "Base Extrude"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if f.Name() != "Base Extrude" {
		t.Errorf("after Rename, feature name = %q, want Base Extrude", f.Name())
	}
	_ = block
}

// TestWorkPlaneRenameParity checks the renameable/fixed split the old isRenameableNode switch
// encoded: a user work plane is renameable, but a grounded origin datum is not (#1264, #1630).
func TestWorkPlaneRenameParity(t *testing.T) {
	s, def := emptyPartSession(t)
	user := def.WorkPlanes().AddByPlaneAndOffset(feature.OriginXYPlane, func() float64 { return 2 })
	def.Recompute()
	if h := (WorkPlaneHandle{Plane: user}); !h.Renameable() {
		t.Error("a user work plane should be renameable")
	}
	origin, ok := def.WorkPlaneByName("XY Plane")
	if !ok {
		t.Fatal("origin XY plane not found")
	}
	if h := (WorkPlaneHandle{Plane: origin}); h.Renameable() {
		t.Error("the grounded origin XY plane must not be renameable")
	}
	if err := (WorkPlaneHandle{Plane: user}).Rename(s, "Mount Face"); err != nil {
		t.Fatalf("Rename user plane: %v", err)
	}
	if user.Name() != "Mount Face" {
		t.Errorf("work plane name = %q, want Mount Face", user.Name())
	}
}

// TestSketchNameCapabilities covers 2D and 3D sketch handles' NodeName so the browser label
// path has parity with the old nodeName switch.
func TestSketchNameCapabilities(t *testing.T) {
	_, def := emptyPartSession(t)
	sk := def.Sketches().Add(sketch.XYPlane())
	if got := (SketchHandle{Sketch: sk}).NodeName(); got != sk.Name() {
		t.Errorf("2D sketch NodeName = %q, want %q", got, sk.Name())
	}
	s3 := def.Sketches3D().Add()
	if got := (Sketch3DHandle{Sketch3D: s3}).NodeName(); got != s3.Name() {
		t.Errorf("3D sketch NodeName = %q, want %q", got, s3.Name())
	}
}

// TestFeatureActivateOpensEditor proves the NodeActivatable capability performs the feature's
// double-click action (opening its edit tool) — the behavior head/ui's old switch dispatched
// through openEditOnDoubleClick (#1630).
func TestFeatureActivateOpensEditor(t *testing.T) {
	s, _ := newPartWithBlock(t, 2)
	f := activePartDef(t, s).Features().Item(0) // the block's extrude
	FeatureHandle{Feature: f}.Activate(s)
	tool := s.ActiveTool()
	if tool == nil || tool.Name() != "Extrude" {
		t.Errorf("activating the extrude feature should re-open its Extrude tool, got %v", tool)
	}
}

// TestSketchHandleRenameCapability: a 2D sketch handle reports itself renameable and renames
// through the NodeRenameable capability (#1630), covering the SketchHandle Renameable/Rename seam.
func TestSketchHandleRenameCapability(t *testing.T) {
	s, def := emptyPartSession(t)
	sk := def.Sketches().Add(sketch.XYPlane())
	h := SketchHandle{Sketch: sk}
	if !h.Renameable() {
		t.Error("a 2D sketch node should be renameable")
	}
	if err := h.Rename(s, "Profile"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if sk.Name() != "Profile" {
		t.Errorf("sketch name = %q, want Profile", sk.Name())
	}
}

// TestSketch3DHandleRenameCapability: a 3D sketch handle reports itself renameable and renames
// through the NodeRenameable capability (#1630), covering the Sketch3DHandle Renameable/Rename seam.
func TestSketch3DHandleRenameCapability(t *testing.T) {
	s, def := emptyPartSession(t)
	s3 := def.Sketches3D().Add()
	h := Sketch3DHandle{Sketch3D: s3}
	if !h.Renameable() {
		t.Error("a 3D sketch node should be renameable")
	}
	if err := h.Rename(s, "Wire Path"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if s3.Name() != "Wire Path" {
		t.Errorf("3D sketch name = %q, want Wire Path", s3.Name())
	}
}

// TestWorkPlaneNodeName: the WorkPlaneHandle reports its plane's current name via NodeName,
// covering the NodeName seam the browser label path uses (#1630).
func TestWorkPlaneNodeName(t *testing.T) {
	_, def := emptyPartSession(t)
	wp := def.WorkPlanes().AddByPlaneAndOffset(feature.OriginXYPlane, func() float64 { return 2 })
	def.Recompute()
	if got := (WorkPlaneHandle{Plane: wp}).NodeName(); got != wp.Name() {
		t.Errorf("work plane NodeName = %q, want %q", got, wp.Name())
	}
}

// userWorkAxis returns a fresh user work axis on an empty part (a two-plane intersection), the
// fixture the work-axis capability tests drive.
func userWorkAxis(t *testing.T) (*Session, *feature.WorkAxis) {
	t.Helper()
	s, def := emptyPartSession(t)
	axis := def.WorkAxes().AddByPlaneIntersection(feature.OriginXYPlane, feature.OriginYZPlane)
	def.Recompute()
	return s, axis
}

// TestWorkAxisRenameCapability: a user work axis self-describes its name, reports itself
// renameable, and renames through the capability (#1630).
func TestWorkAxisRenameCapability(t *testing.T) {
	s, axis := userWorkAxis(t)
	h := WorkAxisHandle{Axis: axis}
	if h.NodeName() != axis.Name() {
		t.Errorf("NodeName = %q, want %q", h.NodeName(), axis.Name())
	}
	if !h.Renameable() {
		t.Error("a user work axis should be renameable")
	}
	if err := h.Rename(s, "Spin Axis"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if axis.Name() != "Spin Axis" {
		t.Errorf("axis name = %q, want Spin Axis", axis.Name())
	}
}

// TestOriginWorkAxisNotRenameable: a grounded origin coordinate-system axis reports itself not
// renameable through the capability (its name is fixed, #1264/#1630).
func TestOriginWorkAxisNotRenameable(t *testing.T) {
	_, def := emptyPartSession(t)
	xAxis, ok := def.WorkGeometry().AxisByRef(feature.OriginXAxis)
	if !ok {
		t.Fatal("origin X axis not found")
	}
	if (WorkAxisHandle{Axis: xAxis}).Renameable() {
		t.Error("an origin coordinate-system axis must not be renameable")
	}
}

// userWorkPoint returns a fresh user work point on an empty part, the fixture the work-point
// capability tests drive.
func userWorkPoint(t *testing.T) (*Session, *feature.WorkPoint) {
	t.Helper()
	s, def := emptyPartSession(t)
	point := def.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(1, 2, 3) })
	def.Recompute()
	return s, point
}

// TestWorkPointRenameCapability: a user work point self-describes its name, reports itself
// renameable, and renames through the capability (#1630).
func TestWorkPointRenameCapability(t *testing.T) {
	s, point := userWorkPoint(t)
	h := WorkPointHandle{Point: point}
	if h.NodeName() != point.Name() {
		t.Errorf("NodeName = %q, want %q", h.NodeName(), point.Name())
	}
	if !h.Renameable() {
		t.Error("a user work point should be renameable")
	}
	if err := h.Rename(s, "Pivot"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if point.Name() != "Pivot" {
		t.Errorf("point name = %q, want Pivot", point.Name())
	}
}

// TestOriginWorkPointNotRenameable: the grounded origin centre point reports itself not
// renameable through the capability (its name is fixed, #1264/#1630).
func TestOriginWorkPointNotRenameable(t *testing.T) {
	_, def := emptyPartSession(t)
	center, ok := def.WorkGeometry().WorkPointByRef(feature.OriginCenter)
	if !ok {
		t.Fatal("origin centre point not found")
	}
	if (WorkPointHandle{Point: center}).Renameable() {
		t.Error("the origin centre point must not be renameable")
	}
}
