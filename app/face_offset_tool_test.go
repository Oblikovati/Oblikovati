// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati/kernel/ops"
)

// TestFaceOffsetToolEndToEnd drives the Offset Face UI: start the tool, click the block's
// top face, set a +1 offset, OK — and asserts the solid grew along the normal. Block 4×4×2
// (vol 32); offsetting the top +1 → 4×4×3 = 48.
func TestFaceOffsetToolEndToEnd(t *testing.T) {
	s, block := newPartWithBlock(t, 4)
	top := topFaceOf(t, block)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: top, Body: block}})

	off := NewFaceOffsetTool()
	s.StartTool(off)
	s.Click(100, 100)
	off.SetDistance(1)
	if !off.CanCommit() {
		t.Fatal("offset tool not ready after face + distance")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}

	def := activePartDef(t, s)
	body := def.SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("offset body not a valid solid: %+v", r)
	}
	if got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErrApp(got, 48) > 1e-6 {
		t.Errorf("offset volume = %g, want 48", got)
	}
	if s.ActiveTool() != nil {
		t.Error("tool should have closed after OK")
	}
}

// TestFaceOffsetViaRibbonCommand drives the Offset Face from its ribbon command.
func TestFaceOffsetViaRibbonCommand(t *testing.T) {
	s, block := newPartWithBlock(t, 4)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: topFaceOf(t, block), Body: block}})
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Modify.FaceOffset"); err != nil {
		t.Fatalf("execute Modify.FaceOffset: %v", err)
	}
	if _, ok := s.ActiveTool().Tool().(*FaceOffsetTool); !ok {
		t.Fatal("Offset Face command did not start the tool")
	}
	s.Click(1, 1)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	def := activePartDef(t, s)
	if v := ops.BodyGeometryProperties(def.SurfaceBodies().Item(0), ops.DefaultQuality()).Volume; v <= 32 {
		t.Errorf("offset did not grow the body: volume %g, want > 32", v)
	}
}

// TestFaceOffsetToolNeedsFace checks the tool is not committable until a face is picked.
func TestFaceOffsetToolNeedsFace(t *testing.T) {
	s, block := newPartWithBlock(t, 4)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: topFaceOf(t, block), Body: block}})
	off := NewFaceOffsetTool()
	s.StartTool(off)
	if off.CanCommit() {
		t.Error("offset ready with no face picked")
	}
	s.Click(0, 0)
	if !off.CanCommit() {
		t.Error("offset not ready after picking a face")
	}
}
