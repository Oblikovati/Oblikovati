// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"path/filepath"
	"strings"
	"testing"
)

// THE CAPPED-END BRIDGE AND THE MITER TRANSIT CHAIN — blend/simple O8, P4, P5 (the two-offset-cylinder
// family: pcylinder r5 ∩ pcylinder r5 offset (3,0,6), cut, tscale 10; fixtures CFI_h1234fiq / CFI_i1234fis).
//
// One retrim capability GREENS O8 (kernel/ops/fillet_miter_chain_retrim.go): a bitten TWO-ARC "lens"
// loop (the cap of two intersecting cylinders) is a legitimate wire — retrimBittenLoop's ≥3-segment
// guard was the only blocker.
//
// P4 and P5 do NOT green. Their fixture's raw STEP-imported base solid (before any fillet runs) already
// carries a degenerate zero-length edge on the wall's own loop at (100,50,0) — one pre-existing edge
// used TWICE (once each direction) bracketing it — proven present on the UNFILLETED body: tessellating
// the raw wall face alone comes out short of its own analytic area (~30680 vs the wall's true extent),
// independent of any fillet. This is an import/base-topology defect (STEP round-trip or the underlying
// two-cylinder-intersection boolean), NOT a curved-miter capability gap — see wave-report-C.md "P4/P5
// seam" for the full derivation. Building a miter body on top of it produces a VALID-LOOKING solid that
// is silently wrong: a ~10.6% short face and 10-28 free edges in the weld (measured before the guards
// below existed). ringHasRepeatedPoint (fillet_miter_chain_retrim.go) and sharedRetrimIsSound
// (fillet_miter_curved_weld.go) catch this — any contact host (bitten OR left passthrough) whose own
// loop revisits a point declines the corner honestly rather than certifying over untrustworthy material.
//
// Two capabilities remain real and GENERAL (not specific to O8/P4/P5, and exercised by prior greens):
//   - the capped-end bridge (bridgedRunoutHostFace) and the miter transit chain (tubeCapTransitArc,
//     reconcileOuterRailWithTrim) — the shapes that would green a P4/P5-class case once its host's own
//     base topology is sound;
//   - a closed (full-circle) seam arc traversed in reverse is reversed in its OWN parameters
//     (reversedWholeArc, fillet_blend_faces.go) rather than a three-point re-fit, which is ill-posed on
//     coincident endpoints and would otherwise zero the rim and crack the shell.
//
// Every number below is DRAWEXE 8.0.0's own `explode result F` + `sprops result_i 1.e-12`, captured
// with the case's own script (plain `explode`; none of these fixtures is in the nexplode set).
// Identification is by area + `mksurface`/`dump`, never `bounding`.

// miterChainPerFaceTol is the per-face relative budget: mesh quantization plus the one modelling
// divergence of the family — OCCT ships the seam-line arm as a vrational BSpline blend; ours is the
// EXACT rolling-ball cylinder, so the two arm faces differ by their own discretization only.
const miterChainPerFaceTol = 5e-3

// TestO8LensCapSingleArmRunout pins simple/O8 on the FIVE faces DRAWEXE ships: the outer wall, the
// pocket wall, the two lens caps (equal 2904.39 — the full-height cut makes top and bottom congruent),
// and the seam-line arm band.
func TestO8LensCapSingleArmRunout(t *testing.T) {
	t.Parallel()
	body := gridCaseBody(t, corpusRecord(t, "simple", "O8"))
	assertShippedPerFaceAgainstDrawexe(t, drawexeFaceCase{
		name: "simple/O8",
		// s1 wall (Cylinder) · s2 pocket wall (Cylinder) · top cap · bottom cap (Planes, congruent
		// lens complements) · seam-line arm (DRAWEXE: BSpline blend; ours: exact cylinder).
		drawexe:    []float64{11932.3, 7781.79, 2904.39, 2904.39, 897.788},
		totalArea:  26420.7,
		perFaceTol: miterChainPerFaceTol,
	}, body, nil, 1e-3)
	assertBodyIsWatertightAndSimple(t, body, "simple/O8")
}

// TestP4DeclinesOnPreexistingDegenerateHost is the regression pin for the ringHasRepeatedPoint guard:
// simple/P4's wall carries the pre-existing degenerate loop (see file doc), so the cylinder-arm's own
// outer-host chain retrim must decline HONESTLY rather than certify a ~10.6%-short, non-watertight
// solid. Falsified by mutation (verbatim in wave-report-C.md): with the guard's `|| ringHasRepeatedPoint(...)`
// removed, this case flips from FAIL(faulty) to a WRONG PASS at area 57417.4 vs the DRAWEXE 61058.6
// target (-5.96%) and 10-28 free edges in the weld.
func TestP4DeclinesOnPreexistingDegenerateHost(t *testing.T) {
	t.Parallel()
	assertMiterHonestlyDeclines(t, "simple", "P4", "single-arm runout: host geom.Cylinder retrim declined")
}

// TestP5DeclinesOnPreexistingDegenerateHost is the regression pin for the sharedRetrimIsSound guard:
// simple/P5's wall is the corner's OWN shared face, so the far-away pre-existing degenerate loop
// survives sharedMiterRetrim's local splice untouched and must decline the corner rather than certify
// over it. Falsified by mutation (verbatim in wave-report-C.md): with the guard removed, this case
// flips from FAIL(faulty) to a WRONG PASS at area 57165.7 vs the DRAWEXE 60665.6 target (-5.77%) and
// 12-18 free edges in the weld.
func TestP5DeclinesOnPreexistingDegenerateHost(t *testing.T) {
	t.Parallel()
	assertMiterHonestlyDeclines(t, "simple", "P5", "shared host geom.Cylinder's own boundary is degenerate")
}

// assertMiterHonestlyDeclines runs one corpus case's real fillet and requires it to decline with a
// reason containing wantSubstr — never a crash, never a silently wrong PASS. The shared assertion for
// the P4/P5 pre-existing-defect guards above.
func assertMiterHonestlyDeclines(t *testing.T, grid, name, wantSubstr string) {
	t.Helper()
	r := corpusRecord(t, grid, name)
	body, err := importInput(filepath.Join(CorpusFixtureDir(), r.InputStep))
	if err != nil {
		t.Fatalf("%s/%s: import failed: %v", grid, name, err)
	}
	sets, ok := scoreLocate(r, body)
	if !ok {
		t.Fatalf("%s/%s: could not locate picked edges", grid, name)
	}
	_, filletOK, reason := runFillet(body, sets)
	if filletOK {
		t.Fatalf("%s/%s: expected an honest decline (pre-existing degenerate host), got a PASS", grid, name)
	}
	if !strings.Contains(reason, wantSubstr) {
		t.Fatalf("%s/%s: decline reason %q does not contain %q — a different capability path may now be firing", grid, name, reason, wantSubstr)
	}
}
