// SPDX-License-Identifier: GPL-2.0-only

package occtparity

// curvedWeldPins are the CURVED-ARM half of byteIdentityPins (see fingerprint_pins_test.go for what the
// pins are, how they were captured, and the arm64 cross-platform caveat that applies to every one of
// them): the curved-arm/miter greens, the single-arm curved run-outs, the closed cone-cap rim bands, the
// torus-host arm, the concave cylinder arms, the sphere zone, and the Arc3d-rim widen.
func curvedWeldPins() []fingerprintPin {
	return []fingerprintPin{
		{"B3", 190756.470897506602, 31274, 0x2dff54b187389df4, ""},
		// ★ RE-CAPTURED by the reverseSegmentCurve locus fix (fillet_orient.go): the shell-orientation pass
		// used to reverse a flipped face's loop segment by RE-FITTING it through three points, which is exact
		// only for a circle and replaced every other rail with a CIRCLE THROUGH ITS ENDPOINTS. Eight pinned
		// bodies carried the result. Measured off-surface residual of the shipped edge, before → after:
		// D4/D5/D9/E4 1.245 → ≤8.2e-11 (the oblique spiric run-out rail, 24.9% of r off its own torus),
		// N7 0.1269 → 2.1e-10, N4 0.0676 → 1.7e-8 (1.4% of r off its own corner patch), F4 0.0108 → 2.0e-11,
		// O1 0.0012 → 1.5e-8. Every changed FACE moved CLOSER to its DRAWEXE per-face oracle — D5's cap plane
		// +0.230% → −0.0008%, its torus arm +0.141% → −0.018%, E4's arm +0.102% → −0.031% (and its 2-edge
		// fold, previously blamed on the CDT, is GONE), N7's torus arm +0.090% → −0.007%, N4's patch mesh
		// +0.177% → +0.052% — so each body's GROSS per-face error dropped (D5 31.20 → 14.93, E4 62.21 →
		// 47.89) even where the whole-body NET moved, which is cancellation unwinding, not a regression.
		// See reverse-segment-fix-report.md. Same cross-platform-risk caveat as above applies.
		{"N7", 963883.358200046350, 31476, 0x23551593d78be76d, ""},
		{"C2", 510191.885601512506, 121529, 0x1b3bc10e4d60136e, ""},
		{"C6", 1559718.567225983134, 139844, 0x613415837489e018, ""},
		{"D5", 3434184.031168804504, 481302, 0x3996c779fd9029cf, ""},
		{"D9", 10320424.304559277371, 458676, 0xe77fec118a9d9098, ""},
		{"E4", 3464194.722067414317, 368074, 0x2db0d8bfb5ab2752, ""},
		{"C8", 64859.163480644333, 103176, 0x2fb676739745a2a0, ""},
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
		{"D8", 10329479.872831704095, 420310, 0x99a3fe794df55d67, ""},
		{"D4", 3435725.927260315977, 437186, 0xed8148b42b860ccd, ""},
		{"E3", 3479448.946229149587, 430000, 0xa042a580a37de75, ""},
		// The convex CLOSED cone-cap rim band (Miter-A1 / J1): a full torus band welded through the
		// host-agnostic rim rebuild (fillet_curved_closed_rim.go). Captured on this HEAD; it locks the
		// closed-band assembly so any later slice that touches the demux or the rim rebuild fails loud if
		// it perturbs the J1 body. Same cross-platform-risk caveat as above applies.
		//
		// RE-CAPTURED (rim-station-report.md), and PROVEN improved, not merely different: its cone face
		// spans rims of r=99.999929 (1024 stations) and r=51.893513 (512), and the seam-bridged band GRID
		// tiled BOTH at 1024 — so the 512-station rim it shares with the torus band cracked the whole way
		// round. Its welded mesh leaked 1536 free edges at PropertyQuality (ops.FreeEdgeCount over
		// CalculateBodyFacets(body,q).Mesh) and now leaks ZERO at BOTH gate qualities. The cone is lofted
		// rim-to-rim instead, so tris drop by exactly 512 (2048 → 1536 on that face, 69116 → 68604 on the
		// body — the arithmetic of one grid row lost, and nothing else moved). Its cone mesh area moves
		// 94648.882214885 → 94648.777612282 against the EXACT frustum closed form π(r1+r2)(L1−L2) =
		// 94649.056895806, i.e. an inscribed deficit of 1.845e-06 → 2.951e-06 — the unavoidable price of
		// honouring the coarser rim's own chords, and 34× inside PropertyQuality's own ~1e-4 promise. The
		// body volume is unchanged to 2.3e-14 (3661547.474067255 → 3661547.474067171).
		{"J1", 3661547.474067171, 68604, 0x4769bd2a8f5353fe, ""},
		// E7: the convex latitude-cut Torus∧Plane rim welded through the single-arm runout (torusArmEdge /
		// fillet_torusarm.go). Captured on this HEAD; it locks the E7 arm+host-torus retrim so any later slice
		// that perturbs the torus-host arm or its contact circle fails loud. Same cross-platform-risk caveat.
		{"E7", 2027042.935824126238, 82754, 0x9a211786edd20722, ""},
		// The CONCAVE Cylinder∧Plane single-arm runout (Group A / N3·M4·N9): a reentrant axis-parallel line
		// edge whose fillet ADDS the fill wedge (fillet_arm_concave.go — void-side arm + concave GROW retrim
		// of both hosts and both caps). They lock the concave arm surface, its void/foot root gate, and the
		// concave grow retrim so any later concave slice (3-pick corners, torus arms) fails loud if it
		// perturbs an N3/M4/N9 body. Same cross-platform-risk caveat as above applies.
		//
		// RE-CAPTURED (planar-retrim-selfcross-report.md), and PROVEN improved, not merely different: all
		// three carried the cap cross-section arc BACKWARDS between its own vertices (matchArcFeet swapped
		// the feet without reversing the curve), so discretizeEdge tiled a doubled-back cap boundary. The
		// TRIANGLE COUNTS are unchanged (1808 / 10024 / 788) — same topology, corrected positions — and each
		// case's per-face areas now rank-pair EXACTLY against DRAWEXE's own blend of the same input:
		//   N3 face#6 10955.6 → 10946.3 (oracle 10946.3), body +0.0143% → −0.00012%;
		//   N9 face#5 1307.31 → 1298.6 (oracle 1298.57), restoring equality with its MIRROR face#4 (which
		//     already read 1298.6 — the two must be equal by the body's own symmetry), body +0.0071% → −0.0063%;
		//   M4 face#1 9752.18 → 9631.8 (oracle 9631.79), all ten faces exact, body +0.197% → −0.00011%.
		{"N3", 1047315.289309629, 1808, 0x32d906d85cedf489, ""},
		{"N9", 1111539.398763902, 10024, 0x9756a231d600c2f6, ""},
		// M4 is the reentrant BORE (axis at (50,−10), plane through the axis): its result mesh is bit-exact
		// (hash+tris pin it), but its whole-body signed-tetra volume sum is TessellateBody-order-sensitive
		// at ~1e-4 (the off-origin bore's large cancellation, see volTolFor), so its raw volume keeps the
		// loose 5e-4 tolerance; N3/N9 (centred) stay bit-stable at 1e-9.
		//
		// Its volume moved 1002658.6 → 981589.896267823 (−2.10%), and CLOSED FORM says the new value is the
		// right one: the input is the 1e6 box less the pocket the R=30 cylinder at (50,−10) cuts through
		// z ∈ [0,50] (cross-section ∫_{21.7157}^{50}(−10+√(900−(x−50)²))dx = 412.510382, so 979374.4809),
		// plus the concave fill wedge between the x=50 plane, that cylinder and the r=10 fillet arc
		// (95.096189 − 61.418485 + 10.619449 = 44.297154 per unit length, ×50 = 2214.8577) — total
		// 981589.338598. The new pin is +5.68e-07 of that; the OLD pin was +2.146e-02, i.e. it had been
		// pinning a 21069-unit volume error. Note N3's and N9's volumes barely moved (rel ~6e-15) while
		// their AREAS did: the doubled-back cap boundary's spurious triangles come in opposite-signed pairs,
		// so they cancel in the SIGNED tetra volume and only add in UNSIGNED area — which is exactly why a
		// volume pin could not see this defect and an area pin could.
		{"M4", 981589.896267823, 788, 0x7c2683f742a4c291, ""},
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
		//
		// A2 RE-CAPTURED (rim-station-report.md), and PROVEN improved, not merely different — the same
		// defect and the same repair as J1 above: its cone spans rims of r=49.999857 (512 stations) and
		// r=98.081269 (1024), the band grid tiled both at 1024, and the 512-station rim it shares with the
		// plane cap leaked 1536 free edges at PropertyQuality → now ZERO at BOTH gate qualities. tris
		// 134668 → 134156, again exactly −512. Cone mesh area 92225.372724545 → 92225.270652647 against the
		// exact frustum 92225.542932716 (deficit 1.845e-06 → 2.952e-06). Its VOLUME rises
		// 17169519.448758263 → 17169519.822057653 (+2.2e-08): a signed-tetra sum over a mesh that was not a
		// closed surface had no well-defined value, and the closed one is the one to pin.
		{"A2", 17169519.822057653, 134156, 0x567be90454112343, "bfuseblend"},
		{"A3", 15606148.623794555664, 305166, 0xf29512989e349b62, "bfuseblend"},
		// E1/E2 (simple): the 90° sphere-sector corner fillets whose CURVED survivor wall (a sphere
		// meridian rim) was chorded by transformLoop's ENDS branch, collapsing the sphere face ~in half
		// (curved-host-collapse-rootcause.md). The survivor-rim carry (fillet_survivor_rim.go) trims the
		// rim to its retained sub-arc, greening both for the first time (E1 area drift 0.37%, E2 0.47%).
		// Captured on THIS HEAD; they lock the survivor-arc carry so any later slice that perturbs the
		// planar end-corner path fails loud. Same cross-platform-risk caveat as above applies.
		{"E1", 3482804.688278918155, 403430, 0x35d1edf3df93b025, ""},
		{"E2", 3479597.279921139590, 440986, 0x91f20e3fc5921b91, ""},
		// D3 (simple): a 270°-latitude/90°-longitude sphere-sector trihedral fillet, greened as a bonus by
		// the quadrant-gated survivor-rim carry (E1/E2/D3 share the mechanism; its 118–146° sphere meridians
		// are carried, its ≤π/2 rims stay chorded). Its mesh (hash+tris) is bit-stable, so the AREA gate is
		// stable at 0.29%, but — like M4 — its raw signed-tetra VOLUME sum is TessellateBody-order-sensitive
		// (~1e-3 swing, the host-sphere zone's near-degenerate winding); its volume is pinned to the mid-spread
		// with the loosened volTolFor(D3) tolerance, the mesh pinned exactly. Same cross-platform caveat.
		{"D3", 3434569.497398118954, 456966, 0xad9d1470e61bae54, ""},
		// B1/B9 (simple): already-green PLANAR corner fillets whose curved survivor rims are ≤π/2 (62–67°),
		// so the survivor-rim carry (fillet_survivor_rim.go) keeps their base chord byte-for-byte. An earlier
		// arc-carry (82a64cbb) silently drifted them WITHIN deps (B1 0.0071%→0.0499%, toward-wrong) — caught
		// only by adversarial fingerprinting, not by the pass/fail gate. Pinned here to lock the exact victims
		// so any future shared-planar-path edit that re-drifts them within tolerance fails loud.
		{"B1", 192217.801964343031, 62382, 0x3af571d015487e6d, ""},
		{"B9", 517453.708967529470, 46506, 0x36779eafffa7208b, ""},
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
		{"N5", 1047341.372749311, 37558, 0x058a8f3407ef5143, ""},
		// K6/L4 (simple): the P2 trihedral corner-setback greens (fillet_corner_setback.go). Each is a
		// box − pocket whose single trihedral corner joins THREE CONCAVE fillets at three mutually-orthogonal
		// planar faces; both were RED (K6 +1.19%, L4 +1.38%) because solvePlanarBlend placed the corner
		// sphere on the MATERIAL side (the reflection of the true void sphere through the vertex), twisting
		// the three bands and over-keeping host tabs. Flipping the sphere to the VOID side retracts each band
		// by r to the void tangent circle and re-trims the three hosts. K6/L4 are the ONLY bodies whose mesh
		// changes across ALL SIX grids (base 4687de4e → P2 HEAD; verified sweep — no green→green re-weld this
		// slice, unlike P1's N5). Pinned here to lock the void-sphere flip so any later corner slice (P3
		// torus, P4 non-orthogonal) fails loud if it perturbs a K6/L4 body. Same cross-platform-risk caveat.
		//
		// ★ RE-CAPTURED (patchgridcap slice, §region): the void corner ball's B-rep octant was right, but
		// its MESH covered the 7/8 COMPLEMENT — the sphere-patch mesher picks its region from the loop's
		// ABSOLUTE winding, which orientFilletShell leaves arbitrary on the planar path (the same defect
		// assembleCornerBlendBody already fixed on the curved path; assembleFilletFaces now routes through
		// it). PROVEN improved on four axes, each vs live DRAWEXE 8.0.0 at `1.e-12`: corner-sphere face
		// 274.350377/274.337703 → 39.264523/39.264226 vs `sprops` 39.2699 (= 25π/2, the octant closed form;
		// the old mesh swept to 274.8868 = 7/8·4π·25, the complement's own closed form); mesh solid angle at
		// the ball centre 7π/2 → π/2 exactly; whole-body mesh area K6 63968.63 → 63733.54 vs DRAWEXE
		// 63733.6, L4 59968.62 → 59733.55 vs 59733.6 — the complement excess gone. That excess is +235.083
		// at the MESH level (clamped complement 274.350377 − honest octant 39.267264), which is the delta
		// this whole-body reconciliation uses; at the B-rep/DRAWEXE level it is +235.619 (274.889 − 39.2699,
		// visible only once the clamp lifts). The two differ by the clamp deficit — do not conflate them;
		// raw volume K6 959144.890948543 → 958623.105106086 vs `vprops` 958623, L4 945198.518972720 →
		// 944676.776286043 vs 944677 (the +521.7 ≈ 4π·5³/3 = 523.6 ball-cover excess, less inscription,
		// gone). Triangle drop is the 7/8→octant fan alone. Captured twice, bit-stable.
		{"K6", 958623.095907601411, 13500, 0x9f571a346bbcbe79, ""},
		{"L4", 944676.766143134912, 13460, 0xbe4dc64eb7a2440c, ""},
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
		{"M2", 1028376.990494292, 31120, 0xec9b614b4061faf0, ""},
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
		{"A8", 192243.676911436545, 46266, 0x9116c7b73631849c, ""},
		{"A6", 328038.064438616973, 46102, 0x70a6582da86f44cc, ""},
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
		{"I3", 979673.649152592, 24510, 0xa53bc1d42eac4d66, ""},
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
		// Wave-E torus-rim greens (torus_rim_spiric_gate_test.go carries their DRAWEXE per-face tables):
		// J5/A5/A6 = the concave closed-rim cove band on a TORUS host with a latitude cap
		// (fillet_torusarm_concave.go — external a+r for J5/A5, internal a−r for A6); J3/A4 = the
		// SPIRIC closed-rim canal on a meridian cap (fillet_spiric_spine.go / fillet_spiric_rim.go +
		// the torusTubeBandLoftMesh tube-wrapping host loft). Captured on THIS HEAD; they lock the
		// concave torus arm, the spiric station loft, the closed-rim weld routing, and the tube-band
		// mesher, so any later slice touching those fails loud. Same cross-platform-risk caveat.
		{"J5", 6757909.464672484435, 1181692, 0x21f4472585f96261, ""},
		{"J3", 7395592.451696694829, 1115132, 0x60907356101f946a, ""},
		{"A4", 15409136.952526209876, 1180684, 0xee50b6c93f5cf261, "bfuseblend"},
		{"A5", 117038179.720218241215, 788492, 0xcd9879937ec3c456, "bfuseblend"},
		{"A6", 113037851.868033841252, 526348, 0xb5bf46a9e57af953, "bfuseblend"},
		// M5 (simple): the notch-wall concave cove sign (W-DH capability wave). A box − quarter-cylinder
		// notch filleted r5 on three ALL-CONCAVE edges at one trihedral vertex: the concave circle-edge
		// cove arm (concaveTorusArmSurface) hardcoded major = R+r (the boss convention), but a notch's
		// material sits OUTSIDE the wall — the ball rolls INSIDE it, maj = R−r = 25 (DRAWEXE ground truth,
		// cove torus centre (50,−10,45)), and the corner is a plain r-sphere ON that spine at
		// (45,14.4949,45). ε now comes from cylinderHostRadialSign (concaveTorusWallSign), the same n_C·r̂
		// read the concave LINE arm always used; every boss cove (ε=+1) is byte-identical (full-corpus
		// sweep: exactly one verdict change, zero area drift elsewhere). Reconciled per face vs DRAWEXE
		// sprops 1e-12 on all 13 faces (worst −0.008%); volume 980008.93 vs vprops 980008. Captured on
		// THIS HEAD; it locks the ε=−1 cove + all-concave sphere-corner weld so a later concave-arm or
		// corner slice fails loud if it perturbs the M5 body. Same cross-platform-risk caveat as above.
		{"M5", 980008.933761107619, 24756, 0x6718a941b785974b, ""},
		// B3 (bfuseblend): the multi-rim weld (fillet_curved_multirim.go, W-DH capability wave). A
		// through-cylinder fused to a box leaves TWO closed concave Cylinder∧Plane rims (one per exit
		// face, y=0 and y=100); the multi-pick op used to floor at "trihedral corner needs 3 arms" —
		// there is no corner, the rims never touch. Both rims now route through the proven single-rim
		// band rebuild SEQUENTIALLY (the second rim's ReferenceKey resolves on the intermediate body
		// because the rim rebuild carries untouched Lineage verbatim — ADR-0043 doing the composition).
		// Reconciled per face vs DRAWEXE sprops 1e-12 on all 12 faces (4×10000 · 2×6151.65/6151.55 ·
		// 2×8482.25/8482.3 · 2×2827.36/2827.43 · 2×1570.04/1570.1, worst +0.0016%); volume
		// 1284834.85 vs vprops 1.28484e6. The −0.998% against the corpus number 78849.6 is the
		// HISTORIC reference's drift: our body reads 78062.6 vs local DRAWEXE 8.0's own 78062.8
		// (−0.0003%). Captured on THIS HEAD; locks the sequential-rim composition so a later rim or
		// key-continuity slice fails loud if it perturbs the B3 body. Same cross-platform caveat.
		{"B3", 1284834.848345890874, 135180, 0xac0646576d972b5d, "bfuseblend"},
	}
}
