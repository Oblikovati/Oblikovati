// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"testing"

	"oblikovati.org/kernel/topo"
)

// TestByteIdentityFingerprints pins the canonical mesh fingerprint + volume of the prior-green corner
// bodies the CN-C8 brief names for do-no-harm (B3, N7, C2, C6, D5, D9, E4) plus C8 itself. CN-C8 changes no
// shared weld/retrim/geometry code (it is verification + a forensic receipt), so every one of these must
// stay bit-identical to the base worktree 6123169d. These VALUES were captured on base 6123169d and re-run
// on the CN-C8 HEAD; if a future shared-code edit perturbs any of them, this fails loud with the delta. The
// RAW volumes match the brief's references exactly (B3 190756.470897507, N7 963883.383205631).
func TestByteIdentityFingerprints(t *testing.T) {
	for _, tc := range byteIdentityPins() {
		t.Run(tc.name, func(t *testing.T) {
			fp := bodyMeshFingerprint(pinnedBody(t, tc.grid, tc.name))
			if fp.Hash != tc.hash || fp.Triangles != tc.tris {
				t.Fatalf("%s fingerprint drifted: hash=%#x tris=%d, want hash=%#x tris=%d (shared geometry changed)",
					tc.name, fp.Hash, fp.Triangles, tc.hash, tc.tris)
			}
			// RAW volume compared at rel volTolFor(name) — 1e-9 for a well-conditioned body (re-run noise
			// ~1e-13, ~4 decades of margin). A body with heavy signed-tetra cancellation (M4's off-origin
			// bore) is order-sensitive at ~1e-4: TessellateBody yields the SAME triangle SET (hash+tris stay
			// EXACT) in a map-order-dependent SEQUENCE and FP addition is non-associative — a pre-existing
			// kernel property, not a fillet defect (N3/N9, centred boss geometry, stay bit-stable at 1e-9).
			if rel := relErr(fp.Volume, tc.vol); rel > volTolFor(tc.name) {
				t.Fatalf("%s volume %.12f != base %.12f (rel %.3g, tol %.1g)", tc.name, fp.Volume, tc.vol, rel, volTolFor(tc.name))
			}
		})
	}
}

// volTolFor is the raw-volume relative tolerance for a pinned body: the bit-stable 1e-9 default, loosened
// for the two cases whose signed-tetra volume sum is TessellateBody-order-sensitive (their mesh — hash+tris
// — is still pinned exactly): M4's off-origin bore (~1e-4) and D3's host-sphere near-degenerate zone winding
// (~1e-3 swing; 3e-3 leaves ~3× margin over the observed spread).
func volTolFor(name string) float64 {
	switch name {
	case "M4":
		return 5e-4
	case "D3":
		return 3e-3
	}
	return 1e-9
}

// fingerprintPin is one body's captured order-independent fingerprint. grid is the corpus grid the case
// lives in ("" means "simple"); it disambiguates cases whose name collides across grids (e.g. A2/A3 exist
// in BOTH simple and bfuseblend).
type fingerprintPin struct {
	name string
	vol  float64
	tris int
	hash uint64
	grid string // "" means the "simple" grid; set for cross-grid name collisions (A2/A3 in bfuseblend)
}

// pinnedBody resolves a pinned case's result body in its grid (defaulting to "simple"), running the real
// fillet feature — the grid-aware counterpart of caseResultBody.
func pinnedBody(t *testing.T, grid, name string) *topo.Body {
	t.Helper()
	if grid == "bfuseblend" {
		return bfuseblendResultBody(t, name)
	}
	return caseResultBody(t, name)
}

