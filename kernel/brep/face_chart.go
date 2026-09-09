// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/predicates"
	"oblikovati.org/math"
)

// A trimmed face's PARAMETRIC boundary, carried from the producer that wound it (ADR-0063).
//
// On a periodic surface the 3D loops do not determine the face. A cylinder band's two rim circles
// bound the strip between them AND the strip the other way round the seam; a torus's two spiric ovals
// bound both bands they separate. Every reader that asks "is this (u,v) point on this face" therefore
// has to choose, and until this existed each of them chose by its own rule: trim_region.go had four
// (an azimuth-wrapping rim, a tube-wrapping ring, which of two periodic windows holds material, and
// the outerless complement), classify_trim_uv.go had two more (leave a wrapping loop open; fall back
// to the loop's handedness when the rings bound nothing). They disagreed, which is the run of defects
// ADR-0062 collected.
//
// OCCT does not have the choice to make: it stores the seam edge in the wire twice with two pcurves,
// so every wire is a CLOSED contour in (u,v) and one uniform rule — signed area, then even-odd —
// covers every face it can build. This is that fact in the shape this kernel's B-rep wants: the 3D
// loops stay the true geometric boundary (two rim circles, not a keyhole), and the closed contour is
// carried beside them as the face's chart.
//
// It cannot be reassembled afterwards, which is worth stating because the cheap version was tried and
// measured. loopToUV unwraps each loop onto whichever turn it started on, so two rims of one band land
// on different branches; a shoelace survives that (the closing chords supply the seams) but even-odd
// does not, and the concatenation reads as a zigzag. Where two lobes TOUCH — the oblique figure-eight —
// there is no seam-free circuit to reassemble at all. The information exists only where the
// arrangement produced it, so that is where it is recorded.

// chartOfKept traces the kept arrangement cells' boundary WITHOUT folding the artificial seams, so
// every contour it returns closes in the covering space.
//
// It is the same trace the face emission takes, with one bit turned off. keptBoundaryEdges welds u=2π
// onto u=0 (and, on a torus, v=2π onto v=0), which turns a wrapping region's two seam traversals into
// reverse twins that cancel — deliberately, because the seam bounds nothing real in 3D. Unfolded, they
// survive, and the same walk that yields two open rim polylines yields the one closed contour that
// runs rim → seam → rim → seam. That contour is the chart.
func chartOfKept(kept []Face2D) [][]dedge {
	return chainLoops(keptBoundaryEdges(kept, false, false))
}

// chartContours converts traced (u,v) contours into the surface's OWN parameters and orders them
// outer-first. origin is the chart's parameter origin in surface parameters — the (u,v) the
// arrangement's (0,0) sits at, which is where each side placed its seams (uvSide.seamOrigin).
//
// The offset is added WITHOUT wrapping. A contour that spans the seam has to stay continuous to be a
// polygon at all, so the chart lives on the branch [origin, origin+2π] and a query is taken into that
// branch by chartContains rather than the contour being folded into [0, 2π).
func chartContours(loops [][]dedge, origin math.Point2) [][]math.Point2 {
	out := make([][]math.Point2, 0, len(loops))
	for _, lp := range loops {
		if len(lp) < 3 {
			continue // fewer than three vertices bounds no area
		}
		poly := make([]math.Point2, 0, len(lp))
		for _, e := range lp {
			poly = append(poly, math.P2(float64(e.a.X)+float64(origin.X), float64(e.a.Y)+float64(origin.Y)))
		}
		out = append(out, dropCollinearVertices(poly))
	}
	sort.SliceStable(out, func(i, j int) bool {
		return stdmath.Abs(polySignedArea(out[i])) > stdmath.Abs(polySignedArea(out[j]))
	})
	return out
}

// dropCollinearVertices removes vertices that lie exactly on the segment between their neighbours.
//
// It is lossless — the predicate is exact, so a vertex is dropped only when the polygon is unchanged by
// dropping it — and it matters because most of a chart is straight. A band rim is constant v and a seam
// constant u, so each arrives as dozens of sampled points along one line and leaves as one segment. The
// contours are read per query by every containment test and per quadrature cell by the flux integrator,
// so their length is the cost of the chart.
func dropCollinearVertices(poly []math.Point2) []math.Point2 {
	if len(poly) < 4 {
		return poly
	}
	out := make([]math.Point2, 0, len(poly))
	for i, n := 0, len(poly); i < n; i++ {
		a, b, c := poly[(i-1+n)%n], poly[i], poly[(i+1)%n]
		if predicates.Orient2D(float64(a.X), float64(a.Y), float64(b.X), float64(b.Y), float64(c.X), float64(c.Y)) != 0 {
			out = append(out, b)
		}
	}
	if len(out) < 3 {
		return poly // every vertex collinear: a degenerate contour, kept as it came
	}
	return out
}

