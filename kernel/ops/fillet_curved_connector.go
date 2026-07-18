// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// D9-T2 — the REFLEX far-vertex host splices. OCCT blend/simple/D9 is a 270° (reflex) sphere-host
// trihedral corner: its cylinder arm's contact ruling on the top cap terminates at the far-vertex
// STATION (−10,0,129.9038), which — because the wedge is reflex — lies INTERIOR to the cap's 270°
// sector, on no loop edge (d9-rail-bundle-forensic.md §1). The far termination therefore adds no new
// FACE (parity holds) but a new EDGE: the host∩capping trace segment station→farVertex, shared by the
// retrimmed top cap and the EXTENDED lon-0 capping face (§2). This file owns that shared connector
// authority and the two splices that consume it — one on the corner-host retrim (the top cap, Splice 1)
// and one on the far-runout capping face (lon-0, Splice 2) — so both faces carry the SAME station/far-
// vertex points and assembleBody welds them watertight (it welds by shared points).

// cornerConnector is a reflex arm's shared far-vertex splice authority: the NEW straight host∩capping
// edge from the arm's interior ruling station (strictly inside a corner host's loop, off every edge) to
// its stamped far vertex (the surviving loop vertex on the same trace). Both flanking host faces consume
// THIS same station/farVertex pair, so the connector edge welds 2-incident.
type cornerConnector struct {
	station   math.Point3 // the interior ruling outer end, off every corner-host loop edge (D9 cap: (−10,0,129.9038))
	farVertex math.Point3 // the surviving cap∧capping loop vertex the connector closes to (D9: (0,0,129.9038))
}

// reflexConnectors scans the welded arms for a REFLEX far termination — a cylinder arm whose ruling
// station on a corner host lands strictly INSIDE that host's loop (a >180° wedge, D9's cylinder arm),
// off every loop edge — and returns the shared connector authority for each. It is EMPTY for every convex
// corner (each arm's station is a real loop crossing, on-loop), so the splice branches below are
// new-branch-only and every existing green stays byte-identical. arms/bundles/w.arms are index-aligned.
func reflexConnectors(arms []edgeFillet, bundles []armRails, w cornerWeld, tol float64) []cornerConnector {
	var out []cornerConnector
	for i, ef := range arms {
		if c, ok := armReflexConnector(ef, bundles[i], w.arms[i], w, tol); ok {
			out = append(out, c)
		}
	}
	return out
}

// armReflexConnector builds the connector for one arm when its host rail's outer end is an interior
// station off its host loop (the reflex case). The interior station only arises for a CYLINDER arm's
// straight ruling (armRulingEnd's rulingStationOuter fallback); a torus arm carves a circular rail that
// always lands on the host rim. The far vertex is the arm's stamped runout authority. ok=false otherwise.
func armReflexConnector(ef edgeFillet, b armRails, set armSetback, w cornerWeld, tol float64) (cornerConnector, bool) {
	if _, isCyl := set.arm.(geom.Cylinder); !isCyl || !set.runoutKnown {
		return cornerConnector{}, false // interior-station reflex termination only occurs on a cylinder-arm ruling
	}
	if stationOffLoop(ef.a, b.hostA.from, w, tol) {
		return cornerConnector{station: b.hostA.from, farVertex: set.farVertex}, true
	}
	if stationOffLoop(ef.b, b.hostB.from, w, tol) {
		return cornerConnector{station: b.hostB.from, farVertex: set.farVertex}, true
	}
	return cornerConnector{}, false
}

// stationOffLoop reports whether the ruling outer end p lies OFF the host's bitten loop — on no edge and
// no vertex of it — the signature of a reflex interior station (D9's cap ruling ends at (−10,0,z), azimuth
// 180°, radius 10 ≪ the sector rim 72.27). It reads the SAME bitten loop the retrim walks (bittenLoop),
// so the detection matches what farPathSegs sees; a station coincident with any loop edge (every convex
// green) is on-loop and returns false.
func stationOffLoop(host *topo.Face, p math.Point3, w cornerWeld, tol float64) bool {
	star, ok := bittenLoop(host, w.center, tol)
	if !ok {
		return false
	}
	segs := segsFromLoop(star)
	if len(segs) < 3 {
		return false
	}
	ring := insertSplits(segs, []math.Point3{p}, tol)
	return indexOfSegFrom(ring, p, tol) < 0
}

// connectorAt returns the reflex connector whose station coincides with p (a retrim far-path endpoint),
// or ok=false when p is an ordinary on-loop outer end.
func connectorAt(conns []cornerConnector, p math.Point3, tol float64) (cornerConnector, bool) {
	for _, c := range conns {
		if float64(p.DistanceTo(c.station)) <= tol {
			return c, true
		}
	}
	return cornerConnector{}, false
}

// cornerFarPath is retrimCornerHost's surviving far boundary from fromP to toP (outerB→outerA). It is
// farPathSegs VERBATIM for every convex corner (empty conns) EXCEPT when a reflex connector bridges an
// off-loop interior station to its far vertex (Splice 1, D9's top cap): then the path is the connector
// edge plus farPathSegs walked from the on-loop far vertex, so the loop closes through the new host∩
// capping edge instead of an original edge that does not exist at the interior station.
func cornerFarPath(segs []endSeg, fromP, toP, v math.Point3, tol float64, conns []cornerConnector) ([]endSeg, bool) {
	if c, ok := connectorAt(conns, fromP, tol); ok {
		return farPathThroughConnector(segs, c, toP, v, tol, true)
	}
	if c, ok := connectorAt(conns, toP, tol); ok {
		return farPathThroughConnector(segs, c, fromP, v, tol, false)
	}
	return farPathSegs(segs, fromP, toP, v, tol)
}

