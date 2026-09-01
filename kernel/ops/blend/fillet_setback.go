// SPDX-License-Identifier: GPL-2.0-only

package blend

import "oblikovati.org/kernel/topo"

// roundThirdEdges augments the pick list so every 2-edge corner becomes a full 3-edge sphere blend by
// adding the sharp third edge at the corner's CONSTANT radius (CornerRound) — reusing the watertight
// 3-edge blend instead of a degenerate 2-edge sphere.
func roundThirdEdges(picks []filletPick) []filletPick {
	return augmentThirdEdges(picks, func(e *topo.Edge, _ uint64, r float64) filletPick {
		return filletPick{edge: e, r0: r, r1: r}
	})
}

// setbackThirdEdges augments the pick list so each 2-edge corner becomes a smooth sphere whose sharp
// third edge is filleted with a VARIABLE taper — the corner radius at the vertex, running out to 0 at
// the edge's far end (CornerSetback) — a smooth corner that fades back to the sharp edge. The run-out
// apex at each far end is handled by the variable-fillet path.
func setbackThirdEdges(picks []filletPick) []filletPick {
	return augmentThirdEdges(picks, taperFromCorner)
}

// augmentThirdEdges adds, for each 2-edge corner (a vertex where exactly two picked edges meet leaving
// one sharp edge), that sharp third edge to the selection — its pick built by mk from the corner
// vertex and the corner radius. Added edges are de-duplicated (an edge may be the third edge of
// corners at both its ends).
func augmentThirdEdges(picks []filletPick, mk func(third *topo.Edge, vid uint64, r float64) filletPick) []filletPick {
	already := map[*topo.Edge]bool{}
	byVertex := map[uint64][]filletPick{}
	for _, p := range picks {
		already[p.edge] = true
		byVertex[p.edge.StartVertex().ID()] = append(byVertex[p.edge.StartVertex().ID()], p)
		byVertex[p.edge.EndVertex().ID()] = append(byVertex[p.edge.EndVertex().ID()], p)
	}
	out := picks
	added := map[*topo.Edge]bool{}
	for vid, ps := range byVertex {
		if len(ps) != 2 {
			continue // only a 2-edge corner has a single sharp edge to round
		}
		third := thirdSharpEdge(vertexByID(edgesOf(ps), vid), ps)
		if third == nil || already[third] || added[third] {
			continue
		}
		added[third] = true
		out = append(out, mk(third, vid, radiusAtVertex(ps[0], vid)))
	}
	return out
}

// taperFromCorner builds a variable pick on edge e that carries radius r at the vertex vid (the corner)
// and runs out to 0 at the far end.
func taperFromCorner(e *topo.Edge, vid uint64, r float64) filletPick {
	if e.StartVertex().ID() == vid {
		return filletPick{edge: e, r0: r, r1: 0}
	}
	return filletPick{edge: e, r0: 0, r1: r}
}

// thirdSharpEdge returns the lone edge at v that is none of the picked edges — the sharp edge a 2-edge
// corner leaves between the two filleted edges' outer faces. nil unless exactly one such edge exists.
func thirdSharpEdge(v *topo.Vertex, ps []filletPick) *topo.Edge {
	picked := map[*topo.Edge]bool{}
	for _, p := range ps {
		picked[p.edge] = true
	}
	var third *topo.Edge
	for _, e := range v.Edges() {
		if picked[e] {
			continue
		}
		if third != nil {
			return nil // more than one sharp edge — not a clean 3-edge corner
		}
		third = e
	}
	return third
}
