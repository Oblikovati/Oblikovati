// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// edgeDirections resolves, for an ordered tangent-continuous edge run, whether each edge is
// traversed along its intrinsic parameter (lo→hi) or against it (hi→lo). It threads a frontier
// vertex through the run: the entry of the first edge is its endpoint not shared with the next
// edge (for a lone edge, its start), and each edge's exit becomes the next edge's entry.
func edgeDirections(edges []*topo.Edge) []bool {
	fwd := make([]bool, len(edges))
	frontier := entryVertex(edges)
	for i, e := range edges {
		fwd[i] = e.StartVertex() == frontier
		frontier = spineOtherVertex(e, frontier)
	}
	return fwd
}

// entryVertex returns the vertex the run starts at: for a multi-edge run, the endpoint of the
// first edge not shared with the second; otherwise the first edge's start.
func entryVertex(edges []*topo.Edge) *topo.Vertex {
	if len(edges) < 2 {
		return edges[0].StartVertex()
	}
	shared := sharedVertex(edges[0], edges[1])
	return spineOtherVertex(edges[0], shared)
}

// sharedVertex returns the vertex common to two consecutive edges (a's end that touches b).
func sharedVertex(a, b *topo.Edge) *topo.Vertex {
	if a.EndVertex() == b.StartVertex() || a.EndVertex() == b.EndVertex() {
		return a.EndVertex()
	}
	return a.StartVertex()
}

// spineOtherVertex returns e's endpoint that is not v (v itself for a closed edge).
func spineOtherVertex(e *topo.Edge, v *topo.Vertex) *topo.Vertex {
	if e.StartVertex() == v {
		return e.EndVertex()
	}
	return e.StartVertex()
}

// gaussLegendre5 are the 5-point Gauss-Legendre nodes/weights on [-1,1]: exact for polynomials up
// to degree 9 and effectively exact for the constant speed of lines and circular arcs (the common
// spine edges), accurate for NURBS.
var gaussLegendre5 = struct{ x, w [5]float64 }{
	x: [5]float64{-0.9061798459386640, -0.5384693101056831, 0, 0.5384693101056831, 0.9061798459386640},
	w: [5]float64{0.2369268850561891, 0.4786286704993665, 0.5688888888888889, 0.4786286704993665, 0.2369268850561891},
}

// arcLength integrates |dP/dt| over [lo,hi] by 5-point Gauss-Legendre — the curve's true arc
// length (exact for constant-speed lines and arcs).
func arcLength(c geom.Curve3, lo, hi float64) float64 {
	if hi <= lo {
		return 0
	}
	half := (hi - lo) / 2
	mid := (hi + lo) / 2
	sum := 0.0
	for k := 0; k < 5; k++ {
		t := mid + half*gaussLegendre5.x[k]
		sum += gaussLegendre5.w[k] * float64(c.TangentAt(t).Length())
	}
	return sum * half
}

// paramAtArcLength inverts the arc length: it returns the parameter t in [lo,hi] whose distance
// along the curve from lo equals target, by Newton iteration (s'(t) = |dP/dt|). Clamped at the
// ends. Used to map a spine abscissa back to an edge-curve parameter.
func paramAtArcLength(c geom.Curve3, lo, hi, target float64) float64 {
	full := arcLength(c, lo, hi)
	if target <= 0 || full == 0 {
		return lo
	}
	if target >= full {
		return hi
	}
	t := lo + (hi-lo)*target/full // constant-speed seed
	for iter := 0; iter < 12; iter++ {
		speed := float64(c.TangentAt(t).Length())
		if speed < 1e-14 {
			break
		}
		dt := (arcLength(c, lo, t) - target) / speed
		t = clampf(t-dt, lo, hi)
		if stdmath.Abs(dt) < 1e-12 {
			break
		}
	}
	return t
}

// clampf constrains x to [lo,hi].
func clampf(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}
