// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The winding certificate: does every loop of a face walk with the face's material on its LEFT, seen
// along the face's outward normal? That is the contract every producer signs — an outer loop
// counter-clockwise about the outward normal, a hole clockwise — and it is what the tessellator, the
// analytic integrator and the boolean's own classifiers read the face's region from.
//
// Validate's per-edge test cannot see it. A shell whose every edge is used twice in opposite directions
// is consistently oriented as a use-graph, and it stays so when one face is wound against its normal
// together with the neighbour that shares its edge: the two errors cancel pairwise. The torus tangent
// cut shipped exactly that — one lobe of the figure-eight lid inverted with the torus edge it borders —
// as a valid solid, and only the tessellated volume showed it. This is the post-condition on the
// EMISSION the stage-4 clip left named and unmade (ADR-0061).

// FaceWindingConsistent reports whether every loop of f winds with the face's material on its left
// about the outward normal. certain is false when the face carries a loop that turns a period of its
// surface and no chart to read that loop's sense against, so the question cannot be answered from the
// loops alone; ok is then true by default and means nothing.
//
// Example: if ok, certain := brep.FaceWindingConsistent(f); certain && !ok { /* an inverted face */ }
func FaceWindingConsistent(f *topo.Face) (ok, certain bool) {
	cf := curvedFaceOf(f)
	if len(cf.loops) == 0 {
		return true, true // a whole surface has no winding to get wrong
	}
	uPer, vPer := surfacePeriodic(cf.surface)
	rings := make([][]math.Point2, len(cf.loops))
	for i, loop := range cf.loops {
		rings[i] = loopToUV(cf.surface, loop, uPer, vPer)
	}
	outer := outwardSenseInUV(cf, rings[0][0])
	var wrapping [][]math.Point2
	for _, ring := range rings {
		if ringTravelsAPeriod(ring, cf.surface, uPer, vPer) {
			wrapping = append(wrapping, ring)
		}
	}
	for i, ring := range rings {
		if !ringTravelsAPeriod(ring, cf.surface, uPer, vPer) && !closedRingWindsAs(rings, i, uPer, vPer, outer, len(wrapping) > 0) {
			return false, true
		}
	}
	if len(wrapping) == 0 {
		return true, true
	}
	if len(cf.chart) > 0 {
		return wrappingRingsAgreeWithChart(wrapping, cf.chart, cf.surface, uPer, vPer, outer)
	}
	return wrappingRingsPartition(wrapping, cf.surface, uPer, vPer, outer)
}

// wrappingRingsAgreeWithChart reads every period-turning ring against the chart; the first ring the
// chart cannot place makes the answer uncertain.
func wrappingRingsAgreeWithChart(rings [][]math.Point2, chart [][]math.Point2, s geom.Surface, uPer, vPer bool, outer float64) (ok, certain bool) {
	for _, ring := range rings {
		ok, certain = wrappingRingAgreesWithChart(ring, chart, s, uPer, vPer, outer)
		if !certain || !ok {
			return ok, certain
		}
	}
	return true, true
}

// wrappingRingsPartition certifies a face WITHOUT a chart from its period-turning rings alone: the
// rims of a band. A ring that turns the azimuth bounds the material on its left, so its direction says
// whether the face lies above or below it in v; ordered by v, the rims must then alternate — the lowest
// with material above it, the next with material below, and so on to the highest — and come in pairs,
// or the rings bound no band at all. A face whose rings turn the OTHER axis reads the same way with u
// and v exchanged; one that turns both, or mixes axes, is not decidable here.
func wrappingRingsPartition(rings [][]math.Point2, s geom.Surface, uPer, vPer bool, outer float64) (ok, certain bool) {
	alongU, mixed := ringsTurnU(rings, s, uPer, vPer)
	if mixed || len(rings)%2 != 0 {
		return true, false
	}
	type rim struct{ across, travel float64 }
	rims := make([]rim, len(rings))
	for i, ring := range rings {
		first, last := ring[0], ring[len(ring)-1]
		if alongU {
			rims[i] = rim{across: ringMean(ring, ringChartV), travel: float64(last.X - first.X)}
		} else {
			rims[i] = rim{across: ringMean(ring, ringChartU), travel: -float64(last.Y - first.Y)}
		}
	}
	sort.Slice(rims, func(a, b int) bool { return rims[a].across < rims[b].across })
	for k, r := range rims {
		// Walking +u with material on the left puts the material at +v (above); the v-turning case is
		// mirrored by the travel sign above, so one rule serves both axes.
		materialAbove := r.travel*outer > 0
		if materialAbove != (k%2 == 0) {
			return false, true
		}
	}
	return true, true
}

