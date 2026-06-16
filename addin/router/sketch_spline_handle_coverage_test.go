// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// TestSetSplineHandle2D activates a 2D spline fit-point handle, edits its tangent/weight,
// then deactivates it — plus the bad-tangent-arity error.
func TestSetSplineHandle2D(t *testing.T) {
	r, s := seededSession(t)
	var sp wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"spline","points":[[0,0],[2,1],[4,0],[6,1]]}`, &sp)

	base := wire.SetSplineHandleArgs{SketchIndex: 0, Spline: sp.EntityID, FitPointIndex: 1}
	withTangent := base
	withTangent.Active, withTangent.Tangent, withTangent.Weight = true, []float64{1, 0}, 1
	call(t, r, s, "sketch.setSplineHandle", mustJSON(t, withTangent), nil)

	weightOnly := base
	weightOnly.Active, weightOnly.Weight = true, 2
	call(t, r, s, "sketch.setSplineHandle", mustJSON(t, weightOnly), nil)

	off := base
	call(t, r, s, "sketch.setSplineHandle", mustJSON(t, off), nil) // Active=false deactivates

	bad := base
	bad.Active, bad.Tangent = true, []float64{1, 0, 0}
	if err := tryCall(t, r, s, "sketch.setSplineHandle", mustJSON(t, bad)); err == nil {
		t.Error("a 3-component tangent should error for a 2D handle")
	}
}

// TestSetSplineHandle3D does the same for a 3D-sketch spline (3-component tangent).
func TestSetSplineHandle3D(t *testing.T) {
	r, s := seededSession(t)
	var created wire.CreateSketch3DResult
	call(t, r, s, "sketch3d.create", `{}`, &created)
	var sp wire.AddSketch3DEntityResult
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"spline","points":[[0,0,0],[1,1,0],[2,0,1],[3,1,0]]}`, &sp)

	on := wire.SetSplineHandleArgs{SketchIndex: created.SketchIndex, Spline: sp.EntityID, FitPointIndex: 1, Active: true, Tangent: []float64{1, 0, 0}, Weight: 1}
	call(t, r, s, "sketch3d.setSplineHandle", mustJSON(t, on), nil)
	off := wire.SetSplineHandleArgs{SketchIndex: created.SketchIndex, Spline: sp.EntityID, FitPointIndex: 1}
	call(t, r, s, "sketch3d.setSplineHandle", mustJSON(t, off), nil)

	bad := wire.SetSplineHandleArgs{SketchIndex: created.SketchIndex, Spline: sp.EntityID, FitPointIndex: 1, Active: true, Tangent: []float64{1, 0}}
	if err := tryCall(t, r, s, "sketch3d.setSplineHandle", mustJSON(t, bad)); err == nil {
		t.Error("a 2-component tangent should error for a 3D handle")
	}
}
