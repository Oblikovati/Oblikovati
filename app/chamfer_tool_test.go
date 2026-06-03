// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/model/compdef"
	"github.com/Oblikovati/oblikovati/model/feature"
)

// activePartDef returns the active document's part component definition.
func activePartDef(t *testing.T, s *Session) *compdef.PartComponentDefinition {
	t.Helper()
	return s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
}

// verticalEdgeOf returns one vertical edge handle of the block (start/end share X,Y).
func verticalEdgeOf(t *testing.T, b *topo.Body) EdgeHandle {
	t.Helper()
	for _, e := range b.Edges() {
		a, c := e.StartVertex().Point(), e.EndVertex().Point()
		if a.X == c.X && a.Y == c.Y {
			return EdgeHandle{Edge: e}
		}
	}
	t.Fatal("no vertical edge found")
	return EdgeHandle{}
}

// TestChamferToolEndToEnd drives the Chamfer UI: start the tool, click a vertical edge of
// a 2×2×2 block, set the distance, OK — and asserts the bevel removed the wedge volume.
func TestChamferToolEndToEnd(t *testing.T) {
	s, block := newPartWithBlock(t, 2) // 2×2×2, vol 8
	edge := verticalEdgeOf(t, block)
	s.SetPicker(stubPicker{sel: edge})

	ch := NewChamferTool()
	s.StartTool(ch)
	s.Click(50, 50)
	ch.SetDistance(0.5)
	if !ch.CanCommit() {
		t.Fatal("chamfer not ready after edge + distance")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}

	def := activePartDef(t, s)
	body := def.SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("chamfered body not a valid solid: %+v", r)
	}
	want := 8 - 0.5*0.5*0.5*2 // 8 − wedge (½·d²·length) = 7.75
	if got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErrApp(got, want) > 1e-6 {
		t.Errorf("chamfer volume = %g, want %g", got, want)
	}
	if s.ActiveTool() != nil {
		t.Error("tool should have closed after OK")
	}
}

func TestChamferViaRibbonCommand(t *testing.T) {
	s, block := newPartWithBlock(t, 2)
	s.SetPicker(stubPicker{sel: verticalEdgeOf(t, block)})
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Modify.Chamfer"); err != nil {
		t.Fatalf("execute Modify.Chamfer: %v", err)
	}
	if _, ok := s.ActiveTool().Tool().(*ChamferTool); !ok {
		t.Fatal("Chamfer command did not start the chamfer tool")
	}
	s.Click(1, 1)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	def := activePartDef(t, s)
	if v := ops.BodyGeometryProperties(def.SurfaceBodies().Item(0), ops.DefaultQuality()).Volume; v >= 8 {
		t.Errorf("chamfer did not remove material: volume %g, want < 8", v)
	}
}

// TestChamferToolSeedsCornerPreference checks the tool adopts the session's default
// corner treatment on Start, and that the chosen treatment reaches the committed feature.
func TestChamferToolSeedsCornerPreference(t *testing.T) {
	s, block := newPartWithBlock(t, 2)
	s.SetPicker(stubPicker{sel: verticalEdgeOf(t, block)})
	s.SetChamferFlatCorners(false)

	ch := NewChamferTool()
	s.StartTool(ch)
	if ch.FlatCorners() {
		t.Error("tool should have seeded pointy corners from the session preference")
	}
	s.Click(1, 1)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if def := ch.AddedFeature().Definition().(*feature.ChamferFeature).Definition(); def.FlatCorners {
		t.Error("committed chamfer should carry the pointy preference")
	}
}

func TestChamferToolNeedsEdge(t *testing.T) {
	s, block := newPartWithBlock(t, 2)
	s.SetPicker(stubPicker{sel: verticalEdgeOf(t, block)})
	ch := NewChamferTool()
	s.StartTool(ch)
	if ch.CanCommit() {
		t.Error("chamfer ready with no edge picked")
	}
	s.Click(0, 0)
	if !ch.CanCommit() {
		t.Error("chamfer not ready after picking an edge")
	}
}