// byteIdentityPins are the base-6123169d fingerprints, captured by running bodyMeshFingerprint in a git
// worktree at 6123169d and confirmed bit-identical on the CN-C8 HEAD (see cnc8-report.md §byte-identity)
// and again on the CN6 HEAD (cn6-report.md §byte-identity; CN6 is a test-harness-only change).
//
// CROSS-PLATFORM RISK (record only — do NOT fix here; for the pre-PR cross-platform-stability pass,
// CLAUDE.md hard-coded-values rule): these are hard-coded 64-bit hashes of quantized tessellated
// coordinates. arm64 FMA/libm rounding could shift a vertex across a 1e-6·scale quantization boundary
// and flip a hash → a FALSE red off linux/amd64. Fine for the corpus's linux/amd64 runs. The pre-PR
// pass must re-capture on arm64 and either confirm identical or widen the quantum / drop the hash pins
// to the (rounding-robust) volume+triangle-count pins there. No arm64 re-capture is attempted here.
func byteIdentityPins() []fingerprintPin {
	return []fingerprintPin{
		{"B3", 190756.470897506602, 31274, 0x2dff54b187389df4, ""},
		{"N7", 963883.383205630700, 31472, 0x8bc99405bb8efa01, ""},
		{"C2", 510191.885601512506, 121529, 0x1b3bc10e4d60136e, ""},
		{"C6", 1559718.455869767116, 121634, 0xe1e51a060a4c1b02, ""},
		{"D5", 3432799.391629283316, 64666, 0x7ef4f7950c6a410d, ""},
		{"D9", 10302524.214111814275, 81106, 0x2eecacfc3279f7d6, ""},
		{"E4", 3460484.448986444157, 59660, 0xf9912504516fdf8a, ""},
		{"C8", 64858.504095408265, 39992, 0x66606a97c42af3b2, ""},
		// The 8 single-arm curved-RUNOUT greens (R1 fb1b0ca4 · R2a ac0249e9 · R2b 3f09eafe · R3
		// 455b9543): B6/C9/C1 perpendicular, M7 plane-host inner-loop, C5/D8 reflex 270° major-sector,
		// D4/E3 oblique cap. Captured on the R3 HEAD 455b9543; they lock the whole single-arm runout
		// construction so any later slice that touches fillet_curved_single_runout*.go fails loud if it
		// perturbs a runout body. Same cross-platform-risk caveat as above applies.
		{"B6", 585938.532761026, 2564, 0x98fea16cb0b47c11, ""},
		{"C9", 74878.585980316, 33282, 0x83be9eb887bda640, ""},
		{"C1", 516328.521818100, 91108, 0x367226063c9789b0, ""},
		{"M7", 1070772.658316682, 1556, 0xbda4ca6e02f5e3fa, ""},
		{"C5", 1565080.392898256, 67716, 0x481e407c7b323a88, ""},
		{"D8", 10309699.178065298, 46272, 0x47fb6c5622612116, ""},
		{"D4", 3434287.473928707, 23974, 0x34f745da81d00eaf, ""},
		{"E3", 3478295.925438132, 52086, 0x8309a22efbfc928f, ""},
		// The convex CLOSED cone-cap rim band (Miter-A1 / J1): a full torus band welded through the
		// host-agnostic rim rebuild (fillet_curved_closed_rim.go). Captured on this HEAD; it locks the
		// closed-band assembly so any later slice that touches the demux or the rim rebuild fails loud if
		// it perturbs the J1 body. Same cross-platform-risk caveat as above applies.
		{"J1", 3661547.474067254923, 69116, 0x33544bb82303b62c, ""},
		// E7: the convex latitude-cut Torus∧Plane rim welded through the single-arm runout (torusArmEdge /
		// fillet_torusarm.go). Captured on this HEAD; it locks the E7 arm+host-torus retrim so any later slice
		// that perturbs the torus-host arm or its contact circle fails loud. Same cross-platform-risk caveat.
		{"E7", 2027042.935824126238, 82754, 0x9a211786edd20722, ""},
		// The CONCAVE Cylinder∧Plane single-arm runout (Group A / N3·M4·N9): a reentrant axis-parallel line
		// edge whose fillet ADDS the fill wedge (fillet_arm_concave.go — void-side arm + concave GROW retrim
		// of both hosts and both caps). Captured on this HEAD; they lock the concave arm surface, its
		// void/foot root gate, and the concave grow retrim so any later concave slice (3-pick corners, torus
		// arms) fails loud if it perturbs an N3/M4/N9 body. Same cross-platform-risk caveat as above applies.
		{"N3", 1047315.289309623, 1808, 0x2f37905510c0ca77, ""},
		{"N9", 1111539.398763901, 10024, 0x31a0f264cc6c4fba, ""},
		// M4 is the reentrant BORE (axis at (50,−10), plane through the axis): its result mesh is bit-exact
		// (hash+tris pin it), but its whole-body signed-tetra volume sum is TessellateBody-order-sensitive
		// at ~1e-4 (the off-origin bore's large cancellation, see volTolFor). So its raw volume is pinned to
		// the mid-spread value with a loose 5e-4 tolerance; N3/N9 (centred) stay bit-stable at 1e-9.
		{"M4", 1002658.6, 788, 0xd0bbc06232535594, ""},
		// J2: the large spherical ZONE reaching an enclosed pole (psphere -90..45), filleted at r=10. Its
		// sphere face used to mesh the WRONG (small north cap) region — whole-body area 8525 — because
		// sphereCapFan's newellUnit axis is biased by the seam+pole samples of a pole-reaching zone;
		// sphere_zone_mesh.go rebuilds the fan on the rim circle's exact normal + the pole vertex, greening
		// J2 (area 30620.3, deps 0.01) for the first time. Captured on this HEAD; it locks the sphere-zone
		// fan so any later slice that perturbs the J2 body fails loud. Same cross-platform-risk caveat.
		{"J2", 492546.204479767184, 165886, 0xd8868b57405355f7, ""},
		// A2/A3 (bfuseblend): the CONCAVE closed sphere/cone cap rim rounds into a full torus COVE band —
		// the concave dual of J1 (fillet_curved_closed_rim_concave*.go: external-tangency R+r arm, concave
		// contact circles, winding-flipped band, outward-growing plate hole). Captured on THIS HEAD; they
		// lock the concave cove-band assembly so any later concave slice (the S2/S5 cove-onto-sidewall
		// follow-up) fails loud if it perturbs an A2/A3 body. Same cross-platform-risk caveat as above.
		{"A2", 17169519.448758263141, 134668, 0x172eb5eb3a475c53, "bfuseblend"},
		{"A3", 15606148.623794555664, 305166, 0xf29512989e349b62, "bfuseblend"},
		// E1/E2 (simple): the 90° sphere-sector corner fillets whose CURVED survivor wall (a sphere
		// meridian rim) was chorded by transformLoop's ENDS branch, collapsing the sphere face ~in half
		// (curved-host-collapse-rootcause.md). The survivor-rim carry (fillet_survivor_rim.go) trims the
		// rim to its retained sub-arc, greening both for the first time (E1 area drift 0.37%, E2 0.47%).
		// Captured on THIS HEAD; they lock the survivor-arc carry so any later slice that perturbs the
		// planar end-corner path fails loud. Same cross-platform-risk caveat as above applies.
		{"E1", 3481520.754844990, 12276, 0x98e05724da9bfa49, ""},
		{"E2", 3479438.313038055, 18196, 0xf2f6195ae0b5e804, ""},
		// D3 (simple): a 270°-latitude/90°-longitude sphere-sector trihedral fillet, greened as a bonus by
		// the quadrant-gated survivor-rim carry (E1/E2/D3 share the mechanism; its 118–146° sphere meridians
		// are carried, its ≤π/2 rims stay chorded). Its mesh (hash+tris) is bit-stable, so the AREA gate is
		// stable at 0.29%, but — like M4 — its raw signed-tetra VOLUME sum is TessellateBody-order-sensitive
		// (~1e-3 swing, the host-sphere zone's near-degenerate winding); its volume is pinned to the mid-spread
		// with the loosened volTolFor(D3) tolerance, the mesh pinned exactly. Same cross-platform caveat.
		{"D3", 3407560.0, 17274, 0xe4772e04bfb33d22, ""},
		// B1/B9 (simple): already-green PLANAR corner fillets whose curved survivor rims are ≤π/2 (62–67°),
		// so the survivor-rim carry (fillet_survivor_rim.go) keeps their base chord byte-for-byte. An earlier
		// arc-carry (82a64cbb) silently drifted them WITHIN deps (B1 0.0071%→0.0499%, toward-wrong) — caught
		// only by adversarial fingerprinting, not by the pass/fail gate. Pinned here to lock the exact victims
		// so any future shared-planar-path edit that re-drifts them within tolerance fails loud.
		{"B1", 192177.835243989, 30206, 0xa0cfc51a98aea8ab, ""},
		{"B9", 517216.198381525, 16130, 0x92904b222d8b4115, ""},
		// L1/L7 (simple): the P1 dihedral corner-setback greens (fillet_corner_setback.go). L1 is the tracer
		// (4 orthogonal concave dihedral miters on a box+boss); L7 is a 10°-rotated boss whose miter corners
		// are still θ=90° in 3D, so the same orthogonal setback closes it. Both were RED (reflected-seam
		// over-keep) before P1 and are pinned here on the P1 HEAD to lock the concave-miter seam re-sample +
		// host re-trim so any later corner slice (P2 sphere octant, P3 torus, P4 non-orthogonal) fails loud if
		// it perturbs an L1/L7 body. Same cross-platform-risk caveat as above applies.
		{"L1", 1064907.332517385, 10084, 0xfc5957778e077e81, ""},
		{"L7", 1064002.621311044, 10174, 0x8c0079d3d630ccb5, ""},
		// N5 (simple): the ONE previously-GREEN case the P1 pass legitimately RE-WELDS — a rotated boss with
		// two orthogonal (dot=0) concave dihedral miters (L7's family), green ONLY by tolerance at base (area
		// rel 0.3337%, reflected-seam over-keep). The setback moves it TOWARD OCCT (rel 0.1043%) and it stays
		// watertight (valid+holes+solid, every edge 2-incident, fold-free). It is a green→green within-deps
		// change invisible to the pass/fail gate (the survivor-arc lesson), so it is pinned HERE on the P1
		// HEAD — captured base→HEAD-verified as the sole changed green across ALL SIX grids — locking the
		// re-weld so a future shared-corner-path edit or a P2–P4 slice cannot silently re-drift it. Same
		// cross-platform-risk caveat as above applies.
		{"N5", 1046146.284621348, 56914, 0xb27cf1e52c17e2ee, ""},
		// K6/L4 (simple): the P2 trihedral corner-setback greens (fillet_corner_setback.go). Each is a
		// box − pocket whose single trihedral corner joins THREE CONCAVE fillets at three mutually-orthogonal
		// planar faces; both were RED (K6 +1.19%, L4 +1.38%) because solvePlanarBlend placed the corner
		// sphere on the MATERIAL side (the reflection of the true void sphere through the vertex), twisting
		// the three bands and over-keeping host tabs. Flipping the sphere to the VOID side retracts each band
		// by r to the void tangent circle and re-trims the three hosts. K6/L4 are the ONLY bodies whose mesh
		// changes across ALL SIX grids (base 4687de4e → P2 HEAD; verified sweep — no green→green re-weld this
		// slice, unlike P1's N5). Pinned here to lock the void-sphere flip so any later corner slice (P3
		// torus, P4 non-orthogonal) fails loud if it perturbs a K6/L4 body. Same cross-platform-risk caveat.
		{"K6", 959144.890948543, 9898, 0xfb501e5efc54048e, ""},
		{"L4", 945198.518972720, 9676, 0x6852a95e3f8b3508, ""},
		// K9/M2 (simple): the P3 mixed-sense trihedral corner-setback greens (fillet_corner_torus.go). Each
		// is a box+boss (K9) / box+cyl (M2) whose single trihedral corner joins 2 CONCAVE + 1 CONVEX fillets
		// at three mutually-orthogonal planar faces; both were RED (K9 +1.17%, M2 +1.02%) because solveBlend
		// forced a SPHERE (area 274.35) where the rolling ball pivots AROUND the convex edge and sweeps a
		// TORUS (axis = the convex fillet axis, major R=2r=10, minor r=5, patch area (25π/2)(π−1)=84.10). The
		// pass builds the torus patch, retracts the 3 bands to its 4 contact arcs and re-trims the shared host
		// plane along the top-contact arc. K9 whole-body 66160.6 (=OCCT to the digit); M2 64088.4. Pinned on
		// the P3 HEAD to lock the torus corner so any later corner slice (P4 non-orthogonal) fails loud if it
		// perturbs a K9/M2 body. Same cross-platform-risk caveat as above applies.
		{"K9", 1064212.575467, 8860, 0x8b7d80c80805b0fb, ""},
		{"M2", 1028376.359922, 16636, 0x1b7133a8b9fdd976, ""},
		// L6 (simple): the ONE previously-GREEN case the P3 pass legitimately RE-WELDS — a 10°-rotated box
		// corner that is ALSO a mixed-sense trihedral (2 concave + 1 convex on 3 orthogonal planar faces),
		// green ONLY by tolerance at base (sphere, area rel 0.9033%). The torus re-weld moves it EXACTLY onto
		// OCCT (rel 0.0000%, whole-body 65867.9) and stays watertight (valid+closed+holes+solid, every edge
		// 2-incident, fold-free, torus patch area 84.09). It is a green→green within-deps change invisible to
		// the pass/fail gate (the survivor-arc / N5 lesson), captured base 6dd604be → P3-HEAD-verified as the
		// sole real changed green across ALL SIX grids (D3/D7's volume-only diff is hash-identical
		// TessellateBody summation-order noise, positions unchanged). Pinned here so a future shared-corner-
		// path edit cannot silently re-drift it. Same cross-platform-risk caveat as above applies.
		{"L6", 1063051.564952, 8864, 0x72af706c3b9fc8f1, ""},
		// A8/A6 (simple): the P4 convex-wedge run-off setback greens (fillet_convex_wedge_setback.go). Each
		// is a WEDGE (polyhedral prism) whose three CONVEX edges meet at one same-sense trihedral corner. The
		// corner (material-side octant sphere + corner-side setback s=r·cot(θ/2)) is already OCCT-exact via
		// the shared blend path; the miss was an OBLIQUE band running off onto an un-filleted face, whose
		// rail overshot the run-off plane by a tab (A8 +2.75%, +742 area). The pass clips that rail at its
		// far-plane pierce (setbackTrihedralCorner) ONLY when it overshoots and the body is all-planar. A8
		// was RED; A6 is the ONE previously-GREEN sibling legitimately RE-WELDED TOWARD OCCT (rel +0.79%→
		// −0.03%, whole-body 30056.8, watertight+fold-free) — a green→green within-deps change invisible to
		// the pass/fail gate (N5 lesson). Both are the ONLY real changed greens across ALL SIX grids vs base
		// e544759a (D3/M4's volume-only diff is hash-identical TessellateBody summation-order noise). Pinned
		// so a later corner slice fails loud if it perturbs an A8/A6 body. Same cross-platform-risk caveat.
		{"A8", 191752.356135391, 12326, 0x5fdf710707588ca3, ""},
		{"A6", 327951.529353541, 25000, 0xb8b3f0fc0125b58, ""},
		// I3 (simple): a straight LineSegment fillet between a planar ANNULAR-SECTOR host and a triangular
		// end-cap. transformLoop's `subs` branch (the A/B tangent-point pull-back) hard-coded the LEAVING
		// survivor edge's curve to nil, CHORDING the sector's r=300 outer rim — slicing the host plane
		// 38270→14084 (−63%) and folding the neighbour cone (conformCylConeFaces) to whole-body +2.55%
		// (i3-recon-rootcause.md). The subs-branch survivor-arc carry (fillet_survivor_rim.go: addSubstVertex
		// + trimCarriedSubArcs, model-scale-relative segment gate so N5's minor boss rim stays byte-identical)
		// keeps the rim, and a conformance do-no-harm guard (conformance_repair.go, never let the cyl/cone
		// re-mesh ADD folds) stops the cone inflation — greening I3 for the first time (whole-body 100120,
		// rel 0.82%; sector plane recovered to 38270). Captured on THIS HEAD; it locks the subs-branch carry
		// so any later planar-retrim slice that re-chords the rim fails loud. Same cross-platform-risk caveat.
		{"I3", 980429.069826266, 19046, 0x35414382a0c5c37c, ""},
		// I9 (simple): the rim-fillet pick gate widen (fillet_rim.go's isClosedCircularEdge) — a plain
		// solid cylinder rim whose picked edge is a closed geom.Arc3d (SweepAngle≈2π), the shape every
		// STEP-imported full circle actually has (kernel/exchange never emits geom.Circle). Before the
		// widen, loneRimPick/resolveRim's geom.Circle-only gate rejected the edge, the pick fell through
		// to loneArcPick, and cylSideEdgeAt declined the self-closed vertex — FAIL(faulty) "arc end is
		// not a cylinder/side tangent vertex". The widen routes it to the SAME FilletCylinderRim band
		// path a procedural geom.Circle rim already uses, greening I9 for the first time (area 61438.17,
		// rel 0.0013%, deps 0.01). Captured on THIS HEAD; it locks the Arc3d-rim path through the closed-
		// rim band assembly so any later slice that touches fillet_rim.go/sphere_zone_mesh.go's shared
		// isClosedCircularEdge predicate fails loud if it perturbs the I9 body. Same cross-platform-risk
		// caveat as above applies. See .superpowers/sdd/rim-arc3d-widen-report.md.
		{"I9", 1171624.810038584052, 67580, 0xab1686e36b3b9396, ""},
		// S6/S9/T3 (simple): the Group-A one-boss setback greens (fillet_setback_extract.go /
		// fillet_setback_close.go single-boss branch, #2007). Each is a box + ONE crossing boss whose
		// footprint used to protrude past the shrunken host outer loop (HolesContained=false — the malformed
		// B-rep poison pill the do-no-harm baseline left). The single-boss tiling (2 plain cyl wings + one
		// central run-out patch that absorbs the footprint, boss WALL kept intact, both edge faces re-clipped
		// single-loop) welds each to a watertight, hole-contained SOLID.
		//
		// RE-CAPTURED on the run-out envelope slice (.superpowers/sdd/runout-envelope-report.md), which
		// legitimately CHANGED all three bodies. The run-out patch is now the exact rolling-ball envelope
		// (an exact-station canal loft), not a Coons fill: coons4-audit.md measured that fill 9.0% of r
		// (S6) / 11.6% (S9) / 9.7% (T3) away from OCCT's own surface on deep-interior samples, and it now
		// reads 0.007% / 0.022% / 0.014%. The pInner rail is the receded tangency CONTACT LOCUS instead of
		// a straight line at the plain fillet's contact (S6's front plane went 360.000 → 360.916 against
		// OCCT's 360.919), and every boss's host-side footprint arc is chorded span-proportionally
		// (hostArcDensifies, now unconditional). Whole-body area: S6 +0.074% → +0.039%, S9 +0.267% →
		// +0.073%, T3 +0.114% → +0.043%. Re-captured ONLY after the per-face DRAWEXE reconciliation
		// confirmed every face moved toward the oracle. Same cross-platform-risk caveat as above.
		{"S6", 18039.724864028, 15982, 0x18a1d92e1931c91b, ""},
		{"S9", 115876.963113864, 89826, 0x93b72952c31ab329, ""},
		{"T3", 130529.141817073, 80716, 0x1ae414df7f42fdd8, ""},
		// U3 (simple): the Group-B obstacle-path dip-detection green (fillet_obstacle_detect.go's
		// dipArcOrder, #2007). U3 is a box + ONE oblique/y-elongated EllipticalCylinder mid-span boss — the
		// SAME shape as the already-green T6 (oblique EllipticalCylinder) — but dipsPast was handed the
		// crossings in unconditional ascending rim-index order, which happened to select the 53-of-64-sample
		// BULGE arc instead of the true 11-sample dip arc (the one wrapping through rim sample 0, the curve's
		// arbitrary t=0 seam point); dipArcOrder now picks whichever of the two crossing-bounded arcs is
		// actually shorter — the genuinely local mid-span excursion — before both the dip test and the
		// downstream rebuild (buildObstacleFeature/buildNotchedHost/mergeObstacleRim, all keyed on the SAME
		// nodes[0]→nodes[1] "Task 2" dip-range convention) consume it. Routes through the already-proven T6
		// obstacle rebuild (rebuildableTube already admits EllipticalCylinder): watertight, hole-contained
		// SOLID at area +0.116%. Captured on THIS HEAD (u3-dipspast-report.md); locks the obstacle rebuild so
		// a later slice (Group C U4, dual-host) fails loud if it perturbs the U3 body. Same cross-platform-
		// risk caveat as above.
		{"U3", 17894.503322426473, 9452, 0xc38f33331071f8ae, ""},
		// U4 (simple): the Group-C dual-host multi-rail corner-blend green (kernel/ops/fillet_obstacle_dual*.go,
		// #2007 Group C — the U4-5 weld). U4 is a box + TWO bosses (Cylinder r8 on y=−20, oblique
		// EllipticalCylinder on x=10) whose footprints BOTH dip into ONE r=5 convex-edge fillet (qualifying==2),
		// the dual-host composition neither the single-boss setback tiling (S6/S9/T3) nor the dipArcOrder
		// obstacle fix (U3) reached. The dual assembler welds 2 notched hosts + 2 walls split at every panel
		// seam + 2 cylinder wings + 4 corner-blend panels (2 coons4 slivers, 2 exact-station canal cores) into a
		// watertight, hole-contained SOLID matching the DRAWEXE 16-FACE / every-face-WIRE:1 topology at whole-
		// body area 6583.29 (rel 0.001%), every fillet panel tessellating fold-free to its per-face oracle
		// (3.039 sliver / 30.334 core) in production. Captured on THIS HEAD (u4-5-report.md); bit-stable at
		// 1e-9. Locks the dual-host weld so any later slice that perturbs the U4 body fails loud. Same
		// cross-platform-risk caveat as above applies.
		{"U4", 32647.755284293769, 23848, 0xf588dd3b13096eb9, ""},
		// K1/Z1 (simple): the CONCAVE bore-lip rim mirror (fillet_rim_concave.go's rimWithCapOrientation).
		// K1 is a genuine bore lip (cylinder radius 30, r=5): the plate material sits OUTSIDE the bore, so
		// the rolling ball is tangent to the wall from the material side at R+r=35 (torus major 35, pinned
		// by TestConcaveRimBuildsRPlusRTorus below), not R−r — the CONCAVE mirror of solveRim's convex R−r.
		// Z1 (cylinder radius 10, r=1) turned out NOT to be a bore lip at all: its rim is a plain CONVEX
		// rim (material INSIDE the cylinder, like I9), but its cap face is stored bottom-up
		// (capF.Reversed()==true — a tower standing on its base, not a boss capped from above), so
		// solveRim's raw pl.Normal() (which every prior corpus convex rim happened to already read as
		// outward) placed torusCenter on the WRONG side of the cap and failed the R−r probe for a reason
		// unrelated to concavity. The SAME cap-orientation fix (outwardPlaneNormal, already used by the
		// S2/S5 concave arm builder) resolves Z1 at R−r=9 (torus major 9), not R+r — see
		// TestConvexRimReversedCapBuildsRMinusRTorus. Both were FAIL(faulty) "concave rim fillet (a bore
		// lip) is not yet supported" before this fix. Captured on THIS HEAD; they lock the concave-rim
		// weld + the cap-orientation retry so any later rim slice fails loud if it perturbs a K1/Z1 body.
		// Same cross-platform-risk caveat as above applies.
		{"K1", 716214.454630634282, 67600, 0xcb1edbf4a4562d86, ""},
		{"Z1", 7840.004567698298, 34808, 0x84c49b798620dc55, ""},
		// N1 (simple): the FIRST R+r bore/notch-wall trihedral corner green (corner-blend-weld Pieces
		// 1+2). N1 is box − r20 cylinder notch at a corner, r=5 fillet on 3 edges. The engine used to
		// hard-code the rolling-ball offset at R−r (a convex boss cap) and mirrored the whole corner into
		// the removed void — the cylinder arm's plane contact landed off the host face and the weld
		// declined. Piece 1 added the ε-gated R+r sense (corner sphere (75.505,5,95), torus major 25 —
		// OCCT-exact); Piece 2 fixed the assembly point-welder so a pass-through face welds to its rebuilt
		// far-runout neighbours at unchanged corners (the fillet spreads onto the box's OWN faces, mixing
		// ided pass-through corners with anonymous rebuilt ones — addID now adopts an anonymously-claimed
		// cell). Result: a watertight 11-FACE solid matching the DRAWEXE oracle (area 58091.81, rel 1.5e-6
		// of 58091.9). Captured on THIS HEAD; it locks the bore-corner weld so any later piece (L9 plane
		// retrim, the concave cluster) fails loud if it perturbs the N1 body. Same cross-platform-risk
		// caveat as above applies.
		{"N1", 967610.719379026, 23816, 0x5f18c6f4452d68a8, ""},
		// L9 (simple): the SECOND R+r bore/notch-wall corner green (corner-blend-weld Piece 3). L9 is box
		// − r30 QUARTER-cylinder notch, r=5 fillet on 3 edges. Its bore torus arm (major R+r=35) runs out
		// against a flat RADIAL notch face, and — because the R+r cap contact reaches r PAST the rim — the
		// fillet's far cross-section EXTENDS the adjacent flat faces OUTWARD past the rim ((50,20)→(50,25)),
		// the opposite of a convex inward corner bite. Piece 3 adds that outward extension: the corner
		// far-path grows a collinear survivor edge to the rail landing (extendStraightSegToLanding), and the
		// far-runout bite grows the loop through the off-rim tip (boreExtendBite) — both no-ops for a convex
		// green, so byte-identical. Result: a watertight 13-FACE solid matching the DRAWEXE oracle (area
		// 61832.36, rel 5.8e-7 of 61832.4). Captured on THIS HEAD; it locks the bore far-cap extension so any
		// later piece fails loud if it perturbs the L9 body. Same cross-platform-risk caveat as above.
		{"L9", 958250.799883365, 23018, 0x2eb42ba563f1966e, ""},
		// M8 (simple): the mixed-sense curved-host 2r-torus corner WELD (corner-blend-weld Slice-1b,
		// fillet_curved_mixed_weld.go). M8 is box + boss (R=25), r=5 fillet on 3 edges meeting at one curved-
		// host trihedral vertex: ONE convex Cyl∧Plane arm + a concave cove torus arm (major R+r=30) + a planar
		// Plane∧Plane arm. No single ball is tangent to the boss wall at both R−r (the convex arm) and R+r (the
		// cove arm), so it is NOT a sphere corner — OCCT builds an analytic 2r-TORUS corner patch (axis = the
		// convex fillet axis, major R=2r=10, minor r=5, C=(55,30.635,105), DRAWEXE-exact). The weld consumes the
		// committed corner arcs (fillet_curved_corner_torus.go): it trims the three arm faces at arcs a/b/c,
		// retrims the two walls (their rails meet at a triple point) + the top plane (closing around arc d), and
		// grows/recedes the far-runout caps into a watertight 14-FACE solid matching the DRAWEXE oracle (area
		// 67959.44, rel 8e-7 of 67959.5; corner-patch mesh 97.63 vs analytic U·r·(R·π/2−r)=97.65). Captured on
		// THIS HEAD; it locks the mixed-sense curved corner weld so any later slice fails loud if it perturbs
		// the M8 body. Same cross-platform-risk caveat as above applies.
		{"M8", 1074028.429945154, 38194, 0x82d94d8c57eed140, ""},
		// W6/W8 (simple) + A1 (bfuseblend): the CONCAVE BOSS-BASE rim family (fillet_rim_concave.go's
		// solveConcaveBossRim). A boss/pin (a cylinder standing on a plate) filleted at its base, where the
		// wall meets the plate in a REENTRANT corner. solveRim's convex R−r round built a watertight solid
		// with the DRAWEXE-exact face count but the WRONG footprint (the round bit the corner INWARD: torus
		// major R−r, plate hole shrinking, wall extended by r) — a STABLE over-size (W6 +3.8%, W8 +5.5%,
		// Property==fine). OCCT builds the concave cove: torus major R+r (W6 25, W8 35, A1 110), the plate
		// hole OPENING to R+r and the wall receding by r. solveConcaveBossRim detects the concave dihedral
		// and offsets the tube centre AXIALLY toward the boss (the direction the wall rises — the one thing
		// the K1 bore-lip mirror got backwards); a cap-fit gate keeps the deep #2012 spillers (R8/W9) on the
		// unchanged solveRim ladder. Result areas now MATCH OCCT to −0.000%/−0.0002% (W6 60998.06 of
		// 60998.1; W8 64789.23 of 64789.3; A1 451532.30 of 451533.0). Captured on THIS HEAD
		// (area-sizing-report.md); they lock the boss-base cove so any later rim slice fails loud if it
		// perturbs a W6/W8/A1 body. Same cross-platform-risk caveat as above applies.
		{"W6", 1013278.147300069, 67596, 0x39c85597736e6fa1, ""},
		{"W8", 1083723.369024332, 67852, 0x2e95a5402d3049f6, ""},
		{"A1", 18223007.902061027, 135180, 0xd04c71795cfaf4cb, "bfuseblend"},
		// F4 (simple): the elliptic-prism vein tracer — a convex STRAIGHT RULING edge where a plane meets a
		// right EllipticalCylinder wall (an oblique-prism side), filleted r=10. Its constant-radius rolling-
		// ball fillet is an EXACT right circular cylinder (the straight spine collapses the canal to a
		// cylinder; fillet_ellipticalarm.go), welded through the single-arm curved runout with the ellipse-
		// aware host retrim (fillet_curved_retrim_ellipse.go). Was FLAT-REFUSED (curvedAdjacentError). Pinned
		// on THIS HEAD so any later slice touching the elliptic arm / ellipse retrim fails loud if it perturbs
		// the F4 body. Same cross-platform-risk caveat as above applies.
		{"F4", 3104847.833035429008, 26988, 0xa242e8277723b567, ""},
		// J6/J8 (simple): the CLOSED elliptic-rim canal band — the elliptic vein's other half. F4's edge was a
		// straight RULING (straight spine → exact right circular cylinder); these are CLOSED rims on the same
		// oblique-extrusion wall, whose rolling-ball spine is a closed NON-ANALYTIC curve, so the fillet is a
		// genuine variable-section canal (fillet_elliptic_rim_canal.go), lofted from exact stations and welded
		// through the shared rim rebuild. Both were FLAT-REFUSED (curvedAdjacentError). Independent geometric
		// receipt for these two values: DRAWEXE `vprops` on OCCT's OWN blend gives 1.56049e6 (J6) and 772435
		// (J8) — ours are −0.0029% / −0.0059% off, i.e. the same solid. (Their AREA carries a documented
		// per-case deviation; see corpus.json — that is an oracle-NUMBER defect, not a geometry one.) Pinned on
		// THIS HEAD so any later slice touching the canal band / rim rebuild fails loud if it perturbs them.
		// Same cross-platform-risk caveat as above applies.
		//
		// RE-CAPTURED once, for the canal band mesher's v-labelling fix (canalRailRow): the band's rail rows
		// were labelled with a v re-derived from chord length ALONG THE RAIL instead of each sample's own
		// (isocurve == surface-v) parameter, which sheared every boundary strip along the ring. Only the FOUR
		// bodies carrying a canal band (J6/J8 here + A7/B1 below) moved; every other pin stayed byte-identical,
		// and each of the four kept its EXACT triangle count (the mesh topology is unchanged — only the
		// mislabelled vertices moved onto their true stations). Receipt: the meshed band area against the band
		// SURFACE's own integral went J6 +0.2417%→−0.0035%, A7 +0.0223%→−0.0030%, J8 +0.0020%→−0.0055%,
		// B1 −0.0034%→−0.0039% — i.e. all four now sit on the physically correct INSCRIBED side at the same
		// ~0.004% faceting deficit, where before J6/A7 were inflated by the shear. All four volumes also moved
		// CLOSER to the DRAWEXE vprops reference.
		{"J6", 1560445.460018513724, 108252, 0x1350ffdad16d2295, ""},
		{"J8", 772389.432451401255, 84988, 0x3294c0c986b48523, ""},
		// A7/B1 (bfuseblend): the CONCAVE members of the same vein — the SAME two shapes as T5/U2 (an
		// elliptic boss / an oblique pipe standing on a plate) but on a plate wide enough that the fillet's
		// foot ring FITS, so the band stays a single closed piece. They green off the geometric convexity
		// gate (fillet_elliptic_rim_spine.go's quadrant probe), which is load-bearing here: the imported
		// elliptic face's Reversed flag mis-classifies these rims. Topology matches DRAWEXE exactly (9 faces
		// / 11 vertices / 17 edges each) and volume to 1e-6 (A7 3.67231e7, B1 1.42995e7 vs 1.42994e7).
		// Re-captured with J6/J8 above for the same canal-band v-labelling fix — see that receipt.
		{"A7", 36723071.024155125022, 110356, 0x988878689a64481e, "bfuseblend"},
		{"B1", 14299454.139366334304, 133748, 0xfb85dc13f23feabb, "bfuseblend"},
		// N4 (simple): the first case welded by the GENERAL corner-weld layer (cornerweld_*.go) — a full r20
		// cylinder standing on a 100³ box's vertical corner, filleted r5 on three edges meeting at one
		// trihedral vertex. It needed two layer stage variants: the CONVEX cap-rim arm's far vertex is a G1
		// SEAM on the boss wall rather than a cap (only the 90° piece of the 270° rim was picked, and the rim
		// continues tangentially), so the arm runs THROUGH it to the end of the tangent chain and splits into
		// one face per host-face span. DRAWEXE 8.0.0 receipt (`restore CFI_e5678fil.rle s ; tscale s 0 0 0 10 ;
		// explode s e ; blend result s 5 s_4 5 s_13 5 s_2`): valid SOLID, 14 faces / 14 wires, sprops area
		// 64287.2, vprops volume 1.04694e6 — ours 14 faces, area 64287.18 (−0.000035%), volume 1046938.9
		// (−0.000109%).
		//
		// RE-CAPTURED when the corner fill became the true rolling-ball CANAL (fillet_curved_mixed_canal.go).
		// The previous pin (1046910.385015469277 / 55856 / 0x8df4717458b48c1d) belonged to the chord-projected
		// coons4 fill, whose corner patch was 27% short of result_5 (59.273 vs 80.733) with the vertical plane
		// over-reading by the near-opposite amount — the whole-body +0.008% was CANCELLATION, not agreement.
		// Every face now reconciles individually: patch 80.875 (+0.18%), vplane 8674.80 (−0.0005%), both torus
		// bands within 0.02%, volume 26× closer to vprops. It locks the layer so any later class builder
		// (O1, H7, …) or executor change fails loud if it perturbs N4. Same cross-platform-risk caveat applies.
		{"N4", 1046938.855499812000, 53496, 0xcee49118703be08, ""},
	}
}
