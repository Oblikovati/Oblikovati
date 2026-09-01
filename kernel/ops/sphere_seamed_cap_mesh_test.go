// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Regression for the seamed-hemisphere density defect (blend/simple S6/S7, sphere-notch-report.md).
// An imported boss hemisphere's outer loop is its SUBDIVIDED equator (a chain of coplanar arcs — the
// blend's runout terminations split it) plus the parametric seam as ONE edge used twice. That shape
// missed every fan recognizer (sphereCapFan needs the whole boundary coplanar; sphereZoneCapFan needs
// a single full-circle rim edge) and fell to spherePatchMesh, whose patchGridCap-clamped interior
// under-reported the hemisphere's area by ~0.66 of 2πR² = 1061.86 at EVERY chord tolerance — a flat,
// non-converging deficit that two review waves mis-read as a trim notch. sphereSeamedCapFan routes it
// to the latitude-ring fan instead. This fixture reproduces the seamed multi-arc face without a STEP
// round-trip (a named fake), so the kernel guards the fix on its own.

// seamedHemisphereFace builds a hemisphere face (R=13, centre origin, pole +z) whose outer loop is
// the equator split into nArcs Arc3d edges plus ONE seam meridian edge used twice — S6's shipped
// sphere-face topology (measured: 48 equator arcs + seam×2, loop order seam-down, CCW, seam-up).
func seamedHemisphereFace(t *testing.T, radius float64, nArcs int) *topo.Face {
	t.Helper()
	sphere, err := geom.NewSphere(math.P3(0, 0, 0), radius)
	if err != nil {
		t.Fatalf("NewSphere(R=%.3f): %v", radius, err)
	}
	pole := math.P3(0, 0, math.Scalar(radius))
	lin := func(s string, i int) topo.Lineage { return topo.NewLineage(topo.Tok("seamcap", s, i)) }
	bld := topo.NewBuilder(true, lin("body", 0))
	vp := bld.AddVertex(pole, lin("vp", 0))
	rims := make([]*topo.Vertex, nArcs)
	for i := range nArcs {
		a := 2 * stdmath.Pi * float64(i) / float64(nArcs)
		rims[i] = bld.AddVertex(equatorPt(radius, a), lin("vr", i))
	}
	seam := bld.AddEdge(seamMeridian(t, radius), vp, rims[0], lin("seam", 0))
	uses := []topo.Use{topo.Fwd(seam)}
	for i := range nArcs {
		a0 := 2 * stdmath.Pi * float64(i) / float64(nArcs)
		a1 := 2 * stdmath.Pi * float64(i+1) / float64(nArcs)
		arc, err := geom.Arc3dByThreePoints(equatorPt(radius, a0), equatorPt(radius, (a0+a1)/2), equatorPt(radius, a1))
		if err != nil {
			t.Fatalf("equator arc %d: %v", i, err)
		}
		e := bld.AddEdge(arc, rims[i], rims[(i+1)%nArcs], lin("arc", i))
		uses = append(uses, topo.Fwd(e))
	}
	uses = append(uses, topo.Rev(seam))
	bld.AddFace(sphere, lin("sph", 0), topo.OuterLoop(uses...))
	return bld.Build().Faces()[0]
}

func equatorPt(radius, azimuth float64) math.Point3 {
	return math.P3(math.Scalar(radius*stdmath.Cos(azimuth)), math.Scalar(radius*stdmath.Sin(azimuth)), 0)
}

// seamMeridian is the great-circle quarter arc from the +z pole down to the equator's azimuth-0 point.
func seamMeridian(t *testing.T, radius float64) geom.Arc3d {
	t.Helper()
	mid := math.P3(math.Scalar(radius/stdmath.Sqrt2), 0, math.Scalar(radius/stdmath.Sqrt2))
	arc, err := geom.Arc3dByThreePoints(math.P3(0, 0, math.Scalar(radius)), mid, math.P3(math.Scalar(radius), 0, 0))
	if err != nil {
		t.Fatalf("seam meridian: %v", err)
	}
	return arc
}

