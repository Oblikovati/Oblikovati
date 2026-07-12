// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// endCornerFan is the topology-free snapshot the runout solver consumes for a fillet edge that
// terminates at a vertex where N>3 faces meet. It carries geometry (points/normals/surfaces) and
// opaque uint64 ids only — never live *topo.* — so the solver depends on geom+math alone.
type endCornerFan struct {
	filletEdge uint64
	faceA      uint64
	faceB      uint64
	radius     float64
	center     math.Point3  // rolling-ball centre at this runout end (corner.cen)
	axis       math.Vector3 // fillet cylinder axis direction (unit), oriented toward the vertex
	apex       math.Point3  // the original runout vertex point
	ta, tb     math.Point3  // arc endpoints where the cap is tangent to faceA, faceB
	fan        []fanFace    // far faces F3..FN, cyclically ordered from the A flank to the B flank
	farEdges   []fanEdge    // the interior edges between consecutive fan faces, aligned with fan gaps
}

// fanFace is one far face incident to the runout vertex (neither A nor B).
//
// entryEdge/exitEdge use 0 as the A/B-flank sentinel ("no far edge here, use fan.ta/fan.tb instead"
// — see boundaryPoint/loopEntersFromEntry). Safe only because real topo edge ids are never 0:
// kernel/topo/lineage.go mints ids from idSeq (zero-valued atomic.Uint64) via nextID() =
// idSeq.Add(1), so the first live id is 1 and 0 is never assigned to a real edge.
type fanFace struct {
	face      uint64
	normal    math.Vector3 // material-outward
	entryEdge uint64       // the far edge (or A/B flank sentinel 0) bounding this face on the A side
	exitEdge  uint64       // the far edge (or B flank sentinel 0) bounding this face on the B side
}

// fanEdge is a far edge shared by two consecutive fan faces; the runout crosses it once and it must
// be split there so both faces weld the identical point.
type fanEdge struct {
	edge      uint64
	from, to  math.Point3
	leftFace  uint64
	rightFace uint64
}

// vertexFaces returns the distinct faces incident to v, derived from its edges (topo.Vertex has no
// Faces() of its own — a vertex only tracks the edges that meet there).
//
// Example: for the valence-5 runout vertex on the occtparity V3 fixture,
// len(vertexFaces(v)) == 5.
func vertexFaces(v *topo.Vertex) []*topo.Face {
	seen := map[uint64]bool{}
	var out []*topo.Face
	for _, e := range v.Edges() {
		for _, f := range e.Faces() {
			if !seen[f.ID()] {
				seen[f.ID()] = true
				out = append(out, f)
			}
		}
	}
	return out
}

// vertexValence is the number of distinct faces meeting at v — the count the runout detector uses
// to decide whether a fillet-edge end needs the n-valent fan treatment (N>3) instead of the
// existing trihedral-corner path.
func vertexValence(v *topo.Vertex) int { return len(vertexFaces(v)) }

// classifyEndCorners partitions the fils' simple end corners: valence>3 ends with all-planar far
// faces become endCornerFans and their vertices are marked owned (so filletMaps' trihedral ends
// path skips them, Task 6); every other corner — blend/miter/runout, trihedral (valence<=3), or a
// non-planar far face — is left untouched for the existing addCornerRound path.
//
// Example: on the occtparity simple/V3 fixture the valence-5 pick end yields one fan (3 far faces),
// while the valence-3 end yields none.
func classifyEndCorners(fils []edgeFillet) ([]endCornerFan, map[uint64]bool) {
	var fans []endCornerFan
	owned := map[uint64]bool{}
	for _, ef := range fils {
		for _, c := range []corner{ef.c0, ef.c1} {
			fan, ok := fanForEndCorner(ef, c)
			if !ok {
				continue
			}
			fans = append(fans, fan)
			owned[c.vertex.ID()] = true
		}
	}
	return fans, owned
}

// fanForEndCorner returns the fan for a SIMPLE end corner at a >3-valent, all-planar vertex; ok=false
// for blend/miter/runout corners, trihedral (valence<=3) ends, or non-planar far faces — all of which
// stay on the shipping trihedral path.
func fanForEndCorner(ef edgeFillet, c corner) (endCornerFan, bool) {
	if c.blend || c.miter || c.runout || vertexValence(c.vertex) <= 3 {
		return endCornerFan{}, false
	}
	return buildEndCornerFan(ef, c)
}

// buildEndCornerFan orders the far faces cyclically from the A flank to the B flank around the runout
// vertex and snapshots the geometry. Returns ok=false if a far face is non-planar (quadric far faces
// are deferred to the trihedral path in this slice).
func buildEndCornerFan(ef edgeFillet, c corner) (endCornerFan, bool) {
	chain, ok := orderedFarChain(c.vertex, ef.a, ef.b)
	if !ok {
		return endCornerFan{}, false
	}
	faces, ok := fanFacesOf(chain)
	if !ok {
		return endCornerFan{}, false // a far face is non-planar
	}
	return endCornerFan{
		filletEdge: ef.edge.ID(), faceA: ef.a.ID(), faceB: ef.b.ID(), radius: ef.cyl.Radius,
		center: c.cen, axis: ef.cyl.AxisDir.AsVector(), apex: c.vertex.Point(), ta: c.ta, tb: c.tb,
		fan: faces, farEdges: farEdgesOf(chain),
	}, true
}

