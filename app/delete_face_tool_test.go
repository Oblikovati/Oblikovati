// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati/kernel/ops"
	"oblikovati/kernel/topo"
)

// chamferFaceHandleOf returns a handle to the body's chamfer (diagonal-normal) face.
func chamferFaceHandleOf(t *testing.T, b *topo.Body) FaceHandle {
	t.Helper()
	for _, f := range b.Faces() {
		if n := f.Geometry().NormalAt(0, 0); stdmath.Abs(n.X) > 0.1 && stdmath.Abs(n.Y) > 0.1 {
			return FaceHandle{Face: f, Body: b}
		}
	}
	t.Fatal("no chamfer face")
	return FaceHandle{}
}

// chamferOneEdge bevels a vertical edge of the active part's block via the Chamfer tool and
// returns the resulting body — a setup with a chamfer face for the Delete Face tests.
func chamferOneEdge(t *testing.T, s *Session, block *topo.Body) *topo.Body {
	t.Helper()
	s.SetPicker(stubPicker{sel: verticalEdgeOf(t, block)})
	ch := NewChamferTool()
	s.StartTool(ch)
	s.Click(50, 50)
	ch.SetDistance(0.5)
	if err := s.OK(); err != nil {
		t.Fatalf("chamfer OK: %v", err)
	}
	return activePartDef(t, s).SurfaceBodies().Item(0)
}

// TestDeleteFaceToolEndToEnd drives the Delete Face UI: chamfer a 2×2×2 block's edge (vol
// 7.75), then start the tool, click the chamfer face, OK — and asserts the body healed back
// to the sharp box (vol 8), a valid solid.
func TestDeleteFaceToolEndToEnd(t *testing.T) {
	s, block := newPartWithBlock(t, 2)
	chamfered := chamferOneEdge(t, s, block)

	s.SetPicker(stubPicker{sel: chamferFaceHandleOf(t, chamfered)})
	d := NewDeleteFaceTool()
	s.StartTool(d)
	s.Click(50, 50)
	if !d.CanCommit() {
		t.Fatal("delete-face not ready after picking a face")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}

	body := activePartDef(t, s).SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("healed body not a valid solid: %+v", r)
	}
	if got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErrApp(got, 8) > 1e-6 {
		t.Errorf("healed volume = %g, want 8", got)
	}
	if s.ActiveTool() != nil {
		t.Error("tool should have closed after OK")
	}
}

// TestDeleteFaceViaRibbonCommand drives Delete Face from its ribbon command.
func TestDeleteFaceViaRibbonCommand(t *testing.T) {
	s, block := newPartWithBlock(t, 2)
	chamfered := chamferOneEdge(t, s, block)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	s.SetPicker(stubPicker{sel: chamferFaceHandleOf(t, chamfered)})
	if err := s.Execute("Modify.DeleteFace"); err != nil {
		t.Fatalf("execute Modify.DeleteFace: %v", err)
	}
	if _, ok := s.ActiveTool().Tool().(*DeleteFaceTool); !ok {
		t.Fatal("Delete Face command did not start the tool")
	}
	s.Click(1, 1)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if v := ops.BodyGeometryProperties(activePartDef(t, s).SurfaceBodies().Item(0), ops.DefaultQuality()).Volume; relErrApp(v, 8) > 1e-6 {
		t.Errorf("delete-face did not heal to 8: volume %g", v)
	}
}

// TestDeleteFaceToolNeedsFace checks the tool is not committable until a face is picked.
func TestDeleteFaceToolNeedsFace(t *testing.T) {
	s, block := newPartWithBlock(t, 2)
	chamfered := chamferOneEdge(t, s, block)
	s.SetPicker(stubPicker{sel: chamferFaceHandleOf(t, chamfered)})
	d := NewDeleteFaceTool()
	s.StartTool(d)
	if d.CanCommit() {
		t.Error("delete-face ready with no face picked")
	}
	s.Click(0, 0)
	if !d.CanCommit() {
		t.Error("delete-face not ready after picking a face")
	}
}
