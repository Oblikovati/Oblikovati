// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/ops"
)

// TestReplaceFaceToolEndToEnd drives the Replace Face UI: click the top face to replace,
// switch to target mode, click the top face as the target (identity replace), OK — and
// asserts a valid solid results (vol 8). Geometric correctness for non-identity targets is
// covered by the kernel ReplaceFaces tests.
func TestReplaceFaceToolEndToEnd(t *testing.T) {
	t.Parallel()
	s, block := newPartWithBlock(t, 2) // 2×2×2
	top := topFaceOf(t, block)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: top, Body: block}})

	r := NewReplaceFaceTool()
	s.StartTool(r)
	s.Click(50, 50) // pick the face to replace
	if r.CanCommit() {
		t.Fatal("replace ready before a target is picked")
	}
	r.SetPickingTarget(true)
	s.Click(50, 50) // pick the target face
	if !r.CanCommit() {
		t.Fatal("replace not ready after face + target")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}

	body := activePartDef(t, s).SurfaceBodies().Item(0)
	if rr := ops.Validate(body); !rr.Valid || !body.IsSolid() {
		t.Fatalf("replaced body not a valid solid: %+v", rr)
	}
	if got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErrApp(got, 8) > 1e-6 {
		t.Errorf("identity replace volume = %g, want 8", got)
	}
	if s.ActiveTool() != nil {
		t.Error("tool should have closed after OK")
	}
}

// TestReplaceFaceViaRibbonCommand starts the tool from its ribbon command.
func TestReplaceFaceViaRibbonCommand(t *testing.T) {
	t.Parallel()
	s, block := newPartWithBlock(t, 2)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: topFaceOf(t, block), Body: block}})
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Modify.ReplaceFace"); err != nil {
		t.Fatalf("execute Modify.ReplaceFace: %v", err)
	}
	if _, ok := s.ActiveTool().Tool().(*ReplaceFaceTool); !ok {
		t.Fatal("Replace Face command did not start the tool")
	}
}

// TestReplaceFaceNeedsTarget checks the tool needs both a face and a target.
func TestReplaceFaceNeedsTarget(t *testing.T) {
	t.Parallel()
	s, block := newPartWithBlock(t, 2)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: topFaceOf(t, block), Body: block}})
	r := NewReplaceFaceTool()
	s.StartTool(r)
	s.Click(0, 0) // a face to replace
	if r.CanCommit() {
		t.Error("replace ready with no target")
	}
	r.SetPickingTarget(true)
	s.Click(0, 0) // the target
	if !r.CanCommit() {
		t.Error("replace not ready after face + target")
	}
}
