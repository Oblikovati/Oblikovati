// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/compdef"
	"github.com/Oblikovati/oblikovati/model/feature"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// patchedPartWithCutPlane gives a part holding a 4×4 surface patch (on XY) and a work plane at
// x=2 (YZ offset 2, normal +X) to trim it with.
func patchedPartWithCutPlane(t *testing.T) (*Session, *compdef.PartComponentDefinition, *feature.WorkPlane) {
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
	feature.NewBoundaryPatchFeatures(def.Features()).Add(sk, 0, feature.PatchFree)
	wp := def.WorkPlanes().AddByPlaneAndOffset(feature.OriginYZPlane, func() float64 { return 2 })
	def.Recompute()
	return s, def, wp
}

// TestSurfaceTrimToolEndToEnd drives the Trim UI: pick the x=2 plane, keep the +X side, OK — and
// asserts the patch is trimmed to x∈[2,4].
func TestSurfaceTrimToolEndToEnd(t *testing.T) {
	s, def, wp := patchedPartWithCutPlane(t)

	trim := NewSurfaceTrimTool()
	s.StartTool(trim)
	trim.Pick(s, WorkPlaneHandle{Plane: wp})
	if !trim.CanCommit() {
		t.Fatal("trim tool not ready after picking a plane")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	box := def.SurfaceBodies().Item(0).RangeBox()
	if stdmath.Abs(float64(box.Min.X)-2) > 1e-6 || stdmath.Abs(float64(box.Max.X)-4) > 1e-6 {
		t.Errorf("trimmed x-span = [%v,%v], want [2,4]", box.Min.X, box.Max.X)
	}
}

func TestSurfaceTrimViaRibbonCommand(t *testing.T) {
	s, _, _ := patchedPartWithCutPlane(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Surface.Trim"); err != nil {
		t.Fatalf("execute Surface.Trim: %v", err)
	}
	if _, ok := s.ActiveTool().Tool().(*SurfaceTrimTool); !ok {
		t.Fatal("Trim command did not start the surface trim tool")
	}
}