// keptChart is one connected kept COMPONENT's chart: its closed contours, and the keys of its own
// (folded) boundary edges so the face emitted from it can claim it.
type keptChart struct {
	contours [][]math.Point2
	edges    map[[4]int64]bool
}

// chartsOfComponents charts every connected component of the kept region.
//
// The component, not the emitted loop group, is the unit. They are the same faces — a connected set of
// cells has one outer boundary, and two disjoint outers are two components — but a component's chart
// carries contours its loops do not: the genus-1 complement's outer contour is the whole parameter
// rectangle, made of nothing but seam, and dropArtificialLoops rightly removes it from the 3D loops
// because it bounds no geometry. In the CHART it bounds the face (ADR-0063). Claiming contours by the
// loops that survived would have left that face with its hole and no outer, and read every point of it
// inverted.
func chartsOfComponents(c uvSide, kept []Face2D) []keptChart {
	uPer, vPer := c.uPeriodic(), c.vPeriodic()
	comps := keptComponents(kept, uPer, vPer)
	out := make([]keptChart, 0, len(comps))
	for _, comp := range comps {
		kc := keptChart{contours: chartContours(chartOfKept(comp), c.seamOrigin()), edges: map[[4]int64]bool{}}
		for _, d := range keptBoundaryEdges(comp, uPer, vPer) {
			kc.edges[chartEdgeKey(d)] = true
		}
		out = append(out, kc)
	}
	return out
}

// chartForGroup returns the chart of the component an emitted face's boundary loops came from.
//
// The match is exact rather than geometric: the group's dedges ARE the component's boundary dedges, so
// one shared key identifies it.
func chartForGroup(charts []keptChart, group [][]dedge) [][]math.Point2 {
	for _, kc := range charts {
		for _, lp := range group {
			for _, d := range lp {
				if kc.edges[chartEdgeKey(d)] {
					return kc.contours
				}
			}
		}
	}
	return nil
}

// chartEdgeKey quantises a dedge's directed (u,v) endpoints onto the arrangement's own weld grid, so
// the folded and unfolded traces of the same edge key alike.
func chartEdgeKey(d dedge) [4]int64 {
	q := func(x math.Scalar) int64 { return int64(stdmath.Round(float64(x) / seamWeldGrid)) }
	return [4]int64{q(d.a.X), q(d.a.Y), q(d.b.X), q(d.b.Y)}
}

// chartContains is a face's parametric membership: even-odd over the carried contours, with the query
// first taken into the chart's own branch on each periodic axis.
//
// One rule, and no question anywhere in it about whether a loop wraps. A contour that runs a whole turn
// closes through the seam like any other, so the ray that crosses it counts the crossings that are
// there.
func chartContains(contours [][]math.Point2, q math.Point2, uPer, vPer bool) bool {
	return chartContainsIndexed(contours, nil, q, uPer, vPer)
}

// chartContainsIndexed is chartContains with the caller's prebuilt segment index, which answers the
// same even-odd count without scanning every contour (face_chart_index.go). ix may be nil.
func chartContainsIndexed(contours [][]math.Point2, ix *chartIndex, q math.Point2, uPer, vPer bool) bool {
	if len(contours) == 0 {
		return false
	}
	inBranch := chartBranchOf(contours, q, uPer, vPer)
	if ix != nil {
		return ix.contains(inBranch)
	}
	return pointInLoops2D(contours, inBranch)
}

// chartBranchOf moves the query by whole periods onto the branch the chart was recorded on. A chart
// spans at most one period per axis (the arrangement runs over one parameter rectangle), so the branch
// beginning at the chart's own lower bound is the only one that can hold the point.
func chartBranchOf(contours [][]math.Point2, q math.Point2, uPer, vPer bool) math.Point2 {
	u0, _, v0, _, ok := polyBounds(contours)
	if !ok {
		return q
	}
	u, v := float64(q.X), float64(q.Y)
	if uPer {
		u = u0 + wrapToPeriod(u-u0)
	}
	if vPer {
		v = v0 + wrapToPeriod(v-v0)
	}
	return math.P2(u, v)
}

// polySignedArea is a closed (u,v) contour's shoelace area: positive counter-clockwise.
func polySignedArea(poly []math.Point2) float64 {
	sum := 0.0
	for i, n := 0, len(poly); i < n; i++ {
		a, b := poly[i], poly[(i+1)%n]
		sum += float64(a.X)*float64(b.Y) - float64(b.X)*float64(a.Y)
	}
	return sum / 2
}
