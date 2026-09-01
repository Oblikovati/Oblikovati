// SPDX-License-Identifier: GPL-2.0-only

package compdef_test

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/depend"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// liftedBlockPart builds a two-feature part: feature 0 is a plain block on XY (independent of
// any parameter); feature 1 is a block on a work plane offset from XY by the "lift" parameter,
// with the sketch wired to the plane exactly as the router wires a live work-plane sketch
// (SetPlaneHost + SetHostFootprint). It returns the def and the lift parameter's id.
func liftedBlockPart(t *testing.T) (*compdef.PartComponentDefinition, string) {
	t.Helper()
	def := compdef.NewPartComponentDefinition()
	lift, err := def.Parameters().AddUserParameter("lift", "20 mm") // 2 cm
	if err != nil {
		t.Fatalf("AddUserParameter: %v", err)
	}

	base := def.Sketches().Add(sketch.XYPlane())
	rectangle(base, 4, 3)
	feature.NewExtrudeFeatures(def.Features()).
		AddByDistanceExtent(base, 0, ops.NewBody, func() float64 { return 5 })

	wp := def.WorkPlanes().AddByPlaneAndOffset(feature.OriginXYPlane, lift.ModelValue)
	lifted := def.Sketches().Add(wp.Plane())
	lifted.SetPlaneHost(func() sketch.Plane { return wp.Plane() })
	lifted.SetHostFootprint(func() []depend.Key { return wp.ParameterFootprint() })
	rectangle(lifted, 4, 3)
	feature.NewExtrudeFeatures(def.Features()).
		AddByDistanceExtent(lifted, 0, ops.NewBody, func() float64 { return 5 })

	return def, lift.Name()
}

// A work-plane offset reaches geometry through the sketch hosted on that plane. Now that the
// plane's offset parameter is folded into the hosted sketch's footprint (ADR-0044), editing it
// must rebuild ONLY the feature that consumes the hosted sketch (and its tail) — not the
// independent base block — while the lifted block still follows the moved plane. This pins both
// the targeting (feature 0 untouched) and the correctness (geometry follows).
func TestWorkPlaneOffsetParameterEditTargetsHostedSketch(t *testing.T) {
	t.Parallel()
	def, _ := liftedBlockPart(t)
	def.Recompute()
	before := recomputeCounts(def)
	if z := def.RangeBox().Min.Z; z < -1e-6 || z > 1e-6 {
		t.Fatalf("base block min.z = %v, want 0 before the lift edit", z)
	}

	if err := def.Parameters().SetExpression(userParamID(t, def, "lift"), "50 mm"); err != nil { // 5 cm
		t.Fatalf("SetExpression: %v", err)
	}
	def.RecomputeAfterChange()

	// Feature 0 is independent of lift and must NOT rebuild; feature 1 (consumes the hosted
	// sketch) and its tail must.
	after := recomputeCounts(def)
	if after[0] != before[0] {
		t.Errorf("independent base feature rebuilt on work-plane-offset edit: count %d→%d, want unchanged", before[0], after[0])
	}
	if after[1] != before[1]+1 {
		t.Errorf("hosted feature not rebuilt: count %d→%d, want +1", before[1], after[1])
	}
	// The lifted block followed its work plane: the model now spans z ∈ [0, 10] (base 0..5,
	// lifted 5..10), so the top moved from 7 cm to 10 cm.
	if z := def.RangeBox().Max.Z; z < 9.999 || z > 10.001 {
		t.Errorf("after lift→50 mm, model max.z = %v, want ~10 (lifted block at 5..10 cm)", z)
	}
}

// The targeted path must produce EXACTLY the geometry a full rebuild would. We make the same
// edit two ways — the incremental RecomputeAfterChange, then a forced full rebuild
// (MarkAllDirty) — and assert the bounding box is identical. A targeted path that wrongly
// skipped a dependent feature would diverge here; this is the differential guard that makes
// narrowing the wholesale fallback safe (ADR-0044, the silent-stale failure class).
func TestWorkPlaneOffsetTargetedMatchesFullRebuild(t *testing.T) {
	t.Parallel()
	def, _ := liftedBlockPart(t)
	def.Recompute()

	if err := def.Parameters().SetExpression(userParamID(t, def, "lift"), "50 mm"); err != nil {
		t.Fatalf("SetExpression: %v", err)
	}
	def.RecomputeAfterChange() // incremental
	targeted := def.RangeBox()

	def.Features().MarkAllDirty()
	def.Recompute() // forced full rebuild of the same state
	full := def.RangeBox()

	if !boxesEqual(targeted, full) {
		t.Errorf("targeted rebuild geometry %v != full rebuild geometry %v (the targeted path skipped a dependent feature)", targeted, full)
	}
}

// boxesEqual compares two range boxes within a small absolute tolerance.
func boxesEqual(a, b math.Box) bool {
	const tol = 1e-6
	return a.Min.IsEqualTo(b.Min, tol) && a.Max.IsEqualTo(b.Max, tol)
}
