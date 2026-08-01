// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"path/filepath"
	"strings"
	"testing"
)

// A1 — THE FAR-RUNOUT NEAR-BOUNDARY RECONCILIATION — blend/simple W3, W4 (fixture CCV_1_c12gsf.rle: a
// slot cut INTO a cylindrical boss R=1, filleted r=0.2 on both slot walls at their shared corner).
//
// R5's own two capability generalizations, still live and unrelated to this fix, are what let W3/W4
// build an arm/bridge topology at all: (1) curvedMiterTorusArm/curvedMiterCylinderArm's convex branches
// (fillet_miter_concave.go) had hardcoded ε=+1 (the boss convention, majorR=R−r) — W3/W4's corner is a
// NOTCH wall (DRAWEXE ships Radii 1.2/0.2 = R+r, ε=−1); both branches now read ε from
// cylinderHostRadialSign, the same per-wall sign the concave rim arm already used. (2) sharedRailAnchor's
// cap-bridge (fillet_miter_curved_weld.go) called capBridgeArc directly, which only builds an Arc3d on a
// CYLINDER shared face — sharedFaceBridgeSeg now dispatches on the shared face's own geometry (a straight
// segment on a Plane, exact by construction), falling back to capBridgeArc unchanged for the Cylinder
// case; miterTrimChain's own far-cap bridge had the identical bug fixed the same way. Both are indirectly
// re-verified below (a wrong ε or a declined bridge would fail TestW3TorusArmMiterPerFace's own arm-area
// and watertightness checks).
//
// R5 (this wave) built the boss/notch radial-sign generalization and the shared-face/capping bridge
// generalization that let W3/W4 reach the assembled-weld stage for the first time, and added
// miterWeldMeshDefect as a do-no-harm guard when the assembled mesh self-crossed (W4) or retraced (W3)
// at a ~2e-5 residual. R5 attributed the residual to "the far cross-section runout's own foot,
// fillet_far_runout.go" — reasonable from where R5 stood (out of scope to patch), but MEASUREMENT (this
// session) shows that attribution does not survive: fillet_far_runout.go's perpendicular fast path is a
// verbatim pass-through of h0.from/h1.from (see its own docstring, "the host rails already end on the
// feet ... pass through unchanged") — it never computes a foot at all for this fixture's regime.
//
// THE ACTUAL ROOT (traced by direct instrumentation, then confirmed against the STEP source): the boss
// cylinder's own stored axis position and the corner vertex the picked edges terminate at are two
// INDEPENDENTLY stored quantities in simple/W4.step — CARTESIAN_POINT #133/#249 (3, ., 0.9999, the
// cylinder's axis) vs #86 (2.000000005003, 0, 1, the vertex) — a genuine ~1e-4 gap baked into the SOURCE
// file (confirmed identically in the DRAWEXE-native CCV_1_c12gsf.rle: Locations block 3 carries an
// explicit -0.0001 z translation). Every step from that input to the torus arm's ball-tangent foot
// (armBallCenter → armRunoutFoot in fillet_curved_single_runout.go, called from buildMiterArmSide in
// fillet_miter_curved_weld.go) is closed-form and exact GIVEN it — torusArmSurface's
// projectOntoPlane(cyl.Origin, pl) faithfully carries cyl.Origin's own z (0.9999) into the arm's centre,
// and the far-vertex-seeded ball centre then projects to a foot ~2e-5 off the shared face's ORIGINAL
// boundary segment it is genuinely incident to (measured: foot z=1.0000200000000004 vs the segment's own
// z=1 exactly). This is NOT a bug in the fillet math — it is a normal amount of STEP-file construction
// slop the fillet kernel must be robust to, exactly the way OCCT's own tolerant B-rep model is.
//
// THE FIX: sharedRailAnchor (fillet_miter_curved_weld.go), miterTrimChain (fillet_miter_curved_hostfaces.go),
// and chainedHostRetrim (fillet_miter_chain_retrim.go) each answered "is this analytically-derived foot
// already part of the pre-existing host loop" using tol — res.Weld()·r, calibrated for FLOAT-NOISE
// coincidence (~1e-9 relative) between two computations that should be bit-identical. The foot here and
// the loop it is incident to are NOT two computations of the same thing; they are independently-stored
// geometry that only needs to agree to the model's own construction precision. Re-answering that specific
// question at snapTol — res.Sew(), this project's OWN pre-existing "close a gap between independently-
// derived geometry" tolerance (kernel/geom/resolution.go, documented "deliberately generous") — makes
// the foot correctly recognized as on-loop, so no bridge/chain is built past it at all: byte-identical
// in SHAPE to every other on-loop torus-arm case in the corpus (the docstring's own "the torus side of
// P5" precedent). tol stays untouched everywhere else (loop selection, the consumed-vertex/repeated-point
// guards, the bridge curve's own exactness certificate) — this is a two-line semantic change (which
// tolerance answers which question), not a blanket loosening.
//
// ★ W3 GREENS outright on this fix alone (TestW3TorusArmMiterPerFace). W4 does NOT: fixing the
// reconciliation lets it reach a fully-assembled body for the first time, which surfaces a SECOND,
// entirely unrelated defect — miterSeamBottomCyl/nearestCircleRoot (fillet_miter_curved_cylouter.go)
// picks the wrong one of up to four torus∩cylinder contact-circle crossings for this corner's unusually
// long second edge, so the sampled seam's LAST chord (station 7 → sBot) jumps 0.85·r off the torus
// surface — a genuine fold, invisible to Validate, caught by TestEveryLoopSegmentLiesOnItsFace. This is
// NOT the near-boundary residual this fix closes; it is upstream, in the seam sampler this A1 slice does
// not own (R3/R4 territory per the wave map — "the single largest, most mathematically dense file in the
// miter subsystem"). Two things landed against it: (1) walkCurvedSeam now seeds each interior station's
// branch bias from the PREVIOUS accepted sample (a continuation-method fix) instead of the fixed corner
// vertex — a genuine, independently-verified improvement (removes a discontinuous MID-walk jump) that
// does not by itself change either case's worst residual, since the dominant defect is the LAST chord,
// not a mid-walk one; (2) curvedMiterSeamOffSurface (fillet_miter_seam_offsurface.go), a new do-no-harm
// floor scoped to curvedMiterBody, so W4 declines HONESTLY (TestW4DeclinesOnSeamOffSurfaceDefect) rather
// than ship the fold. A same-shaped fix to miterSeamBottomCyl's OWN branch pick (seed from sTop instead
// of vp) was tried and REVERTED — it picks a different crossing that miterOuterRail/curvedHostArc cannot
// build a contact rail to, a second failure mode in code this fix does not own; it needs a dedicated
// investigation, not a rushed one-line swap.
//
// tolblend_simple/C6, C8 (R1's independent hit on fillet_far_runout.go, "multi-face-far-vertex") do NOT
// flip: measured directly (TestC6C8DeclineForAnUnrelatedReason below), both decline on "cannot round an
// edge bordering a curved (geom.BSplineSurface) face" — a BSpline-host capability gap this fix does not
// touch, confirmed by the corpus scoreboard (tolblend_simple PASS stayed at 5, unchanged from baseline).

