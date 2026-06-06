// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"math"
	"testing"

	"oblikovati/kernel/ops"
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
