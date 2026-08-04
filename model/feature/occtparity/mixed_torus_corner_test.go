// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// mixedTorusCase is one 270°-sector, 3-pick corpus case (B5 cylinder host, C4 cone host, D7 sphere
// host): the reflex vertical edge is CONCAVE and the two cap edges CONVEX, so their shared trihedral
// vertex is the MIRRORED mixed-sense corner — 1 concave + 2 convex. Every number here is a DRAWEXE
// 8.0.0 receipt from OCCT's own tests/blend script (see offsurface-loopseg-report.md §4).
type mixedTorusCase struct {
	name        string
	center      [3]float64 // the corner torus centre: the concave spine at the cap's setback level
	capZ        float64    // the cap plane the two convex bands share
	occtConcave float64    // DRAWEXE per-face area of the concave (pivot) band
	occtCap     float64    // DRAWEXE per-face area of the receded cap plane
	occtFar     float64    // DRAWEXE per-face area of the far cap plane
}

// exactMixedCornerTorusArea is the corner patch's EXACT area, closed form. The patch is the same on
// ALL THREE cases — a 90°(u)×90°(v) window of the R=2r=20, r=10 torus, independent of the host, which is
// itself the check that the corner is host-agnostic. ∫∫ r(R + r·cos v) du dv over Δu=π/2, v∈[π/2, π]
// gives r·Δu·(R·Δv + r(sin v₂ − sin v₁)) = 10·(π/2)·(10π − 10) = 50π² − 50π. DRAWEXE 8.0.0 reports
// 336.401 for the same face — the exact value to its 6 printed figures — so the analytic constant is
// gated on instead of the oracle's rounded mesh (occt-oracle-not-religion: gate on the exact invariant
// when OCCT is the looser party).
var exactMixedCornerTorusArea = 50*stdmath.Pi*stdmath.Pi - 50*stdmath.Pi

// mixedTorusRadius is the corpus radius every one of these three cases blends at.
const mixedTorusRadius = 10.0

// occtMixedCornerFaceCount is DRAWEXE's `nbshapes result` FACE count on all three cases: 5 base faces
// (host wall, two caps, two sector flanks) + 3 fillet bands + the ONE corner patch.
const occtMixedCornerFaceCount = 9

// TestMixedSenseCornerIsTheDRAWEXETorus is the oracle gate for the 1-concave/2-convex trihedral corner.
// Before the mixed-sense generalization these three cases put a corner SPHERE solved r inside all three
// planes at the vertex — the convex-only answer. On a 270° sector that lands 2r√2 off the concave arm's
// spine, so the concave band's own top cross-section arc sat 12.36 (94% of r) off its own cylinder and
// the sphere patch meshed 1097.40 against OCCT's 336.401 (+226%). DRAWEXE dumps the corner as an
// analytic torus centred ON the concave spine with major R=2r, minor r; this asserts exactly that
// surface, its DRAWEXE area, and the DRAWEXE areas of the three faces the corner reshapes.
func TestMixedSenseCornerIsTheDRAWEXETorus(t *testing.T) {
	t.Parallel()
	for _, tc := range mixedTorusCases() {
		t.Run(tc.name, func(t *testing.T) {
			body := caseResultBody(t, tc.name)
			assertWatertight(t, tc.name, body, occtMixedCornerFaceCount)
			assertCornerTorusGeometry(t, tc, body)
			assertMixedTorusPerFaceAreas(t, tc, body)
		})
	}
}

// mixedTorusCases are the three corpus cases carrying this corner, with their DRAWEXE receipts.
func mixedTorusCases() []mixedTorusCase {
	return []mixedTorusCase{
		// pcylinder s 50 100 270; blend 10 s_9 s_6 s_5 — checkprops -s 44513.5
		{"B5", [3]float64{10, -10, 90}, 100, 1413.72, 4883.03, 5911.95},
		// pcone s 90 40 150 270; blend 10 s_9 s_6 s_5 — checkprops -s 89830.3
		{"C4", [3]float64{10, -10, 140}, 150, 2199.11, 2964.17, 19106.6},
		// psphere s 15 -60 60 270; tscale 10; blend 10 s_9 s_6 s_5 — checkprops -s 275019
		{"D7", [3]float64{10, -10, 119.90381056766}, 129.90381056766, 3923.97, 11743.9, 13275.1},
	}
}

// assertCornerTorusGeometry fails unless the body carries EXACTLY one corner patch, it is an analytic
// geom.Torus with the derived frame (centre on the concave spine, axis along the concave edge, major
// R=2r, minor r), and ZERO corner spheres — the sphere is what the convex-only solve used to force.
func assertCornerTorusGeometry(t *testing.T, tc mixedTorusCase, b *topo.Body) {
	t.Helper()
	tori, spheres := cornerPatches(b)
	if len(tori) != 1 || spheres != 0 {
		t.Fatalf("%s: %d corner tori + %d corner spheres, want exactly 1 torus and 0 spheres", tc.name, len(tori), spheres)
	}
	tor, eps := tori[0], ops.ResolutionForBody(b).Weld()
	want := [3]float64{tc.center[0], tc.center[1], tc.center[2]}
	got := [3]float64{tor.Center.X, tor.Center.Y, tor.Center.Z}
	for i := range want {
		if stdmath.Abs(got[i]-want[i]) > eps {
			t.Fatalf("%s: corner torus centre %v, want %v (the concave spine at the cap setback) within %.3g", tc.name, got, want, eps)
		}
	}
	if stdmath.Abs(stdmath.Abs(tor.AxisDir.Z())-1) > eps {
		t.Fatalf("%s: corner torus axis %v, want ±Z (the concave edge direction)", tc.name, tor.AxisDir)
	}
	if stdmath.Abs(tor.MajorRadius-2*mixedTorusRadius) > eps || stdmath.Abs(tor.MinorRadius-mixedTorusRadius) > eps {
		t.Fatalf("%s: corner torus R=%.9f r=%.9f, want R=2r=%.1f r=%.1f", tc.name, tor.MajorRadius, tor.MinorRadius, 2*mixedTorusRadius, mixedTorusRadius)
	}
}

