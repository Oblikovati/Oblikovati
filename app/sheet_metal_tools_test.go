// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"math"
	"testing"

	"oblikovati.org/kernel/topo"
	gmath "oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// sheetMetalSession returns a session whose active part is in the sheet-metal environment.
func sheetMetalSession(t *testing.T) (*Session, *compdef.PartComponentDefinition) {
	t.Helper()
	s := NewSession()
	if _, err := compdef.AddPart(s.Workspace(), "Sheet1", true); err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	part, err := activePart(s)
	if err != nil {
		t.Fatalf("activePart: %v", err)
	}
	if _, err := part.EnableSheetMetal(); err != nil {
		t.Fatalf("EnableSheetMetal: %v", err)
	}
	return s, part
}

// squareProfile adds a side×side square sketch on XY and returns a profile handle for it.
func squareProfile(part *compdef.PartComponentDefinition, side float64) ProfileHandle {
	sk := part.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(gmath.P2(0, 0))
	c1 := sk.Points().Add(gmath.P2(side, 0))
	c2 := sk.Points().Add(gmath.P2(side, side))
	c3 := sk.Points().Add(gmath.P2(0, side))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	return ProfileHandle{Sketch: sk, ProfileIndex: 0}
}

// topXEdge returns a top X-aligned edge of the body — a valid edge to flange from.
func topXEdge(t *testing.T, body *topo.Body) *topo.Edge {
	t.Helper()
	var best *topo.Edge
	bestZ := math.Inf(-1)
	for _, e := range body.Edges() {
		a, b := e.StartVertex().Point(), e.EndVertex().Point()
		alongX := math.Abs(a.X-b.X) > 1e-6 && math.Abs(a.Y-b.Y) < 1e-6
		if alongX && a.Z > bestZ {
			best, bestZ = e, a.Z
		}
	}
	if best == nil {
		t.Fatal("no X-aligned edge on the body")
	}
	return best
}

// TestSheetMetalCommandsEnable the Sheet Metal commands all sit on the Sheet Metal tab and
// enable only on a sheet-metal part.
func TestSheetMetalCommandsEnable(t *testing.T) {
	s, _ := sheetMetalSession(t)
	if !hasActiveSheetMetalPart(s) {
		t.Error("hasActiveSheetMetalPart should be true with a sheet-metal part active")
	}
	if hasActiveSheetMetalPart(newSessionWithPart(t)) {
		t.Error("hasActiveSheetMetalPart should be false with an ordinary part active")
	}
	cmds := sheetMetalTabCommands()
	if len(cmds) != 19 {
		t.Errorf("Sheet Metal tab has %d commands, want 19", len(cmds))
	}
	for _, c := range cmds {
		if c.tab != "Sheet Metal" {
			t.Errorf("command %q is on tab %q, want Sheet Metal", c.displayName, c.tab)
		}
		// Convert is the entry point: enabled on an ORDINARY part, off once converted. Every
		// other command is the inverse — enabled only inside the environment.
		if c.id == "SheetMetal.Convert" {
			if c.IsEnabled(s) {
				t.Error("Convert should be disabled on an already sheet-metal part")
			}
			if !c.IsEnabled(newSessionWithPart(t)) {
				t.Error("Convert should be enabled on an ordinary part (the way in)")
			}
			continue
		}
		if !c.IsEnabled(s) {
			t.Errorf("command %q should be enabled on a sheet-metal part", c.displayName)
		}
		if c.IsEnabled(newSessionWithPart(t)) {
			t.Errorf("command %q should be disabled on an ordinary part", c.displayName)
		}
	}
}

// TestSheetMetalToolFlow a Face → Flange → Unfold → Refold flow through the tools yields a
// healthy part at each step — the core authoring path the ribbon drives.
func TestSheetMetalToolFlow(t *testing.T) {
	s, part := sheetMetalSession(t)

	face := NewSheetMetalFaceTool()
	face.Start(s)
	face.Pick(s, squareProfile(part, 4))
	if err := face.Commit(s); err != nil {
		t.Fatalf("Face: %v", err)
	}

	flange := NewSheetMetalFlangeTool()
	flange.Start(s)
	flange.Pick(s, EdgeHandle{Edge: topXEdge(t, part.Features().Result()[0])})
	flange.SetHeight(1)
	if err := flange.Commit(s); err != nil {
		t.Fatalf("Flange: %v", err)
	}

	unfold := NewSheetMetalUnfoldTool()
	if !unfold.CanCommit() {
		t.Fatal("Unfold should be ready to commit")
	}
	if err := unfold.Commit(s); err != nil {
		t.Fatalf("Unfold: %v", err)
	}
	if !unfold.AddedFeature().Health().OK() {
		t.Errorf("unfold unhealthy: %s", unfold.AddedFeature().Health().Reason)
	}

	refold := NewSheetMetalRefoldTool()
	if err := refold.Commit(s); err != nil {
		t.Fatalf("Refold: %v", err)
	}
	if !refold.AddedFeature().Health().OK() {
		t.Errorf("refold unhealthy: %s", refold.AddedFeature().Health().Reason)
	}
}

