// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"sort"
	"testing"
)

// Scoreboard tallies a fake evaluator's verdicts.
func TestScoreboardTally(t *testing.T) {
	recs := []Record{{Grid: "simple"}, {Grid: "simple"}, {Grid: "simple", TODO: "x"}}
	got := Scoreboard(recs, func(r Record) Outcome {
		if r.TODO != "" {
			return SkipTODO
		}
		return Pass
	})
	if got[Pass] != 2 || got[SkipTODO] != 1 {
		t.Fatalf("tally: %+v", got)
	}
}

// TestOCCTBlendScoreboard prints a per-grid PASS/FAIL/SKIP table over the real corpus and pins the
// HONEST rollup at the isWatertightSolid bar (watertight.go): the simple grid's green count, the
// all-grid green rollup, and SkipQuarantine, so a future gate change or corpus regeneration that
// silently drifts the count fails loud here. It is non-gating on individual CASE outcomes (the
// per-grid table tests in package feature_test are the parity gate — they must never be loosened) but
// IS gating on these aggregate rollup numbers.
//
// Hardened at the #2007 audit (green-gate-validate-audit-report.md): the area-only gate counted 5
// malformed-hole-loop false greens (S6/S9/T3/U3/U4) as PASS; isWatertightSolid closes that gap and
// quarantine.go holds them out. The honest rollup dropped 91→86 simple / 93→88 all-grid, and
// SkipQuarantine rose 1→6 (H6 + the 5). The single-boss setback tiling (#2007) then genuinely GREENED
// S6/S9/T3 (watertight, footprint absorbed), restoring 86→89 simple / 88→91 all-grid and dropping
// SkipQuarantine 6→3 (H6 + U3/U4). The dipArcOrder obstacle-path fix (u3-dipspast-report.md) then
// GREENED U3 (dipsPast was testing the wrong of the two crossing-bounded arcs), restoring 89→90 simple /
// 91→92 all-grid and dropping SkipQuarantine 3→2 (H6 + U4, the residual dual-host engine gap). The U4-5
// dual-host multi-rail weld then GREENED U4, restoring 90→91 simple / 92→93 all-grid and dropping
// SkipQuarantine 2→1 (H6 only). The concave bore-lip rim mirror (fillet_rim_concave.go's
// rimWithCapOrientation) then GREENED K1 (a genuine bore lip, R+r) and Z1 (a plain convex rim whose cap
// face is stored bottom-up — the SAME cap-orientation fix resolves it at R−r), restoring 91→93 simple /
// 93→95 all-grid with SkipQuarantine unchanged at 1.
// The band∩obstacle imprint walk (kernel/ops/fillet_band_imprint.go) then GREENED Y3 (the 100³ box with a
// 10×10 through-slot, filleted r=20 along the y=0 ∧ z=100 edge): the setback line lands on the slot's
// FLOOR, so the receded host boundary ran through the slot's own profile and the assembled shell was
// FAIL(faulty). The walk imprints the band on the slot's three walls and re-trims all five affected faces
// together, restoring 111→112 simple / 116→117 all-grid, SkipQuarantine unchanged at 1. Y2 and Y4 were
// already PASS and stay PASS — their +0.159 % / +0.645 % body-area errors were hidden INSIDE the 1 % gate,
// and are now zero to their closed forms (slot_band_imprint_test.go).
func TestOCCTBlendScoreboard(t *testing.T) {
	dir := CorpusFixtureDir()
	byGrid := map[string]map[Outcome]int{}
	for _, r := range Corpus() {
		if byGrid[r.Grid] == nil {
			byGrid[r.Grid] = map[Outcome]int{}
		}
		byGrid[r.Grid][ScoreCase(r, dir)]++
	}

	grids := make([]string, 0, len(byGrid))
	for g := range byGrid {
		grids = append(grids, g)
	}
	sort.Strings(grids)

	order := []Outcome{Pass, PassDeviation, FailArea, FailFaulty, SkipTODO, SkipImportDivergence, Incomplete, SkipQuarantine}
	total := map[Outcome]int{}
	t.Logf("OCCT blend parity scoreboard (non-gating):")
	for _, g := range grids {
		line := g
		for _, o := range order {
			n := byGrid[g][o]
			total[o] += n
			line += "  " + o.String() + "=" + itoa(n)
		}
		t.Log(line)
	}
	sum := 0
	tline := "TOTAL"
	for _, o := range order {
		tline += "  " + o.String() + "=" + itoa(total[o])
		sum += total[o]
	}
	t.Log(tline)
	passRollup := 0
	for o, n := range total {
		if o.IsPass() {
			passRollup += n
		}
	}
	t.Logf("PASS rollup (Pass + PASS(deviation)) = %d", passRollup)
	if sum != len(Corpus()) {
		t.Fatalf("scoreboard counted %d cases, want %d", sum, len(Corpus()))
	}
	assertHardenedRollup(t, byGrid, passRollup, total[SkipQuarantine])
}

