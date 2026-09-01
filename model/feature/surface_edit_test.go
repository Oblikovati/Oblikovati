// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/geom"
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
	t.Parallel()
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
	t.Parallel()
	fs := NewPartFeatures(nil)
	patchSurface(fs)
	pf := NewTrimFeatures(fs).AddByPlane(math.P3(10, 0, 0), math.V3(1, 0, 0), true)
	fs.Recompute()
	if pf.Health().Status != health.Sick {
		t.Errorf("trim that removes everything = %v, want sick", pf.Health().Status)
	}
}

func TestExtendFeatureGrowsSurface(t *testing.T) {
	t.Parallel()
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

// TestExtendFeatureMultiEdge extends the bottom (y=0) and top (y=4) edges of the 4×4 patch by 2
// each in one feature: the y-span grows to [-2,6] (#1878).
func TestExtendFeatureMultiEdge(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	patchSurface(fs)
	fs.Recompute()
	var bottom, top []byte
	for _, e := range fs.Result()[0].Edges() {
		a, b := e.StartVertex().Point(), e.EndVertex().Point()
		if a.Y == 0 && b.Y == 0 {
			bottom = e.ReferenceKey()
		}
		if approxEq(a.Y, 4) && approxEq(b.Y, 4) {
			top = e.ReferenceKey()
		}
	}
	pf := NewExtendFeatures(fs).AddExtend(&ExtendDefinition{EdgeKeys: [][]byte{bottom, top}, Distance: func() float64 { return 2 }})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("multi-edge extend sick: %+v", pf.Health())
	}
	box := fs.Result()[0].RangeBox()
	if !approxEq(box.Min.Y, -2) || !approxEq(box.Max.Y, 6) {
		t.Errorf("extended y-span = [%v,%v], want [-2,6]", box.Min.Y, box.Max.Y)
	}
}

// TestExtendFeatureToPlane extends the bottom edge of the 4×4 patch until it reaches the plane
// y=-3: the y-span grows to [-3,4] (#1878).
func TestExtendFeatureToPlane(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	patchSurface(fs)
	fs.Recompute()
	var bottom []byte
	for _, e := range fs.Result()[0].Edges() {
		if e.StartVertex().Point().Y == 0 && e.EndVertex().Point().Y == 0 {
			bottom = e.ReferenceKey()
		}
	}
	target, _ := geom.NewPlane(math.P3(0, -3, 0), math.V3(0, 1, 0))
	pf := NewExtendFeatures(fs).AddExtend(&ExtendDefinition{EdgeKeys: [][]byte{bottom}, TargetPlane: &target})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("extend-to-plane sick: %+v", pf.Health())
	}
	if box := fs.Result()[0].RangeBox(); !approxEq(box.Min.Y, -3) || !approxEq(box.Max.Y, 4) {
		t.Errorf("extended y-span = [%v,%v], want [-3,4]", box.Min.Y, box.Max.Y)
	}
}

