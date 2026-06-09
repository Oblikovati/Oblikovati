// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// patchRect adds a boundary-patch surface over the rectangle [x0,x1]×[0,2] on XY.
func patchRect(def *compdef.PartComponentDefinition, x0, x1 float64) {
	sk := def.Sketches().Add(sketch.XYPlane())
	a := sk.Points().Add(math.P2(math.Scalar(x0), 0))
	b := sk.Points().Add(math.P2(math.Scalar(x1), 0))
	c := sk.Points().Add(math.P2(math.Scalar(x1), 2))
	d := sk.Points().Add(math.P2(math.Scalar(x0), 2))
	sk.Lines().Add(a, b)
	sk.Lines().Add(b, c)
	sk.Lines().Add(c, d)
	sk.Lines().Add(d, a)
	feature.NewBoundaryPatchFeatures(def.Features()).Add(sk, 0, feature.PatchFree)
}

// twoAdjacentPatches gives a part with two coplanar patches sharing the x=2 edge.
func twoAdjacentPatches(t *testing.T) (*Session, *compdef.PartComponentDefinition) {
	t.Helper()
	s, def := emptyPartSession(t)
	patchRect(def, 0, 2)
	patchRect(def, 2, 4)
	def.Recompute()
	if def.SurfaceBodies().Count() != 2 {
		t.Fatalf("setup: %d surface bodies, want 2", def.SurfaceBodies().Count())
	}
	return s, def
}

// TestStitchToolEndToEnd drives the Stitch UI: with two adjacent patches present, OK welds them
// into one surface quilt (two faces, shared edge no longer a boundary).
func TestStitchToolEndToEnd(t *testing.T) {
	s, def := twoAdjacentPatches(t)

	s.StartTool(NewStitchTool())
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("after stitch: %d bodies, want 1 (one quilt)", def.SurfaceBodies().Count())
	}
	if n := len(def.SurfaceBodies().Item(0).Faces()); n != 2 {
		t.Errorf("stitched quilt has %d faces, want 2", n)
	}
}

func TestStitchViaRibbonCommand(t *testing.T) {
	s, _ := twoAdjacentPatches(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Surface.Stitch"); err != nil {
		t.Fatalf("execute Surface.Stitch: %v", err)
	}
	if _, ok := s.ActiveTool().Tool().(*StitchTool); !ok {
		t.Fatal("Stitch command did not start the stitch tool")
	}
}
