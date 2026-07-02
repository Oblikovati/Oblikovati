// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"math"
	"testing"

	"oblikovati.org/kernel/ops"
)

// Moving the top face of a block outward grows its volume.
func TestMoveFaceTool(t *testing.T) {
	s, block := newPartWithBlock(t, 4)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: topFaceOf(t, block), Body: block}})
	def := activePartDef(t, s)
	vol0 := ops.BodyGeometryProperties(def.SurfaceBodies().Item(0), ops.DefaultQuality()).Volume

	tool := NewMoveFaceTool()
	tool.dz = 2
	s.StartTool(tool)
	s.Click(100, 100) // pick the top face
	if err := s.OK(); err != nil {
		t.Fatalf("move face OK: %v", err)
	}
	if s.ActiveTool() != nil {
		t.Fatal("move-face tool should deactivate after OK")
	}
	body := def.SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid {
		t.Fatalf("moved body invalid: %+v", r)
	}
	vol1 := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume
	if vol1 <= vol0 {
		t.Fatalf("moving top face +Z should grow volume: %g → %g", vol0, vol1)
	}
}

// Combining two bodies (Join) yields a single body.
func TestCombineTool(t *testing.T) {
	s, def, src := extrudedPart(t)
	pat := NewFeatureRectPatternTool() // default 2×1 → a second body
	s.StartTool(pat)
	s.feedPick(src)
	if err := s.OK(); err != nil {
		t.Fatalf("pattern setup OK: %v", err)
	}
	if def.SurfaceBodies().Count() != 2 {
		t.Fatalf("setup wanted 2 bodies, got %d", def.SurfaceBodies().Count())
	}

	tool := NewCombineTool() // Join
	s.StartTool(tool)
	s.feedPick(BodyHandle{Body: def.SurfaceBodies().Item(0)})
	s.feedPick(BodyHandle{Body: def.SurfaceBodies().Item(1)})
	if err := s.OK(); err != nil {
		t.Fatalf("combine OK: %v", err)
	}
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("after join = %d bodies, want 1", def.SurfaceBodies().Count())
	}
}

// Moving a body translates it.
func TestMoveBodyTool(t *testing.T) {
	s, def, _ := extrudedPart(t)
	minX0 := float64(def.SurfaceBodies().Item(0).RangeBox().Min.X)

	tool := NewMoveBodyTool()
	tool.dx = 10
	s.StartTool(tool)
	s.feedPick(BodyHandle{Body: def.SurfaceBodies().Item(0)})
	if err := s.OK(); err != nil {
		t.Fatalf("move body OK: %v", err)
	}
	minX1 := float64(def.SurfaceBodies().Item(0).RangeBox().Min.X)
	if math.Abs(minX1-(minX0+10)) > 1e-6 {
		t.Fatalf("body min.X = %g, want %g (shifted by 10)", minX1, minX0+10)
	}
}

// TestModifyToolsDraftFeature asserts Move Face, Combine and Move Bodies build the draft
// the commit gate inspects (#1626): no draft before the tool is commit-ready, a non-nil
// draft once it is. Combine and Move resolve their body operands from the session at
// draft time, exactly as Commit does.
func TestModifyToolsDraftFeature(t *testing.T) {
	s, def, src := extrudedPart(t)
	pat := NewFeatureRectPatternTool() // default 2×1 → a second body for Combine
	s.StartTool(pat)
	s.feedPick(src)
	if err := s.OK(); err != nil {
		t.Fatalf("pattern setup OK: %v", err)
	}

	moveFace := NewMoveFaceTool()
	moveFace.Pick(s, FaceHandle{Face: topFaceOf(t, def.SurfaceBodies().Item(0)), Body: def.SurfaceBodies().Item(0)})
	if _, ok := moveFace.DraftFeature(s); ok {
		t.Error("move face: draft ready with a zero move vector")
	}
	moveFace.dz = 2
	if draft, ok := moveFace.DraftFeature(s); !ok || draft == nil {
		t.Errorf("move face: no draft once commit-ready (ok=%v)", ok)
	}

	combine := NewCombineTool()
	combine.Pick(s, BodyHandle{Body: def.SurfaceBodies().Item(0)})
	if _, ok := combine.DraftFeature(s); ok {
		t.Error("combine: draft ready with one body picked")
	}
	combine.Pick(s, BodyHandle{Body: def.SurfaceBodies().Item(1)})
	if draft, ok := combine.DraftFeature(s); !ok || draft == nil {
		t.Errorf("combine: no draft once commit-ready (ok=%v)", ok)
	}

	moveBody := NewMoveBodyTool()
	moveBody.Pick(s, BodyHandle{Body: def.SurfaceBodies().Item(0)})
	if _, ok := moveBody.DraftFeature(s); ok {
		t.Error("move bodies: draft ready with a zero move vector")
	}
	moveBody.dx = 10
	if draft, ok := moveBody.DraftFeature(s); !ok || draft == nil {
		t.Errorf("move bodies: no draft once commit-ready (ok=%v)", ok)
	}
}

func TestDirectEditCommandsRegistered(t *testing.T) {
	s := NewSession()
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	for _, id := range []string{"Modify.Combine", "Modify.MoveFace", "Modify.MoveBodies"} {
		if _, ok := s.Commands().ByID(id); !ok {
			t.Errorf("command %q not registered", id)
		}
	}
}
