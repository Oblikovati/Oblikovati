// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// M20 api-parity integration: the exact scenarios visually confirmed against the live
// renderer (the head/cmd/m20live driver), pinned headlessly so each feature's geometry
// stays valid. One per feature: F20 Move, F18 Pattern, F17 Bend, F16 WorkSurface.

// TestM20MoveOpsIntegration folds a rotate-then-slide composed move into one valid solid.
func TestM20MoveOpsIntegration(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(subd.ToBody(subd.Box(4, 3, 1), "block"))
	NewModifyFeatures(fs).AddMoveOps(0, []MoveOperation{
		RotateAboutLineOp(math.P3(0, 0, 0), math.V3(0, 0, 1), constFloat(stdmath.Pi/6)),
		AlongRayOp(math.V3(1, 0, 0), constFloat(3)),
	})
	fs.Recompute()
	if got := fs.Result(); len(got) != 1 || !ops.Validate(got[0]).Valid {
		t.Fatalf("moved part = %d bodies, want 1 valid", len(got))
	}
	if box := fs.Result()[0].RangeBox(); box.Min.X < 1 { // slid +3 along X (after a 30° turn)
		t.Errorf("moved block min X = %g, want shifted along +X", box.Min.X)
	}
}

// TestM20PatternBoundaryIntegration clips a 3×3 grid to the 6 occurrences inside the boundary.
func TestM20PatternBoundaryIntegration(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	src := NewBaseFeatures(fs).AddBase(subd.ToBody(subd.Box(1, 1, 1), "cell"))
	boundary, err := NewPatternBoundary(math.P3(0, 0, 0), math.V3(0, 0, 1),
		[]math.Point3{{X: -1, Y: -1}, {X: 7, Y: -1}, {X: 7, Y: 5}, {X: -1, Y: 5}}, types.IncludeByCentroid)
	if err != nil {
		t.Fatalf("NewPatternBoundary: %v", err)
	}
	rect := NewPatternFeatures(fs).AddRectangular([]ID{src.ID()},
		func() int { return 3 }, func() int { return 3 }, math.V3(3, 0, 0), math.V3(0, 3, 0))
	rect.Definition().Options = PatternOptions{Boundary: boundary}
	fs.Recompute()
	if got := len(fs.Result()); got != 6 {
		t.Errorf("clipped 3×3 pattern = %d bodies, want 6 (far row dropped by the boundary)", got)
	}
}

// TestM20BendIntegration folds a flat bar 90° into one valid solid that rises above the bar.
func TestM20BendIntegration(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(subd.ToBody(subd.Box(10, 3, 1), "bar"))
	bendSk := sketch.NewSketches().Add(planeAtZ(1)) // bend line on the top face
	bendSk.Lines().AddByTwoPoints(math.P2(5, 0), math.P2(5, 3))
	NewBendPartFeatures(fs).Add(&BendPartDefinition{
		Sketch: bendSk, LineIndex: 0, BendType: types.RadiusAndAngleBend,
		Radius: constFloat(1), Angle: constFloat(stdmath.Pi / 2),
	})
	fs.Recompute()
	if got := fs.Result(); len(got) != 1 || !ops.Validate(got[0]).Valid {
		t.Fatalf("bent part = %d bodies, want 1 valid", len(got))
	}
	if box := fs.Result()[0].RangeBox(); box.Max.Z < 4 {
		t.Errorf("bent bar top Z = %g, want the flange folded up (>4)", box.Max.Z)
	}
}

// TestM20WorkSurfaceIntegration produces a boundary-patch sheet (an open surface body, not a
// solid) — the kind gathered into the WorkSurfaces collection.
func TestM20WorkSurfaceIntegration(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	sk.AddRectangleByCorners(math.P2(0, 0), math.P2(4, 3))
	NewBoundaryPatchFeatures(fs).Add(sk, 0, PatchFree)
	fs.Recompute()
	if got := surfaceBodiesOf(fs.Result()); len(got) != 1 {
		t.Fatalf("work-surface part = %d sheet bodies, want 1", len(got))
	}
	if fs.Result()[0].IsSolid() {
		t.Error("boundary patch produced a solid, want an open surface body")
	}
}
