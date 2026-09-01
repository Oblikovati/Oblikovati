// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/compdef"
)

// TestCoreCavityToolEndToEnd parts a 6×6×2 block at z=1: two valid solids (core below,
// cavity above) replace the block.
func TestCoreCavityToolEndToEnd(t *testing.T) {
	t.Parallel()
	s, _ := newPartWithBlock(t, 6)
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)

	tool := NewCoreCavityTool()
	s.StartTool(tool)
	if !tool.CanCommit() {
		t.Fatal("core/cavity tool should commit with its defaults")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if def.SurfaceBodies().Count() != 2 {
		t.Fatalf("after core/cavity: %d bodies, want 2", def.SurfaceBodies().Count())
	}
	for i := range 2 {
		b := def.SurfaceBodies().Item(i)
		if r := ops.Validate(b); !r.Valid || !b.IsSolid() {
			t.Fatalf("mold half %d not a valid solid: %+v", i, r)
		}
	}
}

// A parting outside the block must surface the model error through the tool.
func TestCoreCavityToolRejectsPartingOutsideBlock(t *testing.T) {
	t.Parallel()
	s, _ := newPartWithBlock(t, 6) // block z 0..2

	tool := NewCoreCavityTool()
	s.StartTool(tool)
	tool.position = 5
	if err := s.OK(); err == nil {
		t.Fatal("a parting above the block must error")
	}
}

// TestCoreCavityViaRibbonCommand asserts the Mold panel command starts the tool.
func TestCoreCavityViaRibbonCommand(t *testing.T) {
	t.Parallel()
	s, _ := newPartWithBlock(t, 6)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Mold.CoreCavity"); err != nil {
		t.Fatalf("execute Mold.CoreCavity: %v", err)
	}
	if _, ok := s.ActiveTool().Tool().(*CoreCavityTool); !ok {
		t.Fatal("Core/Cavity command did not start the core/cavity tool")
	}
}
