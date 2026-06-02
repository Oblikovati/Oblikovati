// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/health"
)

// patchSurface seeds the running state with a 4×4 planar surface patch (at z=0) via
// a boundary patch.
func patchSurface(fs *PartFeatures) {
	NewBoundaryPatchFeatures(fs).Add(squareSketch(4), 0, PatchFree)
}

func TestTrimFeatureKeepsOneSide(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
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
	fs := NewPartFeatures(nil, nil)
	patchSurface(fs)
	pf := NewTrimFeatures(fs).AddByPlane(math.P3(10, 0, 0), math.V3(1, 0, 0), true)
	fs.Recompute()
	if pf.Health().Status != health.Sick {
		t.Errorf("trim that removes everything = %v, want sick", pf.Health().Status)
	}
}

func TestExtendFeatureDefers(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	patchSurface(fs)
	pf := NewExtendFeatures(fs).Add(Ref{}, func() float64 { return 2 })
	fs.Recompute()
	if pf.Health().Status != health.Warning {
		t.Errorf("extend = %v, want warning (geometry deferred)", pf.Health().Status)
	}
	// The target surface passes through unchanged.
	if len(fs.Result()) != 1 {
		t.Errorf("extend passthrough has %d bodies, want 1", len(fs.Result()))
	}
}

func TestSurfaceOffsetFeatureMovesAlongNormal(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
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
	fs := NewPartFeatures(nil, nil)
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
	fs := NewPartFeatures(nil, nil)
	// A 1×1×1 cube: all separations are 1, none within a 0.5 threshold.
	NewExtrudeFeatures(fs).AddByDistanceExtent(squareSketch(1), 0, ops.NewBody, func() float64 { return 1 })
	pf := NewMidSurfaceFeatures(fs).AddByThickness(0.5)
	fs.Recompute()
	if pf.Health().Status != health.Sick {
		t.Errorf("mid-surface with no thin pair = %v, want sick", pf.Health().Status)
	}
}

func TestSurfaceEditGoesSickWithNoTarget(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	pf := NewTrimFeatures(fs).AddByPlane(math.P3(0, 0, 0), math.V3(1, 0, 0), true)
	fs.Recompute()
	if pf.Health().Status != health.Sick {
		t.Errorf("trim with no target body = %v, want sick", pf.Health().Status)
	}
}
