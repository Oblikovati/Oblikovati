// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// FR5 tessellation-fold gate for the far-runout sphere-slice solids OCCT blend/simple/{D5,E4}. It PINS
// every built face's Property-quality mesh area to OCCT's per-face oracle (captured live from the locally
// built DRAWEXE 8.0.0: run the case, `explode result f`, `sprops` each face — the per-face areas sum to the
// case total 134780/134002, confirming the mapping) and gates each face for FOLDS (winding-consistent, no
// over-enclosing crease). Tessellation correctness is the repo's highest priority (CLAUDE.md), so a folded
// mesh — which over-counts area — must be caught here, not shipped.
//
// ★ THE FORMER "KNOWN RESIDUAL" WAS NOT A CDT DEFECT. fr5-report.md recorded E4 as carrying one permanent
// 2-edge fold on ONE of its two congruent torus arms (mesh 6236.80 vs the oracle 6230.42, +0.10%; its
// mirror twin clean at 6228.11) and attributed it to the metric-scaled CDT leaving an un-flippable needle,
// noting only that the two mirror arms got different interior-node phase. The real cause was upstream and
// explains the asymmetry exactly: the shell-orientation pass FLIPS one of the two arms, and reversing a
// flipped face's loop segment used to re-fit it through three points, which replaced the arm's oblique
// spiric run-out rail with a CIRCLE lying **1.245 off the torus** (24.9% of r). The CDT was tiling a
// boundary that was not on the surface. With reverseSegmentCurve preserving the locus (fillet_orient.go),
// the rail measures 5.0e-12 off its own torus, BOTH arms mesh fold-free, and the previously folded arm
// reads 6228.48 (−0.031%, i.e. 3.3× closer to the oracle than the folded 6236.80). So this gate now
// demands ZERO folding faces on both cases, with no exception.

// oracleArmAreas are OCCT's per-face sprops areas for the two torus ARM faces of each case (DRAWEXE 8.0.0).
var oracleArmAreas = map[string][]float64{
	"D5": {581.921, 5110.61}, // arms Rmajor 72.271 and 139.642
	"E4": {6230.42, 6230.42}, // two congruent Rmajor-139.642 arms
}

// oracleHostSphere is OCCT's per-face sprops area for the retrimmed radius-150 HOST sphere.
var oracleHostSphere = map[string]float64{"D5": 57537.1, "E4": 57646.0}

// oracleTotal is OCCT's whole-result surface area (checkprops -s), reconfirmed live via the oracle wrapper.
var oracleTotal = map[string]float64{"D5": 134780, "E4": 134002}

const (
	armAreaRelTol = 0.005 // torus-arm mesh vs OCCT per-face oracle: a curved-band property-mesh lands ≤0.04% off
	// capPlaneRelTol pins the OBLIQUE CAP PLANE — the face the run-out rail bites, and the one the
	// three-point re-fit moved most. It read 4377.35 / 4305.28 (+0.230% / +0.233%) while the shared rail was
	// a circle off the torus; with the locus preserved it reads 4367.27 / 4295.20 (−8e-6 / −2e-5), so 1e-3
	// leaves ~50× headroom while still failing 2.3× on the re-fit geometry this replaced.
	capPlaneRelTol = 1e-3
)

// oracleCapPlane is OCCT's per-face sprops area for the OBLIQUE CAP PLANE the torus arm runs out onto
// (DRAWEXE 8.0.0, fr5-report.md §1) — identified by being the only plane of that area in each case.
var oracleCapPlane = map[string]float64{"D5": 4367.31, "E4": 4295.29}

func TestD5E4TessellationFoldGate(t *testing.T) {
	for _, name := range []string{"D5", "E4"} {
		t.Run(name, func(t *testing.T) {
			body := caseResultBody(t, name)
			var meshTotal float64
			foldingFaces := 0
			for _, f := range body.Faces() {
				m := ops.TessellateFace(f, ops.PropertyQuality())
				area := ops.MeshArea(m)
				meshTotal += area
				assertFaceFinitePositive(t, name, f, area)
				if ops.FoldEdgeCount(m) > 0 {
					foldingFaces++
					assertNoFold(t, name, f, m, area)
				}
				assertOraclePin(t, name, f, area)
			}
			assertFoldingFaceBudget(t, name, foldingFaces)
			assertMeshTotalMatchesOracle(t, name, meshTotal)
		})
	}
}

// assertFaceFinitePositive fails if any face meshes to a non-finite or non-positive area (a collapsed or
// mis-wound face).
func assertFaceFinitePositive(t *testing.T, name string, f *topo.Face, area float64) {
	t.Helper()
	if area <= 0 || stdmath.IsInf(area, 0) || stdmath.IsNaN(area) {
		t.Fatalf("%s %T face meshed to %.4f, want a finite positive area", name, f.Geometry(), area)
	}
}

