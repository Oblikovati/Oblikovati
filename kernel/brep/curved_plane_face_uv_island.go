// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
)

// Closed-conic ISLAND imprints for the exact-frame planar face (planeFaceUV, ADR-0058 / #3460). The mixed
// boolean's plane∩plane imprints are straight segments that terminate on the frame; a ruled wall's
// plane∩wall section is instead a CLOSED conic that sits wholly inside the frame — the circle a boss or a
// through-cylinder cuts into a plate face. An island needs no crossing injection (it never meets a frame
// edge, which pairUVWallImprints proves in closed form before the trim runs), only a sampling that carries
// the analytic curve, so the kept boundary re-emits the exact conic — the SAME curve the wall's own
// arrangement re-emits, which is what welds the two faces.

// splitImprintByKind separates a mixed imprint list into the STRAIGHT segments (plane∩plane lines, split
// against the frame's conic edges) and the CLOSED conic islands (a ruled wall's section, gated by
// wallSectionIsland to lie wholly inside the frame). Keeping them apart is what lets the straight path
// keep reading PointAt(0)/PointAt(1) as segment ends — a conic's are two points one radian apart.
func splitImprintByKind(imprint []geom.Curve3) (straight, islands []geom.Curve3) {
	for _, cv := range imprint {
		switch cv.(type) {
		case geom.Circle, geom.EllipseFull:
			islands = append(islands, cv)
		default:
			straight = append(straight, cv)
		}
	}
	return straight, islands
}

// conicIslandSegs samples one closed conic imprint over its WHOLE domain into tagged (u,v) segments that
// carry the source curve and its endpoint parameters, so a kept boundary run re-emits the exact analytic
// conic instead of the sampled chords.
func (c *planeFaceUV) conicIslandSegs(cv geom.Curve3) []uvSeg {
	lo, hi := cv.Domain()
	segs := make([]uvSeg, 0, imprintSampleCount)
	prevT, prev := lo, to2D(c.plane, cv.PointAt(lo))
	for i := 1; i <= imprintSampleCount; i++ {
		t := lo + (hi-lo)*float64(i)/imprintSampleCount
		p := to2D(c.plane, cv.PointAt(t))
		segs = append(segs, uvSeg{a: prev, b: p, curve: cv, tA: prevT, tB: t, kind: segImprint})
		prevT, prev = t, p
	}
	return segs
}

// islandSegs samples every closed conic imprint of the face.
func (c *planeFaceUV) islandSegs(islands []geom.Curve3) []uvSeg {
	var out []uvSeg
	for _, cv := range islands {
		out = append(out, c.conicIslandSegs(cv)...)
	}
	return out
}
