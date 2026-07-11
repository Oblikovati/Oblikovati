// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// edgeSamples is the chord count approximating an edge's arc-length centroid and length. It
// must match the oracle's edgeloc sampling (test-utilities/occt-blend/oracle/oracle.tcl) so
// both kernels compute the same values for the same physical edge.
const edgeSamples = 64

// locateEdge returns the body edge that OCCT picked, re-found by geometry so we do not depend
// on our topology's edge ordering. It matches on the arc-length CENTROID (parameterization-
// invariant, unlike the mid-parameter point, which STEP import shifts on curved edges) and
// requires the arc LENGTH to agree — disambiguating edges that share a centroid (e.g.
// concentric circles). tol scales to the body (see importTol).
//
// Example:
//
//	e, err := locateEdge(body, pick.Locator, importTol(body))
func locateEdge(b *topo.Body, loc Locator, tol float64) (*topo.Edge, error) {
	target := math.P3(loc.Centroid[0], loc.Centroid[1], loc.Centroid[2])
	var best *topo.Edge
	bestD := stdmath.Inf(1)
	for _, e := range b.Edges() {
		cen, length := edgeCentroidLength(e)
		if !lengthsAgree(length, loc.Length, tol) {
			continue
		}
		if d := float64(cen.DistanceTo(target)); d < bestD {
			bestD, best = d, e
		}
	}
	if best == nil || bestD > tol {
		return nil, fmt.Errorf("locateEdge: no edge within %.3g of centroid %v len %.4g (closest %.3g)", tol, loc.Centroid, loc.Length, bestD)
	}
	return best, nil
}

// lengthsAgree accepts two arc lengths as the same edge when they differ by under a relative
// slack, with tol as an absolute floor for tiny edges.
func lengthsAgree(a, b, tol float64) bool {
	return stdmath.Abs(a-b) <= 0.01*stdmath.Max(a, b)+tol
}

// edgeCentroidLength approximates an edge's arc-length centroid and total length by chord
// sampling over its parameter domain — parameterization-invariant, matching the oracle.
func edgeCentroidLength(e *topo.Edge) (math.Point3, float64) {
	c := e.Geometry()
	lo, hi := c.Domain()
	var sum math.Vector3
	var length float64
	prev := c.PointAt(lo)
	for i := 1; i <= edgeSamples; i++ {
		t := lo + (hi-lo)*float64(i)/float64(edgeSamples)
		p := c.PointAt(t)
		seg := float64(p.DistanceTo(prev))
		sum = sum.Add(prev.Midpoint(p).AsVector().Scale(seg))
		length += seg
		prev = p
	}
	if length == 0 {
		return c.PointAt(lo), 0
	}
	return sum.Scale(1 / length).AsPoint(), length
}