// assertOraclePin pins a face's mesh area to OCCT's per-face oracle: the two torus ARMS (the brief's focus,
// replacing FR4's count+positivity fallback) and the radius-150 HOST sphere. Planes/cylinder/corner-sphere
// carry their own analytic gate in d5_e4_watertight_test.go (the E4 corner sphere is a known +4% B-rep
// construction divergence from OCCT, not a tessellation defect, so it is not oracle-pinned here).
func assertOraclePin(t *testing.T, name string, f *topo.Face, area float64) {
	t.Helper()
	if tor, ok := f.Geometry().(geom.Torus); ok {
		pinTorusArm(t, name, tor, area)
		return
	}
	if sph, ok := f.Geometry().(geom.Sphere); ok && sph.Radius >= 100 {
		if want := oracleHostSphere[name]; stdmath.Abs(area-want) > armAreaRelTol*want {
			t.Fatalf("%s host-sphere mesh area %.2f, want OCCT %.1f within %.1f%%", name, area, want, armAreaRelTol*100)
		}
	}
	pinObliqueCapPlane(t, name, f, area)
}

// pinObliqueCapPlane pins the plane the arm runs out onto — the face whose trim is the run-out rail the
// reversal used to degrade (see this file's ★ note). Only the cap plane is pinned: the other planes are
// untouched by the rail and already covered by the whole-body total.
func pinObliqueCapPlane(t *testing.T, name string, f *topo.Face, area float64) {
	t.Helper()
	want, ok := oracleCapPlane[name]
	if _, isPlane := f.Geometry().(geom.Plane); !ok || !isPlane || stdmath.Abs(area-want) > 0.02*want {
		return // not this case's cap plane (every other plane is an order of magnitude away in area)
	}
	if rel := stdmath.Abs(area-want) / want; rel > capPlaneRelTol {
		t.Fatalf("%s oblique cap-plane mesh area %.4f, want OCCT %.2f within %.2f%% (rel %.4f%%) — the arm's "+
			"run-out rail no longer trims it on the torus", name, area, want, capPlaneRelTol*100, rel*100)
	}
}

// pinTorusArm asserts a torus-arm mesh area matches its NEAREST OCCT per-face oracle within armAreaRelTol.
func pinTorusArm(t *testing.T, name string, tor geom.Torus, area float64) {
	t.Helper()
	best, bestErr := 0.0, stdmath.Inf(1)
	for _, o := range oracleArmAreas[name] {
		if e := stdmath.Abs(area - o); e < bestErr {
			best, bestErr = o, e
		}
	}
	if best == 0 { // no oracle entry for this case: guard the best-relative ratio below (÷0 → NaN/Inf)
		t.Fatalf("no matching torus-arm oracle for case %q (torus Rmaj %.3f, mesh area %.2f)", name, tor.MajorRadius, area)
	}
	if bestErr > armAreaRelTol*best {
		t.Fatalf("%s torus arm (Rmaj %.3f) mesh area %.2f, no OCCT oracle within %.1f%% (nearest %.2f, err %.3f%%)",
			name, tor.MajorRadius, area, armAreaRelTol*100, best, 100*bestErr/best)
	}
}

// assertNoFold fails on ANY folding face. E4's former single-arm exception is gone: it was the
// three-point re-fit putting that arm's run-out rail 1.245 off the torus (see this file's ★ note), not a
// CDT limit, so a fold here is once again an unqualified defect.
func assertNoFold(t *testing.T, name string, f *topo.Face, m *ops.Mesh, area float64) {
	t.Helper()
	t.Fatalf("%s fold: %T face (area %.2f) has %d fold edges — a tessellation defect (E4's former known "+
		"residual was the reversed run-out rail leaving the torus; it is fixed, see the ★ note)",
		name, f.Geometry(), area, ops.FoldEdgeCount(m))
}

// assertFoldingFaceBudget asserts both cases are entirely fold-free.
func assertFoldingFaceBudget(t *testing.T, name string, foldingFaces int) {
	t.Helper()
	if foldingFaces != 0 {
		t.Fatalf("%s has %d folding faces, want 0", name, foldingFaces)
	}
}

// assertMeshTotalMatchesOracle asserts the summed per-face mesh area equals OCCT's whole-result area within
// the corpus deps (0.01 relative) — the fold-free total (a large fold would inflate this).
func assertMeshTotalMatchesOracle(t *testing.T, name string, meshTotal float64) {
	t.Helper()
	want := oracleTotal[name]
	if rel := stdmath.Abs(meshTotal-want) / want; rel > 0.01 {
		t.Fatalf("%s total mesh area %.2f, want OCCT %.0f within deps 0.01 (rel %.4f)", name, meshTotal, want, rel)
	}
}