// farChain is the A->B ordered fan of far faces and the interior far edges between them.
type farChain struct {
	faces []*topo.Face
	edges []*topo.Edge // len == len(faces)-1
}

// orderedFarChain walks the faces around v from A's flank to B's flank via shared at-v edges,
// collecting the far faces (every incident face except A and B) in cyclic order and the interior
// edges between consecutive far faces (len(edges) == len(faces)-1). ok=false on a non-simple ring
// (dead end, or the walk wraps back to A without reaching B).
func orderedFarChain(v *topo.Vertex, a, b *topo.Face) (farChain, bool) {
	start, _, ok := farNeighbourAcross(v, a, b)
	if !ok {
		return farChain{}, false
	}
	chain := farChain{faces: []*topo.Face{start}}
	return walkFarChain(v, a, b, chain, a.ID(), start)
}

// walkFarChain steps nextFar from cur (arrived via prevID) until it reaches b, appending interior
// far faces/edges to chain. The step cap is len(v.Edges()): a valid fan visits at most one face
// per incident edge, so this bounds the walk even on a non-manifold edge — Edge.Faces() returning
// 3+ faces could otherwise spin nextFar forever (Task 7 owns explicit non-manifold rejection; this
// is just a hang guard).
func walkFarChain(v *topo.Vertex, a, b *topo.Face, chain farChain, prevID uint64, cur *topo.Face) (farChain, bool) {
	for step := 0; step < len(v.Edges()); step++ {
		nf, ne, ok := nextFar(v, cur, prevID)
		if !ok || nf == a {
			return farChain{}, false
		}
		if nf == b { // ne is the B-side flank edge, not an interior far edge
			return chain, true
		}
		chain.edges = append(chain.edges, ne)
		chain.faces = append(chain.faces, nf)
		prevID, cur = cur.ID(), nf
	}
	return farChain{}, false
}

// farNeighbourAcross returns the far face across a's non-fillet at-v edge (the A-flank far face,
// neither a nor b) plus that edge — the entry of the fan. The fillet edge a|b is skipped because
// its other face is b. ok=false if a has no such flank edge at v.
func farNeighbourAcross(v *topo.Vertex, a, b *topo.Face) (*topo.Face, *topo.Edge, bool) {
	for _, e := range v.Edges() {
		if !edgeHasFace(e, a) {
			continue
		}
		if o := otherFace(e, a); o != nil && o != b {
			return o, e, true
		}
	}
	return nil, nil, false
}

// nextFar returns the face sharing an at-v edge with cur that is not prevID (the next step of the
// cyclic walk around v) plus the shared edge. ok=false at a dead end.
func nextFar(v *topo.Vertex, cur *topo.Face, prevID uint64) (*topo.Face, *topo.Edge, bool) {
	for _, e := range v.Edges() {
		if !edgeHasFace(e, cur) {
			continue
		}
		if o := otherFace(e, cur); o != nil && o.ID() != prevID {
			return o, e, true
		}
	}
	return nil, nil, false
}

// fanFacesOf snapshots the chain's far faces in order; ok=false if any is non-planar (the runout
// solver only handles planar far faces in this slice — quadric far faces are deferred).
func fanFacesOf(chain farChain) ([]fanFace, bool) {
	out := make([]fanFace, 0, len(chain.faces))
	for i, f := range chain.faces {
		pl, isPlane := f.Geometry().(geom.Plane)
		if !isPlane {
			return nil, false
		}
		out = append(out, fanFaceOf(f, pl, chain.edges, i))
	}
	return out, true
}

// fanFaceOf snapshots far face f (plane pl) at position i in the chain, wiring its entry/exit far
// edges. Position 0 has no entry (the A flank, sentinel 0); the last face has no exit (the B flank).
func fanFaceOf(f *topo.Face, pl geom.Plane, edges []*topo.Edge, i int) fanFace {
	ff := fanFace{face: f.ID(), normal: outwardPlaneNormal(f, pl)}
	if i > 0 {
		ff.entryEdge = edges[i-1].ID()
	}
	if i < len(edges) {
		ff.exitEdge = edges[i].ID()
	}
	return ff
}

// farEdgesOf snapshots the interior far edges with the two far faces each separates (edge i lies
// between chain.faces[i] and chain.faces[i+1], aligned with the fan gaps).
func farEdgesOf(chain farChain) []fanEdge {
	out := make([]fanEdge, 0, len(chain.edges))
	for i, e := range chain.edges {
		out = append(out, fanEdge{
			edge: e.ID(), from: e.StartVertex().Point(), to: e.EndVertex().Point(),
			leftFace: chain.faces[i].ID(), rightFace: chain.faces[i+1].ID(),
		})
	}
	return out
}

// validateRunoutFans honest-rejects any n-valent runout whose spread fails the validity certificate
// (self-intersecting, over-radius, tangent-degenerate) — the n-valent analogue of validateFilletRadii
// / #1800. Without this pre-pass, buildSpreadMaps would quietly skip the bad fan (it has no other way
// to fail the whole op) and the rebuild would ship an open shell instead of erroring.
func validateRunoutFans(fils []edgeFillet) error {
	fans, _ := classifyEndCorners(fils)
	for _, fan := range fans {
		if _, err := solveRunoutSpread(fan); err != nil {
			return err
		}
	}
	return nil
}