// TestSeamedHemisphereMeshesToItsClosedFormArea is the defect gate: at PropertyQuality the seamed
// hemisphere must mesh to within 0.01% of 2πR² (PropertyQuality's own documented contract). The
// clamped stereo-CDT path this face used to take reads ~0.06% under — 6× outside contract — at EVERY
// chord tolerance, so this fails RED the moment the routing is lost (mutation-proven).
func TestSeamedHemisphereMeshesToItsClosedFormArea(t *testing.T) {
	t.Parallel()
	const radius = 13.0
	face := seamedHemisphereFace(t, radius, 6)
	m := TessellateFace(face, PropertyQuality())
	closed := 2 * stdmath.Pi * radius * radius
	if got := MeshArea(m); stdmath.Abs(got-closed)/closed > 1e-4 {
		t.Fatalf("seamed hemisphere meshed %.6f, closed form 2πR²=%.6f (rel %.3g > 1e-4) — "+
			"the density-capped patch path is back", got, closed, stdmath.Abs(got-closed)/closed)
	}
	if top := meshMaxZ(m); stdmath.Abs(top-radius) > 1e-9*radius {
		t.Fatalf("fan did not reach the pole: max z=%.6f, want %.6f", top, radius)
	}
}

// meshMaxZ returns the largest vertex z (the fan must carry the pole).
func meshMaxZ(m *Mesh) float64 {
	top := stdmath.Inf(-1)
	for _, p := range m.Positions {
		top = stdmath.Max(top, float64(p.Z))
	}
	return top
}

// TestSeamedCapFanDeclinesForeignShapes drives the recognizer's decline arms directly: a shape it
// does not understand must fall through to the existing paths, never mesh a wrong fan.
func TestSeamedCapFanDeclinesForeignShapes(t *testing.T) {
	t.Parallel()
	const radius = 13.0
	t.Run("plain multi-arc rim without a seam declines (sphereCapFan's shape)", func(t *testing.T) {
		face := coplanarRimFace(t, radius)
		if _, ok := sphereSeamedCapFan(face, face.Geometry(), PropertyQuality()); ok {
			t.Fatal("seamed-cap fan claimed a seamless coplanar rim — that is sphereCapFan's face")
		}
	})
	t.Run("doubled edge that is not a pole seam declines (a slit, not a seam)", func(t *testing.T) {
		face := slitRimFace(t, radius)
		if _, ok := sphereSeamedCapFan(face, face.Geometry(), PropertyQuality()); ok {
			t.Fatal("seamed-cap fan claimed a doubled edge that never reaches the pole")
		}
	})
	t.Run("the seamed hemisphere itself is claimed", func(t *testing.T) {
		face := seamedHemisphereFace(t, radius, 6)
		if _, ok := sphereSeamedCapFan(face, face.Geometry(), PropertyQuality()); !ok {
			t.Fatal("seamed-cap fan declined the exact shape it exists for")
		}
	})
}

// coplanarRimFace is a hemisphere bounded by two equator arcs only (no seam edge in the loop).
func coplanarRimFace(t *testing.T, radius float64) *topo.Face {
	t.Helper()
	sphere, _ := geom.NewSphere(math.P3(0, 0, 0), radius)
	lin := func(s string, i int) topo.Lineage { return topo.NewLineage(topo.Tok("plaincap", s, i)) }
	bld := topo.NewBuilder(true, lin("body", 0))
	v0 := bld.AddVertex(equatorPt(radius, 0), lin("v", 0))
	v1 := bld.AddVertex(equatorPt(radius, stdmath.Pi), lin("v", 1))
	a0, _ := geom.Arc3dByThreePoints(equatorPt(radius, 0), equatorPt(radius, stdmath.Pi/2), equatorPt(radius, stdmath.Pi))
	a1, _ := geom.Arc3dByThreePoints(equatorPt(radius, stdmath.Pi), equatorPt(radius, 3*stdmath.Pi/2), equatorPt(radius, 2*stdmath.Pi))
	e0 := bld.AddEdge(a0, v0, v1, lin("e", 0))
	e1 := bld.AddEdge(a1, v1, v0, lin("e", 1))
	bld.AddFace(sphere, lin("sph", 0), topo.OuterLoop(topo.Fwd(e0), topo.Fwd(e1)))
	return bld.Build().Faces()[0]
}

