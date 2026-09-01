// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"sort"

	opstol "oblikovati.org/kernel/ops/internal/tol"
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
	tol := opstol.ForPoints(pts).Weld()
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

// assembleDualObstacleSet is the qualifying==2 (dual-host) counterpart to assembleObstacleSet
// (derivation §3.2/§3.3/§4-U4-5, #2007 Group C): obstacleFacesFor's dualObstacleRoute calls it instead
// of the single-host assembler. U4-5 turns the whole dual-host pipeline ON — it welds the full watertight
// body from the pieces U4-1..U4-4b built and verified: the two notched hosts (each absorbing its own
// footprint), the two split obstacle walls (each rim split at its own nodes AND at every interior panel
// seam so the wall welds edge-for-edge to the panels), the two outer cylinder wings, and the four
// corner-blend panels (the two B-only slivers via coons4, the two dual-host cores via the exact-station
// canal loft). The four panels partition the union band at the stations {±6.633 wing weld, ±6.240
// sliver-core seam, 0 core-core seam}; every shared boundary is traced from the SAME node/dip-rim/seam
// point both neighbours use (dualPanelBoundary), so the shell closes with no orphan seam (ADR-0042).
//
// Before building the host notches, the wing/sliver tangent-seam split points on host A's receded front
// (the ±6.633 wing-tangent points, where the plain fillet cylinder gives way to the sliver — boss A does
// not reach there but boss B does, derivation §1.1 "B ⊃ A") are injected into the shared edgeInserts map
// so the notch front edge is subdivided there and the wing / sliver A-tangent lines each weld to a real
// straight segment, not a mid-air point (the derivation §3.2 orphan-seam caveat, pinned by U4-1's
// TestBuildDualClosureU4WingsWeldNoOrphanSeam).
//
// ok=false honest-rejects the WHOLE edge (ADR-3, never a wings-without-panels hole) when any piece
// declines: a non-U4 dual-host shape the panel/closure construction does not cover, a panel loop that
// does not resolve to a certified fill, or a degenerate section — the caller then falls to the existing
// do-no-harm baseline, corpus byte-identical.
func assembleDualObstacleSet(ef edgeFillet, dets []obstacleDetection, spans []panelSpan, maps filletRebuildMaps, res opstol.Resolution) (obstacleSet, bool) {
	detA, detB, ok := hostDetections(dets)
	if !ok {
		return obstacleSet{}, false
	}
	panels := dualPanelSpans(spans)
	if len(panels) != 4 {
		return obstacleSet{}, false
	}
	// Build every piece that does NOT depend on the shared edgeInserts map FIRST — the four panels, the two
	// split walls and the two wings. These fully validate the shape (all panels resolve to a certified
	// fill, both obstacle walls are rebuildable tubes, both nodes bracket the wings), so a non-U4 dual-host
	// edge that spuriously reaches here (e.g. a two-boss runout/setback body) honest-rejects BEFORE the
	// front-seam inserts are injected — never leaving the shared map mutated for the path it falls back to
	// (the S1/S7 do-no-harm regression the earlier speculative injection caused).
	panelFaces, ok := buildDualPanelFaces(ef, dets, panels, res)
	if !ok {
		return obstacleSet{}, false
	}
	wallA, wallB, wings, ok := buildDualWallsAndWings(ef, dets, spans, panels, res)
	if !ok {
		return obstacleSet{}, false
	}
	return weldDualSet(ef, dets, detA, detB, wallA, wallB, wings, panelFaces, maps)
}

// weldDualSet finishes the confirmed U4-class assembly: it subdivides the inner host's front seam (only
// now, the shape is confirmed), builds the two notches against it, and packs the obstacleSet (both notched
// hosts + both split walls in `replace`, the two wings + four panels in `extra`). It rolls the front-seam
// injection back if a notch declines, so a late failure still leaves the shared map pristine for the
// edge's fallback path (do-no-harm).
func weldDualSet(ef edgeFillet, dets []obstacleDetection, detA, detB obstacleDetection,
	wallA, wallB filletFace, wings, panelFaces []filletFace, maps filletRebuildMaps) (obstacleSet, bool) {
	injectWingTangentInserts(ef, dets, partitionUnionStations(dets, ef), maps)
	notchA, okA := buildNotchedHost(detA, maps)
	notchB, okB := buildNotchedHost(detB, maps)
	if !okA || !okB {
		clearWingTangentInserts(ef, dets, maps)
		return obstacleSet{}, false
	}
	return obstacleSet{
		replace: map[uint64]filletFace{
			detA.host.ID(): notchA, detB.host.ID(): notchB,
			detA.obstacleWall.ID(): wallA, detB.obstacleWall.ID(): wallB,
		},
		extra: append(append([]filletFace{}, wings...), panelFaces...),
	}, true
}

// dualPanelSpans expands partitionUnionStations' three spans (sliver, core, sliver) into the FOUR panel
// spans the welded body is built from by splitting the middle CORE span at z=0 (splitCoreSpan, the U4-4
// core-core seam lever): [sliverLo, coreLo, coreHi, sliverHi]. nil (len!=4) when the input is not the
// U4-shaped 3-span partition — an honest-reject signal for a dual-host shape this build does not cover.
func dualPanelSpans(spans []panelSpan) []panelSpan {
	if len(spans) != 3 {
		return nil
	}
	halves, ok := splitCoreSpan(spans[1])
	if !ok {
		return nil
	}
	return []panelSpan{spans[0], halves[0], halves[1], spans[2]}
}
