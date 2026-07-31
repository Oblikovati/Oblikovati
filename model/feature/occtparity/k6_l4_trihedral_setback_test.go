// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestK6L4TrihedralSetbackWatertight is the per-case whole-body gate for the two OCCT tests/blend/simple
// cases the P2 trihedral corner-setback pass greens: K6 and L4, each a box − pocket whose single
// trihedral corner joins THREE CONCAVE fillets at three mutually-orthogonal planar faces. Both were RED
// (K6 +1.19%, L4 +1.38% — the pre-P2 body placed the corner sphere on the MATERIAL side, the reflection
// of the true void sphere through the vertex, twisting the three bands and leaving un-retracted host
// tabs). Flipping the sphere to the VOID side retracts each band by r to the void tangent circle and
// re-trims the three hosts. It asserts — WITHOUT relying on the area-only TestOCCTBlendSimple — that each
// result is a watertight fold-free solid with the void sphere-octant patch and the retracted corner
// station, so a regression that drops the setback (or re-reflects the sphere) fails loud here.
func TestK6L4TrihedralSetbackWatertight(t *testing.T) {
	for _, tc := range []struct {
		name      string
		faces     int
		area      float64     // OCCT checkprops whole-body reference area
		sphereCtr math.Point3 // the VOID-side octant sphere centre (was the material-side reflection)
	}{
		{"K6", 15, 63733.6, math.P3(35, 15, 15)},
		// L4's pocket is rotated, so the void octant centre is not round — the literal carries full
		// precision (not 6-decimal) so the model-relative Weld tolerance (M35), tighter than the old
		// 1e-6·100, still resolves it as the void point (17.3 = 2·r·√3 from the material reflection).
		{"L4", 13, 59733.6, math.P3(77.63097278769045, 49.22836645711565, 35)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := caseResultBody(t, tc.name)
			assertWatertight(t, tc.name, body, tc.faces)
			assertWholeBodyFoldFree(t, tc.name, body)
			assertWholeBodyArea(t, tc.name, body, tc.area)
			assertVoidCornerSphere(t, tc.name, body, tc.sphereCtr)
		})
	}
}

// assertVoidCornerSphere checks the body carries exactly one radius-r (=5) corner sphere and that its
// centre is the VOID-side octant point — the crux the P2 flip fixes. A regression that re-reflects the
// sphere to the material side moves the centre by 2·r·√3 and fails here.
func assertVoidCornerSphere(t *testing.T, name string, body *topo.Body, want math.Point3) {
	t.Helper()
	found := 0
	for _, f := range body.Faces() {
		sph, ok := f.Geometry().(geom.Sphere)
		if !ok || sph.Radius > 10 { // skip any large host sphere; the corner blend is r=5
			continue
		}
		found++
		if d := sph.Center.DistanceTo(want); d > ops.ResolutionForBody(body).Weld() {
			t.Fatalf("%s corner sphere centre %v, want void-side %v (off by %.4f — sphere re-reflected to the material side?)",
				name, sph.Center, want, d)
		}
		assertVoidOctantMeshed(t, name, f, sph)
	}
	if found != 1 {
		t.Fatalf("%s has %d corner spheres, want exactly 1 void-side octant patch", name, found)
	}
}

// The void corner sphere's MESH region + resolution gate (patchgridcap-report.md §region). The B-rep
// octant was right all along, but the sphere-patch mesher picks its region from the loop's ABSOLUTE
// winding, which orientFilletShell leaves arbitrary on the planar path — so K6/L4's corner ball meshed
// the 7/8 COMPLEMENT (Ω = 7π/2, area 274.35 = 7/8·4πr² − clamp deficit; DRAWEXE 8.0.0 `sprops … 1.e-12`
// says 39.2699, and the swept complement mesh converges to 274.889 = 7/8·4π·25 exactly). The 1% corpus
// deps swallowed the +235.08 whole. assembleFilletFaces now routes through assembleCornerBlendBody
// (the curved path's complement detector); this pins both truths so neither regresses:
//   - REGION: signed solid angle at the centre = π/2 (the octant, covered exactly once);
//   - RESOLUTION: mesh area within voidOctantAreaCeil of the closed form 25π/2 = 39.269908.
const (
	voidOctantAreaCeil = 0.0032 // 1.2× the measured deficit (K6/L4 0.00264/0.00266 once the density
	// budget honours the octant's grid; 0.0054/0.0057 at the old 80-step clamp), abs units² — down-only
	voidOctantSolidAngle = 1e-6 // rel bound on |ΣΩ − π/2| (measured < 1e-12)
)

// assertVoidOctantMeshed pins the corner ball's MESH to the octant: area against 25π/2 and solid
// angle π/2 — the complement mesh reads ~274 area / 7π/2 and fails both, loud.
func assertVoidOctantMeshed(t *testing.T, name string, f *topo.Face, sph geom.Sphere) {
	t.Helper()
	mesh := ops.TessellateFace(f, ops.PropertyQuality())
	closed := stdmath.Pi * sph.Radius * sph.Radius / 2 // octant: 4πr²/8
	if got := ops.MeshArea(mesh); stdmath.Abs(got-closed) > voidOctantAreaCeil {
		t.Errorf("%s: corner sphere meshes %.6f vs octant closed form πr²/2=%.6f (|Δ|=%.4f > %.4f) — the complement region is back",
			name, got, closed, stdmath.Abs(got-closed), voidOctantAreaCeil)
	}
	omega := meshSolidAngleAt(mesh, sph.Center)
	if rel := stdmath.Abs(omega-stdmath.Pi/2) / (stdmath.Pi / 2); rel > voidOctantSolidAngle {
		t.Errorf("%s: corner sphere mesh covers solid angle %.9f, want π/2=%.9f (rel %.3g) — wrong region or a fold/overlap",
			name, omega, stdmath.Pi/2, rel)
	}
}