// ringsTurnU reports whether the rings all turn the u axis (alongU) or all the v axis, and whether they
// mix axes or turn both.
func ringsTurnU(rings [][]math.Point2, s geom.Surface, uPer, vPer bool) (alongU, mixed bool) {
	uHalf, vHalf := 0.0, 0.0
	if lo, hi := s.UDomain(); uPer {
		uHalf = (hi - lo) / 2
	}
	if lo, hi := s.VDomain(); vPer {
		vHalf = (hi - lo) / 2
	}
	sawU, sawV := false, false
	for _, ring := range rings {
		first, last := ring[0], ring[len(ring)-1]
		turnsU := uPer && stdmath.Abs(float64(last.X-first.X)) >= uHalf
		turnsV := vPer && stdmath.Abs(float64(last.Y-first.Y)) >= vHalf
		if turnsU && turnsV {
			return false, true
		}
		sawU, sawV = sawU || turnsU, sawV || turnsV
	}
	return sawU, sawU && sawV
}

// ringMean is the mean of one coordinate over a ring.
func ringMean(ring []math.Point2, coord func(math.Point2) float64) float64 {
	acc := 0.0
	for _, p := range ring {
		acc += coord(p)
	}
	return acc / float64(len(ring))
}

// outwardSenseInUV is the sign a loop's (u,v) shoelace must have to run counter-clockwise about the
// face's OUTWARD normal: the chart's handedness (∂P/∂u × ∂P/∂v against the surface's own normal)
// turned round for a reversed face, whose outward normal is the surface normal's opposite.
func outwardSenseInUV(cf curvedFace, at math.Point2) float64 {
	du, dv := cf.surface.DerivativesAt(float64(at.X), float64(at.Y))
	handed := float64(du.Cross(dv).Dot(cf.surface.NormalAt(float64(at.X), float64(at.Y))))
	sense := 1.0
	if handed < 0 {
		sense = -1
	}
	if cf.reversed {
		sense = -sense
	}
	return sense
}

// ringTravelsAPeriod reports whether an unwrapped ring travels a period along a periodic axis — a
// rim, a parallel — rather than closing on itself. Net travel is zero or a whole period, so the
// half-period mark classifies without a tolerance.
func ringTravelsAPeriod(ring []math.Point2, s geom.Surface, uPer, vPer bool) bool {
	first, last := ring[0], ring[len(ring)-1]
	if uPer {
		lo, hi := s.UDomain()
		if stdmath.Abs(float64(last.X-first.X)) >= (hi-lo)/2 {
			return true
		}
	}
	if vPer {
		lo, hi := s.VDomain()
		if stdmath.Abs(float64(last.Y-first.Y)) >= (hi-lo)/2 {
			return true
		}
	}
	return false
}

// closedRingWindsAs reports whether the i-th ring, which closes in (u,v), winds as its nesting asks:
// with the outward sense when it is an outer loop (enclosed by an even number of the face's other
// closed rings), against it when it is a hole. On a BAND — a face some ring of which turns a period —
// every closed ring is a hole: the band's rims are its outer boundary, and a closed ring lies inside
// the material they bound, whatever the closed rings contain among themselves.
func closedRingWindsAs(rings [][]math.Point2, i int, uPer, vPer bool, outer float64, inBand bool) bool {
	area := ringShoelace(rings[i])
	if area == 0 {
		return false // a loop bounding no region is not a face boundary
	}
	depth := 0
	if inBand {
		depth++
	}
	probe := rings[i][len(rings[i])/2]
	for j, other := range rings {
		if j != i && !ringTurnsAPeriodByTravel(other, uPer, vPer) && pointInRing(ringOnBranchOf(other, probe, uPer, vPer), probe) {
			depth++
		}
	}
	want := outer
	if depth%2 == 1 {
		want = -outer
	}
	return (area > 0) == (want > 0)
}

// ringOnBranchOf shifts a closed ring by whole periods so it lies on the branch nearest to a point.
// Each loop is unwrapped from its own first sample, so two loops of one face can sit a period apart in
// the covering space — a wall's outer loop unwrapped through negative u and its bore holes at the
// positive azimuth they were built on — and containment asked across that gap read every hole as an
// outer loop wound the wrong way.
func ringOnBranchOf(ring []math.Point2, at math.Point2, uPer, vPer bool) []math.Point2 {
	uPeriod, vPeriod := 0.0, 0.0
	if uPer {
		uPeriod = twoPi
	}
	if vPer {
		vPeriod = twoPi
	}
	mean := math.P2(ringMean(ring, ringChartU), ringMean(ring, ringChartV))
	shifted := ontoBranchOf(mean, at, uPeriod, vPeriod)
	du, dv := shifted.X-mean.X, shifted.Y-mean.Y
	if du == 0 && dv == 0 {
		return ring
	}
	out := make([]math.Point2, len(ring))
	for i, p := range ring {
		out[i] = math.P2(float64(p.X+du), float64(p.Y+dv))
	}
	return out
}

