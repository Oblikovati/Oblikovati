// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/health"
)

// patchSurface seeds the running state with a 4×4 planar surface patch (at z=0) via
// a boundary patch.
func patchSurface(fs *PartFeatures) {
	NewBoundaryPatchFeatures(fs).Add(squareSketch(4), 0, PatchFree)
}

func TestTrimFeatureKeepsOneSide(t *testing.T) {
	fs := NewPartFeatures(nil)
	patchSurface(fs)
	pf := NewTrimFeatures(fs).AddByPlane(math.P3(2, 0, 0), math.V3(1, 0, 0), true)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("trim went unhealthy: %+v", pf.Health())
	}
	box := fs.Result()[0].RangeBox()
	if !approxEq(box.Min.X, 2) || !approxEq(box.Max.X, 4) {
		t.Errorf("trimmed x-span = [%v,%v], want [2,4]", box.Min.X, box.Max.X)
	}
}

func TestTrimFeatureGoesSickWhenNothingKept(t *testing.T) {
	fs := NewPartFeatures(nil)
	patchSurface(fs)
	pf := NewTrimFeatures(fs).AddByPlane(math.P3(10, 0, 0), math.V3(1, 0, 0), true)
	fs.Recompute()
	if pf.Health().Status != health.Sick {
		t.Errorf("trim that removes everything = %v, want sick", pf.Health().Status)
	}
}

func TestExtendFeatureGrowsSurface(t *testing.T) {
	fs := NewPartFeatures(nil)
	patchSurface(fs) // 4×4 patch on z=0
	fs.Recompute()
	var key []byte // the bottom boundary edge (both endpoints at y=0)
	for _, e := range fs.Result()[0].Edges() {
		if e.StartVertex().Point().Y == 0 && e.EndVertex().Point().Y == 0 {
			key = e.ReferenceKey()
		}
	}
	if key == nil {
		t.Fatal("no bottom edge on the patch")
	}
	pf := NewExtendFeatures(fs).Add(key, func() float64 { return 2 })
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("extend went sick: %+v", pf.Health())
	}
	box := fs.Result()[0].RangeBox()
	if !approxEq(box.Min.Y, -2) || !approxEq(box.Max.Y, 4) {
		t.Errorf("extended y-span = [%v,%v], want [-2,4]", box.Min.Y, box.Max.Y)
	}
}

func TestExtendFeatureSickOnLostEdge(t *testing.T) {
	fs := NewPartFeatures(nil)
	patchSurface(fs)
	NewExtendFeatures(fs).Add([]byte("ghost"), func() float64 { return 2 })
	fs.Recompute()
	if fs.Item(fs.Count()-1).Health().Status != health.Sick {
		t.Error("extend on a lost edge should be sick")
	}
}

func TestSurfaceOffsetFeatureMovesAlongNormal(t *testing.T) {
	fs := NewPartFeatures(nil)
	patchSurface(fs)
	pf := NewSurfaceOffsetFeatures(fs).AddByDistance(func() float64 { return 3 })
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("offset went unhealthy: %+v", pf.Health())
	}
	box := fs.Result()[0].RangeBox()
	if !approxEq(box.Min.Z, 3) || !approxEq(box.Max.Z, 3) {
		t.Errorf("offset patch z = [%v,%v], want flat at 3", box.Min.Z, box.Max.Z)
	}
}

func TestMidSurfaceFeatureExtractsThinWall(t *testing.T) {
	fs := NewPartFeatures(nil)
	// A 4×4×1 thin plate.
	NewExtrudeFeatures(fs).AddByDistanceExtent(squareSketch(4), 0, ops.NewBody, func() float64 { return 1 })
	pf := NewMidSurfaceFeatures(fs).AddByThickness(2)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("mid-surface went unhealthy: %+v", pf.Health())
	}
	if len(fs.Result()) != 1 || fs.Result()[0].IsSolid() {
		t.Errorf("mid-surface result = %d bodies (solid=%v), want 1 surface", len(fs.Result()), fs.Result()[0].IsSolid())
	}
	mf := pf.Definition().(*MidSurfaceFeature)
	if mf.Thicknesses().Count() != 1 {
		t.Fatalf("recorded %d thicknesses, want 1", mf.Thicknesses().Count())
	}
	if v := mf.Thicknesses().Item(0).Value; !approxEq(v, 1) {
		t.Errorf("recorded thickness = %v, want 1", v)
	}
	if z := fs.Result()[0].RangeBox().Center().Z; !approxEq(z, 0.5) {
		t.Errorf("mid patch center z = %v, want 0.5", z)
	}
}

func TestMidSurfaceFeatureGoesSickOnNoThinPair(t *testing.T) {
	fs := NewPartFeatures(nil)
	// A 1×1×1 cube: all separations are 1, none within a 0.5 threshold.
	NewExtrudeFeatures(fs).AddByDistanceExtent(squareSketch(1), 0, ops.NewBody, func() float64 { return 1 })
	pf := NewMidSurfaceFeatures(fs).AddByThickness(0.5)
	fs.Recompute()
	if pf.Health().Status != health.Sick {
		t.Errorf("mid-surface with no thin pair = %v, want sick", pf.Health().Status)
	}
}

func TestSurfaceEditGoesSickWithNoTarget(t *testing.T) {
	fs := NewPartFeatures(nil)
	pf := NewTrimFeatures(fs).AddByPlane(math.P3(0, 0, 0), math.V3(1, 0, 0), true)
	fs.Recompute()
	if pf.Health().Status != health.Sick {
		t.Errorf("trim with no target body = %v, want sick", pf.Health().Status)
	}
}
