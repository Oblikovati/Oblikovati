// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// scallopWall builds the partial cylinder wall closing the edge notch: the INSIDE-plate arc [ta,tb] of the
// drill, extruded between the two cap planes — an iso-(u,v) rectangle (θ∈[ta,tb]×v∈[low,high]) bounded by the
// two cap notch arcs (top/bottom) and two straight vertical generators (the crossing columns). Its normal is
// REVERSED (radially inward, into the removed notch) because the remaining solid lies OUTSIDE the cylinder —
// the opposite of the boss wall, whose solid is inside. The arc edges reuse the CAP circles (circLow/circHigh)
// so the wall welds to the cap notches by shared-edge discretization; the generators reuse the exact column
// points so they weld to the split side faces.
func scallopWall(axisPt math.Point3, ua math.Vector3, radius float64, circLow, circHigh geom.Circle, ta, tb float64) curvedFace {
	cyl, _ := geom.NewCylinderWithRef(axisPt, ua, circLow.RefDir.AsVector(), radius)
	sideB := geom.NewLineSegment(circLow.PointAt(tb), circHigh.PointAt(tb))
	sideA := geom.NewLineSegment(circHigh.PointAt(ta), circLow.PointAt(ta))
	loop := curvedLoop{edges: []loopEdge{
		{curve: circLow, t0: ta, t1: tb},  // bottom notch arc (welds to the low cap)
		{curve: sideB, t0: 0, t1: 1},      // up the column at tb
		{curve: circHigh, t0: tb, t1: ta}, // top notch arc, reversed (welds to the high cap)
		{curve: sideA, t0: 0, t1: 1},      // down the column at ta
	}}
	// The loop as written winds CCW about the cylinder's NATURAL (radially-outward) normal; the scallop wall
	// faces INWARD (reversed=true), so reverse the winding to run CCW about the inward normal — else every
	// shared edge runs the same way as its neighbour and the manifold gate rejects the body (#1591).
	return curvedFace{surface: cyl, reversed: true, loops: []curvedLoop{reverseCurvedLoop(loop)}, lineage: topo.NewLineage(topo.Tok("brep", "scallopwall", 0))}
}

// splitSideFace splits the drill-breached side face into the two pieces OUTSIDE the removed strip, clipping
// the planar (convex) face at each crossing column and snapping the clip vertices to the exact column points
// so they weld to the wall generators. The two pieces get deterministic spatial lineage (ADR-0043) keyed on
// which side of the strip they fall, so a downstream reference resolves stably. ok=false if either piece
// degenerates (the strip covers the whole face — out of scope).
func splitSideFace(f curvedFace, cols [4]math.Point3) ([]curvedFace, bool) {
	ring := sampleCurvedLoop(f.loops[0])
	if len(ring) < 3 {
		return nil, false
	}
	edgeDir := unit(cols[0].VectorTo(cols[2])) // along the clipped edge, column A → column B
	weld := geom.ResolutionForSize(2 * float64(cols[0].DistanceTo(cols[2]))).Weld()
	pieceA := snapToColumns(clipHalfPlane(ring, cols[0], edgeDir), cols, weld)         // keep the side away from B
	pieceB := snapToColumns(clipHalfPlane(ring, cols[2], negate(edgeDir)), cols, weld) // keep the side away from A
	if len(pieceA) < 3 || len(pieceB) < 3 {
		return nil, false
	}
	return []curvedFace{sidePiece(f, pieceA, 0), sidePiece(f, pieceB, 1)}, true
}

// sidePiece builds one split side sub-face: a planar face on the original surface with the clipped ring as its
// outer loop, preserving the original orientation and carrying a deterministic split ordinal in its lineage.
func sidePiece(orig curvedFace, ring []math.Point3, k int) curvedFace {
	edges := make([]loopEdge, len(ring))
	for i := range ring {
		seg := geom.NewLineSegment(ring[i], ring[(i+1)%len(ring)])
		edges[i] = loopEdge{curve: seg, t0: 0, t1: 1}
	}
	return curvedFace{surface: orig.surface, reversed: orig.reversed, loops: []curvedLoop{{edges: edges}}, lineage: topo.NewLineage(topo.Tok("brep", "scallopside", k))}
}

// clipHalfPlane returns the convex polygon clipped to the half-space {p : (p−through)·dir ≤ 0} — a
// Sutherland-Hodgman clip of a planar ring, introducing an intersection vertex at each edge crossing.
func clipHalfPlane(ring []math.Point3, through math.Point3, dir math.Vector3) []math.Point3 {
	var out []math.Point3
	n := len(ring)
	for i := range n {
		a, b := ring[i], ring[(i+1)%n]
		sa := float64(through.VectorTo(a).Dot(dir))
		sb := float64(through.VectorTo(b).Dot(dir))
		if sa <= 0 {
			out = append(out, a)
		}
		if (sa < 0) != (sb < 0) {
			t := sa / (sa - sb)
			out = append(out, a.TranslateBy(a.VectorTo(b).Scale(math.Scalar(t))))
		}
	}
	return out
}

// snapToColumns replaces each ring vertex within a weld of a crossing column by the EXACT column point (so the
// side pieces weld byte-identically to the wall's generators), then drops any consecutive duplicate that snap
// collapses.
func snapToColumns(ring []math.Point3, cols [4]math.Point3, weld float64) []math.Point3 {
	snapped := make([]math.Point3, len(ring))
	for i, p := range ring {
		snapped[i] = p
		for _, c := range cols {
			if float64(p.DistanceTo(c)) < weld {
				snapped[i] = c
				break
			}
		}
	}
	return dedupConsecutive(snapped, weld)
}

// dedupConsecutive drops each point coincident (within weld) with its predecessor, wrapping the ring.
func dedupConsecutive(ring []math.Point3, weld float64) []math.Point3 {
	var out []math.Point3
	for i, p := range ring {
		prev := ring[(i-1+len(ring))%len(ring)]
		if float64(p.DistanceTo(prev)) >= weld {
			out = append(out, p)
		}
	}
	return out
}

// negate returns −v.
func negate(v math.Vector3) math.Vector3 { return v.Scale(-1) }
