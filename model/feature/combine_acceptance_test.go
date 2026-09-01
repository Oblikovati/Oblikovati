// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// PBI-172 (#469) acceptance for the pure-boolean modify features: Combine join/cut/intersect
// of *intersecting* blocks (the disjoint case in TestCombineJoinsTwoBodiesForReal does not
// exercise the arrangement boolean), recompute under a driving parameter, and recipe round
// trip. Split-by-plane acceptance lives in split_solid_test.go.

// box2x2x2 returns an axis-aligned 2×2×2 prism whose −X face is at x=dx (so dx=0 and dx=1
// overlap in 1×2×2 = volume 4).
func box2x2x2(dx float64, feat string) *topo.Body {
	poly := []math.Point2{{X: dx, Y: 0}, {X: dx + 2, Y: 0}, {X: dx + 2, Y: 2}, {X: dx, Y: 2}}
	return buildPrism(poly, sketch.XYPlane(), span{near: 0, far: 2}, 0, feat)
}

// TestCombineJoinIntersectingUnionVolume: A∪B of two overlapping 2³ blocks is one valid,
// manifold solid of the union volume 8 + 8 − 4 = 12.
func TestCombineJoinIntersectingUnionVolume(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(box2x2x2(0, "a"))
	NewBaseFeatures(fs).AddBase(box2x2x2(1, "b"))
	join := NewModifyFeatures(fs).AddCombine(0, 1, ops.Join)
	fs.Recompute()

	if !join.Health().OK() {
		t.Fatalf("join sick: %+v", join.Health())
	}
	if len(fs.Result()) != 1 {
		t.Fatalf("join result = %d bodies, want 1", len(fs.Result()))
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() || !r.Manifold {
		t.Fatalf("union body not a valid manifold solid: %+v", r)
	}
	if v := query.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErr(v, 12) > 1e-6 {
		t.Errorf("A∪B volume = %g, want 12", v)
	}
}

// TestCombineIntersectOverlapVolume: A∩B of the two overlapping blocks is exactly the 1×2×2
// overlap, volume 4.
func TestCombineIntersectOverlapVolume(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(box2x2x2(0, "a"))
	NewBaseFeatures(fs).AddBase(box2x2x2(1, "b"))
	inter := NewModifyFeatures(fs).AddCombine(0, 1, ops.Intersect)
	fs.Recompute()

	if !inter.Health().OK() {
		t.Fatalf("intersect sick: %+v", inter.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("intersection body not a valid solid: %+v", r)
	}
	if v := query.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErr(v, 4) > 1e-6 {
		t.Errorf("A∩B volume = %g, want 4", v)
	}
}

// TestCombineCutRoundTrip: a Cut combine survives the full recipe codec (operation + indices)
// and rebuilds the same A−B = 4 solid. Tool bodies come from two extrudes so the whole program
// serializes (raw base bodies have no codec).
func TestCombineCutRoundTrip(t *testing.T) {
	t.Parallel()
	skA := sketch.NewSketches().Add(sketch.XYPlane())
	addRect(skA, 0, 0, 2, 2)
	skB := sketch.NewSketches().Add(sketch.XYPlane())
	addRect(skB, 1, 0, 3, 2)
	idx := sketchList{sks: []*sketch.Sketch{skA, skB}}

	fs := NewPartFeatures(nil)
	NewExtrudeFeatures(fs).AddByDistanceExtent(skA, 0, ops.NewBody, func() float64 { return 2 })
	NewExtrudeFeatures(fs).AddByDistanceExtent(skB, 0, ops.NewBody, func() float64 { return 2 })
	NewModifyFeatures(fs).AddCombine(0, 1, ops.Cut)

	data, err := fs.MarshalRecipe(idx)
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, idx, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	cd := fresh.Item(2).Definition().(*CombineFeature).Definition()
	if cd.Operation != ops.Cut || cd.TargetIndex != 0 || len(cd.ToolIndices) != 1 || cd.ToolIndices[0] != 1 {
		t.Fatalf("restored combine = op %v target %d tools %v; want Cut 0 [1]", cd.Operation, cd.TargetIndex, cd.ToolIndices)
	}
	fresh.Recompute()
	if v := query.BodyGeometryProperties(fresh.Result()[0], ops.DefaultQuality()).Volume; relErr(v, 4) > 1e-6 {
		t.Errorf("restored A−B volume = %g, want 4", v)
	}
}

// TestCombineRecomputesOnParameterChange: a Cut whose tool body is driven by an extrude
// distance re-evaluates when that parameter changes (A−B grows as the tool shrinks).
func TestCombineRecomputesOnParameterChange(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	// Target A: a fixed 2×2×2 block (vol 8).
	NewBaseFeatures(fs).AddBase(box2x2x2(0, "a"))
	// Tool B: a 1×2 footprint at x∈[1,2] extruded up by a mutable height; the overlap with A
	// is 1×2×min(2,height).
	skB := sketch.NewSketches().Add(sketch.XYPlane())
	addRect(skB, 1, 0, 2, 2)
	height := 2.0
	toolB := NewExtrudeFeatures(fs).AddByDistanceExtent(skB, 0, ops.NewBody, func() float64 { return height })
	NewModifyFeatures(fs).AddCombine(0, 1, ops.Cut)

	fs.Recompute()
	if v := query.BodyGeometryProperties(fs.Result()[0], ops.DefaultQuality()).Volume; relErr(v, 4) > 1e-6 {
		t.Fatalf("height 2 ⇒ A−B = %g, want 4 (removed 1×2×2)", v)
	}
	height = 1.0        // shrink the tool: overlap becomes 1×2×1 = 2, so A−B = 6
	fs.MarkDirty(toolB) // a parameter change invalidates the driven feature and its dependents
	fs.Recompute()
	if v := query.BodyGeometryProperties(fs.Result()[0], ops.DefaultQuality()).Volume; relErr(v, 6) > 1e-6 {
		t.Errorf("height 1 ⇒ A−B = %g, want 6 (removed 1×2×1)", v)
	}
}

// addRect adds a closed axis-aligned rectangle [x0,x1]×[y0,y1] (one profile) to a sketch.
func addRect(sk *sketch.Sketch, x0, y0, x1, y1 float64) {
	c0 := sk.Points().Add(math.P2(x0, y0))
	c1 := sk.Points().Add(math.P2(x1, y0))
	c2 := sk.Points().Add(math.P2(x1, y1))
	c3 := sk.Points().Add(math.P2(x0, y1))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
}
