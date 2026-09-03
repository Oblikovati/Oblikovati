// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
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
func splitImprintByKind(imprint []geom.Curve3) (straight, islands, open []geom.Curve3) {
	for _, cv := range imprint {
		switch {
		case isClosedIslandImprint(cv):
			islands = append(islands, cv)
		case openConicKind(cv):
			open = append(open, cv)
		default:
			straight = append(straight, cv)
		}
	}
	return straight, islands, open
}

// isClosedIslandImprint reports the imprints that bound a region on their OWN — the island kind: a
// curve whose ends meet. A bounded ARC is the open kind instead, whatever it runs on (ADR-0060).
//
// The property is closure, not conic-ness. A torus's spiric oval closes on itself exactly as a circle
// does, and the sampling below drives off Domain and PointAt alone, so it needs nothing a conic has that
// a spiric has not. Testing for a conic sent every spiric to the STRAIGHT bucket, where a closed curve
// becomes a zero-length segment between its coincident ends and the face kept nothing (ADR-0061 stage 3).
func isClosedIslandImprint(cv geom.Curve3) bool { return geom.CurveIsClosed(cv) }

// islandCurveSegs samples one closed island imprint over its WHOLE domain into tagged (u,v) segments
// that carry the source curve and its endpoint parameters, so a kept boundary run re-emits the exact
// analytic curve instead of the sampled chords.
func (c *planeFaceUV) islandCurveSegs(cv geom.Curve3) []uvSeg {
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

// islandSegs samples every closed island imprint of the face.
func (c *planeFaceUV) islandSegs(islands []geom.Curve3) []uvSeg {
	var out []uvSeg
	for _, cv := range islands {
		out = append(out, c.islandCurveSegs(cv)...)
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
	allConic := true
	for _, cv := range islands {
		pc, ok := toPlaneConic(cv, c.plane)
		if !ok {
			allConic = false
			continue
		}
		conics = append(conics, pc)
	}
	// A CONIC island is tested in closed form. Any other analytic island — a torus's spiric oval — is
	// tested by walking itself, which is exact evaluation of an exact curve: the property is geometric,
	// not a property of being a conic, and requiring one declined every spiric outright
	// (ADR-0061 stage 3).
	if allConic {
		for i, cv := range islands {
			_ = cv
			if !conicClearOfSegments(c, conics[i], straight) {
				return false
			}
		}
		return conicsNestedOrApart(conics)
	}
	for _, cv := range islands {
		if !islandWalkClearOfSegments(c, cv, straight) {
			return false
		}
	}
	return islandsWalkNestedOrApart(c, islands)
}

// islandWalkClearOfSegments reports one island staying clear of every straight imprint, by walking the
// island and measuring to each segment.
func islandWalkClearOfSegments(c *planeFaceUV, cv geom.Curve3, straight []geom.Curve3) bool {
	for _, imp := range straight {
		a2, b2 := to2D(c.plane, imp.PointAt(0)), to2D(c.plane, imp.PointAt(1))
		for _, p := range islandWalk(c, cv) {
			if pointSegmentDistance2D(p, a2, b2) <= c.res.Sew() {
				return false
			}
		}
	}
	return true
}

// islandsWalkNestedOrApart reports every island pair being wholly apart or wholly nested, by walking one
// against the other's sampled ring: a pair that CROSSES has samples on both sides.
func islandsWalkNestedOrApart(c *planeFaceUV, islands []geom.Curve3) bool {
	for i := range islands {
		for j := range islands {
			if i == j {
				continue
			}
			ring := islandWalk(c, islands[j])
			in, out := 0, 0
			for _, p := range islandWalk(c, islands[i]) {
				if pointInRing2D(p, ring) {
					in++
				} else {
					out++
				}
			}
			if in > 0 && out > 0 {
				return false
			}
		}
	}
	return true
}

// islandWalk samples one island into its (u,v) ring.
func islandWalk(c *planeFaceUV, cv geom.Curve3) []math.Point2 {
	lo, hi := cv.Domain()
	out := make([]math.Point2, 0, imprintSampleCount)
	for k := 0; k < imprintSampleCount; k++ {
		out = append(out, to2D(c.plane, cv.PointAt(lo+(hi-lo)*float64(k)/imprintSampleCount)))
	}
	return out
}

// pointInRing2D is the even-odd test of a point against a sampled closed ring.
func pointInRing2D(p math.Point2, ring []math.Point2) bool {
	in := false
	for i, n := 0, len(ring); i < n; i++ {
		a, b := ring[i], ring[(i+1)%n]
		if (a.Y > p.Y) == (b.Y > p.Y) {
			continue
		}
		x := float64(a.X) + float64(p.Y-a.Y)/float64(b.Y-a.Y)*float64(b.X-a.X)
		if float64(p.X) < x {
			in = !in
		}
	}
	return in
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