// TestExtendOptionsRoundTrip pins #1878 serialization: multi-edge, extend-to-plane target, and the
// natural flag survive the recipe codec; a legacy single-edge recipe still restores.
func TestExtendOptionsRoundTrip(t *testing.T) {
	t.Parallel()
	target, _ := geom.NewPlane(math.P3(0, -3, 0), math.V3(0, 1, 0))
	fs := NewPartFeatures(nil)
	NewExtendFeatures(fs).AddExtend(&ExtendDefinition{
		EdgeKeys: [][]byte{[]byte("e-a"), []byte("e-b")}, TargetPlane: &target, Natural: true,
	})
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	if d := data[0].Extend; len(d.Edges) != 2 || len(d.TargetNormal) != 3 || !d.Natural {
		t.Fatalf("serialized extend = %+v", d)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	def := fresh.Item(0).Definition().(*ExtendFeature).Definition()
	if len(def.EdgeKeys) != 2 || def.TargetPlane == nil || !def.Natural {
		t.Errorf("restored extend = %d edges, target %v, natural %v", len(def.EdgeKeys), def.TargetPlane, def.Natural)
	}
}

func TestExtendFeatureSickOnLostEdge(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	patchSurface(fs)
	NewExtendFeatures(fs).Add([]byte("ghost"), func() float64 { return 2 })
	fs.Recompute()
	if fs.Item(fs.Count()-1).Health().Status != health.Sick {
		t.Error("extend on a lost edge should be sick")
	}
}

func TestSurfaceOffsetFeatureMovesAlongNormal(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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

// TestMidSurfaceManualPairsAndRange is #1885: manual face pairs extract the mid-surface without
// auto-pairing, and each pair reports an equal min/max range (1) for the parallel plate walls.
func TestMidSurfaceManualPairsAndRange(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	NewExtrudeFeatures(fs).AddByDistanceExtent(squareSketch(4), 0, ops.NewBody, func() float64 { return 1 })
	fs.Recompute()
	var top, bot []byte
	for _, f := range fs.Result()[0].Faces() {
		if n := f.Geometry().NormalAt(0, 0); n.Z > 0.99 {
			top = f.ReferenceKey()
		} else if n.Z < -0.99 {
			bot = f.ReferenceKey()
		}
	}
	pf := NewMidSurfaceFeatures(fs).AddMidSurface(&MidSurfaceDefinition{Pairs: [][2][]byte{{top, bot}}})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("manual-pair mid-surface sick: %+v", pf.Health())
	}
	if len(fs.Result()) != 1 || fs.Result()[0].IsSolid() {
		t.Errorf("result = %d bodies (solid=%v), want 1 surface", len(fs.Result()), fs.Result()[0].IsSolid())
	}
	th := pf.Definition().(*MidSurfaceFeature).Thicknesses()
	if th.Count() != 1 {
		t.Fatalf("recorded %d thicknesses, want 1", th.Count())
	}
	if it := th.Item(0); !approxEq(it.Min, 1) || !approxEq(it.Max, 1) {
		t.Errorf("thickness range = [%v,%v], want [1,1]", it.Min, it.Max)
	}
}

// TestMidSurfaceBodySelectionKeepsUnselected is #1885: mid-surfacing only body 0 of a two-body
// part leaves body 1 (a solid) untouched alongside the new mid-surface.
func TestMidSurfaceBodySelectionKeepsUnselected(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	NewExtrudeFeatures(fs).AddByDistanceExtent(squareSketch(4), 0, ops.NewBody, func() float64 { return 1 })     // body 0: thin plate
	NewExtrudeFeatures(fs).AddByDistanceExtent(squareSketch(1), 0, ops.NewBody, func() float64 { return 1 })     // body 1: cube (no thin pair)
	pf := NewMidSurfaceFeatures(fs).AddMidSurface(&MidSurfaceDefinition{MaxThickness: 2, BodyIndices: []int{0}}) // only the plate
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("body-selected mid-surface sick: %+v", pf.Health())
	}
	res := fs.Result()
	solids, surfaces := 0, 0
	for _, b := range res {
		if b.IsSolid() {
			solids++
		} else {
			surfaces++
		}
	}
	if solids != 1 || surfaces != 1 {
		t.Errorf("result = %d solids + %d surfaces, want 1 each (cube kept, plate → mid-surface)", solids, surfaces)
	}
}

// TestMidSurfaceOptionsRoundTrip pins #1885 serialization: min/max, body indices, and manual pairs
// survive the recipe codec.
func TestMidSurfaceOptionsRoundTrip(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	NewMidSurfaceFeatures(fs).AddMidSurface(&MidSurfaceDefinition{
		MaxThickness: 3, MinThickness: 1, BodyIndices: []int{0, 2}, Pairs: [][2][]byte{{[]byte("fa"), []byte("fb")}},
	})
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	if d := data[0].MidSurface; d.MinThickness != 1 || len(d.BodyIndices) != 2 || len(d.Pairs) != 1 {
		t.Fatalf("serialized mid-surface = %+v", d)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	def := fresh.Item(0).Definition().(*MidSurfaceFeature).Definition()
	if def.MinThickness != 1 || len(def.BodyIndices) != 2 || len(def.Pairs) != 1 || string(def.Pairs[0][0]) != "fa" {
		t.Errorf("restored mid-surface = %+v", def)
	}
}

func TestMidSurfaceFeatureGoesSickOnNoThinPair(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	fs := NewPartFeatures(nil)
	pf := NewTrimFeatures(fs).AddByPlane(math.P3(0, 0, 0), math.V3(1, 0, 0), true)
	fs.Recompute()
	if pf.Health().Status != health.Sick {
		t.Errorf("trim with no target body = %v, want sick", pf.Health().Status)
	}
}
