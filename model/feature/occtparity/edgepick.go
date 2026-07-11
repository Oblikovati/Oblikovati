// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// locateEdge returns the body edge whose midpoint matches the oracle locator within tol.
// OCCT resolved the pick to a concrete edge; we re-find that edge by geometry so we do not
// depend on our own topology's edge ordering. Ties (equal midpoints) break on direction.
//
// Example:
//
//	e, err := locateEdge(body, pick.Locator, 1e-6)
func locateEdge(b *topo.Body, loc Locator, tol float64) (*topo.Edge, error) {
	target := math.P3(loc.Midpoint[0], loc.Midpoint[1], loc.Midpoint[2])
	var best *topo.Edge
	bestD := stdmath.Inf(1)
	for _, e := range b.Edges() {
		d := topo.DescribeEdge(e).Midpoint.DistanceTo(target)
		if d < bestD {
			bestD, best = d, e
		}
	}
	if best == nil || bestD > tol {
		return nil, fmt.Errorf("locateEdge: no edge within %.3g of midpoint %v (closest %.3g)", tol, loc.Midpoint, bestD)
	}
	return best, nil
}
