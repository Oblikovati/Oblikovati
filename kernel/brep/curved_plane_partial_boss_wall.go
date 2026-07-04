// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// partialBossWall builds the boss side wall as an outward cylinder face whose BASE circle is split at the two
// seat crossings, so its in-seat sub-arc welds to the trimmed seat face and its out-seat sub-arc welds to the
// overhang cap. The seam is placed AT a crossing (ta) so the seam never bisects the out-seat arc — the far
// (top) circle stays whole. Mirrors bossWallFace's winding (seam up, top reversed, seam down, base forward).
func partialBossWall(near math.Point3, axis math.Vector3, radius float64, nearCirc, farCirc geom.Circle, crossings []planeCrossing) curvedFace {
	cyl, _ := geom.NewCylinder(near, axis, radius)
	ta, tb := crossings[0].tConic, crossings[1].tConic
	if tb < ta {
		ta, tb = tb, ta
	}
	seam := geom.NewLineSegment(nearCirc.PointAt(ta), farCirc.PointAt(ta))
	loop := curvedLoop{edges: []loopEdge{
		{curve: seam, t0: 0, t1: 1},
		{curve: farCirc, t0: ta + 1, t1: ta}, // full top circle, reversed, anchored at the seam angle
		{curve: seam, t0: 1, t1: 0},
		{curve: nearCirc, t0: ta, t1: tb},     // in-seat base arc (welds to the trimmed seat)
		{curve: nearCirc, t0: tb, t1: ta + 1}, // out-seat base arc (welds to the overhang cap), wraps the seam
	}}
	return curvedFace{surface: cyl, reversed: false, loops: []curvedLoop{loop}, lineage: topo.NewLineage(topo.Tok("brep", "bosswall", 0))}
}

// partialBossTopCap builds the boss's outer top disc anchored at the seam angle ta, so its full boundary
// circle matches the split-base wall's top edge (also anchored at ta) — a full-circle edge welds only when
// both sides share the same parameter anchoring, so the standard 0-anchored cap would leave the top open.
func partialBossTopCap(center math.Point3, axis math.Vector3, farCirc geom.Circle, ta float64) curvedFace {
	pl, _ := geom.NewPlane(center, axis)
	loop := curvedLoop{edges: []loopEdge{{curve: farCirc, t0: ta, t1: ta + 1}}}
	return curvedFace{surface: pl, reversed: false, loops: []curvedLoop{loop}, lineage: topo.NewLineage(topo.Tok("brep", "bosscap", 0))}
}

// splitFacesAtCrossings inserts each crossing point into any straight face edge that passes through it, so a
// copied target face adjacent to the clipped seat edge welds to the seat/cap sub-edges — curvedStitch does
// not resolve T-junctions itself, so the vertex must already be present on both sides (#1591).
func splitFacesAtCrossings(faces []curvedFace, crossings []planeCrossing) []curvedFace {
	pts := make([]math.Point3, len(crossings))
	for i, cr := range crossings {
		pts[i] = cr.at
	}
	out := make([]curvedFace, len(faces))
	for i, f := range faces {
		out[i] = curvedFace{surface: f.surface, reversed: f.reversed, lineage: f.lineage, outerless: f.outerless, loops: splitLoopsAt(f.loops, pts)}
	}
	return out
}

// splitLoopsAt splits every straight edge of every loop at the crossing points lying on it.
func splitLoopsAt(loops []curvedLoop, pts []math.Point3) []curvedLoop {
	out := make([]curvedLoop, len(loops))
	for i, lp := range loops {
		var edges []loopEdge
		for _, e := range lp.edges {
			edges = append(edges, splitLineEdgeAt(e, pts)...)
		}
		out[i] = curvedLoop{edges: edges}
	}
	return out
}

// splitLineEdgeAt breaks one straight loop edge at any crossing points on its interior (ordered along it),
// re-emitting straight sub-segments through those shared points. Non-straight edges pass through untouched.
func splitLineEdgeAt(e loopEdge, pts []math.Point3) []loopEdge {
	if _, ok := e.curve.(geom.LineSegment); !ok {
		return []loopEdge{e}
	}
	a, b := e.start(), e.end()
	on := pointsOnSegment(a, b, pts)
	if len(on) == 0 {
		return []loopEdge{e}
	}
	verts := append([]math.Point3{a}, on...)
	verts = append(verts, b)
	out := make([]loopEdge, 0, len(verts)-1)
	for i := 1; i < len(verts); i++ {
		out = append(out, loopEdge{curve: geom.NewLineSegment(verts[i-1], verts[i]), t0: 0, t1: 1})
	}
	return out
}

// pointsOnSegment returns the points lying on the interior of segment a→b, ordered by their parameter along
// it (within a stitch tolerance of the line, strictly between the endpoints).
func pointsOnSegment(a, b math.Point3, pts []math.Point3) []math.Point3 {
	tol := geom.ResolutionForSize(float64(a.DistanceTo(b))).Stitch()
	d := a.VectorTo(b)
	l2 := float64(d.Dot(d))
	type sp struct {
		s float64
		p math.Point3
	}
	var hits []sp
	for _, p := range pts {
		s := float64(a.VectorTo(p).Dot(d)) / l2
		if s <= 1e-9 || s >= 1-1e-9 {
			continue
		}
		if a.TranslateBy(d.Scale(math.Scalar(s))).DistanceTo(p) > math.Scalar(tol) {
			continue
		}
		hits = append(hits, sp{s: s, p: p})
	}
	for i := 1; i < len(hits); i++ { // insertion sort by parameter (few hits per edge)
		for j := i; j > 0 && hits[j].s < hits[j-1].s; j-- {
			hits[j], hits[j-1] = hits[j-1], hits[j]
		}
	}
	out := make([]math.Point3, len(hits))
	for i, h := range hits {
		out[i] = h.p
	}
	return out
}
