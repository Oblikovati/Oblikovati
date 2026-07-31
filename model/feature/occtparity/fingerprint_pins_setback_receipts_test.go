// SPDX-License-Identifier: GPL-2.0-only

package occtparity

// The shared re-capture receipts for the NINE setback pins (S1 S4 S6 S7 S9 T1 T3 T4 T7) —
// split from fingerprint_pins_setback_test.go by responsibility (500-line rule): that file holds
// the pins, this one the evidence trail each re-capture must extend. Newest receipt LAST.

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
//
// ═══ FOOTPRINT-RIM CURVE CARRY: the shared re-capture receipt for the same NINE pins ═══
//
// S1 S4 S6 S7 S9 T1 T3 T4 T7 — re-captured again by the t3-plane-sliver slice
// (.superpowers/sdd/t3-plane-sliver-report.md). WHAT CHANGED: the intact-boss footprint rim's
// subdivided segments — wall side (subdivideBossWall → maps.insertCurves) AND host-notch side
// (appendTrimmedArcSegs / appendSeamArc) — now carry each segment's exact sub-span of the footprint
// conic (geom.TrimmedCurve3) instead of shipping a straight chord, so the re-clipped host plane and
// the intact wall bound the TRUE rim. The defect this closes was measured in closed form: T3's host
// plane read 1208.987870 against the fixture-derived 60·52 − (π·25² − (625·acos(22/25) − 22·√141)) =
// 1204.602895 and live DRAWEXE 8.0.0 `sprops result_5 1.e-12` = 1204.6, and the +4.384975 excess
// equals Σ (r²/2)(θ−sinθ) over its 42 rim chords to SEVEN digits (4.38498) — pure inscribed-polygon
// surplus, a trim-loop region defect on a plane that cannot be wrong as a surface.
//
// PROVEN IMPROVED, every case, two oracle axes measured this session (functions and qualities named):
// whole-body mesh area (Σ ops.MeshArea ∘ ops.TessellateFace at PropertyQuality) against OCCT's
// checkprops reference, adoption HEAD → this HEAD:
//
//	S1 +6.396e-05→−1.047e-05 (6.1×)   S4 +9.129e-05→−7.068e-06 (12.9×)  S6 +1.682e-04→−1.594e-04 (1.06×)
//	S7 +1.434e-04→−1.051e-04 (1.36×)  S9 +3.608e-04→+1.196e-06 (302×)   T1 +3.745e-04→−2.378e-06 (157×)
//	T3 +2.653e-04→−7.484e-06 (35×)    T4 +2.816e-04→−6.112e-06 (46×)    T7 +6.075e-05→−6.228e-06 (9.8×)
//
// and mesh volume (ops.BodyGeometryProperties at PropertyQuality) against live DRAWEXE `vprops … 1.e-12`
// re-read this session (6-s.f. print resolution, so ±~4e-6 relative): S1 −4.522e-04→−3.790e-05,
// S4 −3.943e-04→−1.797e-05, S6 −3.581e-04→−3.221e-04, S7 −2.831e-04→−1.771e-04, S9 −1.179e-05→−9.139e-06,
// T1 −2.236e-05→−7.332e-06, T3 −2.816e-06→+2.264e-06 (flat at the oracle's resolution), T4
// −3.059e-05→+1.642e-06, T7 −5.841e-04→−2.879e-05. Per-face against `sprops <face> 1.e-12` re-derived
// live for T1/T3/T4: Σ|Δ| T1 6.0841→0.2660, T3 5.3449→0.3592, T4 6.1115→0.3549, no face regressing;
// the 1204.6 plane itself: 1208.987870 → 1204.671464 on all three (the +0.0686 that remains is the
// PropertyQuality mesh inscribing the now-exact arc boundary, not the B-rep: the shoelace area of the
// shipped loop equals the closed form + chord deficit identity above). T7 re-ranked against its
// nexplode oracle: front host 399.537566→399.192709 (DRAWEXE 399.178), top host 334.081647→333.101499
// (333.086), elliptic wall 2380.950144→2381.645098 against the ∮|C′×v| closed form 2381.677340 —
// the wall gap the adoption could not close was these rim chords. ★ The S6/S7 sphere-notch dissent
// above KEEPS its true extent: S6's sphere face reads 1061.197485 here (further below the untrimmed
// 2π·13² = 1061.858347; S7's reads 1061.193720, slightly toward it) — the notch is still the queued
// slice, and both bodies improve on whole-body area AND volume regardless.
//
// Off-surface residual (worstLoopSegmentOffFace/boundingDiag): S7 3.8e-10, T4 4.6e-9, T7 4.1e-8
// (entries DELETED from knownOffSurfaceDebt), T1 5.96e-6, S1 4.16e-5, S4 8.10e-5, S6 1.75e-4,
// T3 2.39e-4, S9 7.93e-4 (holds). knownNilCurveOfferDebt 272 → 110 (the band-side rim records are
// now two-sided value agreements). Free edges 0 and folds 0 at BOTH gate qualities; face counts
// unchanged; rollup unchanged at 114 simple / 119 all-grid, SkipQuarantine 0.
//
// ═══ RAILB INTERPOLATED CONTACT-LOCUS CARRY: the third re-capture receipt for the same NINE pins ═══
//
// S1 S4 S6 S7 S9 T1 T3 T4 T7 — re-captured by the railB locus slice (railb-locus-report.md). WHAT
// CHANGED: the surf-rst runout band's B rail was a degree-1 polyline through the 7 NODE contacts;
// its chords lay IN the pInner host plane (partner residual ~1e-15) but off the lofted patch by the
// chord sagitta — the sharpest named off-surface family on the setback corpus (S9 7.927231e-04 of
// the bounding diagonal). contactLocusRail now interpolates EVERY solved station contact (73) into
// the loft's own boundary row (geom.CanalFootLocusRail — the identical chord-length params, degree
// and averaged knots assembleCanalLoft interpolates the FeetB pole row with, so the rail IS the
// patch's u=1 boundary isoparm), and BOTH consumers carry its per-segment TrimmedCurve3 sub-spans
// by value: the patch loop (runoutPatchLoops, unchanged machinery) and the host-notch detours
// (plainContactDetour + innerHostSegs, appendArcSegs → appendTrimmedArcSegs).
//
// PROVEN IMPROVED — per-face against live DRAWEXE 8.0.0 `sprops <face> 1.e-12` (per-face rel err
// ≤ 1e-15), blend rebuilt from each case's own STEP fixture with the pick edge matched by lprops
// CoG to the corpus locator at ≤ 9e-16 and the face count matching ours EXACTLY on every case
// (S1/S4/T1/T4/T7 15, S6 10, S9/T3 11, S7 14); ours = ops.MeshArea ∘ ops.TessellateFace at
// PropertyQuality, faces rank-paired by area. The ONLY faces that move on any body are the pInner
// host plane and its runout patch(es) — face-ID-paired movers, complementary by construction — and
// EVERY mover moves TOWARD its oracle row (21 of 21; no face regresses on any case):
//
//	S9 plane 1221.564147→1221.731164 vs 1221.74 (19.9×), patch 366.615511→366.448437 vs 366.439 (18.7×)
//	T3 plane 1327.424000→1327.460278 vs 1327.46 (129×), patch 249.700452→249.664124 vs 249.664 (294×)
//	S6 plane 360.915871→360.918572 vs 360.919 (7.3×), patch 40.330167→40.327438 vs 40.3269 (6.1×)
//	S4 plane 380.511934→380.476164 vs 380.464 (3.9×), flanks 59.909188/59.908856→59.927068/59.926736 vs 59.9262 (20×/32×)
//	S1 plane 181.115405→181.105272 vs 181.089 (1.6×), flanks 26.591670/26.591632→26.596736/26.596699 vs 26.5949 (1.8×)
//	T1 plane 1241.744096→1241.734749 vs 1241.72 (1.6×), flanks 71.259246/71.259243→71.263920/71.263916 vs 71.2647 (7.0×)
//	T4 plane 1036.353431→1036.348498 vs 1036.34 (1.6×), flanks 57.228821/57.228794→57.231287/57.231260 vs 57.2326 (2.9×)
//	T7 plane 399.192709→399.187669 vs 399.178 (1.5×), flanks 33.210883/33.210881→33.213403/33.213401 vs 33.2141 (4.6×)
//
// Summed per-face |Δ| vs the live rows falls on every rank-paired case: S9 0.5355→0.2014, T3
// 0.3592→0.2871, S4 0.1953→0.1265, S1 0.1114→0.0984, T1 0.2660→0.2473, T4 0.3549→0.3451, S6
// 0.7015→0.6960, T7 2.6507→2.6406 (T7's residual is dominated by its rank-1 oracle-quadrature
// entry, t7_adoption_perface_test.go; its own per-face gate PASSES at the 1.24e-5 whole-body
// budget). S7's per-mover magnitudes are ~1.6e-6 absolute — pInner plane +1.608e-6
// (614.205553186→614.205554794), flanks −8.05e-7 each — with a NET sum of ~−2e-9 (an earlier
// revision of this receipt printed the net as the per-mover figure; corrected by the adversarial
// fix wave). Every row is still far below DRAWEXE's 6-s.f. print resolution and sub-quantum vs
// the 1e-6·boundingDiag pin quantization: S7's pin moved only because that sub-quantum vertex
// drift crossed rounding boundaries and its boundary samples slid ALONG the (unchanged to
// 2.5e-8) locus. Whole-body mesh area and raw volume move ≤6e-8 relative on every case
// — flat at the oracle's 6-s.f. resolution (the movers are complementary by construction); the
// adjudicating measure is per-face + off-surface, and both improve. Triangle counts rise S1 +48,
// S4 +76, S6 +36, S9 +112, T1 +16, T3 +68, T4 +12, T7 +16, S7 +0 — the stations the mesher needs
// to follow six locus curves it used to walk as straight chords.
//
// Off-surface residual (worstLoopSegmentOffFace/boundingDiag, the gate's own function):
// S9 7.927231e-4→3.198130e-9, T3 2.393473e-4→4.900577e-10, S6 1.748055e-4→1.834108e-10,
// S4 8.097143e-5→6.279003e-10, S1 4.157203e-5→1.072957e-9, T1 5.963266e-6→2.172440e-7 — all six
// remaining setback entries DELETED from knownOffSurfaceDebt (28 → 22). knownNilCurveOfferDebt
// stays at its floor of 2 (both sides of every locus edge offer the SAME curve by value — silent
// agreements, no adoption). Free edges 0 and folds 0 at BOTH gate qualities; face counts
// unchanged; the whole-corpus DRAWEXE face-count join unchanged; rollup unchanged at 114 simple /
// 119 all-grid, SkipQuarantine 0. Same arm64 cross-platform-risk caveat as every other pin here.
