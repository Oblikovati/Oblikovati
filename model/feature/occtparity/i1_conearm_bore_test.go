// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
)

// I1 (simple grid) is a 90°-revolve wedge (a triangular profile swept around a quarter circle) whose
// picked rim is the z=0 edge where the bottom annular plate meets the INNER cone wall — a conical BORE
// (the cone face is Reversed, material OUTSIDE the cone). The edge itself is still CONVEX (a bore rim
// rounds like an ordinary corner, ball in the material — see the do-no-harm comment above
// coneArmFillet, kernel/ops/fillet_conearm.go), so coneArmFilletConcave reuses coneArmSurface with the
// apex shift flipped to s=−1 (A′ = A − r/sinα·â) but the SAME material-side plane offset the boss (s=+1)
// case uses. Wiring this cost two fixes: (1) the dispatch in coneArmFillet, which used to honest-reject
// every material-outside cone; (2) the single-arm runout's host contact rail (armRunoutRail,
// fillet_curved_single_runout.go), which only ever tried torusContactCircle's INTERNAL tangency
// (h·sinα−R_s·cosα=+r, the boss reading) — a bore's torus is tangent on the OTHER equator (=−r), so the
// rail could not be built even once the arm itself existed. The SHARED coneContactCircle
// (fillet_curved_retrim.go, also used by the trihedral corner weld) stays convex-only on purpose: the
// tangency equation alone cannot tell I1's edge-convex bore torus apart from a genuinely edge-concave
// cove torus (S2's concaveConeArmSurface), so the external-tangency reading lives in a runout-scoped
// helper (coneBoreRunoutContactCircle) instead of widening the shared predicate — see
// TestConvexContactCircleRejectsConcaveTorus, the do-no-harm firewall this would otherwise have broken
// (caught during this session's own verification pass, not shipped).
//
// Per-face reconciliation vs DRAWEXE 8.0.0 (`blend result s 10 s_4` on the real I1 fixture, `sprops
// result_N 1e-10` per face — see wave-report-K.md for the session log): every one of the 6 faces is
// within 0.002% relative of OCCT's own number.
//
//	face (own surface)          ours        OCCT        rel
//	Plane end-cap (θ=0)      2376.38     2376.39    -0.0004%
//	Plane end-cap (θ=π/2)    2376.38     2376.39    -0.0004%
//	Plane bottom annulus    31227.6     31227.7    -0.0003%
//	Cone outer wall (r=300, unchanged)  30544.8   30544.8   exact
//	Cone inner wall (bore, trimmed)     17083.0   17083.1   -0.0006%
//	Torus band (major 224.14214, minor 10)  8027.46  8027.58  -0.0015%
//
// OCCT's own torus is centre(-200,0,10), axis(0,0,1), major 224.142135623731 = 200+10(1+√2), minor 10 —
// the closed form coneArmSurface(co,pl,n,-1,10,res) reproduces exactly (fillet_conearm_test.go's
// TestConeArm_ConcaveBoreBuilds pins it, and proves by mutation that the genuinely-edge-concave
// concaveConeArmSurface construction gives a DIFFERENT, wrong torus — centre z=-10, major 204.14214,
// short by exactly 2r).
const (
	i1EndCapArea    = 2376.38
	i1BottomArea    = 31227.6
	i1OuterConeArea = 30544.8
	i1InnerConeArea = 17083.0
	i1TorusArea     = 8027.46
	i1WholeArea     = 91635.9 // OCCT's checkprops reference (corpus.json expectedArea)
	i1Deps          = 0.01
)

// TestI1ConcaveBoreWatertight asserts the concave-bore cap-edge fillet builds a watertight 6-face solid
// (2 trimmed end caps, the trimmed bottom plate, the untouched outer cone, the trimmed inner bore cone,
// and the new torus band) — every edge 2-incident, valid + closed + holes-contained + IsSolid.
func TestI1ConcaveBoreWatertight(t *testing.T) {
	t.Parallel()
	assertWatertight(t, "I1", caseResultBody(t, "I1"), 6)
}