// a1ArmPerFaceTol is W3's per-face relative budget: mesh quantization only (W3's shipped face count
// matches DRAWEXE's exactly, 8-for-8, so no modeling-difference allowance is needed — the worst
// rank-paired face, the r=0.2 torus arm, differs by 0.19%).
const a1ArmPerFaceTol = 3e-3

// TestW3TorusArmMiterPerFace pins simple/W3 on all EIGHT faces DRAWEXE ships (`blend result s 0.2 s_2
// 0.2 s_6` on CCV_1_c12gsf.rle, DRAWEXE 8.0.0 `explode result F` + `sprops result_i 1.e-12`, largest
// first): the four untouched planar walls (2.14444, 2.10548, 1.99142, 1.9536), the untouched r=1 boss
// wall and the flat end cap (1, 0.583773), and the two arm faces (0.438968 cylinder, 0.284221 torus
// Radii 1.2/0.2 — R+r on host R=1, the notch-wall sign R5 fixed).
func TestW3TorusArmMiterPerFace(t *testing.T) {
	body := gridCaseBody(t, corpusRecord(t, "simple", "W3"))
	assertShippedPerFaceAgainstDrawexe(t, drawexeFaceCase{
		name:       "simple/W3",
		drawexe:    []float64{2.14444, 2.10548, 1.99142, 1.9536, 1, 0.583773, 0.438968, 0.284221},
		totalArea:  10.5019,
		perFaceTol: a1ArmPerFaceTol,
	}, body, nil, 1e-3)
	assertBodyIsWatertightAndSimple(t, body, "simple/W3")
}

// TestW4DeclinesOnSeamOffSurfaceDefect pins simple/W4's honest decline: the near-boundary reconciliation
// this A1 slice owns now builds cleanly (no "assembled weld self-crosses/retraces" — R5's guard is
// silent), but curvedMiterSeamOffSurface catches the SEPARATE, unresolved seam-bottom branch-selection
// defect (see file doc) and declines before a folded torus face ships.
func TestW4DeclinesOnSeamOffSurfaceDefect(t *testing.T) {
	assertMiterHonestlyDeclines(t, "simple", "W4", "off its own face's surface")
}

// TestC6C8DeclineForAnUnrelatedReason pins tolblend_simple/C6, C8's decline reason: R1 hit the same
// fillet_far_runout.go file as R5/A1 and characterized both as "multi-face-far-vertex", but measured
// directly they decline much earlier, on an edge bordering a B-spline-host face — a capability gap this
// A1 near-boundary reconciliation does not touch. Confirms the A1 fix's scope: it greens W3 exactly,
// nothing else in the corpus (the scoreboard's tolblend_simple PASS count is unchanged at 5).
func TestC6C8DeclineForAnUnrelatedReason(t *testing.T) {
	for _, name := range []string{"C6", "C8"} {
		r := corpusRecord(t, "tolblend_simple", name)
		body, err := importInput(filepath.Join(CorpusFixtureDir(), r.InputStep))
		if err != nil {
			t.Fatalf("tolblend_simple/%s: import failed: %v", name, err)
		}
		sets, ok := scoreLocate(r, body)
		if !ok {
			t.Fatalf("tolblend_simple/%s: could not locate picked edges", name)
		}
		_, filletOK, reason := runFillet(body, sets)
		if filletOK {
			t.Fatalf("tolblend_simple/%s: expected an honest decline (BSpline-host capability gap), got a PASS", name)
		}
		want := "bordering a curved (geom.BSplineSurface) face"
		if !strings.Contains(reason, want) {
			t.Fatalf("tolblend_simple/%s: decline reason %q does not contain %q — the A1 fix may now reach "+
				"a different (and un-reconciled) capability path here", name, reason, want)
		}
	}
}