// assertHardenedRollup pins the honest rollup at the isWatertightSolid bar: 104 green in the
// simple grid, 109 green across all grids, and SkipQuarantine=1 (H6 only). The single-boss setback tiling
// greened S6/S9/T3 (86→89 / 88→91, SkipQuarantine 6→3); the dipArcOrder obstacle-path fix then greened
// U3 (89→90 / 91→92, SkipQuarantine 3→2); the U4-5 dual-host multi-rail weld then greened U4 (90→91 /
// 92→93, SkipQuarantine 2→1). The concave bore-lip rim mirror (fillet_rim_concave.go's
// rimWithCapOrientation, K1/Z1) then greened both: K1 is a genuine bore lip (the R+r mirror, material
// outside the bore) and Z1 turned out to be a plain CONVEX rim whose cap face happens to be stored
// bottom-up (capF.Reversed()==true) — solveRim's raw pl.Normal() offset the torus centre to the wrong
// side of the cap, failing its R−r probe for a reason unrelated to concavity; the same cap-orientation fix
// resolves it at R−r, not R+r (91→93 / 93→95), leaving only H6 (concave open-torus, ROOT 2) held. The
// R+r bore/notch-wall trihedral corner (corner-blend-weld Pieces 1+2) then GREENED N1 (box − r20
// cylinder notch; the engine used to mirror the corner into the void at R−r, now solves at R+r and
// welds pass-through faces to their rebuilt neighbours), restoring 93→94 simple / 95→96 all-grid. The
// bore far-cap OUTWARD extension (Piece 3) then GREENED L9 (box − r30 quarter-cylinder notch: the R+r
// torus far cross-section reaches past the rim, growing the adjacent flat faces outward instead of
// biting inward), restoring 94→95 simple / 96→97 all-grid, SkipQuarantine unchanged at 1. The
// mixed-sense curved-host 2r-torus corner WELD (corner-blend-weld Slice-1b, fillet_curved_mixed_weld.go)
// then GREENED M8 (box + boss, one convex Cyl∧Plane arm + a concave cove torus arm + a planar arm meeting
// at a curved-host trihedral vertex: no single ball is tangent to the boss wall at both R−r and R+r, so
// OCCT builds an analytic 2r-torus corner patch — this weld trims the three arm faces at the corner arcs,
// retrims the five bitten hosts, and assembles the 14-face solid), restoring 95→96 simple / 97→98
// all-grid, SkipQuarantine unchanged at 1. The concave BOSS-BASE rim cove (fillet_rim_concave.go's
// solveConcaveBossRim) then GREENED W6/W8 (simple) and A1 (bfuseblend): a boss filleted at its base built a
// watertight, DRAWEXE-exact-face-count solid but with solveRim's convex R−r round biting the reentrant
// corner INWARD (a STABLE +3.8%/+5.5% footprint over-size) instead of OCCT's concave R+r cove — the fix
// offsets the tube centre toward the boss and opens the plate hole to R+r, matching OCCT to −0.000%; a
// cap-fit gate leaves the deep #2012 spillers (R8/W9) on the unchanged ladder — restoring 96→98 simple /
// 98→101 all-grid, SkipQuarantine unchanged at 1. The elliptic-prism ruling fillet (fillet_ellipticalarm.go)
// then GREENED F4 (simple): a convex STRAIGHT ruling edge where a plane meets a right EllipticalCylinder
// wall (an oblique-prism side) was FLAT-REFUSED (curvedAdjacentError) because an elliptic host offset is a
// non-analytic canal — but on a STRAIGHT (∥-axis) edge the rolling-ball spine is a straight line, so the
// fillet collapses to an EXACT right circular cylinder; it is built analytically and welded through the
// existing single-arm curved runout with an ellipse-aware host retrim (fillet_curved_retrim_ellipse.go),
// restoring 98→99 simple / 101→102 all-grid, SkipQuarantine unchanged at 1. The asymmetric planar-miter
// seam (fillet_miter_asymmetric.go) then GREENED P9/V9 (simple): two filleted box-top edges of DIFFERENT
// radii (r1/r0.5) sharing a face were guard-rejected ("must use one radius") because the equal-radius miter
// samples cyl1 ∩ the nF1−nF2 mirror plane — a shortcut valid only when the two cylinders coincide there;
// unequal cylinders do not, so the seam is now the TRUE cyl0 ∩ cyl1 intersection (every point on both
// cylinders, welding the two arm faces watertight), matching OCCT's 8-face solid at −0.21%. The equal-radius
// corner path is untouched (routed only when radii differ), so every fingerprint pin stays byte-identical.
// This restored 99→101 simple / 102→104 all-grid, SkipQuarantine unchanged at 1. The CLOSED elliptic-rim
// canal band (fillet_elliptic_rim_canal.go) then GREENED J6/J8 (simple): a CLOSED rim where a plane meets an
// oblique-extrusion EllipticalCylinder wall was FLAT-REFUSED, because — unlike F4's straight ruling, whose
// spine is a line and whose fillet collapses to a right circular cylinder — a closed elliptic rim's
// rolling-ball spine is a closed NON-ANALYTIC curve, so the envelope is a genuine variable-section canal.
// It is built from the closed-form spine (exact centre + exact wall/plane feet per station), lofted with
// geom.LoftCanalStations and assembled through the SAME host-agnostic rim rebuild the analytic torus bands
// use. BOTH carry a per-case exact-deviation, so they score PassDeviation: OCCT's checkprops number comes
// from `sprops <shape> 1e-4`, which mis-integrates a SurfaceOfLinearExtrusion face by +2.4..4.7% (receipt:
// the UN-BLENDED J8 pipe wall is exactly 38201.978, OCCT plain sprops 38202.0, OCCT sprops 1e-4 39104.4) —
// OCCT's OWN blend measured with its accurate integrator is 77932.8 (J6) / 51078.3 (J8) against our exact
// 77931.429 / 51076.805. The SAME slice also greened bfuseblend A7/B1 outright (no deviation): they are the
// CONCAVE members of the vein — T5/U2's two shapes on a plate wide enough that the foot ring FITS — and they
// land on OCCT's own numbers at −0.041% / −0.020% with DRAWEXE-exact topology (9 faces / 11 vertices / 17
// edges). They green only because the convexity gate reads the SOLID (the imported elliptic face's Reversed
// flag mis-classifies them). This restored 101→103 simple / 104→108 all-grid, SkipQuarantine unchanged at 1.
// The GENERAL corner-weld layer (cornerweld_*.go) then GREENED N4 (simple): a full r20 cylinder standing on a
// 100³ box's vertical corner, filleted r5 on three edges meeting at one trihedral vertex. Its geometry (the
// four corner points, the two on-host rails, the NoFold-certified corner patch) had been committed and
// DRAWEXE-validated for four prior pieces; what was missing was the WELD, and it needed two stage variants
// that now live in the shared layer rather than in N4: (a) the convex cap-rim arm's far vertex is a G1 SEAM,
// not a cap — only the 90° piece of the 270° rim was picked and the rim continues tangentially across the boss
// wall's second face, so the arm runs THROUGH the seam to the end of the tangent chain (farRimContinuation);
// (b) crossing that seam splits the band into one face per host-face span, which is the C4 split. OCCT's own
// blend does exactly this (its oracle carries the band over all 270° as two faces, 76.3° + exactly 180°, and
// recedes BOTH wall faces to z=45), so the "cap-less rim far termination" the recon predicted is really a
// continuation. Ours: watertight 14-face solid, area 64287.18 vs OCCT 64287.2 (−0.000035%), volume 1046938.9
// vs DRAWEXE vprops 1.04694e6 (−0.000109%) — and reconciled PER FACE, since the corner fill became the true
// rolling-ball canal (the earlier +0.008% was two 27%/+29.7 face errors cancelling). This restored 103→104
// simple / 108→109 all-grid, SkipQuarantine unchanged at 1. Slice 2 of the same layer then GREENED O1 — the
// layer's own falsification test, and it passed as a pure CONFIGURATION (cornerweld_class_o1.go): a boss
// cylinder fused to a box, filleted r=5 on three edges meeting at one trihedral vertex, whose two CONCAVE
// arms (a cylinder arm and a cove torus arm, roll-sense regime R2 → R+r) terminate at the corner while the
// CONVEX planar band runs past it, so the corner patch is the rolling-ball canal of a ball riding the boss
// CYLINDER and rolling on the band's tube (the cylinder-on-cylinder sibling of N4's plane-on-torus frame).
// Ours: watertight 12-face solid, area 65101.43 vs OCCT 65104.9 (−0.0053%), volume 1111282.0 vs DRAWEXE
// vprops 1.11166e6 (−0.034%), reconciled per face — eight of twelve faces to ≤2.5e-5 relative, the three
// around the corner carrying OCCT's OWN patch approximation (its interior sits 1.16% of r off the exact
// envelope, and its implied rolling ball misses tangency to the boss wall by 0.174). This restored 104→105
// simple / 109→110 all-grid, SkipQuarantine unchanged at 1. (The mixed-radius TRIHEDRAL
// corner A4, r10/r5/r5, needs a torus corner patch — declined for now, a tracked follow-up.)
// Generalizing the MIXED-SENSE torus corner to its mirrored signature — 1 concave + 2 convex, where the
// CONCAVE band pivots (fillet_corner_torus.go's splitMixedSense) — then GREENED B5 and C4, restoring
// 105→107 simple / 110→112 all-grid, SkipQuarantine unchanged at 1. Both had been carrying a corner sphere
// solved r inside all three planes, which is the convex-only answer: on a 270°-sector corner it lands 2r
// off the concave arm's spine, so that arm's own band boundary sat 12.36 (94% of r) off its own cylinder.
// DRAWEXE 8.0.0 dumps the corner as geom.Torus(centre on the concave spine, axis = concave edge, R=2r, r)
// — see offsurface-loopseg-report.md.
// Trimming a fillet band's FAR END against the wall it stops on, instead of squaring it off in the section
// plane at the filleted edge's end vertex (fillet_farend_trim.go), plus enforcing that a carried rim arc
// spans only its OWN loop segment (fillet_survivor_rim.go's alignCarriedArcsToSegments), then GREENED C7, D7
// and complex/D8, restoring 107→109 simple / 112→115 all-grid, SkipQuarantine unchanged at 1. C7 was the
// extreme run-on: a `pcone s 50 0 120 90` APEX cone whose single cap-edge band ran on to the flank's
// parametric extreme, shipping +17.67% body area — it now measures −0.0022%, with its per-face gross error
// vs DRAWEXE 2204.07 → 0.25. D7's −45.3% "host sphere-zone tessellation defect" turned out to be the
// carried-arc overshoot, not the zone mesher: its body area goes −30.27% → −0.0795% and its per-face gross
// error 83508.25 → 217.95.
// Carrying an ELLIPTIC survivor rim through the corner/tangent-point substitution instead of chording it
// (fillet_survivor_rim_ellipse.go — the circle-only subArcOnParent/arcFrac/projectOntoArcCircle got an
// eccentric-angle sibling) then GREENED F6, restoring 109→110 simple / 115→116 all-grid, SkipQuarantine
// unchanged at 1. F6's top cap rim had been shipping as a straight chord, tiling its top plane at 72422.5
// against DRAWEXE's 31060.7 and its elliptic wall at 23469.8 against 39942.6 (+5.6762% body area, and a
// volume 45% low); it now measures −0.0011% body area with every face inside 0.006% of DRAWEXE. The same
// fix took already-green F7 from the corpus's WORST per-face gross error, 40793 (its 39896.5 elliptic wall
// meshing 60210.7, its 27897.8 top plane meshing 7419) to 2.8, body area −0.1234% → −0.0006%.
// Carrying the rim-fillet HOST SEAM as the retained sub-span of the host's own meridian instead of chording
// it to a straight ruling (fillet_rim_build.go's wallSeamCurve → retainedHostSeamCurve) then GREENED J4,
// restoring 110→111 simple / 116→117 all-grid, SkipQuarantine unchanged at 1. The rebuild had assumed the
// re-aimed host seam is always a straight ruling — true on a cylinder / cone / elliptical-cylinder host, but
// a SPHERE or TORUS host's meridian is an ARC, so J4's torus host shipped a 61.24-long chord 10.43 off
// itself and tiled 394781 against DRAWEXE's 94600.6 (+70.2% body area); it now measures −0.0002% with every
// face inside 0.0004%. The same fix took already-green J2's sphere seam from a 90.38-long chord 28.44 off
// its own sphere — the corpus's LARGEST off-surface residual, rel 0.303 — to 2.7e-14 (its mesh was already
// faithful, because the sphere-zone fan does not read the seam curve; the defect was a corrupt B-rep every
// other consumer would inherit). A mismatch means either a false green slipped through or a case was
// over/under-quarantined — see harden-green-gate-brief.md's "STOP and report" instruction.
//
// ★ THE LAST QUARANTINE IS GONE (112→114 simple / 117→119 all-grid, SkipQuarantine 1→0). The arc band's
// rolling-ball SEAT (kernel/ops/fillet_arc_seat.go — the seat is solved from the picked edge's convexity
// and the faces' OUTWARD normals, where it used to be hard-coded to the convex-shaft cylR−r on the stored
// plane normal) plus the RUN-OUT TERMINATION (fillet_arc_runout.go — the band terminated on the side
// plane's own spiric section, which is what OCCT does) GREENED simple/W2 and freed simple/H6, the corpus's
// last held case. Both are one root: a cylR+r band that the old code seated at cylR−r on the wrong side of
// the cap. Measured against DRAWEXE 8.0.0, face for face: W2 ships OCCT's SEVEN faces at 11.766423 vs
// 11.76665 (−0.0015%, was +12.47% on nine faces), H6 ships OCCT's EIGHT at 555913.3 vs 555915 (−0.00031%,
// was +3.65%). The same termination also took already-green simple/N6 from ten faces to DRAWEXE's NINE
// (−0.0113% → −0.0068%), retiring a setback triangle whose tip stood inside the pocket the cut removed.
// Ratchets: knownMeshLeaks 7 → 6 cases (W2's 3/3 retired; H6 enters the sweep at 0/0 where it hid 642),
// knownRetracingLoops 4 loops/3 cases → 1/1 (W2's 2 and N6's 1 retired), knownSelfCrossingLoops 3 loops/2
// cases → 2/1 (W2's retired; H6 enters at 0 where it hid 2).
//
// ★ complex/D8's green was RETIRED (all-grid 117 → 116, simple unchanged at 111) because it was never
// parity: it was MANUFACTURED by the far-end trim's own arbitrary branch pick and is unreachable by any
// correct single-edge fillet. Receipts, all measured (stopface-reversed-report.md):
//   - OCCT builds a DIFFERENT SOLID. DRAWEXE 8.0.0 `blend result a 30 a_20` ships 18 faces / 40 edges, and
//     every one of its `sprops` numbers decodes as a fillet of the WHOLE tangent-continuous closed top-edge
//     loop (8 edges): two long walls at 462.843×70 = 32399, two short at 162.916×70 = 11404.1, FOUR corner
//     cylinders at 37.699×70 = 2638.94, plus 2+2+4 band faces (21528.2 / 7394.46 / 1215.41) and the
//     68039.2 top. We fillet the ONE picked edge and ship 11 faces, leaving the other three walls and all
//     four corner cylinders at their full ×100 height (46284.3 / 16291.5 / 3769.89) — so the comparison is
//     between two different bodies. Tangent-chain PROPAGATION is the missing capability.
//   - The correct single-edge area is +3.52%, i.e. outside the 1% gate no matter what. Closed form for our
//     own construction: band = 30∫₀^{π/2}(462.843 + 2√(576−(30cos φ − 6)²))dφ = 23340.06 (we mesh 23339.7),
//     top plane 92175.2 (we mesh 92175.2, exact), corner cylinders 3769.911 − 464.006 = 3305.906 each →
//     348183 total, vs OCCT's 336159. Those are the SOLO per-face meshes; the body used to SHIP 344843
//     (−0.90%) because the conformance repair adopted three area-destroying re-meshes on a fold count
//     alone, and it now ships 348296 (+0.03% of its own closed form) with the adoption criterion fixed
//     (kernel/ops/conformance_adopt.go, conform-area-report.md). The case stays FAIL(area) either way:
//     the gap to OCCT is the two different SOLIDS above, not the mesh.
//   - The old +0.0437% was a four-face cancellation. The trim's zigzag station list made the band mesh
//     6142.29 against its true 23340 (−74%) while the two corner cylinders over-read (4887.9 / 4664.51
//     against 3769.89, on faces the fillet should have SHRUNK) and the top plane over-read by 2379 — and
//     the four errors summed to +147 on a 336159 body. The far-end trim itself created that green
//     (see the C7/D7/complex/D8 note above); it is now withdrawn.
//
// The NOTCH-WALL concave cove sign (concaveTorusArmSurface's ε, W-DH capability wave) then GREENED M5
// (simple, box − quarter-cylinder notch, three all-concave r5 edges at one trihedral vertex): the concave
// circle-edge cove arm hardcoded major = R+r (the boss convention), but on a notch the material sits
// OUTSIDE the wall and the ball rolls INSIDE it — DRAWEXE ships the cove as maj = R−r = 25 with a plain
// r-sphere corner ON that spine at (45,14.4949,45), while our R+r arm put the corner ball 2r off its own
// spine and the corner solve honest-declined. ε now comes from cylinderHostRadialSign (the SAME n_C·r̂
// read the concave LINE arm has always used), so every boss cove (ε=+1: M8/O1/N4/B5/C4) is byte-identical
// — the full-corpus sweep shows exactly ONE verdict change (M5 FAIL(faulty)→PASS) and zero area drift
// elsewhere. M5 reconciles per face against DRAWEXE `sprops 1e-12` on all 13 faces (sphere corner
// 44.301 vs 44.3039, cove torus 222.800 vs 222.817, worst face −0.008% — tessellation quadrature), body
// area −0.000% of 61187.1, volume 980008.93 vs vprops 980008, watertight at the full bar. This restored
// 114→115 simple / 119→120 all-grid, SkipQuarantine unchanged at 0.
// The MULTI-RIM weld (fillet_curved_multirim.go, same wave) then GREENED bfuseblend/B3: a
// through-cylinder fused to a box leaves TWO closed concave Cylinder∧Plane rims (one per exit face),
// each individually the proven A1-class boss cove — but the multi-pick op fell through to the trihedral
// weld and floored on "needs 3 arms (got 1)". There is no corner (the rims never touch); ≥2 pairwise
// face-disjoint closed rims now route through the single-rim band rebuild SEQUENTIALLY, the next rim's
// ReferenceKey resolving on the intermediate body (the rim rebuild carries untouched Lineage verbatim).
// Reconciled per face vs DRAWEXE on all 12 faces (worst +0.0016%); the −0.998% vs the corpus number is
// the historic reference's drift — our 78062.6 vs local DRAWEXE 8.0's own 78062.8 (−0.0003%). This
// restored 120→121 all-grid (simple unchanged at 115), SkipQuarantine unchanged at 0.
// W-G's BSpline-host fillet engine (kernel/ops/fillet_bspline_host*.go — the edge-anchored
// section-plane Newton march) greened seven simple cases (G5 G7 G9 I5 I7 J9 V6) and one
// bfuseblend case (B2, a bezier pipe fused to a box): 120→127 simple, 128→136 all-grid. G5/G9
// needed one shared-code fix first — fillet_blend_faces.go's reversedEndSeg dropped a segment's
// srcEdge identity on reversal, which farPathSegs' "other way" branch runs on every retained
// segment of a double-bite capping face (both ends of an open runout landing on ONE side wall);
// without it the assembler's edge catalog fused the retained portion of a bigon host's straight
// closing chord with an unrelated new rail sharing its endpoints into one non-manifold edge.
// Every other new green built clean. I6 (a 3-pick op combining two independent single-arm
// runouts that share a host face with a genuine 2-arm corner) and T8 (concave closed rim; the
// cap's hole loop protrudes past its outer boundary, fillet_rim_build.go, Cluster K's simple/T9
// territory) stay FAIL(faulty), declining into shared dispatch/build code outside this
// cluster's owned files — reported as merge-train seams, not built around. C6/C8 stay
// FAIL(faulty) too: their far vertex touches TWO non-host transverse faces (cappingFaceAtFarVertex,
// fillet_far_runout.go), a genuine multi-face corner termination this wave's single-flat-cap
// Newton engine does not attempt. encoderegularity/A1+A4 remain DEFERRED per the wave brief
// (integration-last, need C+T+D-class welds at 11/51 picks).
func assertHardenedRollup(t *testing.T, byGrid map[string]map[Outcome]int, allGridGreen, skipQuarantine int) {
	t.Helper()
	simpleGreen := byGrid["simple"][Pass] + byGrid["simple"][PassDeviation]
	if simpleGreen != 127 {
		t.Errorf("simple grid green (Pass+PassDeviation) = %d, want 127 (W-DH's notch-wall concave cove "+
			"sign ε=−1 greened M5; W-B's Cylinder∧Cylinder SSI-seam canal engine greened K2/K3/K4 + P1; "+
			"W-T's stripe-junction crossings + exact anchor distance greened Y9; W-G's BSpline-host "+
			"engine greened G5 G7 G9 I5 I7 J9 V6; 114→120→127)", simpleGreen)
	}
	if allGridGreen != 136 {
		t.Errorf("all-grid green (Pass+PassDeviation) = %d, want 136 (127 simple + 9 bfuseblend; the multi-rim "+
			"weld greened bfuseblend/B3, the closed cyl∧cyl seam canal greened B4+B5, and W-G's BSpline-host "+
			"engine greened bfuseblend/B2; complex/D8's coincidental green stays retired; 119→128→136)", allGridGreen)
	}
	if skipQuarantine != 0 {
		t.Errorf("SkipQuarantine = %d, want 0 — the corpus holds NO case; every one of the 475 records is "+
			"now scored on its own merit (quarantine.go, TestTheCorpusHoldsNoCase)", skipQuarantine)
	}
}

// itoa avoids importing strconv for one call.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
