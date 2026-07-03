// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

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