// cornerPatches returns the body's analytic torus corner patches and its corner-sphere count. A corner
// sphere is a radius-r sphere (never the HOST sphere, which is two decades larger on D7).
func cornerPatches(b *topo.Body) ([]geom.Torus, int) {
	var tori []geom.Torus
	spheres := 0
	for _, f := range b.Faces() {
		switch s := f.Geometry().(type) {
		case geom.Torus:
			tori = append(tori, s)
		case geom.Sphere:
			if stdmath.Abs(s.Radius-mixedTorusRadius) < 1e-6*mixedTorusRadius {
				spheres++
			}
		}
	}
	return tori, spheres
}

// assertMixedTorusPerFaceAreas reconciles the four faces the corner treatment reshapes against their
// DRAWEXE per-face receipts at 1e-4 relative (measured ≤2.5e-5 on all three cases, so ~4× headroom).
// The HOST face and the two convex cap bands are deliberately NOT pinned here: they carry the separate
// far-end run-on / sphere-zone-tessellation debts this corner fix does not own (report §2, §5).
func assertMixedTorusPerFaceAreas(t *testing.T, tc mixedTorusCase, b *topo.Body) {
	t.Helper()
	const tol = 1e-4
	assertFaceArea(t, tc.name+" corner torus", cornerTorusArea(b), exactMixedCornerTorusArea, tol)
	assertFaceArea(t, tc.name+" concave pivot band", concaveBandArea(b), tc.occtConcave, tol)
	assertFaceArea(t, tc.name+" receded cap plane", capPlaneArea(b, tc.capZ), tc.occtCap, tol)
	assertFaceArea(t, tc.name+" far cap plane", farCapPlaneArea(b, tc.capZ), tc.occtFar, tol)
}

// assertFaceArea compares one measured face area to its DRAWEXE receipt at a relative tolerance.
func assertFaceArea(t *testing.T, what string, got, want, tol float64) {
	t.Helper()
	if rel := relErr(got, want); rel > tol {
		t.Errorf("%s mesh area %.6f, want DRAWEXE %.6f (rel %.3g, tol %.1g)", what, got, want, rel, tol)
	}
}

// cornerTorusArea is the summed mesh area of every analytic-torus face (exactly one here).
func cornerTorusArea(b *topo.Body) float64 {
	area := 0.0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Torus); ok {
			area += faceMeshArea2(f)
		}
	}
	return area
}

// concaveBandArea is the mesh area of the CONCAVE (pivot) band: the radius-r cylinder whose axis runs
// along the sector's own axis (Z), i.e. the reflex vertical edge's fillet.
func concaveBandArea(b *topo.Body) float64 {
	area := 0.0
	for _, f := range b.Faces() {
		cyl, ok := f.Geometry().(geom.Cylinder)
		if ok && stdmath.Abs(cyl.Radius-mixedTorusRadius) < 1e-6 && stdmath.Abs(stdmath.Abs(cyl.AxisDir.Z())-1) < 1e-9 {
			area += faceMeshArea2(f)
		}
	}
	return area
}

// capPlaneArea is the mesh area of the Z-normal plane at height z — the cap the two convex bands share,
// whose receded corner the torus top-contact arc re-trims.
func capPlaneArea(b *topo.Body, z float64) float64 {
	area := 0.0
	for _, f := range b.Faces() {
		if pl, ok := zNormalPlane(f); ok && stdmath.Abs(pl.Origin.Z-z) < 1e-6 {
			area += faceMeshArea2(f)
		}
	}
	return area
}

// farCapPlaneArea is the mesh area of the OTHER Z-normal plane — the cap at the concave band's far end,
// which the concave fillet's run-out lune extends.
func farCapPlaneArea(b *topo.Body, capZ float64) float64 {
	area := 0.0
	for _, f := range b.Faces() {
		if pl, ok := zNormalPlane(f); ok && stdmath.Abs(pl.Origin.Z-capZ) >= 1e-6 {
			area += faceMeshArea2(f)
		}
	}
	return area
}

// zNormalPlane reports whether f is a planar face whose normal is along ±Z, returning its plane.
func zNormalPlane(f *topo.Face) (geom.Plane, bool) {
	pl, ok := f.Geometry().(geom.Plane)
	if !ok {
		return geom.Plane{}, false
	}
	n := pl.UAxis.AsVector().Cross(pl.VAxis.AsVector())
	return pl, stdmath.Abs(stdmath.Abs(n.Z)-1) < 1e-9
}
