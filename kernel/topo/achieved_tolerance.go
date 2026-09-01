// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	stdmath "math"
)

// A body's boundary is not uniformly exact. Most edges are analytic curves that describe their geometry
// exactly, but an edge stitched from a MARCHED surface–surface intersection is a chord approximation:
// its vertices lie on both surfaces, its chords bow off them. A consumer that measures the body — mass
// properties, closure post-conditions, exchange writers — must be able to ASK how exact the boundary is
// rather than assume it is exact (ground rule: "achieved tolerance is a measured output of an operation,
// stored on the entity"; #3489).

// AchievedBoundaryTolerance returns the largest achieved tolerance over the body's edges, in model
// units: how far the worst edge's stored curve sits off the exact geometry it describes. It is 0 for a
// body whose every edge is analytic or exactly healed — the common case — and >0 exactly when some edge
// came from a marched intersection ([Edge.Tolerance]) or from import healing.
//
// Example — deciding whether a closure residual is a real defect or the boundary's own inexactness:
//
//	if residual > body.AchievedBoundaryTolerance()*someBudget { /* the body is genuinely inconsistent */ }
func (b *Body) AchievedBoundaryTolerance() float64 {
	worst := 0.0
	for _, e := range b.Edges() {
		worst = stdmath.Max(worst, e.Tolerance())
	}
	return worst
}
