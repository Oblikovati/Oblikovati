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

// islandContactOK gates a closed-conic island against everything it must NOT meet inside the frame: a
// straight imprint segment, and another island. The exact-crossing injection covers frame conics only, so
// such a meeting would be resolved on the island's sampled CHORD — off the true conic by the sagitta —
// while the wall's own arrangement places it on the conic, leaving a T-junction the stitch cannot weld.
// Both are named declines, never approximations (#3460).
func islandContactOK(c *planeFaceUV, islands, straight []geom.Curve3) bool {
	conics := make([]planeConic, 0, len(islands))
	for _, cv := range islands {
		pc, ok := toPlaneConic(cv, c.plane)
		if !ok {
			return false
		}
		if !conicClearOfSegments(c, pc, straight) {
			return false
		}
		conics = append(conics, pc)
	}
	return conicsNestedOrApart(conics)
}

// conicClearOfSegments reports one island crossing (or grazing) no straight imprint segment.
func conicClearOfSegments(c *planeFaceUV, pc planeConic, straight []geom.Curve3) bool {
	for _, imp := range straight {
		a2, b2 := to2D(c.plane, imp.PointAt(0)), to2D(c.plane, imp.PointAt(1))
		hits, tangent := conicEdgeHits(pc, a2, b2, c.res)
		if tangent || len(hits) > 0 {
			return false
		}
	}
	return true
}

// conicsNestedOrApart reports every island pair being strictly apart or strictly nested — the two
// arrangements that need no conic∩conic inversion. The test is on each conic's circumscribed (A) and
// inscribed (B) radii, so it is conservative for an ellipse and exact for a circle (A==B).
func conicsNestedOrApart(conics []planeConic) bool {
	for i := range conics {
		for j := i + 1; j < len(conics); j++ {
			if !conicPairSeparated(conics[i], conics[j]) {
				return false
			}
		}
	}
	return true
}

// conicPairSeparated reports one island pair being apart (centres farther than the two circumscribed
// radii) or nested (one's circumscribed disc strictly inside the other's inscribed disc).
func conicPairSeparated(a, b planeConic) bool {
	d := float64(a.center.DistanceTo(b.center))
	return d > a.A+b.A || d+b.A < a.B || d+a.A < b.B
}
