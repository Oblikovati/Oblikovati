// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/compdef"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// partWithTopRegion builds a 6×6×2 block and a 2×2 closed region sketched on its top face.
func partWithTopRegion(t *testing.T) (*Session, *compdef.PartComponentDefinition, ProfileHandle) {
	t.Helper()
	s, _ := newPartWithBlock(t, 6) // block on XY, z 0..2, vol 72
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	top, _ := sketch.NewPlane(math.P3(0, 0, 2), math.V3(1, 0, 0).AsUnit(), math.V3(0, 1, 0).AsUnit())
	es := def.Sketches().Add(top)
	c0 := es.Points().Add(math.P2(2, 2))
	c1 := es.Points().Add(math.P2(4, 2))
	c2 := es.Points().Add(math.P2(4, 4))
	c3 := es.Points().Add(math.P2(2, 4))
	es.Lines().Add(c0, c1)
	es.Lines().Add(c1, c2)
	es.Lines().Add(c2, c3)
	es.Lines().Add(c3, c0)
	return s, def, ProfileHandle{Sketch: es, ProfileIndex: 0}
}

// TestEmbossToolEndToEnd drives the Emboss UI: pick a region, set a depth, OK — and asserts the
// raised emboss added material (block 72 + 2×2×1 = 76).
func TestEmbossToolEndToEnd(t *testing.T) {
	s, def, region := partWithTopRegion(t)
	s.SetPicker(stubPicker{sel: region})

	emb := NewEmbossTool()
	s.StartTool(emb)
	s.Click(100, 100)
	emb.SetDepth(1)
	if !emb.CanCommit() {
		t.Fatal("emboss tool not ready after picking a region + depth")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	body := def.SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("embossed body not a valid solid: %+v", r)
	}
	if v := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErrApp(v, 76) > 0.01 {
		t.Errorf("embossed volume = %g, want ≈76 (72 + 2×2×1)", v)
	}
}

// Engrave mode cuts material (block 72 − 2×2×1 = 68).
func TestEmbossToolEngraves(t *testing.T) {
	s, def, region := partWithTopRegion(t)
	s.SetPicker(stubPicker{sel: region})

	emb := NewEmbossTool()
	s.StartTool(emb)
	s.Click(100, 100)
	emb.SetDepth(1)
	emb.SetEngrave(true)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if v := ops.BodyGeometryProperties(def.SurfaceBodies().Item(0), ops.DefaultQuality()).Volume; relErrApp(v, 68) > 0.01 {
		t.Errorf("engraved volume = %g, want ≈68 (72 − 2×2×1)", v)
	}
}

func TestEmbossViaRibbonCommand(t *testing.T) {
	s, _, _ := partWithTopRegion(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Create.Emboss"); err != nil {
		t.Fatalf("execute Create.Emboss: %v", err)
	}
	if _, ok := s.ActiveTool().Tool().(*EmbossTool); !ok {
		t.Fatal("Emboss command did not start the emboss tool")
	}
}
