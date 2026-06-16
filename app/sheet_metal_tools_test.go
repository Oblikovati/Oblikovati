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

// TestSheetMetalCommandsEnable the Sheet Metal commands enable only on a sheet-metal part.
func TestSheetMetalCommandsEnable(t *testing.T) {
	s, _ := sheetMetalSession(t)
	if !hasActiveSheetMetalPart(s) {
		t.Error("hasActiveSheetMetalPart should be true with a sheet-metal part active")
	}
	if hasActiveSheetMetalPart(newSessionWithPart(t)) {
		t.Error("hasActiveSheetMetalPart should be false with an ordinary part active")
	}
	for _, c := range sheetMetalTabCommands() {
		if !c.IsEnabled(s) {
			t.Errorf("command %q should be enabled on a sheet-metal part", c.displayName)
		}
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