// farPathThroughConnector closes the reflex far path across the connector edge: the off-loop station is
// bridged to its far vertex, and the remainder walks the ORIGINAL loop (farPathSegs) from that on-loop
// far vertex to the other endpoint. atFrom marks the station as the path's from (connector first) or to
// (connector last), so the returned chain stays oriented fromP→toP.
func farPathThroughConnector(segs []endSeg, c cornerConnector, other, v math.Point3, tol float64, atFrom bool) ([]endSeg, bool) {
	if atFrom {
		rest, ok := farPathSegs(segs, c.farVertex, other, v, tol)
		if !ok {
			return nil, false // the far vertex does not reach the other endpoint along the loop — cannot close
		}
		return append([]endSeg{{from: c.station, to: c.farVertex}}, rest...), true
	}
	rest, ok := farPathSegs(segs, other, c.farVertex, v, tol)
	if !ok {
		return nil, false
	}
	return append(rest, endSeg{from: c.farVertex, to: c.station}), true
}

// spliceBite routes one far-runout bite: an ordinary on-loop bite (every green) takes spliceCornerBite
// VERBATIM (byte-identity); a bite whose endpoint is a reflex connector's interior station (off this
// capping face's loop) is an EXTENSION (Splice 2, D9's lon-0 face), not a corner removal.
func spliceBite(segs []endSeg, bite endSeg, conns []cornerConnector, tol float64) ([]endSeg, bool) {
	if c, ok := extensionConnectorFor(conns, bite, tol); ok {
		return extendLoopWithBite(segs, bite, c, tol)
	}
	return spliceCornerBite(segs, bite, tol)
}

// extensionConnectorFor returns the reflex connector whose station is one endpoint of bite — the signal
// that this far bite runs off the capping face and must EXTEND it (across its old boundary), not bite a
// corner off it. ok=false for every bite whose feet both lie on the loop.
func extensionConnectorFor(conns []cornerConnector, bite endSeg, tol float64) (cornerConnector, bool) {
	for _, c := range conns {
		if float64(bite.from.DistanceTo(c.station)) <= tol || float64(bite.to.DistanceTo(c.station)) <= tol {
			return c, true
		}
	}
	return cornerConnector{}, false
}

// extendLoopWithBite splices the far bite as a FACE EXTENSION (D9's lon-0 capping face, forensic §2/§3.3):
// the bite has one endpoint at the interior STATION (off this face's loop, x<0 of the y=0 plane) and one
// at the on-loop foot. It replaces the short loop span between the far vertex F and the on-loop foot with
// the outward detour [F→station (the shared connector), station→foot (the far trim)], extending the face
// across its old boundary to enclose the reflex bump. Declines (floor) when F or the foot is not on the loop.
func extendLoopWithBite(segs []endSeg, bite endSeg, c cornerConnector, tol float64) ([]endSeg, bool) {
	foot, ok := biteFootOffStation(bite, c.station, tol)
	if !ok {
		return nil, false // neither bite endpoint is this connector's station — not the reflex far trim
	}
	ring := insertSplits(segs, []math.Point3{c.farVertex, foot}, tol)
	i, j := indexOfSegFrom(ring, c.farVertex, tol), indexOfSegFrom(ring, foot, tol)
	if i < 0 || j < 0 {
		return nil, false // the far vertex or the on-loop foot is not on this capping loop — cannot extend
	}
	return spliceExtension(ring, i, j, bite, c, foot, tol), true
}

// spliceExtension replaces the SHORT loop span between the far vertex (index i) and the on-loop foot
// (index j) with the outward detour [connector, far trim], keeping the large complementary span — the
// capping-face extension. The short span is picked by enclosed area (the same geometric criterion
// spliceCornerBite removes a bite corner by), so the winding is preserved regardless of loop orientation.
func spliceExtension(ring []endSeg, i, j int, bite endSeg, c cornerConnector, foot math.Point3, tol float64) []endSeg {
	trim := orientedTrim(bite, c.station, tol) // far trim oriented station→foot
	spanIJ, spanJI := segsForward(ring, i, j), segsForward(ring, j, i)
	chord := endSeg{from: foot, to: c.farVertex}
	if cornerBiteArea(spanIJ, chord) <= cornerBiteArea(spanJI, chord) {
		detour := []endSeg{{from: c.farVertex, to: c.station}, trim} // F→station→foot replaces the small F→foot span
		return append(detour, spanJI...)
	}
	rev := reverseEndSegs([]endSeg{trim})[0]                             // foot→station
	return append(spanIJ, rev, endSeg{from: c.station, to: c.farVertex}) // …→foot→station→F
}

// biteFootOffStation returns the bite endpoint that is NOT the station (the on-loop foot), or ok=false
// when neither endpoint is the station (this bite is not the reflex connector's far trim).
func biteFootOffStation(bite endSeg, station math.Point3, tol float64) (math.Point3, bool) {
	if float64(bite.from.DistanceTo(station)) <= tol {
		return bite.to, true
	}
	if float64(bite.to.DistanceTo(station)) <= tol {
		return bite.from, true
	}
	return math.Point3{}, false
}

// orientedTrim returns the far trim oriented station→foot (reversing the arc — via reverseEndSegs, which
// re-derives an Arc3d through its three points — when the incoming bite runs foot→station), so the
// extension detour walks the trim in its outward direction.
func orientedTrim(bite endSeg, station math.Point3, tol float64) endSeg {
	if float64(bite.from.DistanceTo(station)) <= tol {
		return bite
	}
	return reverseEndSegs([]endSeg{bite})[0]
}
