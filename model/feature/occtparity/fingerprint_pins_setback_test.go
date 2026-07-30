// SPDX-License-Identifier: GPL-2.0-only

package occtparity

// setbackObstaclePins are the SETBACK / MID-SPAN-OBSTACLE half of byteIdentityPins (see
// fingerprint_pins_test.go for what the pins are, how they were captured, and the arm64 cross-platform
// caveat that applies to every one of them): the one- and two-boss setback families, the obstacle-path
// bodies, the far-run-out and rail-bundle greens, and the general corner-weld layer's own cases.
func setbackObstaclePins() []fingerprintPin {
	return []fingerprintPin{
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
		// ★ RE-CAPTURED with S1/S4/S7/T1/T4/T7 below — see the NIL-CURVE-OFFER ADOPTION receipt at the
		// bottom of this file. These nine are the whole population the edge catalog's nil-vs-curve
		// adoption moves, and every one is PROVEN improved per-face against live DRAWEXE 8.0.0.
		{"S6", 18039.238279760, 16292, 0x6a1c3475f4697bae, ""},
		{"S9", 115877.633872810, 90366, 0xc6487a5cc1a6f0c5, ""},
		{"T3", 130529.632368156, 81256, 0xf22027c50f378b81, ""},
		// S1/S4/S7/T1/T4/T7 (simple): the TWO-boss members of the same setback family — a box with a boss
		// on EACH of the picked edge's two host planes, tiled left flank / central / right flank
		// (fillet_setback_extract.go). Their nine-case cohort S6/S9/T3 was pinned above from the day the
		// single-boss branch greened; these six never were, and until this commit their ONLY guard was the
		// corpus's 1% whole-body area gate — which coons4-audit.md proved is structurally blind to the
		// defect they carried: S7's run-out patch was 18.5% of r wrong in SHAPE while its per-face area was
		// 0.03% RIGHT, and the whole body read +0.73% by CANCELLATION. The run-out envelope slice
		// (runout-envelope-report.md) made all six geometrically correct — deep-interior bidirectional
		// Hausdorff against OCCT's own surface now 0.009–0.023% of r, down from 9–19%, with the cancellation
		// gone (Σ|Δ| 0.038–0.060% of the body vs a 1% gate) — and these pins are what stop a later slice
		// from silently undoing it. Captured on THIS HEAD, bit-stable across re-runs at 1e-9.
		//
		// They are the byte-identity half of a two-part guard; the geometric half is the DRAWEXE per-face
		// oracle in kernel/ops/fillet_runout_oracle_test.go (spine stations, per-face `sprops` areas, and —
		// the load-bearing one, because area alone would not have caught S7 — points DRAWEXE evaluated on
		// OCCT's OWN patch surface asserted to lie on ours). Same cross-platform-risk caveat as above.
		// ★ RE-CAPTURED with S6/S9/T3 above — NIL-CURVE-OFFER ADOPTION receipt at the bottom of this file.
		{"S1", 12595.501472741, 17144, 0xb7b695fd2b6721ca, ""},
		{"S4", 36853.861302685, 22770, 0x1a793294ff6e2f08, ""},
		{"S7", 33547.700513286, 20052, 0xb7e6bfe83effd506, ""},
		{"T1", 117950.363024610, 93596, 0xf64469fb6ae1b453, ""},
		{"T4", 133680.910978887, 90144, 0x9ff14e461d806c8f, ""},
		{"T7", 36278.796710477, 23068, 0x211eab265438be21, ""},
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
		// ★ RE-CAPTURED (rimcrossings-report.md), and PROVEN improved, not merely different: its two
		// boundary nodes were solved on the 64-CHORD SAMPLED rim, landing 3.719953e-03 off the exact
		// rim∩band crossing; analyticNode re-solves them on the rim CURVE and both now sit on the closed
		// form z = ±√44 = ±6.633249580711 to 3.6e-15 (measured against lineCircleRoots, the runout path's
		// OWN independent exact solver, not this slice's bisection). The TRIANGLE COUNT is unchanged (9452)
		// — same topology, corrected positions — and the two faces the nodes bound, the cylinder wings, move
		// 65.739870 → 65.710655 against DRAWEXE 8.0.0's own sprops 65.7123, i.e. +0.0420 % → −0.0025 %, 16.6×
		// closer and now at the inscribed-mesh floor. The other faces move ≤6e-04 absolute as the complement
		// of that, inside U3's own pre-existing +9.5 % corner-patch error (ours 94.11 vs OCCT's 42.9587×2),
		// which this change neither creates nor addresses. Volume 17894.503322426473 → 17894.523018369215
		// (+1.1e-09 relative, against a standing +0.112 % gap to DRAWEXE's vprops 17874.5).
		// ★ RE-CAPTURED AGAIN (obstacle-canal-report.md) — and this is the slice that closed the +9.5 %
		// corner-patch error the paragraph above names as untouched. The obstacle patch is no longer a Coons
		// FillSurface but the exact SURF-RST rolling-ball canal (ball tangent to the fillet wall, through the
		// boss rim), and the wall face's front is no longer the straight tangent seam but the ball's own
		// tangency-foot locus. PROVEN improved on four independent measures, none of them our own number:
		//   - the patch face 94.109908 → 85.941735 against its CLOSED FORM 85.917424 (= DRAWEXE's own
		//     42.9587 x 2), i.e. +9.539 % → +0.0283 %, 337x closer;
		//   - the wall face 300.000000 → 302.998177, recovering a tangency bulge whose closed form is
		//     2.998261 and which the straight seam was missing ENTIRELY (−100 % → −0.0028 %);
		//   - the whole-body volume 17894.523018369215 → 17873.835060138 against DRAWEXE's own vprops
		//     17874.5: +0.1120 % → −0.0037 %, 30x closer;
		//   - the patch's INTERIOR now certifies: MaxBallDev 3.14e-05 (uncertified before — the Coons tier
		//     never populated the field at all) → 7.55e-09, inside the 5.38e-08 model weld.
		// The triangle count FALLS, 9452 → 5850, and that is the leaner mesh of a more faithful surface, not
		// a loss: the canal is degree-2-RATIONAL in u (one exact arc, a 3 x 97 net) where the Coons fill was
		// a degree-3 28 x 28 net, so the same chordal tolerance is met with far fewer u-columns. Face count
		// (11), free edges (0) and folds (0) are unchanged at BOTH gate qualities.
		// ★ RE-CAPTURED AGAIN (rim-node-chord-report.md), and PROVEN improved on four independent measures.
		// The four rim segments the two boundary nodes split used to carry NO curve at all — a straight
		// truncated chord — and on U3 that chord read 1.561e-02 off the corner-blend patch AND 1.197e-02 off
		// the UNTOUCHED obstacle wall, the same chord against two different faces. They now carry the rim
		// conic TRIMMED at the node's own rim parameter (rimNodeTrimsOf), which for U3's circular rim is the
		// EXACT trimmed arc (3.97e-15 off the rim curve, measured):
		//   - worstLoopSegmentOffFace/boundingDiag — the off-surface gate's OWN function — 3.170438e-04 →
		//     1.987829e-06, 159x better, and the debt ceiling shrinks 0.000323 → 0.00000219 with it;
		//   - the patch face 85.941735200 → 85.922547692 against its CLOSED FORM 85.917424 (= DRAWEXE's own
		//     42.9587 x 2): +0.0283 % → +0.00596 %, 4.7x closer;
		//   - the obstacle wall (EllipticalCylinder) 916.819041982 → 916.824497356 — an inscribed rim
		//     polyline under-reports a tube, so replacing chords by arcs must and does ADD area;
		//   - the whole-body volume 17873.835060138 → 17873.945817788 against DRAWEXE's own vprops 17874.5:
		//     −0.00372 % → −0.00310 %, an INDEPENDENT measure from area.
		// The triangle count rises 5850 → 5862, the 12 stations the mesher needs to follow four arcs it used
		// to walk as straight lines. Face count (11), free edges (0) and folds (0) unchanged at BOTH gate
		// qualities. No other case in the corpus changes by a single triangle.
		{"U3", 17873.945817788, 5862, 0x718c0ef4145587bc, ""},
		// U4 (simple): the Group-C dual-host multi-rail corner-blend green (kernel/ops/fillet_obstacle_dual*.go,
		// #2007 Group C — the U4-5 weld). U4 is a box + TWO bosses (Cylinder r8 on y=−20, oblique
		// EllipticalCylinder on x=10) whose footprints BOTH dip into ONE r=5 convex-edge fillet (qualifying==2),
		// the dual-host composition neither the single-boss setback tiling (S6/S9/T3) nor the dipArcOrder
		// obstacle fix (U3) reached. The dual assembler welds 2 notched hosts + 2 walls split at every panel
		// seam + 2 cylinder wings + 4 corner-blend panels (2 coons4 slivers, 2 exact-station canal cores) into a
		// watertight, hole-contained SOLID matching the DRAWEXE 16-FACE / every-face-WIRE:1 topology at whole-
		// body area 6583.29 (rel 0.001%), every fillet panel tessellating fold-free to its per-face oracle
		// (3.039 sliver / 30.334 core) in production. Locks the dual-host weld so any later slice that
		// perturbs the U4 body fails loud. Same cross-platform-risk caveat as above applies.
		// ★ RE-CAPTURED (u4-canal-report.md), and the move is PROVEN an improvement, not merely a
		// difference: the previous pin (32647.755284293769, 23848 tris, 0xf588dd3b13096eb9) was taken on a
		// body whose welded mesh LEAKED 44 free edges at BOTH gate qualities — it was not a closed surface.
		// Routing saddleBandLoftMesh's rim read through discretizeEdge takes that to 0/0. The deltas are
		// exactly what that repair must produce and nothing more: +40 triangles = 2 x 20, the vertices the
		// boss wall's band loft gains on the two straight rails it shares with the sliver panels (which
		// already carried 21 chords each); all sixteen per-face areas unchanged to 2e-6 and the whole-body
		// area unchanged at 6583.287851 (property quality, ops.MeshArea over CalculateBodyFacets); the raw
		// volume moves 7.8e-8 relative, 340x smaller than the standing 2.6e-5 gap to DRAWEXE's 32648.6.
		// ★ RE-CAPTURED AGAIN (rimcrossings-report.md), and PROVEN improved: its boss-B node was solved on
		// the 64-chord sampled rim at z = −6.629434364090, 3.815216e-03 short of the exact ±√44 =
		// ±6.633249580711 that boss B's r=12 mouth circle and the fillet's y=−15 B-tangent give in closed
		// form. analyticNode re-solves it on the rim CURVE: it now reads −6.633249142265, which is 4.4e-07
		// from √44 and 1.2e-11 from DRAWEXE 8.0.0's OWN patch end (−6.63324914227121) — i.e. at the oracle's
		// own floor. The TRIANGLE COUNT is unchanged (23888) — same topology, corrected positions — and every
		// one of the sixteen faces is better or bit-identical, none worse (ops.MeshArea over
		// CalculateBodyFacets(body, PropertyQuality()), rank-paired against DRAWEXE sprops 1e-6):
		// the two SLIVERS −0.8484 %/−0.8064 % → +0.1156 %/+0.1153 % (7.3× closer, the −0.93 % z-span deficit
		// the node caused is gone), the two WINGS +0.0411 %/+0.0431 % → −0.0025 % (16× closer, at the mesh
		// floor), the boss wall +0.15265 % → +0.15259 %; the two CORES are bit-identical because their bound is
		// host A's node, which is NOT refined (coupledNodeStation — see fillet_obstacle_detect_face.go).
		// Summed per-face |Δ| vs DRAWEXE 1.49116 → 1.39527; raw volume 32647.752737870436 →
		// 32647.756822102991, moving TOWARD DRAWEXE's vprops 32648.6 (−0.0025952 % → −0.0025827 %).
		// ★ RE-CAPTURED AGAIN (rim-node-chord fix wave, finding I4), and PROVEN improved on the body's own
		// MIRROR SYMMETRY. spliceRimPoint — the dual path's interior-station insert — dropped the split
		// segment's curve on BOTH halves, which on U4 discarded two of the four trimmed node sub-arcs
		// insertNodesIntoRim had just built (the segments terminating exactly on the receded boundary
		// y = −15) plus an interior rim arc, leaving U4 byte-identical across the whole rim-trim slice. The
		// split now trims the segment's own conic at the station (splitRimSegmentCurve), and exactly TWO of
		// the sixteen faces move. U4's solid is mirror-symmetric about z=0 and DRAWEXE's own face list says so
		// (869.635/869.635, 65.7123/65.7123, 30.3344/30.3344, 3.03906/3.03903) — and the +z CORE panel had been
		// the ONE pair member out of step: 30.351289547 against its −z twin's 30.340810010, a 1.048e-02 split
		// (3.5e-04 relative) where every other mirror pair on this body agrees to 3e-07…7e-06. It now reads
		// 30.340891604, agreeing with its twin to 8.2e-05 — 128× closer — and against DRAWEXE's own
		// tolerance-STABLE 30.3344 (identical at 1e-9 and 1e-12) it goes +0.0557 % → +0.0214 %, landing on the
		// twin's own +0.0211 %. Off-surface residual (worstLoopSegmentOffFace/boundingDiag, the gate's own
		// function) 2.233973e-04 → 1.493883e-04, and its ceiling ratchets down with it. Raw volume
		// 32647.756822103 → 32647.827612193, TOWARD DRAWEXE vprops 32648.6 (−0.0025827 % → −0.0023658 %).
		// Triangles 23888 → 23894: the six stations the mesher needs to follow the recovered arcs. Free edges 0
		// and folds 0 at BOTH gate qualities, face count still 16.
		//
		// The one face moving the other way is the oblique `geom.EllipticalCylinder` pipe wall,
		// 799.083600483 → 799.080823847 (−3.5e-06 relative) against DRAWEXE's 799.138 @1e-12 — a move 120×
		// SMALLER than that face's own `sprops` quadrature spread (799.473 @1e-9 vs 799.138 @1e-12), i.e. the
		// trimmed-oblique-quadric class perface_oracle_test.go's ★★ note describes, not adjudicable by
		// DRAWEXE. Its boundary is now ON the surface it bounds instead of 9.6e-03 off it.
		//
		// Still out of scope and unchanged: host A's COUPLED node is not refined onto its rim
		// (coupledNodeStation), so its r=8 boss rim keeps the honest straight chords — they are U4's new worst
		// off-surface offender at 9.623665e-03, both endpoints on x²+z²=64 at y=−20.
		// ★ RE-CAPTURED ONCE MORE (dual dip-rim slice), and proven improved on that SAME mirror symmetry.
		// The fix wave above did not in fact reach U4's shipped rim: splitRimSegmentCurve recovered the
		// split parameter by inverting the station onto the SEGMENT's own conic and then required that conic
		// to reproduce the station within the model weld. U4's boss-B rim is an imported b-spline whose
		// per-segment circular fit sits ~1e-7 off it while the station is solved on the exact rim to ~1e-12,
		// so a per-segment FIT RESIDUAL was being compared against a WHOLE-MODEL weld: measured on the
		// shipped path, res.Weld() = 6.442049363e-08 (boundingDiag 64.420493634) and the two inversions read
		// 8.723146033e-08 (1.354x) and 7.634559529e-08 (1.185x), so BOTH halves fell back to chords. (The
		// 3.394e-08 an earlier revision of this note quoted was the deleted regression test's own
		// ResolutionForPoints(holeSampled.pts).Weld(), not the shipped res.Weld().)
		// Four of boss B's rim halves therefore shipped straight: 5.002371e-03 / 4.967703e-03 /
		// 2.675363e-03 / 2.674992e-03 off the corner-blend PANELS they bound and 4.301259e-03 /
		// 4.275629e-03 / 2.080290e-03 / 2.079314e-03 off the obstacle WALL — the same four chords against
		// two different faces, which is what makes them the chords' own defect and not either surface's.
		// The trim now reads the RIM's parameterisation (rimSubArcBetween), the one construction
		// rimSegmentArc and nodeSubArcs already use, and the panels' rim sides read the SAME pure function
		// of the same two endpoint values instead of handing all-nil (they were discarded 28 times per
		// build only because the walls REPLACE body faces and so reach the first-writer-wins edge catalog
		// first). Five of the sixteen faces move; eleven are bit-identical.
		//
		// Proven on the mirror twins at SMOOTH tessellation (TessellateFace, ChordTolerance 1e-5, where
		// PropertyQuality's own chordal deficit no longer dominates). ★ 1e-5 is smooth, NOT converged: at
		// ct=1e-6 the mesher emits "tessellate.cap-saturated: interior refinement saturated the 64-cell
		// floor still above chord tol 1e-06" on ALL FOUR BSpline panels, and one core panel's area then
		// jumps ~6.4e-03 (base 30.335378318 @3e-6 → 30.329024770 @1e-6; HEAD 30.333277446 → 30.326923524).
		// That saturation is present IDENTICALLY at base f633c0ce, so it is pre-existing and not caused
		// here, but no reading at or below ct=3e-6 on these panels should be called converged until it is
		// understood (tessellator defect vs legitimate refinement branch — queued). The twins below are
		// mirror images measured at the SAME quality, so their SPLIT is unaffected by it.
		// The two CORE panels are exact mirror
		// images about z=0 (DRAWEXE 30.3344/30.3344) and their split goes 1.856e-05 → 1.035e-06, 17.9×,
		// from 1.9× the wings' own pair split (2.1012e-05 on 65.712, rel 3.198e-07) to 9.4× BELOW it. The
		// asymmetry was the CHORDS' own: the rim's 64 samples are not mirror-symmetric (the +z neighbour
		// sample sits at z=5.656289770, the −z one at z=−5.658082312), so a chord tiling reads differently
		// on the two sides while the exact rim arc reads the same. Raw volume 32647.827612193 →
		// 32647.857558403 against DRAWEXE vprops 32648.6 (stable at 1e-9 and 1e-12): both readings are
		// BELOW the oracle's 6-s.f. band [32648.55, 32648.65], so "closer" is unambiguous — −0.0023658 % →
		// −0.0022741 %. Triangles 23894 → 23830 (the mesher's aspect-aware densification, #2009, needs
		// fewer stations on a boundary that is on the surface). Free edges 0 and folds 0 at BOTH gate
		// qualities, face count still 16.
		//
		// NOT improved, and stated rather than claimed: at the same smooth quality the two core panels go
		// 30.335400037/30.335418597 → 30.333320307/30.333319272 against DRAWEXE's 30.3344 — +1.00e-03/
		// +1.02e-03 to −1.08e-03/−1.08e-03, i.e. across the oracle and slightly FURTHER from it. Pushed to
		// ct=1e-7 the smooth twin reads base 30.335346521 (+9.47e-04) → HEAD 30.333264763 (−1.14e-03): the
		// magnitude GROWS ~19 %, so the per-face DRAWEXE reading marginally DISFAVOURS this change on the
		// two core panels. It is not the adjudicating measure here — what dominates that residual is the
		// queued coupled node, whose own node point is 4.04e-03 off host A's rim and whose chord is
		// 9.62e-03 off the panel, both an order above anything this slice moved; the pin rests on the
		// mirror twins and on volume. What WOULD adjudicate it is a closed form for these panels, which
		// does not exist yet. The two SLIVERS do improve, 3.046316780/3.046337374 →
		// 3.045616229/3.045636721 against 3.03906/3.03903 (+0.2387 % → +0.2157 %). The oblique
		// EllipticalCylinder pipe wall goes 799.101070069 → 799.099981639 against 799.138 @1e-12, a move
		// 308× smaller than that face's own sprops quadrature spread (799.473 @1e-9 vs 799.138 @1e-12) —
		// the trimmed-oblique-quadric class perface_oracle_test.go's ★★ note describes, not adjudicable.
		{"U4", 32647.857558403, 23830, 0x65b96c41502bab5b, ""},
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
		{"F4", 3104847.813129819464, 26988, 0x703c9f4f9e96a8d3, ""},
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
		{"N4", 1046938.850088385167, 53918, 0xe5fe9d9df624b52f, ""},
		// O1 (simple): the SECOND case welded by the general corner-weld layer, and the layer's own
		// falsification test — it cost one ~75-line plan CONFIGURATION (cornerweld_class_o1.go) plus its
		// corner SOLVE, with no new executor stage. A r50×h130 cylinder fused to a 30×50×70 box that
		// protrudes from it, filleted r5 on the three edges meeting at (80,10,90): the concave
		// wall∧cylinder ruling, the concave cove-torus arc where the box top meets the cylinder, and the
		// CONVEX box top∧wall edge. The two concave arms terminate at the corner; the convex band runs past
		// it, so the patch is the rolling-ball canal of a ball riding the boss cylinder at ρ=R+r=55 and
		// rolling on the band's tube at 2r=10 (fillet_curved_mixed_canal_o1.go).
		//
		// DRAWEXE 8.0.0 receipt (`restore CFI_f5678fin.rle s ; tscale s 0 0 0 10 ; explode s e ;
		// blend result s 5 s_7 5 s_6 5 s_14`): valid SOLID, 12 faces / 13 wires / 18 vertices / 27 edges,
		// sprops area 65104.9, vprops volume 1.11166e6 — ours 12 faces, area 65101.43 (−0.0053%), volume
		// 1111282.03 (−0.034%). The residual is OCCT's own corner-patch approximation, not ours: our patch is
		// the exact envelope (G1 to all four analytic neighbours within 1e-6 rad, all four corners identical
		// to DRAWEXE to 1e-10), while OCCT's interior sits up to 1.16% of r off that envelope and its implied
		// rolling ball misses tangency to the boss wall by 0.174. Pinned so any later class builder or
		// executor change fails loud if it perturbs O1. Same cross-platform-risk caveat applies.
		//
		// The two on-host canal rails are registered as ONE seg each. They used to be 8 sub-segs
		// (o1CanalRailPieces) purely to keep the three-point re-fit's h³ error small; with the re-fit gone
		// that workaround is deleted, so this pin encodes the un-subdivided ring — which is also why O1's
		// vertex/edge count drops 30/39 → 16/25, against the oracle's 18/27.
		{"O1", 1111281.884319459088, 29866, 0x617cc8805bca3d1, ""},
	}
}

