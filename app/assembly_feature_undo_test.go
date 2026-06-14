// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// TestAssemblyChamferIsUndoable: now that the assembly feature program serializes (#785), adding a
// chamfer is one undo step — undo removes it (the recipe restores without it, then re-binds), redo
// restores it. This exercises the full RestoreRecipe → ApplyRecipe → ResolveReferences path.
func TestAssemblyChamferIsUndoable(t *testing.T) {
	s, asm, _ := assemblyWithBoxComponent(t, 0)
	trackFromHere(s) // baseline: the box placed, no features

	tool := NewAssemblyChamferTool()
	tool.Pick(s, worldVerticalEdge(t, s))
	tool.distance = 0.2
	if err := tool.Commit(s); err != nil {
		t.Fatalf("chamfer: %v", err)
	}
	if asm.Features().Count() != 1 {
		t.Fatalf("after chamfer: feature count = %d, want 1", asm.Features().Count())
	}
	if err := s.Undo(); err != nil {
		t.Fatalf("undo: %v", err)
	}
	if asm.Features().Count() != 0 {
		t.Errorf("undo should remove the chamfer: feature count = %d, want 0", asm.Features().Count())
	}
	if err := s.Redo(); err != nil {
		t.Fatalf("redo: %v", err)
	}
	if asm.Features().Count() != 1 || asm.Features().Item(0).Kind() != "assemblyChamfer" {
		t.Errorf("redo should restore the chamfer: features = %d", asm.Features().Count())
	}
}

// TestAssemblyExtrudeIsUndoable: a sketch-based extrude undoes too — both the sketch and the
// extrude round-trip through the recipe.
func TestAssemblyExtrudeIsUndoable(t *testing.T) {
	s, asm, occ := assemblyWithBoxComponent(t, 0)
	trackFromHere(s) // baseline: the box, no sketch, no features

	sk, err := s.CreateSketch(sketch.XYPlane())
	if err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}
	sk.AddRectangleByCorners(math.P2(0.5, 0.5), math.P2(1.5, 1.5))
	if err := s.FinishSketch(); err != nil { // records the sketch step
		t.Fatalf("FinishSketch: %v", err)
	}
	tool := NewAssemblyExtrudeTool()
	tool.Pick(s, ProfileHandle{Sketch: sk, ProfileIndex: 0})
	tool.distance = 4
	if err := tool.Commit(s); err != nil { // records the extrude step
		t.Fatalf("extrude: %v", err)
	}
	if got := participantMachinedVolume(asm, occ); got >= 16 {
		t.Fatalf("the extrude did not machine the participant: volume %g", got)
	}

	if err := s.Undo(); err != nil { // undo the extrude
		t.Fatalf("undo: %v", err)
	}
	if asm.Features().Count() != 0 {
		t.Errorf("undo should remove the extrude: feature count = %d, want 0", asm.Features().Count())
	}
	// The sketch survives the extrude's undo (it was created in a prior step).
	if asm.Sketches().Count() != 1 {
		t.Errorf("the sketch should survive the extrude undo: sketch count = %d, want 1", asm.Sketches().Count())
	}
}
