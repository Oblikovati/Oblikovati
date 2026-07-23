// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
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

// dualClosure is U4-1's watertight-closure pieces for a qualifying==2 (dual-host) edge: both hosts'
// notched planes (each absorbing its OWN footprint), both bosses' split walls (each kept intact, split
// at its own two nodes), and the two wing faces welding to the union band's outermost stations
// (derivation §3.2, §4-U4-1). It is the pre-panel foundation U4-3/U4-4 build the coons4 setback panels
// against — U4-1 itself does not weld it into a body (assembleDualObstacleSet still honest-rejects).
type dualClosure struct {
	notchA, notchB filletFace
	wallA, wallB   filletFace
	wings          []filletFace
}

// buildDualClosure builds the dual-host watertight-closure pieces by REUSING the single-host closure
// functions per host/boss (derivation §3.2 items 1-3), exactly as the single-host assembler
// (assembleObstacleSet) already does for its one host/boss: buildNotchedHost for EACH host, and
// buildSplitObstacleWall for EACH boss. The two hosts are two DIFFERENT body faces (host A = face16
// y=-20, host B = face31 x=10) with two DIFFERENT obstacle walls behind their own hole rims, so the two
// notch calls and the two wall-split calls touch entirely disjoint topology — no shared cut region, no
// conflict, despite the slice plan's up-front "MEDIUM risk" caution (derivation §4). ok=false on any
// piece's own honest-reject (ADR-3), so a partial closure never survives to a caller.
func buildDualClosure(ef edgeFillet, dets []obstacleDetection, spans []panelSpan, maps filletRebuildMaps) (dualClosure, bool) {
	if len(dets) != 2 {
		return dualClosure{}, false
	}
	notchA, okA := buildNotchedHost(dets[0], maps)
	notchB, okB := buildNotchedHost(dets[1], maps)
	if !okA || !okB {
		return dualClosure{}, false
	}
	wallA, okWA := buildSplitObstacleWall(dets[0])
	wallB, okWB := buildSplitObstacleWall(dets[1])
	if !okWA || !okWB {
		return dualClosure{}, false
	}
	wings, ok := buildOuterWings(ef, dets, spans)
	if !ok {
		return dualClosure{}, false
	}
	return dualClosure{notchA: notchA, notchB: notchB, wallA: wallA, wallB: wallB, wings: wings}, true
}

// buildOuterWings builds the two wing faces (buildObstacleWings, REUSED unchanged) welding to the
// union band's OUTERMOST stations, generalizing the single-host wing split (derivation §3.2 item 3):
// the outer detection's own obstacleGeom section arcs ARE the true wing end-sections there, because
// B's footprint fully brackets A's over the shared band (derivation §1.1 "B ⊃ A") — so no new wing
// geometry is needed, only calling the existing function against whichever detection owns the outer
// stations (outerDetection).
func buildOuterWings(ef edgeFillet, dets []obstacleDetection, spans []panelSpan) ([]filletFace, bool) {
	outer, ok := outerDetection(ef, dets, spans)
	if !ok {
		return nil, false
	}
	og, ok := computeObstacleGeom(ef, outer)
	if !ok {
		return nil, false
	}
	return buildObstacleWings(ef, outer, og), true
}

// outerDetection returns the detection among dets whose own two nodes ARE the union's outermost
// stations (spans[0].zLo and the last span's zHi) — the host whose footprint fully brackets the
// other's, so its own cylinder cross-sections are exactly the wing end-sections the whole union band
// welds to (derivation §1.1/§3.2 item 3). exactTol is bit-identical-modulo-reordering tolerance, not a
// model-relative one: axisParam is the SAME pure function partitionUnionStations already evaluated to
// build these very station values, so a genuinely matching detection's recomputed value differs from
// the station only by floating-point re-association, not by measurement (mirrors the sibling test
// file's chainTol). ok=false when spans is empty or no detection's nodes match both outer stations — a
// shape family this construction does not cover (report BLOCKED, do not guess a host).
func outerDetection(ef edgeFillet, dets []obstacleDetection, spans []panelSpan) (obstacleDetection, bool) {
	if len(spans) == 0 {
		return obstacleDetection{}, false
	}
	outerLo, outerHi := spans[0].zLo, spans[len(spans)-1].zHi
	const exactTol = 1e-9
	for _, d := range dets {
		lo, hi := axisParam(ef, d.pMinus), axisParam(ef, d.pPlus)
		if lo > hi {
			lo, hi = hi, lo
		}
		if stdmath.Abs(lo-outerLo) <= exactTol && stdmath.Abs(hi-outerHi) <= exactTol {
			return d, true
		}
	}
	return obstacleDetection{}, false
}

// assembleDualObstacleSet is the qualifying==2 (dual-host) counterpart to assembleObstacleSet:
// obstacleFacesFor's dualObstacleRoute calls it instead of the single-host assembler. U4-1 builds the
// watertight-closure pieces (buildDualClosure: both notches, both wall splits, the wings) but STILL
// honest-rejects the whole edge (ADR-3): the coons4 setback panels bridging notch-to-notch (U4-3/U4-4)
// do not exist yet, and welding a partial body (closure without panels) would leave a hole where the
// panels belong — exactly the "wings-without-patch" mistake assembleObstacleSet's own docstring already
// forbids for the single-host case. So a dual-host edge still falls to the existing do-no-harm baseline
// — corpus byte-identical — while U4-1's unit tests exercise buildDualClosure directly to pin the
// closure pieces' topology (both notches WIRE:1, both walls split+intact, wings welding to the outer
// stations with no orphan seam, derivation §3.2 caveat).
//
// U4-3..U4-5 return the real welded set once the coons4 panels exist to fill it.
//
//nolint:unparam // result 0 is always the zero obstacleSet{} through U4-1 (still reject-gated, ADR-3);
func assembleDualObstacleSet(ef edgeFillet, dets []obstacleDetection, spans []panelSpan, maps filletRebuildMaps) (obstacleSet, bool) {
	_, _ = buildDualClosure(ef, dets, spans, maps) // exercised for do-no-harm parity; panels land U4-3/4
	return obstacleSet{}, false                    // U4-3..U4-5 weld the real dual-host body; U4-1 stops here
}
