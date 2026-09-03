// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"slices"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// ISLAND imprints for the exact-frame planar face (planeFaceUV, ADR-0058 / #3460). The mixed boolean's
// plane∩plane imprints are straight segments that terminate on the frame; a curved operand's section is
// instead a CLOSED boundary sitting wholly inside the frame — the circle a boss or a through-cylinder
// cuts into a plate face, the oval a plane cuts out of a torus. An island needs no crossing injection
// (it never meets a frame edge, which pairUVWallImprints proves in closed form before the trim runs),
// only a sampling that carries the analytic curves, so the kept boundary re-emits them exactly — the
// SAME curves the curved side's own arrangement re-emits, which is what welds the two faces.
//
// An island is a CYCLE, not a curve: a section arrives as however many pieces its own construction
// makes, and they are assembled before they are classified (ADR-0062).

// splitImprintByKind separates a mixed imprint list into the STRAIGHT segments (plane∩plane lines, split
// against the frame's curved edges), the CLOSED islands (a curved operand's section, gated by
// wallSectionIsland to lie wholly inside the frame) and the OPEN arcs that cross the frame. Keeping them
// apart is what lets the straight path keep reading PointAt(0)/PointAt(1) as segment ends — a curved
// arc's are two points a chord apart. Open arcs that close on each other are assembled into an island
// first, so a boundary delivered in pieces is classified as the one boundary it is.
func splitImprintByKind(imprint []geom.Curve3) (straight []geom.Curve3, islands []imprintCycle, open []geom.Curve3) {
	var arcs []geom.Curve3
	for _, cv := range imprint {
		switch {
		case isClosedIslandImprint(cv):
			islands = append(islands, imprintCycle{{curve: cv, t0: domLo(cv), t1: domHi(cv)}})
		case openCurvedKind(cv):
			arcs = append(arcs, cv)
		default:
			straight = append(straight, cv)
		}
	}
	cycles, rest := chainImprintCycles(arcs)
	return straight, append(islands, cycles...), rest
}

// imprintArc is one imprint curve walked over a parameter span; t1 < t0 when the cycle traverses it
// backwards, which is how a chain built from arcs of either sense stays continuous.
type imprintArc struct {
	curve  geom.Curve3
	t0, t1 float64
}

// imprintCycle is a CLOSED imprint boundary: one closed curve, or several open arcs that meet end to
// end and come back to where they started.
//
// A section need not arrive as one curve. A plane parallel to a torus axis, offset between the inner
// and outer tube radii, sections it in ONE oval delivered as TWO spiric branches meeting at the oval's
// v-extremes — a bigon. Neither branch closes, so neither is an island on its own, and neither crosses
// the receiving face's boundary either, so neither is an open crossing: the face kept nothing and the
// tool's lid passed through whole (ADR-0062).
//
// So the arcs are ASSEMBLED before they are classified, which is what OCCT's BOPAlgo_BuilderFace does —
// PerformLoops builds wires out of edges, and only then does PerformAreas decide which wire is a growth
// and which a hole. Nothing here knows a spiric from a conic; it knows which ends meet.
type imprintCycle []imprintArc

// chainImprintCycles assembles open arcs that meet end to end into closed cycles, returning the cycles
// and the arcs that do not close.
func chainImprintCycles(arcs []geom.Curve3) ([]imprintCycle, []geom.Curve3) {
	if len(arcs) < 2 {
		return nil, arcs
	}
	res := imprintChainResolution(arcs)
	used := make([]bool, len(arcs))
	var cycles []imprintCycle
	var rest []geom.Curve3
	for i := range arcs {
		if used[i] {
			continue
		}
		cycle, ok := traceImprintCycle(arcs, used, i, res)
		if !ok {
			continue
		}
		cycles = append(cycles, cycle)
	}
	for i, cv := range arcs {
		if !used[i] {
			rest = append(rest, cv)
		}
	}
	return cycles, rest
}

// traceImprintCycle follows arcs from seed's end to the next arc that starts there, until it returns to
// the seed's start. It marks the arcs it consumed only when the walk CLOSES, so a chain that runs out
// leaves its arcs for the open path.
func traceImprintCycle(arcs []geom.Curve3, used []bool, seed int, res geom.Resolution) (imprintCycle, bool) {
	taken := []int{seed}
	cycle := imprintCycle{{curve: arcs[seed], t0: domLo(arcs[seed]), t1: domHi(arcs[seed])}}
	start := arcs[seed].PointAt(domLo(arcs[seed]))
	at := arcs[seed].PointAt(domHi(arcs[seed]))
	for range arcs {
		if samePoint(at, start, res) {
			for _, k := range taken {
				used[k] = true
			}
			return cycle, true
		}
		next, arc, ok := nextImprintArc(arcs, used, taken, at, res)
		if !ok {
			return nil, false
		}
		taken = append(taken, next)
		cycle = append(cycle, arc)
		at = arc.curve.PointAt(arc.t1)
	}
	return nil, false
}

