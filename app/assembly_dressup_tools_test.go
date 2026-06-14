// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/occurrence"
)

// worldVerticalEdge returns an EdgeHandle for a vertical edge of the assembly's (single) placed
// component body — the kind of pick the viewport produces under a SelectEdge filter (#769). The
// edge carries the world body's occurrence-lineage prefix, which the chamfer tool must strip.
func worldVerticalEdge(t *testing.T, s *Session) EdgeHandle {
	t.Helper()
	bodies := s.VisibleBodies()
	if len(bodies) == 0 {
		t.Fatal("assembly has no placed body to pick an edge on")
	}
	for _, e := range bodies[0].Edges() {
		a, b := e.StartVertex().Point(), e.EndVertex().Point()
		if a.X == b.X && a.Y == b.Y && a.Z != b.Z { // vertical: same XY, differing Z
			return EdgeHandle{Edge: e}
		}
	}
	t.Fatal("no vertical edge on the placed box")
	return EdgeHandle{}
}

// participantMachinedVolume sums the assembly-feature machined result volume for an occurrence —
// the volume the participant's body has AFTER the dress-up features run.
func participantMachinedVolume(asm *compdef.AssemblyComponentDefinition, o *occurrence.Occurrence) float64 {
	var v float64
	for _, b := range asm.Features().Result(o) {
		v += ops.BodyGeometryProperties(b, ops.DefaultQuality()).Volume
	}
	return v
}

// TestAssemblyChamferMachinesParticipant is the headline #766 test: picking a vertical edge of a
// placed component (a world-space body carrying the occurrence-lineage prefix) and chamfering it
// removes the analytic wedge from the participant — proving the picked world edge resolves to the
// right component-local edge (a wrong suffix would no-op, leaving the volume unchanged).
func TestAssemblyChamferMachinesParticipant(t *testing.T) {
	s, asm, occ := assemblyWithBoxComponent(t, 0) // box [0,2]×[0,2]×[0,4], volume 16

	tool := NewAssemblyChamferTool()
	tool.Pick(s, worldVerticalEdge(t, s))
	tool.distance = 0.2
	if !tool.CanCommit() {
		t.Fatal("a chamfer with an edge and a positive distance should be committable")
	}
	if err := tool.Commit(s); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if asm.Features().Count() != 1 || asm.Features().Item(0).Kind() != "assemblyChamfer" {
		t.Fatalf("features = %d, want one assemblyChamfer", asm.Features().Count())
	}
	// A 45° flat chamfer of setback 0.2 on a length-4 vertical edge removes a 0.2²/2·4 = 0.08 wedge.
	if got := participantMachinedVolume(asm, occ); stdmath.Abs(got-15.92) > 0.02 {
		t.Errorf("chamfered participant volume = %g, want 15.92 (16 minus a 0.2 chamfer wedge)", got)
	}
}

// TestAssemblyChamferAppliesToAllInstances is the occurrence-relative proof (#735/#766): picking
// one edge on one placed instance chamfers that edge on EVERY instance of the component — a single
// feature, resolved per participant by the component-local suffix.
func TestAssemblyChamferAppliesToAllInstances(t *testing.T) {
	s, asm, occ1 := assemblyWithBoxComponent(t, 0)
	box := s.Workspace().Documents()[0] // the box component document
	occ2, err := asm.PlaceComponentFromFile(s.ActiveDocument(), box, "box:2", math.Translation4(math.V3(20, 0, 0)))
	if err != nil {
		t.Fatalf("place second instance: %v", err)
	}

	tool := NewAssemblyChamferTool()
	tool.Pick(s, worldVerticalEdge(t, s)) // one edge on the first instance
	tool.distance = 0.2
	if err := tool.Commit(s); err != nil {
		t.Fatalf("commit: %v", err)
	}
	for i, occ := range []*occurrence.Occurrence{occ1, occ2} {
		if got := participantMachinedVolume(asm, occ); stdmath.Abs(got-15.92) > 0.02 {
			t.Errorf("instance %d volume = %g, want 15.92 — both instances chamfered from one pick", i, got)
		}
	}
}

// TestAssemblyFilletMachinesParticipant: rounding a component edge removes the convex corner
// material from the participant (volume drops below the box, by a small rounded amount).
func TestAssemblyFilletMachinesParticipant(t *testing.T) {
	s, asm, occ := assemblyWithBoxComponent(t, 0)

	tool := NewAssemblyFilletTool()
	tool.Pick(s, worldVerticalEdge(t, s))
	tool.radius = 0.2
	if err := tool.Commit(s); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if asm.Features().Item(0).Kind() != "assemblyFillet" {
		t.Errorf("feature kind = %q, want assemblyFillet", asm.Features().Item(0).Kind())
	}
	// A radius-0.2 fillet on a length-4 edge removes (1 − π/4)·0.2²·4 ≈ 0.034 of corner material.
	got := participantMachinedVolume(asm, occ)
	if got >= 16 || got < 15.9 {
		t.Errorf("filleted participant volume = %g, want a box minus a small rounded corner (≈15.97)", got)
	}
}

// TestAssemblyDressupNeedsEdge: committing with nothing picked errors rather than adding an empty
// feature.
func TestAssemblyDressupNeedsEdge(t *testing.T) {
	s, _, _ := assemblyWithBoxComponent(t, 0)
	if NewAssemblyChamferTool().CanCommit() {
		t.Error("a chamfer with no edge picked should not be committable")
	}
	if err := NewAssemblyChamferTool().Commit(s); err == nil {
		t.Error("committing a chamfer with no edge should error")
	}
}

// TestComponentEdgeSuffixStripsOccurrencePrefix pins the resolution helper: a world key
// [kind]+"occurrence:occ#3/extrude:edge#2" yields the bare component suffix "extrude:edge#2".
func TestComponentEdgeSuffixStripsOccurrencePrefix(t *testing.T) {
	worldKey := append([]byte{byte(topo.KindEdge)}, []byte("occurrence:occ#3/extrude:edge#2")...)
	if got := string(componentEdgeSuffix(worldKey)); got != "extrude:edge#2" {
		t.Errorf("componentEdgeSuffix = %q, want extrude:edge#2", got)
	}
}