// slitRimFace doubles an equator arc instead of a pole meridian: the doubled edge's far vertex sits
// on the rim plane, nowhere near the pole, so seamEndsAtPole must decline it.
func slitRimFace(t *testing.T, radius float64) *topo.Face {
	t.Helper()
	sphere, _ := geom.NewSphere(math.P3(0, 0, 0), radius)
	lin := func(s string, i int) topo.Lineage { return topo.NewLineage(topo.Tok("slitcap", s, i)) }
	bld := topo.NewBuilder(true, lin("body", 0))
	v0 := bld.AddVertex(equatorPt(radius, 0), lin("v", 0))
	v1 := bld.AddVertex(equatorPt(radius, stdmath.Pi), lin("v", 1))
	vs := bld.AddVertex(equatorPt(radius, -stdmath.Pi/4), lin("v", 2))
	a0, _ := geom.Arc3dByThreePoints(equatorPt(radius, 0), equatorPt(radius, stdmath.Pi/2), equatorPt(radius, stdmath.Pi))
	a1, _ := geom.Arc3dByThreePoints(equatorPt(radius, stdmath.Pi), equatorPt(radius, 3*stdmath.Pi/2), equatorPt(radius, 2*stdmath.Pi))
	slit, _ := geom.Arc3dByThreePoints(equatorPt(radius, 0), equatorPt(radius, -stdmath.Pi/8), equatorPt(radius, -stdmath.Pi/4))
	e0 := bld.AddEdge(a0, v0, v1, lin("e", 0))
	e1 := bld.AddEdge(a1, v1, v0, lin("e", 1))
	es := bld.AddEdge(slit, v0, vs, lin("e", 2))
	bld.AddFace(sphere, lin("sph", 0), topo.OuterLoop(topo.Fwd(es), topo.Rev(es), topo.Fwd(e0), topo.Fwd(e1)))
	return bld.Build().Faces()[0]
}

// TestSpherePatchGridClampIsDiagnosed: when patchGridCellBudget denies the interior Steiner grid the
// spacing the chord tolerance asked for, the mesh must carry CodeTessellateCapSaturated
// (Oblikovati#1412's honest-degradation rule) — the old silent per-axis clamp is what let a
// hemisphere under-report its area at every swept tolerance and read as a trim defect for two review
// waves. A tolerance the budget can honour must stay diagnostic-free. The fixture is a giant patch
// (R=150, 120°-polar cap rim: stereo chart, bbox ≈ 520): PropertyQuality asks ~1897² ≈ 3.6M cells,
// 13.7× over the 2^18 budget; the old R=13/60° fixture (78k cells) is honoured now and moved to the
// coarse (cry-wolf) arm's family of honoured grids.
func TestSpherePatchGridClampIsDiagnosed(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~4s): `make test-corpus`")
	}
	t.Parallel()
	const radius = 150.0
	sph, err := geom.NewSphere(math.P3(0, 0, 0), radius)
	if err != nil {
		t.Fatalf("NewSphere: %v", err)
	}
	ring := make([]math.Point3, 256)
	for i := range ring { // a 120°-polar-angle cap rim: stereo chart, bbox ≈ 520 — over budget at fine tol
		phi := 2 * stdmath.Pi * float64(i) / float64(len(ring))
		s, c := stdmath.Sincos(phi)
		ring[i] = math.P3(math.Scalar(radius*stdmath.Sin(2*stdmath.Pi/3)*c),
			math.Scalar(radius*stdmath.Sin(2*stdmath.Pi/3)*s), math.Scalar(radius*stdmath.Cos(2*stdmath.Pi/3)))
	}
	fine, ok := spherePatchMesh(nil, sph, ring, nil, Quality{ChordTolerance: 1e-3, AngleTolerance: stdmath.Pi / 180})
	if !ok {
		t.Fatal("spherePatchMesh declined the 120° cap rim")
	}
	if !hasDiag(fine.Diagnostics, CodeTessellateCapSaturated) {
		t.Fatalf("grid budget-scaled below chord tol 1e-3 but no %s diagnostic on the mesh", CodeTessellateCapSaturated)
	}
	coarse, ok := spherePatchMesh(nil, sph, ring, nil, Quality{ChordTolerance: 0.5, AngleTolerance: stdmath.Pi / 180})
	if !ok {
		t.Fatal("spherePatchMesh declined the 120° cap rim at the coarse tolerance")
	}
	if hasDiag(coarse.Diagnostics, CodeTessellateCapSaturated) {
		t.Fatal("budget honoured the coarse tolerance yet still reported saturation — the diagnostic would cry wolf")
	}
}
