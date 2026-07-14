// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
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

// setbackBands is the D2 band partition of a runout edge's interfered span: cutLo/cutHi are the
// outer setback stations (where the plain fillet ends), and seams are the interior boundaries
// splitting the flank band(s) (outer boss only) from the central band (every boss). bosses is
// ordered by xSetback descending (outer reach first) — the order bandsFromBosses assumes.
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
	for _, im := range detectRunouts(ef, res) {
		if cb, ok := setbackBossFrom(im, ef, res); ok {
			bosses = append(bosses, cb)
		}
	}
	if len(bosses) == 0 {
		return setbackBands{}, false
	}
	sortBossesByReach(bosses)
	return bandsFromBosses(bosses), true
}

// setbackBossFrom turns one solved runout imprint into a crossingBoss. The boss wall is read via
// the existing otherFace(footprintEdge, host) idiom (fillet_runout_hosts.go's buildHostBAndWall/
// buildHostAAndWall) and kept intact — no split is called. xSetback is read off the imprint's
// already-solved footprint∩band crossings (solveImprint) rather than re-solving the line∩conic
// from scratch — see setbackStation.
func setbackBossFrom(im runoutImprint, ef edgeFillet, res Resolution) (crossingBoss, bool) {
	cut, ok := solveImprint(im, res)
	if !ok {
		return crossingBoss{}, false
	}
	wall := otherFace(im.footprintEdge, im.host)
	if wall == nil {
		return crossingBoss{}, false
	}
	xs := setbackStation(cut, ef.cyl)
	if !nonDegenerateSetback(xs, res) {
		return crossingBoss{}, false
	}
	return crossingBoss{wall: wall.Geometry(), footEdge: im.footprintEdge, host: im.host, xSetback: xs}, true
}

// setbackStation is the boss's D1 setback reach x_s: half the axial span between its already-
// solved footprint∩band crossings (imprintCut.pMinus/pPlus), projected onto the fillet cylinder's
// spine via spineInterval (corner_runout_region.go, itself built on spineParam). detectRunouts'
// crossings already ARE the ±x_s stations — solveImprint's own doc example (an r8 footprint
// crossing the band at (±√48,y)) — so this reads them off instead of re-deriving D1's closed form
// (√(r_b²−(a−R)²)) independently; verified equal to it to 1e-6 on S1
// (TestDetectSetbackBands_S1TwoBosses: cutHi=√48, seam=√20). spineInterval guarantees lo≤hi, so
// the result is always ≥0 — no separate abs() needed anywhere downstream.
func setbackStation(cut imprintCut, cyl geom.Cylinder) float64 {
	lo, hi := spineInterval(cut, cyl)
	return (hi - lo) / 2
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
// reach: the outer band ends at ±the largest x_s (cutLo/cutHi — the plain-segment/runout
// boundary), and every SMALLER reach becomes an interior seam at ±its own x_s, splitting the
// flank band (outer boss only) from the central band (both bosses). N bosses ⇒ up to 2N−1 bands;
// this only computes the seam stations, not the band geometry itself (Task 3+, unwired here).
func bandsFromBosses(bosses []crossingBoss) setbackBands {
	outer := bosses[0].xSetback
	seams := make([]float64, 0, 2*(len(bosses)-1))
	for _, b := range bosses[1:] {
		seams = append(seams, b.xSetback, -b.xSetback)
	}
	return setbackBands{bosses: bosses, cutLo: -outer, cutHi: outer, seams: seams}
}
