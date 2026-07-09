// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// circleSketchOnPlaneZ builds a one-circle sketch on the XY-oriented plane offset to z, so an
// extrude from it stacks coaxially above one from the base XY plane.
func circleSketchOnPlaneZ(z, cx, cy, r float64) *sketch.Sketch {
	ux, _ := math.UnitVector3FromVector(math.V3(1, 0, 0))
	uy, _ := math.UnitVector3FromVector(math.V3(0, 1, 0))
	pl, _ := sketch.NewPlane(math.P3(0, 0, z), ux, uy)
	s := sketch.NewSketches().Add(pl)
	s.Circles().AddByCenterRadius(math.P2(cx, cy), r)
	return s
}

// TestCoaxialCylinderJoinKeepsAnalyticFace is the #1831 regression: joining two coaxial, equal-radius
// cylinders (a stacked/stepped shaft — extrude Ø4 z[0,4] then JOIN Ø4 z[4,7]) must yield ONE analytic
// geom.Cylinder face spanning the merged extent with the exact πr²h volume, not a shattered stack of
// planar facets. The bug: combine()'s curved-boolean gate (exactlyOneCurvedPrimitive) rejected the
// both-operands-cylinder case, so it faceted BOTH cylinders into 24-gon prisms before the union —
// losing analyticity (74 planar faces, 0 cylinders) and under-reporting the volume.
func TestCoaxialCylinderJoinKeepsAnalyticFace(t *testing.T) {
	const r, h1, h2 = 2.0, 4.0, 3.0
	fs := NewPartFeatures(nil)
	ex := NewExtrudeFeatures(fs)
	ex.AddByDistanceExtent(circleSketchAt(0, 0, r), 0, ops.NewBody, func() float64 { return h1 })        // z[0,4]
	ex.AddByDistanceExtent(circleSketchOnPlaneZ(h1, 0, 0, r), 0, ops.Join, func() float64 { return h2 }) // z[4,7]
	fs.Recompute()

	if n := len(fs.Result()); n != 1 {
		t.Fatalf("coaxial join = %d bodies, want 1", n)
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("coaxial-join body not a valid solid: %+v", r)
	}
	if got := cylinderFaceCount(body); got != 1 {
		t.Errorf("coaxial-join analytic cylinder faces = %d, want 1 (the union re-faceted the wall — #1831)", got)
	}
	// 5e-4 discriminates the analytic union (~−0.01% at PropertyQuality) from a shattered 24-gon
	// prism (~−1.1%): only the analytic cylinder passes.
	analytic := stdmath.Pi * r * r * (h1 + h2)
	if v := ops.BodyGeometryProperties(body, ops.PropertyQuality()).Volume; relErr(v, analytic) > 5e-4 {
		t.Errorf("coaxial-join volume = %g, want %g (πr²·%g) — faceted, not the analytic union", v, analytic, h1+h2)
	}
}

// TestUnequalCoaxialCylinderJoinStillFacets guards the gate's precision: a coaxial join of DIFFERENT
// radii (a shoulder — Ø4 over Ø6) is NOT the single-cylinder case, so it must NOT take the analytic
// coaxial path; it stays a valid solid of the correct volume (whether or not it keeps analytic faces).
func TestUnequalCoaxialCylinderJoinStillFacets(t *testing.T) {
	fs := NewPartFeatures(nil)
	ex := NewExtrudeFeatures(fs)
	ex.AddByDistanceExtent(circleSketchAt(0, 0, 3), 0, ops.NewBody, func() float64 { return 4 })       // Ø6 z[0,4]
	ex.AddByDistanceExtent(circleSketchOnPlaneZ(4, 0, 0, 2), 0, ops.Join, func() float64 { return 3 }) // Ø4 z[4,7]
	fs.Recompute()

	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("shoulder body not a valid solid: %+v", r)
	}
	// The shoulder is not a single cylinder, so it stays on the faceted planar path (24-gon prism,
	// ~−1.1% under the analytic value) — a wide bound just confirms it is a sane solid, not the
	// analytic union.
	analytic := stdmath.Pi*3*3*4 + stdmath.Pi*2*2*3
	if v := ops.BodyGeometryProperties(body, ops.PropertyQuality()).Volume; relErr(v, analytic) > 2e-2 {
		t.Errorf("shoulder volume = %g, want ~%g", v, analytic)
	}
}