// nextImprintArc finds an unconsumed arc with an end AT p, oriented to leave from there.
func nextImprintArc(arcs []geom.Curve3, used []bool, taken []int, p math.Point3, res geom.Resolution) (int, imprintArc, bool) {
	for i, cv := range arcs {
		if used[i] || slices.Contains(taken, i) {
			continue
		}
		lo, hi := cv.Domain()
		if samePoint(cv.PointAt(lo), p, res) {
			return i, imprintArc{curve: cv, t0: lo, t1: hi}, true
		}
		if samePoint(cv.PointAt(hi), p, res) {
			return i, imprintArc{curve: cv, t0: hi, t1: lo}, true
		}
	}
	return 0, imprintArc{}, false
}

// imprintChainResolution is the model-relative tolerance the chaining welds ends at: derived from the
// arcs' own extent, so it scales with the part (ADR-0042).
func imprintChainResolution(arcs []geom.Curve3) geom.Resolution {
	pts := make([]math.Point3, 0, 2*len(arcs))
	for _, cv := range arcs {
		lo, hi := cv.Domain()
		pts = append(pts, cv.PointAt(lo), cv.PointAt(hi))
	}
	return geom.ResolutionForPoints(pts)
}

// domLo and domHi are a curve's own domain ends.
func domLo(cv geom.Curve3) float64 { lo, _ := cv.Domain(); return lo }
func domHi(cv geom.Curve3) float64 { _, hi := cv.Domain(); return hi }

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
func (c *planeFaceUV) islandCurveSegs(arc imprintArc) []uvSeg {
	segs := make([]uvSeg, 0, imprintSampleCount)
	prevT, prev := arc.t0, to2D(c.plane, arc.curve.PointAt(arc.t0))
	for i := 1; i <= imprintSampleCount; i++ {
		t := arc.t0 + (arc.t1-arc.t0)*float64(i)/imprintSampleCount
		p := to2D(c.plane, arc.curve.PointAt(t))
		segs = append(segs, uvSeg{a: prev, b: p, curve: arc.curve, tA: prevT, tB: t, kind: segImprint})
		prevT, prev = t, p
	}
	return segs
}

// islandSegs samples every closed island imprint of the face.
func (c *planeFaceUV) islandSegs(islands []imprintCycle) []uvSeg {
	var out []uvSeg
	for _, cyc := range islands {
		for _, arc := range cyc {
			out = append(out, c.islandCurveSegs(arc)...)
		}
	}
	return out
}

// islandContactOK gates a closed-conic island against everything it must NOT meet inside the frame: a
// straight imprint segment, and another island. The exact-crossing injection covers frame conics only, so
// such a meeting would be resolved on the island's sampled CHORD — off the true conic by the sagitta —
// while the wall's own arrangement places it on the conic, leaving a T-junction the stitch cannot weld.
// Both are named declines, never approximations (#3460).
func islandContactOK(c *planeFaceUV, islands []imprintCycle, straight []geom.Curve3) bool {
	conics := make([]planeConic, 0, len(islands))
	allConic := true
	for _, cyc := range islands {
		if len(cyc) != 1 {
			allConic = false // an assembled cycle is not one curve, so no closed form covers it
			continue
		}
		pc, ok := toPlaneConic(cyc[0].curve, c.plane)
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
		for i := range islands {
			if !conicClearOfSegments(c, conics[i], straight) {
				return false
			}
		}
		return conicsNestedOrApart(conics)
	}
	for _, cyc := range islands {
		if !islandWalkClearOfSegments(c, cyc, straight) {
			return false
		}
	}
	return islandsWalkNestedOrApart(c, islands)
}

// islandWalkClearOfSegments reports one island staying clear of every straight imprint, by walking the
// island and measuring to each segment.
func islandWalkClearOfSegments(c *planeFaceUV, cyc imprintCycle, straight []geom.Curve3) bool {
	for _, imp := range straight {
		a2, b2 := to2D(c.plane, imp.PointAt(0)), to2D(c.plane, imp.PointAt(1))
		for _, p := range islandWalk(c, cyc) {
			if pointSegmentDistance2D(p, a2, b2) <= c.res.Sew() {
				return false
			}
		}
	}
	return true
}

// islandsWalkNestedOrApart reports every island pair being wholly apart or wholly nested, by walking one
// against the other's sampled ring: a pair that CROSSES has samples on both sides.
func islandsWalkNestedOrApart(c *planeFaceUV, islands []imprintCycle) bool {
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

// islandWalk samples one island cycle into its (u,v) ring, arc by arc in traversal order.
func islandWalk(c *planeFaceUV, cyc imprintCycle) []math.Point2 {
	out := make([]math.Point2, 0, len(cyc)*imprintSampleCount)
	for _, arc := range cyc {
		for k := 0; k < imprintSampleCount; k++ {
			out = append(out, to2D(c.plane, arc.curve.PointAt(arc.t0+(arc.t1-arc.t0)*float64(k)/imprintSampleCount)))
		}
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
