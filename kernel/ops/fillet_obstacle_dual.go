// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// panelSpan is one interval of the union band along the shared fillet axis, carrying which host(s)
// dip into the fillet over that interval: hostA && hostB both true is a CORE span (both bosses set
// the fillet back — the genuinely new dual-host geometry, derivation §1.3); only one true is a SLIVER
// span (the other host's boundary is still the unmodified fillet cylinder — the existing single-host
// obstacle shape). Stations are the ordered axis-parameters partitionUnionStations derives from the
// two hosts' node crossings. U4-0 only produces the spans — no panel geometry consumes them yet
// (real panels land in U4-3/U4-4; this is do-no-harm foundation, #2007 Group C).
type panelSpan struct {
	zLo, zHi     float64
	hostA, hostB bool
}

// partitionUnionStations projects each detection's two dip nodes onto the shared fillet axis (ef.c0.cen
// + ef.cyl.AxisDir, the same anchor nearFarNodes already uses) and partitions the union interval into
// ordered panelSpans between consecutive distinct stations, marking which host(s) are active (dipping)
// over each span's midpoint. dets holds 1 or 2 detections; with 2 (the dual-host case this slice wires
// up) the spans capture the SLIVER (one host only) / CORE (both hosts) distinction the U4 multi-rail
// corner-blend needs (derivation §1.2/§3.1). Model-relative weld tolerance dedupes near-coincident
// stations (e.g. a node shared by both hosts), never a bare epsilon (ADR-0042).
func partitionUnionStations(dets []obstacleDetection, ef edgeFillet) []panelSpan {
	stations := axisStations(dets, ef)
	spans := make([]panelSpan, 0, len(stations)-1)
	for i := 0; i+1 < len(stations); i++ {
		lo, hi := stations[i], stations[i+1]
		spans = append(spans, panelSpan{
			zLo: lo, zHi: hi,
			hostA: hostActiveAt(dets, ef, true, (lo+hi)/2),
			hostB: hostActiveAt(dets, ef, false, (lo+hi)/2),
		})
	}
	return spans
}

// axisParam projects point p onto the shared fillet axis built from ef.c0.cen + ef.cyl.AxisDir — the
// identical fields nearFarNodes already reads — but re-origins the result at the axis MIDPOINT (half
// the c0→c1 span) rather than at c0.cen itself. nearFarNodes only ever compares two such values (near
// vs far), so either origin preserves its ordering; partitionUnionStations additionally reports the
// stations themselves, and re-origining at the midpoint is what makes a symmetric edge's stations read
// as the signed distance from its own centre (U4: ±6.240/±6.633) instead of an arbitrary c0-relative
// offset — the numbers the derivation (§1.1) and the DRAWEXE oracle both quote.
func axisParam(ef edgeFillet, p math.Point3) float64 {
	axis := ef.cyl.AxisDir.AsVector()
	half := ef.c0.cen.VectorTo(ef.c1.cen).Dot(axis) / 2
	return ef.c0.cen.VectorTo(p).Dot(axis) - half
}

// axisStations projects every detection's two nodes onto the shared axis and returns the DISTINCT
// stations in ascending order, weld-deduplicated at model-relative tolerance (ResolutionForPoints, not
// a bare epsilon — ADR-0042).
func axisStations(dets []obstacleDetection, ef edgeFillet) []float64 {
	var raw []float64
	pts := []math.Point3{ef.c0.cen, ef.c1.cen}
	for _, d := range dets {
		raw = append(raw, axisParam(ef, d.pMinus), axisParam(ef, d.pPlus))
		pts = append(pts, d.pMinus, d.pPlus)
	}
	sort.Float64s(raw)
	tol := geom.ResolutionForPoints(pts).Weld()
	var stations []float64
	for _, s := range raw {
		if len(stations) == 0 || s-stations[len(stations)-1] > tol {
			stations = append(stations, s)
		}
	}
	return stations
}

// hostActiveAt reports whether host hostIsA dips into the fillet at axis station z: true when z falls
// within that host's own node interval [min(pMinus,pPlus), max(pMinus,pPlus)] projected onto the axis.
// A host with no qualifying detection in dets (the single-host slice) is never active — a span never
// claims a host that didn't qualify.
func hostActiveAt(dets []obstacleDetection, ef edgeFillet, hostIsA bool, z float64) bool {
	for _, d := range dets {
		if d.hostIsA != hostIsA {
			continue
		}
		lo, hi := axisParam(ef, d.pMinus), axisParam(ef, d.pPlus)
		if lo > hi {
			lo, hi = hi, lo
		}
		return z >= lo && z <= hi
	}
	return false
}

// assembleDualObstacleSet is the qualifying==2 (dual-host) counterpart to assembleObstacleSet:
// obstacleFacesFor's dualObstacleRoute calls it instead of the single-host assembler. U4-0 is a data-
// model + wiring SLICE only — this is a STUB that always honest-rejects (ADR-3), so a dual-host edge
// still falls to the existing do-no-harm baseline exactly as before this change (corpus byte-
// identical). The real rebuild — notched hosts, split walls, wings, and the setbackSection-bounded
// coons4 panels per span — lands across U4-1..U4-5 (derivation §3.2/§4, #2007 Group C); dets and spans
// are already the shape that build needs, so this stub's signature does not change out from under it.
//
//nolint:unparam // ef/dets/spans are the fixed signature U4-1..U4-5 build against; unused by the U4-0 stub.
func assembleDualObstacleSet(ef edgeFillet, dets []obstacleDetection, spans []panelSpan) (obstacleSet, bool) {
	return obstacleSet{}, false // U4-1..U4-5 build the real dual-host rebuild; U4-0 stops here
}
