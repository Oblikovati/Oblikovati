// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// crossingBoss is one transverse boss that crosses ef's runout edge and is kept INTACT (never
// split — the correction over the old split-boss tiler, setback-patch-derivation.md's "Candidate
// method ii"): the plain fillet is trimmed back to xSetback and the interfered middle is later
// filled with a G¹ setback patch that fairs out to wall along footEdge (Task 3+, unwired here).
type crossingBoss struct {
	wall     geom.Surface // the INTACT boss wall (Cylinder/Cone/Torus/SurfaceOfLinearExtrusion)
	footEdge *topo.Edge   // the footprint edge on the host plane (exact conic curve)
	host     *topo.Face   // the support plane the boss exits
	xSetback float64      // D1 setback reach |x_s| ≥ 0 — see setbackStation
}

// setbackBands is the D2 band partition of a runout edge's interfered span: cutLo/cutHi and seams
// are ABSOLUTE stations on the fillet cylinder's own spine (mid±x_s — NOT half-widths about
// spine-0) where the plain fillet ends and where the interior band boundaries fall. cutLo/cutHi
// are the outer setback stations (where the plain fillet ends), and seams are the interior
// boundaries splitting the flank band(s) (outer boss only) from the central band (every boss).
// bosses is ordered by xSetback descending (outer reach first) — the order bandsFromBosses
// assumes. Downstream (Task 3+) reads cutLo/cutHi/seams directly as spine coordinates to place the
// fillet-end cross-section arcs, so they must be absolute — a half-width representation about an
// arbitrary spine-0 would place those arcs at the wrong stations and the patch rail loops built
// from them would not close when the path is wired (Task 5).
//
// The whole partition assumes every boss shares the SAME transverse midplane — D2's "on the SAME
// centered span" — because bandsFromBosses places every station relative to that ONE shared plane,
// mid: cutLo=mid−outer, cutHi=mid+outer, seams=mid±x_s. mid equals the true plain-fillet ends'
// center ONLY when every boss's own axial midpoint c=(lo+hi)/2 (setbackMidpoint) agrees with the
// others'. mid is NOT v=0 (spine-0) in the fillet cylinder's own spine frame — cyl.Origin sits at
// an arbitrary edge endpoint, so a bare c carries no absolute meaning in isolation, only agreement
// BETWEEN bosses does (verified against S1: both its bosses read c≈10, not 0 — so cutLo/cutHi come
// out as 10∓√48≈{3.0718,16.9282}, NOT ∓√48≈{−6.9282,+6.9282} about a false origin). A boss whose
// midpoint disagrees with the rest is rejected outright by detectSetbackBands' centeredness guard
// (bossesShareTransverseMidplane; this milestone honest-defers rather than emit a wrong-but-
// plausible symmetric split); supporting real disagreement needs a richer per-boss lo/hi
// representation than this single xSetback field carries (future milestone). A LONE boss has
// nothing to disagree with, so this guard cannot catch a single off-center boss — only a multi-
// boss mismatch (same future-milestone gap).
type setbackBands struct {
	bosses       []crossingBoss
	cutLo, cutHi float64
	seams        []float64
}

// setbackNearZeroCoef is the dimensionless floor (a multiple of res.Weld()) below which a
// computed setback reach reads as "no real interference" rather than a genuine crossing: D1's
// closed form (x_s = √(r_b²−(a−R)²)) loses precision as (a−R)→r_b (the derivation's "Numerical
// pitfalls" setback-conditioning guard), so a barely-positive x_s born of float noise at the
// tangency singularity must not pass. 64 sits comfortably above weld-scale noise and far below
// any real corpus-scale reach (S1's smallest is √20≈4.47).
const setbackNearZeroCoef = 64

// detectSetbackBands finds every crossing boss on ef's runout edge (Task 1's detectRunouts,
// unchanged — this reuses it rather than re-deriving boss detection), keeps each boss wall
// INTACT, and computes the D1 setback stations + D2 band partition a later task tiles into
// setback patches. NEW AND UNWIRED: no caller in runoutFacesFor yet (M3 Task 2,
// .superpowers/sdd/task-2-brief.md) — the corpus stays byte-identical until a later task wires
// this in.
func detectSetbackBands(ef edgeFillet, res Resolution) (setbackBands, bool) {
	bosses := make([]crossingBoss, 0, 2)
	mids := make([]float64, 0, 2)
	for _, im := range detectRunouts(ef, res) {
		cb, c, ok := setbackBossFrom(im, ef, res)
		if !ok {
			continue
		}
		bosses = append(bosses, cb)
		mids = append(mids, c)
	}
	if len(bosses) == 0 {
		return setbackBands{}, false
	}
	if !bossesShareTransverseMidplane(mids, ef.cyl) {
		return setbackBands{}, false // off-center boss: honest-defer, see bossesShareTransverseMidplane
	}
	sortBossesByReach(bosses)
	// mids[0]: any already-agreeing mid works (bossesShareTransverseMidplane just verified
	// agreement within tolerance) — the first is as good as the mean and cheaper to compute.
	return bandsFromBosses(bosses, mids[0]), true
}

