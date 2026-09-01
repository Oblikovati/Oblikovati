// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Multi-tool combine and keep-tool-bodies (#1894). Both are about what the FEATURE is, not what
// the boolean computes: one feature against N tools instead of N features, and tools that outlive
// the boolean that used them.

// threeBoxPart seeds a part with a 2×2×2 block at x=0 and two 1-wide tools overlapping it at
// x=1 and x=1.5, so a cut by both removes strictly more than a cut by either.
func threeBoxPart(t *testing.T) *PartFeatures {
	t.Helper()
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(box2x2x2(0, "base"))
	NewBaseFeatures(fs).AddBase(unitTool(1.0, "tool-1"))
	NewBaseFeatures(fs).AddBase(unitTool(1.5, "tool-2"))
	return fs
}

// unitTool returns a 0.5×2×2 prism starting at x — a slab that cuts a slice off the base block.
func unitTool(x float64, feat string) *topo.Body {
	return buildPrism([]math.Point2{{X: x, Y: 0}, {X: x + 0.5, Y: 0}, {X: x + 0.5, Y: 2}, {X: x, Y: 2}},
		sketch.XYPlane(), span{near: 0, far: 2}, 0, feat)
}

// TestCombineCutsByEveryTool: the whole point of the collection is that ONE feature applies all
// the tools. A fold that stopped at the first would still produce a plausible, valid solid — so
// the test measures the volume both slabs remove, not merely that the feature is healthy.
func TestCombineCutsByEveryTool(t *testing.T) {
	t.Parallel()
	fs := threeBoxPart(t)
	cut := NewModifyFeatures(fs).AddCombineTools(0, []int{1, 2}, ops.Cut, false)
	fs.Recompute()

	if !cut.Health().OK() {
		t.Fatalf("multi-tool cut sick: %+v", cut.Health())
	}
	if len(fs.Result()) != 1 {
		t.Fatalf("multi-tool cut left %d bodies, want 1 (both tools consumed)", len(fs.Result()))
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("multi-tool cut result invalid: %+v", r)
	}
	// 8 − two 0.5×2×2 slabs = 8 − 2 − 2 = 4. Cutting by only the first would leave 6.
	if v := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErr(v, 4) > 1e-6 {
		t.Errorf("volume after cutting by both tools = %g, want 4 (6 means only one applied)", v)
	}
}

// TestCombineKeepsToolBodies: with keepTools the tools survive as separate solids, which is what
// lets a later feature reuse them. Their geometry must be untouched — a tool that came back
// already cut would silently change every later use of it.
func TestCombineKeepsToolBodies(t *testing.T) {
	t.Parallel()
	fs := threeBoxPart(t)
	cut := NewModifyFeatures(fs).AddCombineTools(0, []int{1, 2}, ops.Cut, true)
	fs.Recompute()

	if !cut.Health().OK() {
		t.Fatalf("keep-tools cut sick: %+v", cut.Health())
	}
	res := fs.Result()
	if len(res) != 3 {
		t.Fatalf("keep-tools left %d bodies, want 3 (two tools + the result)", len(res))
	}
	for i, b := range res[:2] {
		if v := ops.BodyGeometryProperties(b, ops.DefaultQuality()).Volume; relErr(v, 2) > 1e-6 {
			t.Errorf("kept tool %d volume = %g, want the untouched 2", i, v)
		}
	}
	if v := ops.BodyGeometryProperties(res[2], ops.DefaultQuality()).Volume; relErr(v, 4) > 1e-6 {
		t.Errorf("keep-tools cut volume = %g, want 4", v)
	}
}

// TestCombineRejectsRepeatedAndSelfTools: a repeated tool would boolean twice (harmless for a
// cut, wrong for a join of overlapping solids) and a self-tool would intersect a body with
// itself. Both are recipe mistakes worth naming rather than absorbing.
func TestCombineRejectsRepeatedAndSelfTools(t *testing.T) {
	t.Parallel()
	for name, tools := range map[string][]int{
		"repeated": {1, 1},
		"self":     {0},
		"absent":   {},
		"range":    {1, 7},
	} {
		t.Run(name, func(t *testing.T) {
			fs := threeBoxPart(t)
			bad := NewModifyFeatures(fs).AddCombineTools(0, tools, ops.Cut, false)
			fs.Recompute()
			if bad.Health().OK() {
				t.Errorf("combine with %s tools %v should be sick", name, tools)
			}
		})
	}
}

// extrudedThreeBoxPart seeds the same three blocks as extrudes, so the whole program serializes
// (a raw base body has no codec).
func extrudedThreeBoxPart(t *testing.T) (*PartFeatures, sketchList) {
	t.Helper()
	var sks []*sketch.Sketch
	for _, x := range []float64{0, 1, 1.5} {
		sk := sketch.NewSketches().Add(sketch.XYPlane())
		addRect(sk, x, 0, x+2, 2)
		sks = append(sks, sk)
	}
	fs := NewPartFeatures(nil)
	for _, sk := range sks {
		NewExtrudeFeatures(fs).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 2 })
	}
	return fs, sketchList{sks: sks}
}

// TestCombineToolsRoundTrip: a multi-tool, tool-keeping combine must come back as the same
// feature — and a single-tool one must keep writing the original scalar so an existing document
// is unchanged by this option existing.
func TestCombineToolsRoundTrip(t *testing.T) {
	t.Parallel()
	fs, idx := extrudedThreeBoxPart(t)
	NewModifyFeatures(fs).AddCombineTools(0, []int{1, 2}, ops.Cut, true)
	data, err := fs.MarshalRecipe(idx)
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	d := data[3].Combine
	if len(d.Tools) != 2 || d.Tools[0] != 1 || d.Tools[1] != 2 || !d.KeepTools {
		t.Fatalf("serialized combine = %+v, want tools [1 2] keepTools", d)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, idx, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	cd := fresh.Item(3).Definition().(*CombineFeature).Definition()
	if len(cd.ToolIndices) != 2 || cd.ToolIndices[1] != 2 || !cd.KeepTools {
		t.Errorf("restored combine = %+v, want both tools and keepTools", cd)
	}
}

// TestSingleToolCombineKeepsTheScalarSpelling: the recipe for the ordinary combine must not
// change shape because the multi-tool form now exists.
func TestSingleToolCombineKeepsTheScalarSpelling(t *testing.T) {
	t.Parallel()
	fs, idx := extrudedThreeBoxPart(t)
	NewModifyFeatures(fs).AddCombine(0, 1, ops.Cut)
	data, err := fs.MarshalRecipe(idx)
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	if d := data[3].Combine; d.Tool != 1 || len(d.Tools) != 0 || d.KeepTools {
		t.Errorf("single-tool combine serialized as %+v, want the scalar tool: 1 alone", d)
	}
}

// TestLegacyCombineRecipeRestores: a document written before #1894 carries only the scalar and
// must restore as the one-tool, tool-consuming combine it was.
func TestLegacyCombineRecipeRestores(t *testing.T) {
	t.Parallel()
	legacy := []FeatureData{{Kind: "combine", Combine: &CombineData{Target: 0, Tool: 1, Operation: "cut"}}}
	fs := NewPartFeatures(nil)
	if err := fs.ApplyRecipe(legacy, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe(legacy): %v", err)
	}
	cd := fs.Item(0).Definition().(*CombineFeature).Definition()
	if len(cd.ToolIndices) != 1 || cd.ToolIndices[0] != 1 || cd.KeepTools {
		t.Errorf("legacy combine restored as %+v, want tools [1] consuming", cd)
	}
}
