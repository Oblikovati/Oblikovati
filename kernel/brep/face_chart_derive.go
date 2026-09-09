// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Deriving a chart for a face whose producer recorded none (ADR-0063).
//
// The primitives and the planar constructors record no chart, and neither does an import. Most of them
// need none: their rings already close in the covering space, and a ring that closes bounds one region
// there. The two that do not — a band bounded by two period-turning rims, and a closed-surface face
// whose loops are all holes — are derivable only where the derivation is not a guess, and this refuses
// everywhere else rather than pick a side. Refusing is what the seventeen deleted rules did not do.

// faceChart is the face's chart: the one its producer recorded, or the one its loops determine when no
// producer did. ok=false when the loops do not determine one — the caller then declines rather than
// picking a side (ADR-0063).
//
// Deriving is legitimate exactly where it is not a guess. A face whose rings all CLOSE in the covering
// space already bounds one region there, and the rings are the chart. A band bounded by two rings that
// each turn a whole period is closed by the seam between them, which is determined when the rings run
// opposite ways and the axis they do NOT turn is non-periodic — there is one strip between them then,
// not two. Everything else needs the datum the producer has and this does not.
func faceChart(f curvedFace, uPer, vPer bool) ([][]math.Point2, bool) {
	if f.chart != nil {
		return f.chart, true
	}
	rings := trimPolys(f, uPer, vPer)
	if len(rings) == 0 {
		return nil, true // boundaryless: the face is its whole surface
	}
	return closeRingsIntoChart(f.surface, rings, f.outerless, uPer, vPer)
}

// closeRingsIntoChart turns projected loop rings into closed contours.
func closeRingsIntoChart(s geom.Surface, rings [][]math.Point2, outerless, uPer, vPer bool) ([][]math.Point2, bool) {
	closed, turning := splitTurningRings(rings, uPer, vPer)
	if len(turning) == 0 {
		return withDomainFrame(s, closed, outerless, uPer, vPer)
	}
	if len(turning) != 2 || outerless || (uPer && vPer) {
		return nil, false // more than one band, or a doubly-closed domain: only the producer knows which side
	}
	circuit, ok := bandCircuit(turning[0], turning[1], uPer)
	if !ok {
		return nil, false
	}
	return append([][]math.Point2{circuit}, closed...), true
}

// splitTurningRings separates rings that close in the covering space from rings that travel a whole
// period instead — the band rims, which are closed circuits in 3-D and open polylines here.
func splitTurningRings(rings [][]math.Point2, uPer, vPer bool) (closed, turning [][]math.Point2) {
	for _, ring := range rings {
		switch {
		case uPer && ringTurnsAPeriod(ring, ringChartU):
			turning = append(turning, ring)
		case vPer && ringTurnsAPeriod(ring, ringChartV):
			turning = append(turning, ring)
		default:
			closed = append(closed, ring)
		}
	}
	return closed, turning
}

// withDomainFrame supplies the outer contour of a face whose loops are ALL holes — the closed-surface
// complement topo records as an outerless face. Its outer boundary in the chart is the parameter
// rectangle itself: a full turn on a periodic axis, the surface's own domain on a bounded one (a
// sphere's latitude, which ends at its poles). That rectangle is exactly the all-seam contour the
// arrangement traces and the 3D loops rightly drop.
//
// It declines an unbounded axis, where there is no rectangle to frame with — and no outerless face
// either, since a surface open to infinity has an exterior its rings already run to.
func withDomainFrame(s geom.Surface, closed [][]math.Point2, outerless, uPer, vPer bool) ([][]math.Point2, bool) {
	if !outerless {
		return closed, true
	}
	c := loopCentroid(closed[0])
	ul, uh := s.UDomain()
	vl, vh := s.VDomain()
	u0, u1 := axisWindow(ul, uh, uPer, float64(c.X))
	v0, v1 := axisWindow(vl, vh, vPer, float64(c.Y))
	if !isFiniteRect(u0, u1, v0, v1) {
		return nil, false
	}
	frame := []math.Point2{math.P2(u0, v0), math.P2(u1, v0), math.P2(u1, v1), math.P2(u0, v1)}
	return append([][]math.Point2{frame}, closed...), true
}

// bandCircuit closes two period-turning rims into the one contour that bounds the strip between them:
// rim, seam across, the other rim back, seam across again — the wire OCCT would have stored.
//
// The two rims must run OPPOSITE ways round the period, which is what a consistently wound band gives
// and what makes the strip between them the region they bound. Each is re-cut at the same crossing of
// the period, which is what makes the result a rectangle rather than the sheared parallelogram an
// arbitrary starting sample would give.
//
// The second rim is cut where the first ENDS: a WHOLE turn from the seam, not the travel the samples
// report. loopToUV leaves a rim one sampling step short of its period — the closing step is the edge a
// sample list leaves implicit — so cutting at that travel put the circuit's two seam traversals a step
// apart and sheared it after all (ADR-0061 stage 5, round 3).
func bandCircuit(a, b []math.Point2, alongU bool) ([]math.Point2, bool) {
	coord := chartCoord(alongU)
	netA, netB := ringTravel(a, coord), ringTravel(b, coord)
	if netA*netB >= 0 {
		return nil, false // both the same way round: not the two ends of one band
	}
	seam := coord(a[0])
	ra, okA := recutRing(a, seam, alongU)
	rb, okB := recutRing(b, seam+stdmath.Copysign(twoPi, netA), alongU)
	if !okA || !okB {
		return nil, false
	}
	return append(ra, rb...), true
}