// setbackBossFrom turns one solved runout imprint into a crossingBoss plus its axial midpoint c
// (setbackMidpoint — returned separately, not stored on crossingBoss, since only
// detectSetbackBands' cross-boss centeredness guard needs it). The boss wall is read via the
// existing otherFace(footprintEdge, host) idiom (fillet_faces.go) and kept intact — the wall is
// merely subdivided later by reclipOuterHost/reclipInnerHost (fillet_setback_close.go), never
// split. xSetback is read off the imprint's already-solved footprint∩band crossings (solveImprint)
// rather than re-solving the line∩conic from scratch — see setbackStation.
func setbackBossFrom(im runoutImprint, ef edgeFillet, res Resolution) (crossingBoss, float64, bool) {
	cut, ok := solveImprint(im, res)
	if !ok {
		return crossingBoss{}, 0, false
	}
	wall := otherFace(im.footprintEdge, im.host)
	if wall == nil {
		return crossingBoss{}, 0, false
	}
	xs := setbackStation(cut, ef.cyl)
	if !nonDegenerateSetback(xs, res) {
		return crossingBoss{}, 0, false
	}
	cb := crossingBoss{wall: wall.Geometry(), footEdge: im.footprintEdge, host: im.host, xSetback: xs}
	return cb, setbackMidpoint(cut, ef.cyl), true
}

// setbackStation is the boss's D1 setback reach x_s: half the axial span between its already-
// solved footprint∩band crossings (imprintCut.pMinus/pPlus), projected onto the fillet cylinder's
// spine via spineInterval (fillet_spine.go, itself built on spineParam). detectRunouts'
// crossings already ARE the ±x_s stations — solveImprint's own doc example (an r8 footprint
// crossing the band at (±√48,y)) — so this reads them off instead of re-deriving D1's closed form
// (√(r_b²−(a−R)²)) independently; verified equal to it to 1e-6 on S1
// (TestDetectSetbackBands_S1TwoBosses: cutHi=√48, seam=√20). spineInterval guarantees lo≤hi, so
// the result is always ≥0 — no separate abs() needed anywhere downstream.
//
// This DISCARDS the interval's midpoint c=(lo+hi)/2 (see setbackMidpoint) — only the half-span
// survives here; bandsFromBosses re-attaches the shared c (passed in as mid) when it turns this
// half-span into an ABSOLUTE spine station (mid±x_s), see bandsFromBosses. mid+x_s and mid−x_s
// equal the true plain-fillet ends ONLY when every boss on the edge shares the SAME c (D1's "Fᵢ a
// conic centered on the transverse plane x=0" — a plane shared by all bosses, not necessarily v=0
// in cyl's own frame, see setbackMidpoint); a boss whose c disagrees with the others is rejected by
// bossesShareTransverseMidplane before xSetback is ever trusted downstream.
func setbackStation(cut imprintCut, cyl geom.Cylinder) float64 {
	lo, hi := spineInterval(cut, cyl)
	return (hi - lo) / 2
}

// setbackMidpoint is the boss's axial midpoint c=(lo+hi)/2 on the fillet cylinder's spine — the
// projection setbackStation discards when it keeps only the half-span. D1's closed form assumes
// every boss on the edge shares the same c (setback-patch-derivation.md: "Fᵢ a conic centered on
// the transverse plane x=0"), but that plane carries NO absolute meaning in cyl's own coordinate
// frame: cyl.Origin sits wherever the edge's solve happened to place it (an imported-STEP
// endpoint, not the physical midplane), so c itself is an arbitrary offset — verified against the
// S1 fixture, both its crossing bosses read c≈10, not 0. Only AGREEMENT between bosses'
// setbackMidpoint values is meaningful; bossesShareTransverseMidplane checks exactly that.
func setbackMidpoint(cut imprintCut, cyl geom.Cylinder) float64 {
	lo, hi := spineInterval(cut, cyl)
	return (lo + hi) / 2
}

