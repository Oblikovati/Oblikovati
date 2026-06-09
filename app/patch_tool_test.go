// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// partWithSquareRegion gives an empty part with a 4×4 closed region sketched on XY.
func partWithSquareRegion(t *testing.T) (*Session, *compdef.PartComponentDefinition, ProfileHandle) {
	t.Helper()
	s, def := emptyPartSession(t)
	sk := def.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(0, 0))
	c1 := sk.Points().Add(math.P2(4, 0))
	c2 := sk.Points().Add(math.P2(4, 4))
	c3 := sk.Points().Add(math.P2(0, 4))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	return s, def, ProfileHandle{Sketch: sk, ProfileIndex: 0}
}

// TestPatchToolEndToEnd drives the Patch UI: click a closed region — it auto-commits a surface
// (a one-face sheet body, not a solid).
func TestPatchToolEndToEnd(t *testing.T) {
	s, def, region := partWithSquareRegion(t)
	s.SetPicker(stubPicker{sel: region})

	s.StartTool(NewPatchTool())
	s.Click(1, 1) // pick the region → auto-commit

	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("after patch: %d bodies, want 1", def.SurfaceBodies().Count())
	}
	body := def.SurfaceBodies().Item(0)
	if body.IsSolid() {
		t.Error("patch should be a surface (sheet) body, not a solid")
	}
	if n := len(body.Faces()); n != 1 {
		t.Errorf("patch has %d faces, want 1", n)
	}
	if s.ActiveTool() != nil {
		t.Error("patch should auto-commit and close the tool on pick")
	}
}

func TestPatchViaRibbonCommand(t *testing.T) {
	s, _, region := partWithSquareRegion(t)
	s.SetPicker(stubPicker{sel: region})
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Surface.Patch"); err != nil {
		t.Fatalf("execute Surface.Patch: %v", err)
	}
	if _, ok := s.ActiveTool().Tool().(*PatchTool); !ok {
		t.Fatal("Patch command did not start the patch tool")
	}
}
