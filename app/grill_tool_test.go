// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// partWithGrillSketch builds a 6×6×2 block and a grill sketch on its top face: a 4×4 boundary
// (area 16) bridged by two 0.5-wide ribs drawn as inner rectangles (total 3) ⇒ one profile of
// area 13.
func partWithGrillSketch(t *testing.T) (*Session, *compdef.PartComponentDefinition, ProfileHandle) {
	t.Helper()
	s, _ := newPartWithBlock(t, 6) // block on XY, z 0..2, vol 72
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	top, _ := sketch.NewPlane(math.P3(0, 0, 2), math.V3(1, 0, 0).AsUnit(), math.V3(0, 1, 0).AsUnit())
	es := def.Sketches().Add(top)
	rectOn(es, 1, 1, 5, 5)           // boundary 4×4
	rectOn(es, 2.25, 1.5, 2.75, 4.5) // rib 1 (0.5 × 3)
	rectOn(es, 3.25, 1.5, 3.75, 4.5) // rib 2 (0.5 × 3)
	return s, def, ProfileHandle{Sketch: es, ProfileIndex: 0}
}

// rectOn adds a closed axis-aligned rectangle [x0,x1]×[y0,y1] to a sketch.
func rectOn(sk *sketch.Sketch, x0, y0, x1, y1 float64) {
	c0 := sk.Points().Add(math.P2(x0, y0))
	c1 := sk.Points().Add(math.P2(x1, y0))
	c2 := sk.Points().Add(math.P2(x1, y1))
	c3 := sk.Points().Add(math.P2(x0, y1))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
}

// TestGrillToolEndToEnd drives the Grill UI: pick the boundary, OK — and asserts the vent cut
// through the block left the ribs (block 72 − 13×2 = 46).
func TestGrillToolEndToEnd(t *testing.T) {
	t.Parallel()
	s, def, region := partWithGrillSketch(t)
	s.SetPicker(stubPicker{sel: region})

	g := NewGrillTool()
	s.StartTool(g)
	s.Click(100, 100)
	if !g.CanCommit() {
		t.Fatal("grill tool not ready after picking a boundary")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	body := def.SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("grill body not a valid solid: %+v", r)
	}
	if v := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErrApp(v, 46) > 0.01 {
		t.Errorf("grill volume = %g, want ≈46 (72 − 13×2 vent)", v)
	}
}

// TestGrillViaRibbonCommand confirms the ribbon command starts the grill tool.
func TestGrillViaRibbonCommand(t *testing.T) {
	t.Parallel()
	s, _, _ := partWithGrillSketch(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Create.Grill"); err != nil {
		t.Fatalf("execute Create.Grill: %v", err)
	}
	if _, ok := s.ActiveTool().Tool().(*GrillTool); !ok {
		t.Fatal("Grill command did not start the grill tool")
	}
}