// setbackCenteredCoef is the dimensionless, GENEROUS fraction of the fillet radius R below which
// two bosses' axial midpoints (setbackMidpoint) read as "the same transverse plane" rather than
// genuinely disagreeing — the do-no-harm guard this Task 2 review finding calls for:
// bandsFromBosses hard-assumes every boss shares one plane and silently mis-partitions if that's
// false, so bossesShareTransverseMidplane must reject (ok=false) loudly rather than emit a
// plausible-looking wrong split. 1% of R sits many orders of magnitude above the floating-point
// noise spineParam's dot-product projections carry (~1e-14 relative to model size, the same order
// TestNonDegenerateSetback_NearTangencyClamp's weld-scale floor guards against — confirmed on the
// real S1 fixture, whose two independently-solved bosses agree on c=10 to 4e-15) — genuinely
// agreeing bosses NEVER trip this — yet stays far below any disagreement that would actually
// matter: two bosses on truly different transverse planes differ by a meaningful fraction of
// their own footprint radius or of R, not by a percent of it. R (not res.Weld()) is the scale
// because c lives in the same axial coordinate as R (both measured off the fillet cylinder's own
// frame), so it tracks the LOCAL fillet's size even when the document-wide Resolution's model
// size differs from it.
const setbackCenteredCoef = 0.01

// bossesShareTransverseMidplane is the do-no-harm centeredness guard (Task 2 review finding,
// corrected against real S1 data — see setbackMidpoint for why a literal |c|≈0 check would have
// rejected the very corpus case it must pass): true when every boss's axial midpoint agrees with
// the first, within setbackCenteredCoef·R, so bandsFromBosses' shared-plane assumption (D2: "on
// the SAME centered span") holds. detectSetbackBands calls this over ALL detected bosses before
// trusting any of their xSetback; a false result means honest-defer (ok=false), never a silent
// symmetric split built from bosses that don't actually share a plane. A single boss (len(mids)
// ==1) has nothing to disagree with and trivially passes — this guard cannot detect a LONE
// off-center boss, only a multi-boss mismatch (see setbackBands' doc comment).
func bossesShareTransverseMidplane(mids []float64, cyl geom.Cylinder) bool {
	for _, m := range mids[1:] {
		if !withinCenteredTolerance(m-mids[0], cyl) {
			return false
		}
	}
	return true
}

// withinCenteredTolerance is true when offset sits within setbackCenteredCoef·R of zero — the
// generous, R-relative floor bossesShareTransverseMidplane compares each boss's midpoint
// disagreement against. See setbackCenteredCoef for why 1% of R can never be tripped by
// floating-point noise alone.
func withinCenteredTolerance(offset float64, cyl geom.Cylinder) bool {
	return math.Abs(offset) <= setbackCenteredCoef*cyl.Radius
}

// nonDegenerateSetback rejects a boss reach too close to zero to be a genuine crossing — the D1
// conditioning guard: compare x_s² against the (setbackNearZeroCoef·Weld)² floor rather than
// trusting a barely-positive root near the tangency singularity (a boss just grazing the contact
// line must read as "no interference", not a vanishing band).
func nonDegenerateSetback(xSetback float64, res Resolution) bool {
	floor := setbackNearZeroCoef * res.Weld()
	return xSetback*xSetback >= floor*floor
}

// sortBossesByReach orders bosses by xSetback descending (outer reach first) — the ordering
// bandsFromBosses' D2 partition assumes (xSetback is always ≥0, see setbackStation).
func sortBossesByReach(bosses []crossingBoss) {
	sort.Slice(bosses, func(i, j int) bool { return bosses[i].xSetback > bosses[j].xSetback })
}

// bandsFromBosses computes D2's 3-patch partition from bosses already ordered outer→inner by
// reach, placed at ABSOLUTE positions on the fillet cylinder's own spine about the shared
// transverse midplane mid (NOT spine-0 — see setbackBands' and setbackMidpoint's doc comments for
// why a bare spine coordinate carries no meaning in isolation): the outer band ends at mid∓the
// largest x_s (cutLo/cutHi — the plain-segment/runout boundary), and every SMALLER reach becomes
// an interior seam at mid∓its own x_s, splitting the flank band (outer boss only) from the central
// band (both bosses). N bosses ⇒ up to 2N−1 bands; this only computes the seam stations, not the
// band geometry itself (Task 3+, unwired here).
//
// mid must be the ONE shared transverse midplane every boss agrees on — detectSetbackBands passes
// mids[0] (any of the already-agreeing mids would do: bossesShareTransverseMidplane has already
// verified they agree within tolerance, so the first is as good as the mean and cheaper). Like
// setbackStation, this hard-ASSUMES every boss shares that SAME transverse midplane — it has no
// lo/hi (or c=(lo+hi)/2) of its own to check agreement against, only xSetback (the half-span).
// Callers must reject disagreeing bosses upstream (bossesShareTransverseMidplane, in
// detectSetbackBands) before any boss ever reaches here.
func bandsFromBosses(bosses []crossingBoss, mid float64) setbackBands {
	outer := bosses[0].xSetback
	seams := make([]float64, 0, 2*(len(bosses)-1))
	for _, b := range bosses[1:] {
		seams = append(seams, mid+b.xSetback, mid-b.xSetback)
	}
	return setbackBands{bosses: bosses, cutLo: mid - outer, cutHi: mid + outer, seams: seams}
}