// recutRing rotates a period-turning ring so it begins where it CROSSES `at`, and shifts it onto that
// branch by a WHOLE TURN and by nothing else.
//
// Both halves of that are load-bearing, and getting them wrong is what put a merged band's chart
// 0.0497 rad off its own edges (ADR-0061 stage 5, round 3). Starting at the NEAREST SAMPLE leaves the
// cut up to half a sampling step from the seam; shifting the ring by that raw difference then
// translates every one of the ring's own vertices by the gap, so the chart disagreed with the face's
// boundary EVERYWHERE, not only at the seam. A face's chart is its own (u, v) boundary (ADR-0063): the
// only re-basing it admits is by whole periods, and the one point this may invent is the crossing,
// interpolated on the ring's own chord.
//
// It declines a ring that is not monotone in the coordinate, where "the crossing" is not one place —
// the figure-eight, and every other case only the producer's chart can settle.
func recutRing(ring []math.Point2, at float64, alongU bool) ([]math.Point2, bool) {
	coord := chartCoord(alongU)
	sign := stdmath.Copysign(1, ringTravel(ring, coord))
	if !ringAdvancesOneWay(ring, coord, sign) {
		return nil, false
	}
	turn := ringOverOnePeriod(ring, coord, at, sign, alongU)
	k, ok := crossingSegment(turn, coord, at, sign)
	if !ok {
		return nil, false
	}
	return rotatedAtCrossing(turn, coord, at, k, sign, alongU), true
}

// ringAdvancesOneWay reports whether the ring only ever moves one way in the periodic coordinate.
func ringAdvancesOneWay(ring []math.Point2, coord func(math.Point2) float64, sign float64) bool {
	for i := 1; i < len(ring); i++ {
		if (coord(ring[i])-coord(ring[i-1]))*sign < 0 {
			return false
		}
	}
	return true
}

// ringOverOnePeriod is the ring shifted onto the branch that holds `at` — by a whole number of periods
// — with its closing point appended, so it spans exactly one period.
func ringOverOnePeriod(ring []math.Point2, coord func(math.Point2) float64, at, sign float64,
	alongU bool) []math.Point2 {
	base := sign * twoPi * stdmath.Floor((at-coord(ring[0]))*sign/twoPi)
	out := make([]math.Point2, 0, len(ring)+1)
	for _, p := range ring {
		out = append(out, shiftAlongChart(p, base, alongU))
	}
	return append(out, shiftAlongChart(out[0], sign*twoPi, alongU))
}

// crossingSegment is the segment of a one-period ring that `at` falls in — the first whose far end is
// strictly past it, so the crossing never lands on the segment's own end and duplicates a vertex.
func crossingSegment(turn []math.Point2, coord func(math.Point2) float64, at, sign float64) (int, bool) {
	for k := 0; k+1 < len(turn); k++ {
		if (coord(turn[k+1])-at)*sign > 0 {
			return k, true
		}
	}
	return 0, false
}

// rotatedAtCrossing restarts the one-period ring at the crossing and walks it back round to the
// crossing a period later. Every ORIGINAL vertex keeps its own coordinate, shifted by whole turns.
func rotatedAtCrossing(turn []math.Point2, coord func(math.Point2) float64, at float64, k int,
	sign float64, alongU bool) []math.Point2 {
	q := crossingPoint(turn[k], turn[k+1], coord, at)
	out := append([]math.Point2{q}, turn[k+1:]...)
	for _, p := range turn[1 : k+1] {
		out = append(out, shiftAlongChart(p, sign*twoPi, alongU))
	}
	return append(out, shiftAlongChart(q, sign*twoPi, alongU))
}

// crossingPoint interpolates the ring's own polyline where it reaches `at`.
func crossingPoint(p, q math.Point2, coord func(math.Point2) float64, at float64) math.Point2 {
	span := coord(q) - coord(p)
	if span == 0 {
		return p
	}
	return p.Lerp(q, math.Scalar((at-coord(p))/span))
}

// shiftAlongChart moves a chart sample along the periodic coordinate.
func shiftAlongChart(p math.Point2, d float64, alongU bool) math.Point2 {
	if alongU {
		return math.P2(float64(p.X)+d, float64(p.Y))
	}
	return math.P2(float64(p.X), float64(p.Y)+d)
}

// chartCoord picks the reader for the periodic coordinate a band turns.
func chartCoord(alongU bool) func(math.Point2) float64 {
	if alongU {
		return ringChartU
	}
	return ringChartV
}

// ringTravel is a ring's net signed travel in one coordinate: ±2π for a ring that turns it.
func ringTravel(ring []math.Point2, coord func(math.Point2) float64) float64 {
	net := 0.0
	for i := 1; i < len(ring); i++ {
		net += coord(ring[i]) - coord(ring[i-1])
	}
	return net
}

// ringTurnsAPeriod reports whether a ring travels a whole turn in a coordinate rather than returning to
// where it started — what makes it an open polyline in the covering space.
func ringTurnsAPeriod(ring []math.Point2, coord func(math.Point2) float64) bool {
	return len(ring) >= 3 && stdmath.Abs(ringTravel(ring, coord)) > twoPi/2
}

// ringChartU and ringChartV read a chart sample's two coordinates.
func ringChartU(p math.Point2) float64 { return float64(p.X) }
func ringChartV(p math.Point2) float64 { return float64(p.Y) }

// wrapToPeriod folds a difference onto [0, 2π): how far forward one must travel to reach it.
func wrapToPeriod(d float64) float64 {
	d = stdmath.Mod(d, twoPi)
	if d < 0 {
		d += twoPi
	}
	return d
}
