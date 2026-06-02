// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
)

func approx(a, b float64) bool { return a-b < 1e-9 && b-a < 1e-9 }

// boxFaces returns the six outward-oriented quad surface bodies of a box of the
// given dimensions, anchored at the origin (reusing quadBody from stitch_test.go).
func boxFaces(sx, sy, sz float64) []*topo.Body {
	p := math.P3
	return []*topo.Body{
		quadBody("bottom", p(0, 0, 0), p(0, sy, 0), p(sx, sy, 0), p(sx, 0, 0)),
		quadBody("top", p(0, 0, sz), p(sx, 0, sz), p(sx, sy, sz), p(0, sy, sz)),
		quadBody("front", p(0, 0, 0), p(sx, 0, 0), p(sx, 0, sz), p(0, 0, sz)),
		quadBody("back", p(0, sy, 0), p(0, sy, sz), p(sx, sy, sz), p(sx, sy, 0)),
		quadBody("left", p(0, 0, 0), p(0, 0, sz), p(0, sy, sz), p(0, sy, 0)),
		quadBody("right", p(sx, 0, 0), p(sx, sy, 0), p(sx, sy, sz), p(sx, 0, sz)),
	}
}

func TestTrimByPlaneKeepsHalf(t *testing.T) {
	patch := quadBody("p", math.P3(0, 0, 0), math.P3(4, 0, 0), math.P3(4, 4, 0), math.P3(0, 4, 0))
	trimmed, err := TrimByPlane(patch, math.P3(2, 0, 0), math.V3(1, 0, 0), true, "trim")
	if err != nil {
		t.Fatalf("TrimByPlane: %v", err)
	}
	if trimmed.IsSolid() || len(trimmed.Faces()) != 1 {
		t.Errorf("trim result solid=%v faces=%d, want surface/1", trimmed.IsSolid(), len(trimmed.Faces()))
	}
	box := trimmed.RangeBox()
	if !approx(box.Min.X, 2) || !approx(box.Max.X, 4) {
		t.Errorf("trimmed x-span = [%v,%v], want [2,4]", box.Min.X, box.Max.X)
	}
}

func TestTrimByPlaneEmptyErrors(t *testing.T) {
	patch := quadBody("p", math.P3(0, 0, 0), math.P3(4, 0, 0), math.P3(4, 4, 0), math.P3(0, 4, 0))
	// Keep the x ≥ 10 side — nothing remains.
	if _, err := TrimByPlane(patch, math.P3(10, 0, 0), math.V3(1, 0, 0), true, "trim"); err == nil {
		t.Error("trimming away the whole patch should error")
	}
}

func TestOffsetSurfaceMovesAlongNormal(t *testing.T) {
	patch := quadBody("p", math.P3(0, 0, 0), math.P3(4, 0, 0), math.P3(4, 4, 0), math.P3(0, 4, 0))
	offset, err := OffsetSurface(patch, 3, "offset")
	if err != nil {
		t.Fatalf("OffsetSurface: %v", err)
	}
	box := offset.RangeBox()
	if !approx(box.Min.Z, 3) || !approx(box.Max.Z, 3) {
		t.Errorf("offset z = [%v,%v], want flat at 3", box.Min.Z, box.Max.Z)
	}
}

func TestMidSurfacesThinBox(t *testing.T) {
	// A 4×4×1 thin plate: only the top/bottom faces (separation 1) are a thin pair.
	solid, _ := Stitch(boxFaces(4, 4, 1), 0, false, "box")
	patches, err := MidSurfaces(solid, 2, "mid")
	if err != nil {
		t.Fatalf("MidSurfaces: %v", err)
	}
	if len(patches) != 1 {
		t.Fatalf("got %d mid-surfaces, want 1 (the thin pair)", len(patches))
	}
	if !approx(patches[0].Thickness, 1) {
		t.Errorf("recorded thickness = %v, want 1", patches[0].Thickness)
	}
	// The mid patch sits on z = 0.5, midway between the caps.
	box := patches[0].Body.RangeBox()
	if !approx(box.Min.Z, 0.5) || !approx(box.Max.Z, 0.5) {
		t.Errorf("mid patch z = [%v,%v], want flat at 0.5", box.Min.Z, box.Max.Z)
	}
}

func TestMidSurfacesNoThinPairErrors(t *testing.T) {
	solid, _ := Stitch(boxFaces(1, 1, 1), 0, false, "box")
	if _, err := MidSurfaces(solid, 0.5, "mid"); err == nil {
		t.Error("a cube with all separations 1 should have no pair within 0.5")
	}
}
