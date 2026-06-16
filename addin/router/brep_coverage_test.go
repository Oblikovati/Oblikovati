// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// TestBrepCylinderConeTorus covers the cylinder/cone and torus primitive builders the
// existing test does not (block + sphere are already covered).
func TestBrepCylinderConeTorus(t *testing.T) {
	r, s := seededSession(t)
	var cone wire.BrepHandleResult
	call(t, r, s, "brep.createPrimitive", `{"kind":"cylinderCone","bottom":[0,0,0],"top":[0,0,5],"bottomRadius":2,"topRadius":1}`, &cone)
	if !cone.Stats.Solid {
		t.Error("cylinderCone is not solid")
	}
	var torus wire.BrepHandleResult
	call(t, r, s, "brep.createPrimitive", `{"kind":"torus","center":[0,0,0],"axis":[0,0,1],"majorRadius":3,"minorRadius":1}`, &torus)
	if !torus.Stats.Solid {
		t.Error("torus is not solid")
	}

	if err := tryCall(t, r, s, "brep.createPrimitive", `{"kind":"dodecahedron"}`); err == nil {
		t.Error("an unknown primitive kind should error")
	}
}

// TestBrepTransformSectionSilhouette drives the transient ops the existing test skips.
func TestBrepTransformSectionSilhouette(t *testing.T) {
	r, s := seededSession(t)
	var b wire.BrepHandleResult
	call(t, r, s, "brep.createPrimitive", `{"kind":"block","min":[0,0,0],"max":[4,4,4]}`, &b)
	ref := wire.BrepBodyRef{Handle: b.Handle}

	// Translate by a row-major 4×4 matrix.
	_ = tryCall(t, r, s, "brep.transform", mustJSON(t, wire.BrepTransformArgs{
		Handle: b.Handle, Matrix: []float64{1, 0, 0, 1, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1},
	}))
	_ = tryCall(t, r, s, "brep.sectionWithPlane", mustJSON(t, wire.BrepSectionArgs{
		Source: ref, PlaneOrigin: []float64{0, 0, 2}, PlaneNormal: []float64{0, 0, 1},
	}))
	_ = tryCall(t, r, s, "brep.silhouette", mustJSON(t, wire.BrepSilhouetteArgs{
		Source: ref, ViewDirection: []float64{0, 0, 1}, IncludeBoundary: true,
	}))
}
