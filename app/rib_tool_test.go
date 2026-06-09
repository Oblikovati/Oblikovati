// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// ribbedPart builds a 6×6×2 block and adds a sketch with an open line within the footprint
// (the rib profile), returning the session and part.
func ribbedPart(t *testing.T) (*Session, *compdef.PartComponentDefinition) {
	t.Helper()
	s, _ := newPartWithBlock(t, 6) // block on XY, z 0..2
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	rs := def.Sketches().Add(sketch.XYPlane())
	rs.Lines().AddByTwoPoints(math.P2(1, 3), math.P2(5, 3)) // open path inside the block
	return s, def
}

// TestRibToolEndToEnd drives the Rib UI: with an open profile sketched, start the tool, set
// thickness and depth, OK — and asserts a valid solid that joined the rib to the block.
func TestRibToolEndToEnd(t *testing.T) {
	s, def := ribbedPart(t)

	rib := NewRibTool()
	s.StartTool(rib) // resolves the open profile at Start
	rib.SetThickness(1)
	rib.SetDepth(3) // up through and above the block → joins
	if !rib.CanCommit() {
		t.Fatal("rib tool not ready (profile + thickness + depth)")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("after rib: %d bodies, want 1 (joined)", def.SurfaceBodies().Count())
	}
	body := def.SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("ribbed body not a valid solid: %+v", r)
	}
	if !rib.AddedFeature().Health().OK() {
		t.Fatalf("rib feature sick: %+v", rib.AddedFeature().Health())
	}
}

func TestRibViaRibbonCommand(t *testing.T) {
	s, _ := ribbedPart(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Create.Rib"); err != nil { // ribbon: click the Rib button
		t.Fatalf("execute Create.Rib: %v", err)
	}
	if _, ok := s.ActiveTool().Tool().(*RibTool); !ok {
		t.Fatal("Rib command did not start the rib tool")
	}
}
