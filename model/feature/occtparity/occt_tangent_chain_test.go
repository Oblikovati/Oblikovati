// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// occtChainSmoothTol is the normal-angle band (radians) below which an edge is SMOOTH — i.e. its
// two faces are tangent, so OCCT's ChFi3d::IsTangentFaces accepts it and the blend neither stops on
// it nor blends it. It is deliberately the same 1e-3 rad blend.ClassifyEdgeConvexity uses, and the
// choice is MEASURED, not assumed: re-running the whole sweep at OCCT's own LocalAnalysis G1 epsilon
// (0.1 rad, 100x looser) moves exactly one case's chain (bfuseblend/B7, 2 -> 1, by declassifying the
// SEED) and adds none. The population below is therefore not an artefact of this constant.
const occtChainSmoothTol = 1e-3

// occtChainTurnBack is OCCT's turn-back guard: two chained edges' away-from-the-shared-vertex
// tangents must subtend MORE than a right angle (equivalently, their forward tangents less than
// one), ChFi3d_Builder_1.cxx's `av1v2 < M_PI/2`. It is NOT a tangency test — tangency is carried by
// FaceTangency's "every other edge at the vertex is smooth", which forces both edges onto the same
// two sheets. blend.TangentEdgeChain instead gates on a 1 deg ANTIPARALLEL band, and that difference is
// load-bearing: on complex/C5 it shortens two of the four picks' chains from 5 edges to 2 and 3.
const occtChainTurnBack = stdmath.Pi / 2

// occtTangentChain returns the maximal run of edges OCCT's `blend` propagates a single pick along —
// the SPINE it actually blends, which is why our one-edge result is scored against a solid built
// over this whole run. It is a port of OCCT ChFi3d_Builder::PerformElement's downstream/upstream
// walk plus ChFi3d_Builder::FaceTangency (ChFi3d_Builder_1.cxx): from the seed, at each end vertex
// take the one other CREASE edge, provided every remaining edge at that vertex is SMOOTH (so both
// edges' support faces continue tangentially through the junction) and the run does not turn back.
//
// It lives in the harness, not in kernel/ops, because it models the ORACLE's rule, not ours:
// blend.TangentEdgeChain is our product's "select tangent chain" and carries extra gates (see
// occtChainTurnBack). A ratchet measured with our own walker could not falsify our own walker.
//
// Example:
//
//	chain := occtTangentChain(pickedEdge) // len(chain) == 8 on complex/D8
func occtTangentChain(seed *topo.Edge) []*topo.Edge {
	if !occtChainCrease(seed) || seed.StartVertex() == seed.EndVertex() {
		return []*topo.Edge{seed}
	}
	walk := &occtChainWalk{out: []*topo.Edge{seed}, seen: map[uint64]bool{seed.ID(): true}}
	if walk.grow(seed, seed.EndVertex(), seed.StartVertex()) {
		return walk.out // the run closed a loop; the other direction would retrace it
	}
	walk.grow(seed, seed.StartVertex(), seed.EndVertex())
	return walk.out
}

// occtChainWalk accumulates one propagation and the edges already taken, so neither direction
// revisits an edge or steps back onto the seed.
type occtChainWalk struct {
	out  []*topo.Edge
	seen map[uint64]bool
}

// grow marches from cur across frontier until no edge continues the spine, or the run closes back
// onto target (a periodic spine, e.g. complex/D8's 8-edge top rim).
func (w *occtChainWalk) grow(cur *topo.Edge, frontier, target *topo.Vertex) bool {
	for {
		next := w.step(cur, frontier)
		if next == nil {
			return false
		}
		w.out = append(w.out, next)
		w.seen[next.ID()] = true
		nv := otherEndVertex(next, frontier)
		if nv == target {
			return true
		}
		cur, frontier = next, nv
	}
}

// step returns the edge OCCT continues onto at v, or nil. Among the unvisited creases at v it takes
// the straightest one that clears the turn-back guard, and only when cur and it are the SOLE creases
// there — FaceTangency's invariant.
func (w *occtChainWalk) step(cur *topo.Edge, v *topo.Vertex) *topo.Edge {
	away := tangentAwayFromVertex(cur, v)
	var best *topo.Edge
	bestAng := occtChainTurnBack
	for _, cand := range v.Edges() {
		if cand == cur || w.seen[cand.ID()] || !occtChainCrease(cand) || !soleCreasesAt(v, cur, cand) {
			continue
		}
		if ang := float64(away.AngleTo(tangentAwayFromVertex(cand, v))); ang > bestAng {
			best, bestAng = cand, ang
		}
	}
	return best
}

// soleCreasesAt reports whether cur and cand are the only creases meeting at v — every other edge
// there is smooth, so the two supporting sheets pass tangentially through the junction.
func soleCreasesAt(v *topo.Vertex, cur, cand *topo.Edge) bool {
	for _, e := range v.Edges() {
		if e != cur && e != cand && occtChainCrease(e) {
			return false
		}
	}
	return true
}

// occtChainCrease reports whether an edge is a blendable crease: manifold, with its two
// material-outward face normals more than occtChainSmoothTol apart.
func occtChainCrease(e *topo.Edge) bool {
	uses := e.Uses()
	if len(uses) != 2 {
		return false
	}
	lo, hi := e.Geometry().Domain()
	mid := e.Geometry().PointAt((lo + hi) / 2)
	n1, ok1 := outwardNormalAt(uses[0].Loop().Face(), mid)
	n2, ok2 := outwardNormalAt(uses[1].Loop().Face(), mid)
	return ok1 && ok2 && float64(n1.AngleTo(n2)) >= occtChainSmoothTol
}

// outwardNormalAt is a face's material-outward unit normal nearest p.
func outwardNormalAt(f *topo.Face, p math.Point3) (math.Vector3, bool) {
	u, v := f.Geometry().ParamAt(p)
	n := f.Geometry().NormalAt(u, v)
	l := float64(n.Length())
	if l == 0 || stdmath.IsNaN(l) {
		return math.Vector3{}, false
	}
	n = n.Scale(math.Scalar(1 / l))
	if f.Reversed() {
		return n.Negate(), true
	}
	return n, true
}

// tangentAwayFromVertex is e's unit tangent at v pointing along the edge, away from v.
func tangentAwayFromVertex(e *topo.Edge, v *topo.Vertex) math.Vector3 {
	c := e.Geometry()
	lo, hi := c.Domain()
	t := c.TangentAt(lo)
	if e.EndVertex() == v && e.StartVertex() != v {
		t = c.TangentAt(hi).Negate()
	}
	if l := float64(t.Length()); l > 0 {
		t = t.Scale(math.Scalar(1 / l))
	}
	return t
}

// otherEndVertex returns the endpoint of e that is not v.
func otherEndVertex(e *topo.Edge, v *topo.Vertex) *topo.Vertex {
	if e.StartVertex() == v {
		return e.EndVertex()
	}
	return e.StartVertex()
}
