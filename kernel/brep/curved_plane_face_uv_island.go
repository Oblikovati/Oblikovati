// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"slices"
	"sort"

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
func splitImprintByKind(imprint []geom.Curve3) (straight []geom.Curve3, islands []imprintCycle, touches []math.Point3, open []geom.Curve3) {
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
	split, at := splitIslandsAtTouches(append(islands, cycles...))
	return straight, split, at, rest
}

// splitIslandsAtTouches makes every meeting between island arcs — with each other, or of an arc with
// itself — a shared vertex: an interior meeting splits the arc there, and a meeting AT an arc's end
// replaces that end with the solved point.
//
// A section that touches bounds two lobes, and without a vertex at the meeting the arrangement joins
// them into ONE cell whose boundary is a single self-touching circuit. Its lobes then wind oppositely,
// its boundary integral cancels, and the face it bounds measures nothing: measured on the oblique
// figure-eight, a lid of two 25.267 lobes integrating to 1.41e-06 (ADR-0062).
//
// Both shapes of the defect occur and both are covered. The mixed boolean delivers the figure-eight as
// ONE self-touching spiric, which splits. The half-space composition delivers it as TWO spiric ovals
// that already END at the meeting — and those are the ones that need the solved point rather than a
// split, because the spiric's u(v) = Φ ± arccos w is ill-conditioned there (d(arccos)/dw diverges as
// w → ±1) and the two arcs evaluate their shared point 1.03e-07 apart, past the arrangement's 1e-09
// vertex weld. The meeting is TANGENTIAL either way, so nothing crosses and no sampling finds it;
// geom.CurveTouches solves it.
func splitIslandsAtTouches(islands []imprintCycle) ([]imprintCycle, []math.Point3) {
	cuts, meets, at := islandTouchCuts(islands)
	out := make([]imprintCycle, 0, len(islands))
	for i, cyc := range islands {
		out = append(out, splitCycleAt(withArcMeets(cyc, meets[i]), cuts[i]))
	}
	return out, at
}

// arcEnd names one end of one arc of one island, for recording a solved meeting there.
type arcEnd struct {
	island, arc int
	last        bool
}

// withArcMeets attaches the solved meeting points to a cycle's arc ends.
func withArcMeets(cyc imprintCycle, meets map[int][2]*math.Point3) imprintCycle {
	out := make(imprintCycle, len(cyc))
	copy(out, cyc)
	for ai := range out {
		if m, ok := meets[ai]; ok {
			out[ai].meet0, out[ai].meet1 = m[0], m[1]
		}
	}
	return out
}

// islandTouchCuts solves every meeting and sorts it into an interior cut or an end replacement.
func islandTouchCuts(islands []imprintCycle) ([]map[int][]float64, []map[int][2]*math.Point3, []math.Point3) {
	cuts := make([]map[int][]float64, len(islands))
	meets := make([]map[int][2]*math.Point3, len(islands))
	for i := range islands {
		cuts[i], meets[i] = map[int][]float64{}, map[int][2]*math.Point3{}
	}
	var at []math.Point3
	for _, hit := range islandTouchHits(islands) {
		at = append(at, hit.at)
		recordTouch(cuts, meets, islands, hit.a, hit.ta, hit.at)
		recordTouch(cuts, meets, islands, hit.b, hit.tb, hit.at)
	}
	return cuts, meets, at
}

// islandTouch is one solved meeting: the two arcs, the parameter on each, and the point.
type islandTouch struct {
	a, b   arcEnd
	ta, tb float64
	at     math.Point3
}

// islandTouchHits solves every arc pair, including each arc against itself.
func islandTouchHits(islands []imprintCycle) []islandTouch {
	var out []islandTouch
	for i, ci := range islands {
		for ai, arcA := range ci {
			for j := i; j < len(islands); j++ {
				for bi, arcB := range islands[j] {
					if j == i && bi < ai {
						continue // the pair was taken the other way round
					}
					out = append(out, arcPairTouches(islands, i, ai, arcA, j, bi, arcB)...)
				}
			}
		}
	}
	return out
}

// arcPairTouches solves one pair. The meeting point is the MIDPOINT of the two evaluations, which is
// the symmetric choice and the best estimate of a point neither arc can evaluate exactly.
func arcPairTouches(islands []imprintCycle, i, ai int, arcA imprintArc, j, bi int, arcB imprintArc) []islandTouch {
	same := i == j && ai == bi
	tol := stdmath.Min(islandTouchWeld(arcA.curve), islandTouchWeld(arcB.curve))
	var out []islandTouch
	for _, hit := range geom.CurveTouches(arcA.curve, arcB.curve, same, tol) {
		pa, pb := arcA.curve.PointAt(hit[0]), arcB.curve.PointAt(hit[1])
		out = append(out, islandTouch{
			a: arcEnd{island: i, arc: ai}, b: arcEnd{island: j, arc: bi},
			ta: hit[0], tb: hit[1], at: pa.TranslateBy(pa.VectorTo(pb).Scale(0.5)),
		})
	}
	return out
}