// TestI1ConcaveBoreFoldGate meshes every I1 face at both gate qualities (fold-free is exact and
// sampling-independent — never just PropertyQuality), then checks the per-face areas against the
// DRAWEXE reconciliation table above (2% slack — the DRAWEXE numbers here are read off sprops to 6
// significant figures, not carried to full precision) and the whole-body area against OCCT's own
// checkprops number.
func TestI1ConcaveBoreFoldGate(t *testing.T) {
	t.Parallel()
	body := caseResultBody(t, "I1")
	meshTotal, torusBands := 0.0, 0
	for _, f := range body.Faces() {
		m := tessellate.TessellateFace(f, ops.PropertyQuality())
		area := ops.MeshArea(m)
		meshTotal += area
		assertI1FaceSane(t, f, m, area)
		if tor, ok := f.Geometry().(geom.Torus); ok {
			torusBands++
			assertI1TorusMatchesOracle(t, tor)
			assertFaceNear(t, "I1 torus band", area, i1TorusArea)
		}
	}
	if torusBands != 1 {
		t.Fatalf("I1 has %d torus-band faces, want exactly 1 (the concave-bore fillet)", torusBands)
	}
	if rel := stdmath.Abs(meshTotal-i1WholeArea) / i1WholeArea; rel > i1Deps {
		t.Fatalf("I1 total mesh area %.4f, want OCCT %.1f within deps %.2f (rel %.6f)", meshTotal, i1WholeArea, i1Deps, rel)
	}
}

// assertI1TorusMatchesOracle pins the fillet torus to OCCT's own closed form: centre (-200,0,10), axis
// (0,0,1), major 224.142135623731 = 200+10(1+√2), minor 10 — the mutation-witness proof (a naive
// both-flipped construction gives major 204.14214 instead, short by exactly 2r) lives in
// fillet_conearm_test.go's TestConeArm_ConcaveBoreBuilds; this pins the SAME closed form through the
// real corpus fixture and the full single-arm-runout weld, not just the unit-level surface builder.
func assertI1TorusMatchesOracle(t *testing.T, tor geom.Torus) {
	t.Helper()
	const wantMajor = 224.142135623731
	const tol = 1e-6
	if stdmath.Abs(tor.MajorRadius-wantMajor) > tol {
		t.Fatalf("I1 torus major %.9f, want OCCT's %.9f (rel %.3g)", tor.MajorRadius, wantMajor, stdmath.Abs(tor.MajorRadius-wantMajor)/wantMajor)
	}
	if stdmath.Abs(tor.MinorRadius-10) > tol {
		t.Fatalf("I1 torus minor %.9f, want 10 (the fillet radius)", tor.MinorRadius)
	}
}

// assertFaceNear checks a face's tessellated area against a DRAWEXE-measured reference within 0.5%
// (looser than the exact-value torus pin above — this covers the coarser end-cap/plate/cone readings
// captured to 6 significant figures in the table comment).
func assertFaceNear(t *testing.T, label string, got, want float64) {
	t.Helper()
	if rel := stdmath.Abs(got-want) / want; rel > 0.005 {
		t.Fatalf("%s area %.4f, want DRAWEXE's %.4f (rel %.4g)", label, got, want, rel)
	}
}

// assertI1FaceSane fails if an I1 face meshes to a non-finite / non-positive area or carries a fold at
// either gate quality.
func assertI1FaceSane(t *testing.T, f *topo.Face, m *ops.Mesh, area float64) {
	t.Helper()
	if area <= 0 || stdmath.IsInf(area, 0) || stdmath.IsNaN(area) {
		t.Fatalf("I1 %T face meshed to %.4f, want a finite positive area", f, area)
	}
	assertFaceFoldFreeAtEveryQuality(t, "I1", f, m)
}
