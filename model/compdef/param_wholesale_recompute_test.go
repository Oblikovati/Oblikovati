// SPDX-License-Identifier: GPL-2.0-only

package compdef_test

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// A parameter that reaches geometry through a path the engine cannot attribute to a single
// feature — here a work-plane offset, which moves a hosted sketch and everything built on it —
// must conservatively rebuild the whole program (Oblikovati#1414). Missing it would leave the
// dependent feature on stale geometry, the exact silent-stale failure the targeted path must
// never introduce. This test pins both halves: the rebuild happens (every feature recomputes)
// AND the geometry follows the moved plane.
func TestWorkPlaneOffsetParameterEditRebuildsWholeProgram(t *testing.T) {
	def := compdef.NewPartComponentDefinition()
	lift, err := def.Parameters().AddUserParameter("lift", "20 mm") // 2 cm
	if err != nil {
		t.Fatalf("AddUserParameter: %v", err)
	}

	// Feature 0: a plain block on XY (independent of lift).
	base := def.Sketches().Add(sketch.XYPlane())
	rectangle(base, 4, 3)
	feature.NewExtrudeFeatures(def.Features()).
		AddByDistanceExtent(base, 0, ops.NewBody, func() float64 { return 5 })

	// Feature 1: a block on a work plane offset from XY by the "lift" parameter.
	wp := def.WorkPlanes().AddByPlaneAndOffset(feature.OriginXYPlane, lift.ModelValue)
	lifted := def.Sketches().Add(wp.Plane())
	lifted.SetPlaneHost(func() sketch.Plane { return wp.Plane() })
	rectangle(lifted, 4, 3)
	feature.NewExtrudeFeatures(def.Features()).
		AddByDistanceExtent(lifted, 0, ops.NewBody, func() float64 { return 5 })

	def.Recompute()
	before := recomputeCounts(def)
	if z := def.RangeBox().Min.Z; z < -1e-6 || z > 1e-6 {
		t.Fatalf("base block min.z = %v, want 0 before the lift edit", z)
	}

	if err := def.Parameters().SetExpression(userParamID(t, def, "lift"), "50 mm"); err != nil { // 5 cm
		t.Fatalf("SetExpression: %v", err)
	}
	def.RecomputeAfterChange()

	// The whole program rebuilds: even feature 0, independent of lift, recomputes (the
	// conservative fallback for an unmodelled parameter path).
	after := recomputeCounts(def)
	for i := range before {
		if after[i] != before[i]+1 {
			t.Errorf("feature %d count %d→%d, want a full rebuild (+1)", i, before[i], after[i])
		}
	}
	// The lifted block followed its work plane: the model now spans z ∈ [0, 5+5] (base 0..5,
	// lifted 5..10), so the top moved from 7 cm to 10 cm.
	if z := def.RangeBox().Max.Z; z < 9.999 || z > 10.001 {
		t.Errorf("after lift→50 mm, model max.z = %v, want ~10 (lifted block at 5..10 cm)", z)
	}
}