// ═══ NIL-CURVE-OFFER ADOPTION: the shared re-capture receipt for the NINE pins it moved ═══
//
// S1 S4 S6 S7 S9 T1 T3 T4 T7. Recorded once, here, instead of nine times above.
//
// WHAT CHANGED. `edgeCatalog.use` welds one shared edge per vertex pair from the FIRST consumer to
// reach it. When that consumer had no curve, the edge shipped a straight CHORD and the second
// consumer's real boundary curve was thrown away — 272 shipped edges over ten cases. The catalog now
// adopts the later curve, REVERSED onto the edge's own stored sense when the offering consumer walks
// it backwards (which is all 270 of the ADOPTED records; the other 2 of the 272 are complex/F2's
// curve-FIRST pair, kept, and 0 are declined). A previous slice built this without the reversal, measured
// T3's blend torus inflating 2827.227365 → 13816.882599 and the rollup falling 114 → 110, and recorded
// the policy as refuted; the reversal is the whole difference, and removing it reproduces those
// numbers exactly.
//
// PROVEN IMPROVED, four independent measures, none of them our own number. Whole-body mesh area
// (Σ ops.MeshArea ∘ ops.TessellateFace at PropertyQuality) against OCCT's own checkprops reference —
// confirmed this session as live DRAWEXE 8.0.0 `sprops result 1.e-12`, relative error of mass
// computation ≤ 9e-16 on every case:
//
//	S1 +2.451e-04→+6.396e-05 (3.83×)   S4 +2.346e-04→+9.129e-05 (2.57×)   S6 +2.371e-04→+1.682e-04
//	S7 +2.004e-04→+1.434e-04           S9 +7.332e-04→+3.608e-04 (2.03×)   T1 +4.462e-04→+3.745e-04
//	T3 +3.739e-04→+2.653e-04 (1.41×)   T4 +3.526e-04→+2.816e-04 (1.25×)   T7 +2.193e-04→+6.075e-05 (3.61×)
//
// ★ PER-FACE against live DRAWEXE `sprops <face> 1.e-12`, faces rank-paired by area (never by
// `bounding`). Summed per-face |Δ| falls on EVERY one of the nine: S1 1.9743→1.0003, S4 4.3199→2.7362,
// S6 2.0950→1.9655, S7 2.8441→2.6327, S9 11.0587→5.6612, T1 8.4104→6.0841, T3 7.3596→5.3449,
// T4 7.8348→6.1115, T7 7.0374→4.7997 (1.47×, no face regressing — the standing gate is
// t7_adoption_perface_test.go). ★ An earlier revision of this receipt EXCLUDED T7 as "a pre-existing
// topology divergence, 11 faces to our 15, nothing to rank-pair" — that was a MIS-RUN oracle, not a
// divergence: the T7 fixture picks its edge with `nexplode s E` and the slice ran `explode s E`, which
// silently renumbers the pick and blends a different edge (the 11-face / 7507.53 body). With the
// fixture's own `nexplode`, OCCT's blend is a valid 15-face solid, `sprops 1.e-12` = 7482.15 (rel err
// 9.3e-16), and T7 rank-pairs 15-to-15 (perface_oracle_test.go's ★★ explode-variant note is the durable
// trap record). The faces that move are exactly the ones
// bounded by an adopted curve; every other face is bit-identical. On T3, the case the previous slice
// used to refute the policy: blend torus 2827.227365 → 2826.791716 against 2826.04, BSpline patch
// 251.184828 → 249.700452 against 249.664 (42× closer), and NO cap saturation (65072 → 65162
// triangles, three decades below the 262144 cap the un-reversed build saturated).
//
// ★ MIRROR TWINS — the strongest of the four, because it needs no oracle at all. Each of these
// fixtures is symmetric, so its two blend flank faces must have equal area. They did not: on T1/T3/T4
// the pair read 227.809448 / 227.851401, 4.2e-02 apart (1.8e-04 relative). They now read
// 227.764913 / 227.764925, 1.2e-05 apart — 3500× tighter, and both land on DRAWEXE's 227.774. Same on
// every case: S1 29.099278/29.114950 → 28.950601/28.950604, S4 51.033265/51.128363 →
// 50.837314/50.837321, S7 5.331584/5.342248 → 5.308848/5.308849, S9 235.796757/235.836658 →
// 235.613052/235.613056, T7 33.275269/33.337713 → 33.210881/33.210883. The chord that used to ship was
// not mirror-symmetric; the rim arc is.
//
// MESH VOLUME against live DRAWEXE `vprops … 1.e-12` — an axis independent of area — improves on
// seven: T1 −1.09e-04→−2.24e-05 (4.9×), T3 −6.58e-06→−2.82e-06, T4 −6.95e-05→−3.06e-05, S9
// −1.76e-05→−1.18e-05, S1 −6.71e-04→−4.52e-04, S4 −5.24e-04→−3.94e-04, T7 −2.54e-03→−2.15e-03.
// ★ NOT improved, stated with its TRUE extent (an earlier revision of this receipt half-disclosed
// it as an S6-only VOLUME dissent; the adversarial review measured it as a TWO-case AREA
// regression, and that is what it is): the regression is the SPHERE-HOST ARM, on BOTH sphere
// cases, S6 AND S7, on the host geom.Sphere face itself. OCCT does not trim that sphere at all —
// verified against the closed hemisphere 2π·13² = 1061.858347 by area, CoG z = R/2 AND inertia —
// while we NOTCH it: S6's shipped sphere boundary loop runs 50 edge-uses / 118.39 summed chord
// against the equator's 2π·13 = 81.68 (measured here on the shipped topo loop). Sphere-face area
// at PropertyQuality, base → HEAD (ops.MeshArea ∘ ops.TessellateFace, measured here): S6
// 1061.302202 → 1061.235258, S7 1061.291184 → 1061.179147; at the review's CONVERGED tessellation
// limit: S6 1061.313591 → 1061.273914 (−0.5448 → −0.5844 vs the closed form) and S7 1061.301549 →
// 1061.264052 (−0.5568 → −0.5943). HEAD has MORE triangles and LESS area, so it is a REGION
// change, not mesh noise, sitting on the pre-existing ~0.55 notch defect — the notch is its own
// queued slice (curve-adoption-report.md, concerns), NOT re-fixed under adoption. S6's vprops
// dissent above (−3.31e-04→−3.58e-04) is this same regression seen through volume; S7's volume
// merely masks it (flat at −2.83e-04, a 2.0e-05 absolute move below the oracle's 6-s.f.
// resolution). The pins stand because BOTH bodies improve on every other measure — converged
// cylinders 60× closer, S7's BSplines 323×/640×, twins 1.30e-02 → 4e-06 (review's convergence
// measurements) — and the dissenting face is named, not averaged away.
//
// Off-surface residual (worstLoopSegmentOffFace / boundingDiag, the gate's own function) falls on all
// nine, 1.07–4.58×, and every one of the nine knownOffSurfaceDebt ceilings ratchets down with it.
// Free edges 0 and folds 0 at BOTH gate qualities; face counts unchanged; the whole-corpus DRAWEXE
// face-count join is unchanged. Same arm64 cross-platform-risk caveat as every other pin here.
