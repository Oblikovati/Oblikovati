// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
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
