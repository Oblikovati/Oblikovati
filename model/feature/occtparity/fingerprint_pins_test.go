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
		// single-loop) welds each to a watertight, hole-contained SOLID: S6 (Sphere r13, area +0.074%), S9
		// (Torus(20,5), +0.267%), T3 (Torus(35,10) oblique, +0.114%). S6's sphere host footprint arc is
		// densified span-proportionally (densifyHostArc — one-boss only, so the 2-boss sphere S7 stays byte-
		// identical). Captured on THIS HEAD; they lock the single-boss setback so any later slice (Group B U3,
		// Group C U4) fails loud if it perturbs an S6/S9/T3 body. Same cross-platform-risk caveat as above.
		{"S6", 18044.769951207, 19602, 0x677c6f17793080c, ""},
		{"S9", 116088.833852354, 85614, 0x93ef9087c21bf57a, ""},
		{"T3", 130621.192627563, 91054, 0xfa008dbaf5dd4d2d, ""},
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
	}
}
