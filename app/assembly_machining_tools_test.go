// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// TestAssemblyRevolveMachinesFromSketch: revolving a sketch profile about an axis adds an
// assemblyRevolve feature that machines the participant (its volume drops — a cut removes the
// swept material) — the revolve half of the assembly sketched features (#766).
func TestAssemblyRevolveMachinesFromSketch(t *testing.T) {
	t.Parallel()
	s, asm, occ := assemblyWithBoxComponent(t, 0) // box [0,2]×[0,2]×[0,4]

	sk, err := s.CreateSketch(sketch.XYPlane())
	if err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}
	sk.AddRectangleByCorners(math.P2(0.5, 0.5), math.P2(1.5, 1.5))
	if err := s.FinishSketch(); err != nil {
		t.Fatalf("FinishSketch: %v", err)
	}

	tool := NewAssemblyRevolveTool() // full turn, +Y axis, Cut
	tool.Pick(s, ProfileHandle{Sketch: sk, ProfileIndex: 0})
	if !tool.CanCommit() {
		t.Fatal("a revolve with a profile and a full-turn angle should be committable")
	}
	if err := tool.Commit(s); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if asm.Features().Item(0).Kind() != "assemblyRevolve" {
		t.Fatalf("feature kind = %q, want assemblyRevolve", asm.Features().Item(0).Kind())
	}
	if got := participantMachinedVolume(asm, occ); got <= 0 || got >= 16 {
		t.Errorf("revolved participant volume = %g, want machined material removed (0 < v < 16)", got)
	}
}

// TestAssemblyHoleDrillsParticipant: a parametric hole drills a bore through the participant,
// removing the cylindrical material (volume drops below the box by a small bore).
func TestAssemblyHoleDrillsParticipant(t *testing.T) {
	t.Parallel()
	s, asm, occ := assemblyWithBoxComponent(t, 0) // box [0,2]×[0,2]×[0,4]

	tool := NewAssemblyHoleTool() // drills -Z, d=1, depth=1 by default
	tool.cx, tool.cy, tool.cz = 1, 1, 4
	tool.diameter, tool.depth = 0.5, 2 // a thin bore from the top face down 2
	if !tool.CanCommit() {
		t.Fatal("a hole with positive diameter and depth should be committable")
	}
	if err := tool.Commit(s); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if asm.Features().Item(0).Kind() != "assemblyHole" {
		t.Fatalf("feature kind = %q, want assemblyHole", asm.Features().Item(0).Kind())
	}
	// A Ø0.5 × 2 bore removes ~π·0.25²·2 ≈ 0.39, so the box (16) loses a small amount.
	if got := participantMachinedVolume(asm, occ); got >= 16 || got < 15 {
		t.Errorf("holed participant volume = %g, want a box minus a thin bore (≈15.6)", got)
	}
}

// TestAssemblyMachiningToolsNeedInput: the tools refuse to commit without their inputs.
func TestAssemblyMachiningToolsNeedInput(t *testing.T) {
	t.Parallel()
	if NewAssemblyRevolveTool().CanCommit() {
		t.Error("a revolve with no profile should not be committable")
	}
	bad := NewAssemblyHoleTool()
	bad.diameter = 0
	if bad.CanCommit() {
		t.Error("a hole with zero diameter should not be committable")
	}
}
