// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import "testing"

// THE BOSS/NOTCH RADIAL-SIGN GENERALIZATION AND ITS MESH-DEFECT FLOOR — blend/simple W3, W4 (fixture
// CCV_1_c12gsf.rle: a slot cut INTO a cylindrical boss, r=1, filleted r=0.2 on both slot walls at their
// shared corner).
//
// TWO real, general capability gaps advanced this session (kernel/ops, files this cluster owns):
//
//  1. curvedMiterTorusArm/curvedMiterCylinderArm's CONVEX branches (fillet_miter_concave.go) had
//     hardcoded ε=+1 — the BOSS convention (majorR = R−ε·r = R−r) — with an explicit "bore-miter is a
//     later slice" caveat on both branches. W3/W4's corner is exactly that later slice: the circle-edge
//     arm's DRAWEXE torus is Radii 1.2/0.2 = R+r on host R=1 (ε=−1, a NOTCH wall, not a boss), while the
//     hardcoded-ε=+1 build shipped R−r=0.8 — a wrong analytic surface. Both branches now read ε from
//     cylinderHostRadialSign(e, cyl), the SAME per-wall sign the concave rim arm already used since the
//     W-DH wave's M5 fix, so a boss corner (ε=+1) stays byte-identical and a notch corner (ε=−1) now
//     takes the correct arm. Falsified by mutation: pinning curvedMiterTorusArm's convex branch back to
//     a hardcoded eps=1 reproduces the pre-fix R−r=0.8 torus (DRAWEXE ships 1.2/0.2) — verbatim in
//     wave-report-R5.md.
//  2. sharedRailAnchor's cap-bridge (fillet_miter_curved_weld.go) called capBridgeArc directly, which
//     ONLY builds an Arc3d on a CYLINDER shared face (P5's curved wall) — declining outright whenever
//     the shared face is a PLANE (W3/W4's slot corner: shared=geom.Plane). sharedFaceBridgeSeg now
//     dispatches on the shared face's own geometry (a straight segment on a Plane, exact by
//     construction — both endpoints are guaranteed IN that plane), falling to capBridgeArc unchanged for
//     the Cylinder case. miterTrimChain's OWN far-cap bridge (fillet_miter_curved_hostfaces.go) had the
//     same bug one level down — built against m.shared's geometry when the trim's off-loop foot
//     actually lives on s.run.capping, a DIFFERENT face in general — fixed the same way.
//
// Neither fix touches an already-green case: the eps threading is byte-identical whenever
// cylinderHostRadialSign already reads +1 (every existing boss-cove/boss-rim green — verified via the
// full corpus scoreboard, unchanged outside this file's own cases), and both bridge dispatchers only ADD
// a Plane branch neither had before (a Cylinder shared/capping face still goes through the exact prior
// capBridgeArc call).
//
// With both fixes, W4 (and W3, a smaller sub-case of the same fixture) reach the ASSEMBLED-WELD stage
// for the first time — and there certify as VALID B-rep (Validate: Manifold/Closed/OrientationOK/
// EulerConsistent/HolesContained all true, IsSolid true) and within the corpus's 1% AREA gate. But the
// TESSELLATED mesh is NOT sound: a bridge segment built from an independently-converged near-boundary
// point (the far cross-section runout's own foot, computed elsewhere in fillet_far_runout.go — general
// infra this cluster does not own, outside R5's scope to patch) lands ~2e-5 off the point it should
// exactly coincide with (measured on W4: foot z=1.0000200000000004 vs the adjacent original vertex's
// z=1 exactly). The B-rep welds edge-for-edge regardless (every edge used by exactly two faces, opposite
// orientation — invisible to Validate), but the resulting TESSELLATED loop self-crosses (W4: a whole-
// face-sized pinch, area 1.51684 on a 1.516878-area face) or retraces (W3: an r-sized overlap, 0.2) at
// that residual — a defect the WHOLE-BODY AREA number cannot see: W4's shipped area under the pinch is
// 11.541444 vs DRAWEXE 11.5416 (rel −0.0013%), comfortably inside the 1% gate — the exact "whole-body
// area gates admit WRONG solids" trap this project's own doctrine warns about (per-face DRAWEXE
// reconciliation, not the aggregate, is what would have caught the earlier ε-sign bug too).
//
// miterWeldMeshDefect (fillet_miter_curved_weld.go) is the new do-no-harm floor this session added
// — the SAME two detectors the corpus's own watertightness gate uses (SelfCrossingFaceLoops /
// RetracingFaceLoops), run once more, one layer earlier, on curvedMiterBody's own assembled result — so
// W3/W4 decline HONESTLY instead of certifying the silently-wrong mesh. Falsified by mutation
// (miterWeldMeshDefect's body forced to `return ""` unconditionally, guard disabled): simple/W4 flips
// FAIL(faulty)→PASS with the self-crossing mesh intact (face self-cross area 1.51684, unchanged) at the
// area above, well inside tolerance — the corpus scoreboard's own simple-grid green count moves
// 125→126, caught separately by assertHardenedRollup's exact-count assertion. Reverting the mutation
// restores the honest decline and the 125/141 rollup.
//
// Neither case joins the corpus rollup: the corpus scoreboard stays 125 simple / 141 all-grid (this
// branch's merged baseline), unchanged by this file. The real, upstream-owned residual (the far
// cross-section runout's ~2e-5 imprecision) is the seam for the merge train / fillet_far_runout.go's
// owner: closing it at the source would make miterWeldMeshDefect permanently silent for this family,
// not remove its correctness.

// TestW3W4NotchBossCornerDeclinesOnMeshDefect pins the honest-decline regression for both cases: the
// arm-pair and shared/capping-bridge capabilities now BUILD (no "arms unsupported" / "bridge declined"
// reason), but the assembled weld's own mesh-defect floor catches the residual and declines before a
// wrong solid ships.
func TestW3W4NotchBossCornerDeclinesOnMeshDefect(t *testing.T) {
	assertMiterHonestlyDeclines(t, "simple", "W3", "curved miter: assembled weld retraces")
	assertMiterHonestlyDeclines(t, "simple", "W4", "curved miter: assembled weld self-crosses")
}
