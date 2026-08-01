// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// TestConvexWedgeSetbackWatertight is the per-case whole-body gate for the two OCCT tests/blend/simple
// cases the P4 convex-wedge run-off setback touches: A8 (RED→GREEN) and A6 (the green-by-tolerance
// sibling legitimately re-welded TOWARD OCCT, +0.79%→−0.03%). Each is a WEDGE (a polyhedral prism)
// whose three CONVEX edges meet at one same-sense trihedral corner. The corner itself is already
// OCCT-exact — the material-side octant sphere (frac Ω/4π; A8 area 195.1) and the corner-side band
// setback s=r·cot(θ/2) come free from the shared blend path. The miss is the OTHER end of an OBLIQUE
// band running off onto an un-filleted face: its rail overshot the run-off plane by a tab (A8 +742,
// OCCT clips it exactly at the plane). This asserts — WITHOUT relying on the area-only
// TestOCCTBlendSimple — that each result is a watertight fold-free solid carrying exactly one
// material-side corner sphere of radius r and OCCT's whole-body area (a regression that re-introduces
// the run-off tab jumps the area and fails loud here).
func TestConvexWedgeSetbackWatertight(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		faces      int
		area       float64 // OCCT checkprops whole-body reference area
		sphereArea float64 // material-side octant sphere patch (Ω/4π·4πr²) — the corner is already correct
	}{
		{"A8", 9, 26965.6, 195.09},
		{"A6", 10, 30065.2, 157.06},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := caseResultBody(t, tc.name)
			assertWatertight(t, tc.name, body, tc.faces)
			assertWholeBodyFoldFree(t, tc.name, body)
			assertWholeBodyArea(t, tc.name, body, tc.area)
			assertConvexCornerSphere(t, tc.name, body, tc.sphereArea)
			assertNoRunoffTab(t, tc.name, body)
		})
	}
}

// assertConvexCornerSphere checks the body carries EXACTLY ONE corner sphere and that it is the
// material-side octant of radius r=10 with the derived patch area — proof the convex same-sense corner
// (and its s=r·cot(θ/2) corner-side setback) is intact after the run-off clip. A regression that
// perturbs the sphere (wrong count/radius/area) fails here; the whole-body area gate above independently
// catches a re-introduced run-off tab.
func assertConvexCornerSphere(t *testing.T, name string, body *topo.Body, wantArea float64) {
	t.Helper()
	found := 0
	for _, f := range body.Faces() {
		sph, ok := f.Geometry().(geom.Sphere)
		if !ok {
			continue
		}
		found++
		if stdmath.Abs(sph.Radius-10) > ops.ResolutionForBody(body).Weld() {
			t.Fatalf("%s corner sphere radius %.6f, want r=10", name, sph.Radius)
		}
		if a := faceMeshArea2(f); stdmath.Abs(a-wantArea) > 0.02*wantArea {
			t.Fatalf("%s corner sphere area %.4f, want ≈%.2f (material-side octant frac Ω/4π)", name, a, wantArea)
		}
	}
	if found != 1 {
		t.Fatalf("%s: want exactly 1 convex corner sphere, found %d", name, found)
	}
}

// assertNoRunoffTab is the fix-specific invariant: every planar-face loop vertex must lie ON its own
// plane (within the model-relative weld). The baseline A8 bug put a band's run-off rail at
// (−9.285,96.286,100) — a vertex 9.285 OFF the x=0 face's plane, inflating that face and the body.
// After the clip every vertex sits on its face's plane, so this rejects a re-introduced tab even if a
// future tessellation change masked it in the area total.
func assertNoRunoffTab(t *testing.T, name string, body *topo.Body) {
	t.Helper()
	tol := 1e-6 * boundingDiag(body)
	for _, f := range body.Faces() {
		pl, ok := f.Geometry().(geom.Plane)
		if !ok {
			continue
		}
		n := pl.Normal()
		for _, l := range f.Loops() {
			for _, u := range l.EdgeUses() {
				v := u.Edge().StartVertex().Point()
				if d := stdmath.Abs(n.Dot(pl.Origin.VectorTo(v))); d > tol {
					t.Fatalf("%s: face vertex %v lies %.4g off its plane (a run-off tab)", name, v, d)
				}
			}
		}
	}
}
