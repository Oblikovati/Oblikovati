// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// buildCoreLoop assembles the CORE panel's valence-4 RailLoop (derivation §1.4 core table, §4-U4-4):
// the dual-host span where BOTH bosses' rims set the fillet back. Unlike the sliver (one tangent line
// + one rim + one wing arc + one seam), a core panel's two long-boundary sides are BOTH rim sub-arcs
// (panelSide's active branch, G0) and BOTH ends are setbackSection seams (G0) — there is no wing end,
// because the core span never touches the plain fillet cylinder on either side (derivation §1.3). The
// two seam rails are built ONCE via setbackSection and REUSED verbatim as sides (never re-derived),
// so a core panel's z=±6.240 rail is bit-identical to its sliver neighbour's own seam rail at the same
// station (the corner-weld invariant, ADR-0042) — this is what makes zHi/zLo the SAME setbackSection
// call the sliver already makes, not a fresh construction. Loop order mirrors buildSliverLoop's
// alternating long/short pattern so coons4's opposite-side pairing (s0,s2)=long, (s1,s3)=short holds:
// A-rim(zLo->zHi) -> zHi seam(A->B) -> B-rim reversed(zHi->zLo) -> zLo seam reversed(B->A). ok=false
// when either seam station has no crossing (a zLo/zHi outside a boss's own active band) or a rim
// sub-arc fails to fit (panelSide's own failure modes).
func buildCoreLoop(ef edgeFillet, res Resolution, dets []obstacleDetection, detA, detB obstacleDetection, zLo, zHi float64) (RailLoop, bool) {
	seamLo, ok := setbackSection(zLo, dets, ef, res)
	if !ok {
		return RailLoop{}, false
	}
	seamHi, ok := setbackSection(zHi, dets, ef, res)
	if !ok {
		return RailLoop{}, false
	}
	pALo, pBLo := seamLo.PointAt(0), seamLo.PointAt(1)
	pAHi, pBHi := seamHi.PointAt(0), seamHi.PointAt(1)
	aSide, ok := panelSide(ef, detA, true, pALo, pAHi)
	if !ok {
		return RailLoop{}, false
	}
	bSide, ok := panelSide(ef, detB, true, pBHi, pBLo)
	if !ok {
		return RailLoop{}, false
	}
	sides := []Side{
		aSide,
		{Curve: seamHi, Cont: G0},
		bSide,
		{Curve: geom.ReverseCurve3(seamLo), Cont: G0},
	}
	return RailLoop{Sides: sides, Provenance: topo.Lineage{}}, true
}

// splitCoreSpan derives the z=0 split lever's two half-spans from a core span (derivation §2.3, the
// escalation this build measured as NECESSARY: the whole-span core panel over |z|<6.240 misses the
// oracle by ~76% — the wide all-G0 Coons fill balloons past the true rolling-ball setback surface with
// no G1 ribbon to pull it back, see the U4-4 report — whereas splitting at z=0 and pinning the mid
// setbackSection cuts that miss to ~4.5%). Each half is STILL a core span (both hosts active — z=0 is
// a point strictly inside both hosts' active bands, never a host boundary, derivation §1.2), running
// from span's own zLo/zHi to the shared mid station 0. ok=false unless span is a well-formed core span
// that actually straddles 0 (U4's own core span is exactly z∈[-6.240,+6.240]) — a malformed input this
// build never produces, not a real degenerate case to silently tolerate.
func splitCoreSpan(span panelSpan) ([2]panelSpan, bool) {
	if !span.hostA || !span.hostB || span.zLo >= 0 || span.zHi <= 0 {
		return [2]panelSpan{}, false
	}
	lo := panelSpan{zLo: span.zLo, zHi: 0, hostA: true, hostB: true}
	hi := panelSpan{zLo: 0, zHi: span.zHi, hostA: true, hostB: true}
	return [2]panelSpan{lo, hi}, true
}
