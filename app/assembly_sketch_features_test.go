// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// TestCreateSketchOnAssembly: the sketch environment now opens on an assembly — CreateSketch adds
// the sketch to the assembly and enters the editing environment (#766).
func TestCreateSketchOnAssembly(t *testing.T) {
	s := assemblySession(t)
	asm := s.ActiveDocument().Content().(*compdef.AssemblyComponentDefinition)

	sk, err := s.CreateSketch(sketch.XYPlane())
	if err != nil {
		t.Fatalf("CreateSketch on an assembly: %v", err)
	}
	if !s.InSketch() {
		t.Error("CreateSketch should enter the sketch environment")
	}
	if asm.Sketches().Count() != 1 || asm.Sketches().Item(0) != sk {
		t.Error("the sketch should be added to the assembly")
	}
}

// TestAssemblySketchCommandsOnAssemblyRibbon: the Assemble tab offers Create 2D Sketch, and the
// contextual Sketch tab's tools appear on the assembly ribbon while editing an assembly sketch.
func TestAssemblySketchCommandsOnAssemblyRibbon(t *testing.T) {
	s := assemblySession(t)
	if cmd, ok := s.Commands().ByID("Assembly.CreateSketch"); !ok || !cmd.IsEnabled(s) {
		t.Fatal("the Assemble tab should offer an enabled Create 2D Sketch")
	}
	if _, err := s.CreateSketch(sketch.XYPlane()); err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}
	tab, ok := BuildRibbon(s).Tab("Sketch")
	if !ok {
		t.Fatal("the contextual Sketch tab should appear on the assembly ribbon while sketching")
	}
	if _, ok := tab.Panel("Create"); !ok {
		t.Error("the assembly Sketch tab has no Create panel (Line/Rectangle/…)")
	}
}

// TestAssemblyExtrudeMachinesFromSketch is the end-to-end #766 sketch path: author a sketch on an
// assembly, extrude-cut its profile across a placed component, and confirm an assemblyExtrude
// feature machined the participant (its volume dropped).
func TestAssemblyExtrudeMachinesFromSketch(t *testing.T) {
	s, asm, occ := assemblyWithBoxComponent(t, 0) // box [0,2]×[0,2]×[0,4], assembly active

	sk, err := s.CreateSketch(sketch.XYPlane())
	if err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}
	sk.AddRectangleByCorners(math.P2(0.5, 0.5), math.P2(1.5, 1.5)) // a 1×1 window inside the box footprint
	if err := s.FinishSketch(); err != nil {
		t.Fatalf("FinishSketch: %v", err)
	}
	if sk.Profiles().Count() == 0 {
		t.Fatal("the finished assembly sketch has no profile to extrude")
	}

	tool := NewAssemblyExtrudeTool()
	tool.Pick(s, ProfileHandle{Sketch: sk, ProfileIndex: 0})
	tool.distance = 4  // through the box height
	tool.operation = 0 // Cut
	if !tool.CanCommit() {
		t.Fatal("an extrude with a profile and a positive distance should be committable")
	}
	if err := tool.Commit(s); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if asm.Features().Count() != 1 || asm.Features().Item(0).Kind() != "assemblyExtrude" {
		t.Fatalf("features = %d, want one assemblyExtrude", asm.Features().Count())
	}
	// A 1×1 cut through the 16-unit box removes 1×1×4 = 4 ⇒ 12.
	if got := participantMachinedVolume(asm, occ); stdmath.Abs(got-12) > 0.1 {
		t.Errorf("extrude-cut participant volume = %g, want 12 (16 minus a 1×1×4 pocket)", got)
	}
}