// recordTouch files one side of a meeting: an end replacement when the parameter is an arc's own end,
// an interior cut otherwise.
func recordTouch(cuts []map[int][]float64, meets []map[int][2]*math.Point3, islands []imprintCycle, e arcEnd, t float64, at math.Point3) {
	arc := islands[e.island][e.arc]
	switch {
	case stdmath.Abs(t-arc.t0) <= stdmath.Abs(arc.t1-arc.t0)*arcEndFraction:
		m := meets[e.island][e.arc]
		meets[e.island][e.arc] = [2]*math.Point3{&at, m[1]}
	case stdmath.Abs(t-arc.t1) <= stdmath.Abs(arc.t1-arc.t0)*arcEndFraction:
		m := meets[e.island][e.arc]
		meets[e.island][e.arc] = [2]*math.Point3{m[0], &at}
	default:
		cuts[e.island][e.arc] = append(cuts[e.island][e.arc], t)
	}
}

// atArcEnd reports a parameter sitting on one of an arc's own ends.
func atArcEnd(a imprintArc, t float64) bool {
	span := stdmath.Abs(a.t1 - a.t0)
	return stdmath.Min(stdmath.Abs(t-a.t0), stdmath.Abs(t-a.t1)) <= span*arcEndFraction
}

// arcEndFraction is how near an arc's end a parameter counts as being AT it, as a fraction of the
// arc's own span. It is a parameter-space proportion, not a model distance.
const arcEndFraction = 1e-6

// splitCycleAt subdivides a cycle's arcs at the recorded parameters, dropping cuts that fall on an
// arc's own ends (they are already vertices).
func splitCycleAt(cyc imprintCycle, cuts map[int][]float64) imprintCycle {
	out := make(imprintCycle, 0, len(cyc))
	for ai, arc := range cyc {
		out = append(out, splitArcAt(arc, cuts[ai])...)
	}
	return out
}

// splitArcAt cuts one arc at the given parameters, in the arc's own direction.
func splitArcAt(arc imprintArc, at []float64) imprintCycle {
	inside := make([]float64, 0, len(at))
	for _, t := range at {
		if !atArcEnd(arc, t) && betweenParams(t, arc.t0, arc.t1) {
			inside = append(inside, t)
		}
	}
	if len(inside) == 0 {
		return imprintCycle{arc}
	}
	sort.Float64s(inside)
	if arc.t1 < arc.t0 {
		slices.Reverse(inside)
	}
	out := make(imprintCycle, 0, len(inside)+1)
	prev, prevMeet := arc.t0, arc.meet0
	for _, t := range inside {
		at := arc.curve.PointAt(t)
		out = append(out, imprintArc{curve: arc.curve, t0: prev, t1: t, meet0: prevMeet, meet1: &at})
		prev, prevMeet = t, &at
	}
	return append(out, imprintArc{curve: arc.curve, t0: prev, t1: arc.t1, meet0: prevMeet, meet1: arc.meet1})
}

// betweenParams reports a parameter lying strictly within a span given in either direction.
func betweenParams(t, a, b float64) bool {
	if a > b {
		a, b = b, a
	}
	return t > a && t < b
}

// islandTouchWeld is the model-relative distance at which two visits to a point are the SAME point,
// taken from the curve's own extent — a coincidence of one computation with itself, so the weld class.
func islandTouchWeld(cv geom.Curve3) float64 {
	lo, hi := cv.Domain()
	pts := make([]math.Point3, 0, islandTouchProbe+1)
	for i := 0; i <= islandTouchProbe; i++ {
		pts = append(pts, cv.PointAt(lo+(hi-lo)*float64(i)/islandTouchProbe))
	}
	return geom.ResolutionForPoints(pts).Weld()
}

// islandTouchProbe samples the island to size it. It bounds the curve, nothing more.
const islandTouchProbe = 16

// imprintArc is one imprint curve walked over a parameter span; t1 < t0 when the cycle traverses it
// backwards, which is how a chain built from arcs of either sense stays continuous.
type imprintArc struct {
	curve  geom.Curve3
	t0, t1 float64
	// meet0/meet1 replace the curve's own evaluation at an end where a SOLVED incidence says two arcs
	// meet. Near a pinch the spiric's u(v) = Φ ± arccos w is ill-conditioned — d(arccos)/dw diverges as
	// w → ±1 — so two arcs that meet there evaluate their shared point 1.03e-07 apart, past the
	// arrangement's 1e-09 vertex weld. The arrangement then saw two vertices with a sliver between them
	// and made the two lobes ONE cell (ADR-0062). nil at an ordinary end.
	meet0, meet1 *math.Point3
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
	prevT, prev := arc.t0, c.arcEndPoint(arc, arc.meet0, arc.t0)
	for i := 1; i <= imprintSampleCount; i++ {
		t := arc.t0 + (arc.t1-arc.t0)*float64(i)/imprintSampleCount
		p := to2D(c.plane, arc.curve.PointAt(t))
		if i == imprintSampleCount {
			p = c.arcEndPoint(arc, arc.meet1, arc.t1)
		}
		segs = append(segs, uvSeg{a: prev, b: p, curve: arc.curve, tA: prevT, tB: t, kind: segImprint})
		prevT, prev = t, p
	}
	return segs
}

// arcEndPoint is the (u,v) an arc's end sits at: the SOLVED meeting point where one was found, and the
// curve's own evaluation otherwise. Using the solve is what makes two arcs that meet hand the
// arrangement one vertex rather than two a hair apart.
func (c *planeFaceUV) arcEndPoint(arc imprintArc, meet *math.Point3, t float64) math.Point2 {
	if meet != nil {
		return to2D(c.plane, *meet)
	}
	return to2D(c.plane, arc.curve.PointAt(t))
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