// ringTurnsAPeriodByTravel is ringTurnsAPeriod for a ring whose surface is not at hand: a ring that
// does not return to its first point (within the sampling's own closure) turned a period.
func ringTurnsAPeriodByTravel(ring []math.Point2, uPer, vPer bool) bool {
	first, last := ring[0], ring[len(ring)-1]
	return (uPer && stdmath.Abs(float64(last.X-first.X)) >= stdmath.Pi) ||
		(vPer && stdmath.Abs(float64(last.Y-first.Y)) >= stdmath.Pi)
}

// ringShoelace is the signed area of a closed (u,v) ring: positive counter-clockwise.
func ringShoelace(ring []math.Point2) float64 {
	acc := 0.0
	for i := range ring {
		a, b := ring[i], ring[(i+1)%len(ring)]
		acc += float64(a.X*b.Y - b.X*a.Y)
	}
	return acc / 2
}

// pointInRing is the even-odd test of q against a closed ring.
func pointInRing(ring []math.Point2, q math.Point2) bool {
	inside := false
	for i := range ring {
		a, b := ring[i], ring[(i+1)%len(ring)]
		if (a.Y > q.Y) != (b.Y > q.Y) {
			x := float64(a.X) + float64(q.Y-a.Y)*float64(b.X-a.X)/float64(b.Y-a.Y)
			if x > float64(q.X) {
				inside = !inside
			}
		}
	}
	return inside
}

// wrappingRingAgreesWithChart reads a period-turning ring's sense against the face's chart, whose
// contours carry the material on their left in the surface's own parameters (ADR-0063): at the ring's
// middle sample the ring's direction must run with the nearest contour segment when the outward sense
// is the chart's, and against it otherwise. Without a chart the sense of a wrapping loop is not
// decidable from the loop, and the answer is uncertain.
func wrappingRingAgreesWithChart(ring []math.Point2, chart [][]math.Point2, s geom.Surface, uPer, vPer bool, outer float64) (ok, certain bool) {
	if len(chart) == 0 || len(ring) < 2 {
		return true, false
	}
	uLo, uHi := s.UDomain()
	vLo, vHi := s.VDomain()
	uPeriod, vPeriod := periodOfDomain(uLo, uHi, uPer), periodOfDomain(vLo, vHi, vPer)
	// A sample that lands on a contour VERTEX is nearest to two segments at once, and the wrong one
	// of a rim's corner is the seam running across it; the next sample along is interior to the
	// rim's own contour edge, so the first sample whose nearest segment has a component along the
	// ring is the one read.
	for k := len(ring) / 2; k+1 < len(ring); k++ {
		at, dir := ring[k], ring[k].VectorTo(ring[k+1])
		seg, found := nearestChartSegment(chart, at, uPeriod, vPeriod)
		if !found {
			return true, false
		}
		if along := float64(dir.Dot(seg)); along != 0 {
			return (along > 0) == (outer > 0), true
		}
	}
	return true, false
}

// periodOfDomain is a periodic axis's period, or zero for a bounded one.
func periodOfDomain(lo, hi float64, periodic bool) float64 {
	if !periodic {
		return 0
	}
	return hi - lo
}

// nearestChartSegment returns the direction of the chart segment nearest to a point, the point first
// carried onto each segment's own branch of the periodic axes, so a ring unwrapped on one branch meets
// a contour drawn on another. The distance is to the SEGMENT, not its midpoint: a band's rim runs the
// whole azimuth, and against a midpoint the rim's mid-sample was nearer the seam's short side than the
// rim contour it lies on.
func nearestChartSegment(chart [][]math.Point2, at math.Point2, uPeriod, vPeriod float64) (math.Vector2, bool) {
	best, bestD := math.Vector2{}, stdmath.Inf(1)
	for _, contour := range chart {
		for i := range contour {
			a, b := contour[i], contour[(i+1)%len(contour)]
			q := ontoBranchOf(at, math.P2(float64(a.X+b.X)/2, float64(a.Y+b.Y)/2), uPeriod, vPeriod)
			if d := perpDistToSeg(q, a, b); d < bestD {
				best, bestD = a.VectorTo(b), d
			}
		}
	}
	return best, !stdmath.IsInf(bestD, 1)
}

// ontoBranchOf shifts p by whole periods so it lies on the branch nearest to ref.
func ontoBranchOf(p, ref math.Point2, uPeriod, vPeriod float64) math.Point2 {
	u, v := float64(p.X), float64(p.Y)
	if uPeriod > 0 {
		u -= uPeriod * stdmath.Round((u-float64(ref.X))/uPeriod)
	}
	if vPeriod > 0 {
		v -= vPeriod * stdmath.Round((v-float64(ref.Y))/vPeriod)
	}
	return math.P2(u, v)
}