// TestSheetMetalHemTool the Hem tool folds a healthy hem on a base sheet edge.
func TestSheetMetalHemTool(t *testing.T) {
	s, part := sheetMetalSession(t)
	face := NewSheetMetalFaceTool()
	face.Start(s)
	face.Pick(s, squareProfile(part, 4))
	if err := face.Commit(s); err != nil {
		t.Fatalf("Face: %v", err)
	}
	hem := NewSheetMetalHemTool()
	hem.Start(s)
	hem.Pick(s, EdgeHandle{Edge: topXEdge(t, part.Features().Result()[0])})
	hem.SetLength(0.5)
	if err := hem.Commit(s); err != nil {
		t.Fatalf("Hem: %v", err)
	}
	if !hem.AddedFeature().Health().OK() {
		t.Errorf("hem unhealthy: %s", hem.AddedFeature().Health().Reason)
	}
}

// TestSheetMetalToolsRequireInput every tool's Commit errors before its input is gathered (no
// pick, or no developable flat) — so a half-finished tool never silently commits nothing.
func TestSheetMetalToolsRequireInput(t *testing.T) {
	makers := []func() Tool{
		func() Tool { return NewSheetMetalFaceTool() },
		func() Tool { return NewSheetMetalFlangeTool() },
		func() Tool { return NewSheetMetalHemTool() },
		func() Tool { return NewSheetMetalContourFlangeTool() },
		func() Tool { return NewSheetMetalLoftedFlangeTool() },
		func() Tool { return NewSheetMetalContourRollTool() },
		func() Tool { return NewSheetMetalBendTool() },
		func() Tool { return NewSheetMetalFoldTool() },
		func() Tool { return NewSheetMetalCornerTool() },
		func() Tool { return NewSheetMetalCornerSeamTool() },
		func() Tool { return NewSheetMetalCutTool() },
		func() Tool { return NewSheetMetalUnfoldTool() },
		func() Tool { return NewSheetMetalRefoldTool() },
	}
	for _, mk := range makers {
		s, _ := sheetMetalSession(t)
		tool := mk()
		tool.Start(s)
		if err := tool.Commit(s); err == nil {
			t.Errorf("%s should error before its input is gathered", tool.Name())
		}
		tool.Cancel(s)
	}
}

// TestSheetMetalFaceAndFlangeTools the Face tool thickens a profile into the base wall and the
// Flange tool folds a wall on a resulting edge — both committing healthy features.
func TestSheetMetalFaceAndFlangeTools(t *testing.T) {
	s, part := sheetMetalSession(t)

	face := NewSheetMetalFaceTool()
	face.Start(s)
	face.Pick(s, squareProfile(part, 4))
	if !face.CanCommit() {
		t.Fatal("Face tool should be ready to commit after a profile pick")
	}
	if err := face.Commit(s); err != nil {
		t.Fatalf("Face commit: %v", err)
	}
	if !face.AddedFeature().Health().OK() {
		t.Fatalf("base Face unhealthy: %s", face.AddedFeature().Health().Reason)
	}

	flange := NewSheetMetalFlangeTool()
	flange.Start(s)
	flange.Pick(s, EdgeHandle{Edge: topXEdge(t, part.Features().Result()[0])})
	flange.SetHeight(1)
	if !flange.CanCommit() {
		t.Fatal("Flange tool should be ready to commit after an edge pick")
	}
	if err := flange.Commit(s); err != nil {
		t.Fatalf("Flange commit: %v", err)
	}
	if !flange.AddedFeature().Health().OK() {
		t.Fatalf("flange unhealthy: %s", flange.AddedFeature().Health().Reason)
	}
}
