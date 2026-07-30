// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// extractPanelLoop assembles ONE panel's valence-4 RailLoop for a panelSpan (derivation §1.4/§3.1),
// generalizing extractObstacle to the dual-host union band. Two shapes: the SLIVER (exactly one host
// active — the single-host-B obstacle shape reused wholesale, derivation §1.5, U4-3) and the CORE
// (both hosts active — the genuinely new dual-host loop, derivation §1.3/§4-U4-4). Every rail is
// either REUSED verbatim (wingSection's quarter arc, setbackSection's seam) or built from EXACT
// corner points panelSide pins to those reused rails, so the loop welds to its neighbours (the wing,
// and the neighbouring sliver/core panel at the shared seam station) with no drift (ADR-0042).
func extractPanelLoop(span panelSpan, dets []obstacleDetection, ef edgeFillet, res Resolution) (RailLoop, bool) {
	detA, detB, ok := hostDetections(dets)
	if !ok {
		return RailLoop{}, false
	}
	activeIsB, isSliver := sliverActiveHost(span)
	if !isSliver {
		if !span.hostA || !span.hostB {
			return RailLoop{}, false // neither host active: malformed input, never a real panelSpan
		}
		return buildCoreLoop(ef, res, dets, detA, detB, span.zLo, span.zHi)
	}
	active, inactive := detA, detB
	if activeIsB {
		active, inactive = detB, detA
	}
	wingEnd, seamEnd, ok := sliverStations(span, active, ef)
	if !ok {
		return RailLoop{}, false
	}
	return buildSliverLoop(ef, res, dets, active, inactive, wingEnd, seamEnd)
}

// sliverActiveHost reports whether span is a well-formed single-host SLIVER span (derivation §1.3)
// and, if so, which host is the active (dipping) one. Both hosts active is a CORE span (not this
// slice's shape); neither active never occurs for a real panelSpan (partitionUnionStations always
// marks the active host(s) between consecutive stations).
func sliverActiveHost(span panelSpan) (activeIsB, ok bool) {
	if span.hostA == span.hostB {
		return false, false
	}
	return span.hostB, true
}

// sliverStationExactTol matches an axis station back to the host node it came from: bit-identical
// modulo float re-association, not a model-relative weld — span.zLo/zHi and axisParam(ef, det.p*)
// are the SAME pure function evaluated on the SAME underlying node point (partitionUnionStations'
// own axisStations), so a genuinely matching station differs only by re-association, never by
// measurement (mirrors outerDetection's identical exactTol convention, fillet_obstacle_dual.go).
const sliverStationExactTol = 1e-9

// sliverStations classifies a sliver span's two axis stations: wingEnd is the ACTIVE host's own
// outer node (the plain-fillet weld to wingSection's quarter arc — B ⊃ A, derivation §1.1, so the
// active host's node here is always the union's OUTER station); seamEnd is the INACTIVE host's own
// node (the interior seam bordering the neighbour CORE span, where setbackSection closes the loop).
// ok=false when neither or both ends match the active host's nodes — not a well-formed sliver span.
func sliverStations(span panelSpan, active obstacleDetection, ef edgeFillet) (wingEnd, seamEnd float64, ok bool) {
	loActive := matchesNode(active, ef, span.zLo)
	hiActive := matchesNode(active, ef, span.zHi)
	switch {
	case loActive && !hiActive:
		return span.zLo, span.zHi, true
	case hiActive && !loActive:
		return span.zHi, span.zLo, true
	default:
		return 0, 0, false
	}
}

// matchesNode reports whether z is det's own pMinus or pPlus node station (exact match, see
// sliverStationExactTol).
func matchesNode(det obstacleDetection, ef edgeFillet, z float64) bool {
	return stdmath.Abs(axisParam(ef, det.pMinus)-z) <= sliverStationExactTol ||
		stdmath.Abs(axisParam(ef, det.pPlus)-z) <= sliverStationExactTol
}

// buildSliverLoop assembles the 4-sided loop from its two REUSED end rails — setbackSection at
// seamEnd (the interior seam, shared bit-for-bit with the future core panel) and wingSection's
// quarter arc at wingEnd (the plain-fillet weld) — then the two long-boundary sides panelSide pins
// between their exact corners. Loop order (A→B→C→D→A): A=seam's inactive-side pin, B=the wing arc's
// wall point, C=the wing arc's node, D=seam's active-side pin, closing A via the seam rail reversed.
func buildSliverLoop(ef edgeFillet, res Resolution, dets []obstacleDetection,
	active, inactive obstacleDetection, wingEnd, seamEnd float64) (RailLoop, bool) {
	seam, ok := setbackSection(seamEnd, dets, ef, res)
	if !ok {
		return RailLoop{}, false
	}
	pinInactive, pinActive := seam.PointAt(0), seam.PointAt(1)
	wingArc, wingCyl, wallP, node, ok := sliverWingEnd(ef, active, wingEnd)
	if !ok {
		return RailLoop{}, false
	}
	lineSide, ok := panelSide(ef, inactive, false, pinInactive, wallP)
	if !ok {
		return RailLoop{}, false
	}
	rimSide, ok := panelSide(ef, active, true, node, pinActive)
	if !ok {
		return RailLoop{}, false
	}
	sides := []Side{
		lineSide,
		{Curve: wingArc, Adjacent: wingCyl, Cont: G1},
		rimSide,
		{Curve: geom.ReverseCurve3(seam), Cont: G0}, // D(pinActive) -> A(pinInactive), the shared seam
	}
	return RailLoop{Sides: sides, Provenance: topo.Lineage{}}, true
}

// sliverWingEnd builds the sliver's z=wingEnd end: the FULL fillet quarter arc from the inactive
// host's tangent point (wallP) to the active host's own node — wingSection reused verbatim, the same
// building block computeObstacleGeom uses for the single-host wing sections (derivation §1.3/§1.5:
// the sliver's wing end IS a plain single-host wing cross-section). node is active's own pMinus/pPlus
// at that station (never re-derived), so the arc's node end is bit-identical to the wing face's own
// node and to active's Nodes elsewhere. ok=false when wingSection itself declines (a degenerate
// section, ADR-3) or the reconstructed cylinder fails (wingCylinder).
func sliverWingEnd(ef edgeFillet, active obstacleDetection, wingEnd float64) (arc geom.Arc3d, cyl geom.Cylinder, wallP, node math.Point3, ok bool) {
	hostRadial, wallRadial, midRadial := cornerRadials(ef, active.hostIsA)
	node = activeNodeAt(active, ef, wingEnd)
	arc, sec, ok0 := wingSection(node, hostRadial, wallRadial, midRadial)
	if !ok0 {
		return geom.Arc3d{}, geom.Cylinder{}, math.Point3{}, math.Point3{}, false
	}
	cyl, ok1 := wingCylinder(arc, ef.cyl.AxisDir.AsVector())
	return arc, cyl, sec.wall, node, ok1
}

// activeNodeAt returns active's own node (pMinus or pPlus) at axis station z — precondition:
// sliverStations already confirmed one of them matches (matchesNode), so this always finds one.
func activeNodeAt(active obstacleDetection, ef edgeFillet, z float64) math.Point3 {
	if stdmath.Abs(axisParam(ef, active.pMinus)-z) <= sliverStationExactTol {
		return active.pMinus
	}
	return active.pPlus
}
