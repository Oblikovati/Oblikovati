// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import (
	"math/big"
	"math/rand"
	"testing"
)

// TestForceEdgeRecoversConstraint forces edges that are not present in the Delaunay
// triangulation and checks each is recovered while the triangulation stays a valid
// tiling. It requires that at least one force actually flips (the constraint was
// initially absent), so the recovery path is genuinely exercised.
func TestForceEdgeRecoversConstraint(t *testing.T) {
	face := tri([3]float64{0, 0, 0}, [3]float64{20, 0, 0}, [3]float64{0, 20, 0})
	d := newDelaunayInTriangle(face)
	interior := [][3]float64{
		{14, 2, 0}, {2, 14, 0}, {5, 5, 0}, {9, 9, 0}, {3, 10, 0}, {10, 3, 0}, {7, 2, 0}, {2, 7, 0},
	}
	for _, p := range interior {
		d.Insert(pt(p))
	}
	want := new(big.Rat).Abs(orient2Val(face[0], face[1], face[2], d.axis))

	r := rand.New(rand.NewSource(0x2c2c))
	flipped := false
	for range 40 {
		i := 3 + r.Intn(len(d.verts)-3) // pick interior vertices (skip the 3 corners)
		j := 3 + r.Intn(len(d.verts)-3)
		if i == j {
			continue
		}
		if !d.edgeExists(i, j) {
			flipped = true
		}
		d.forceEdge(i, j)
		if !d.edgeExists(i, j) {
			t.Fatalf("forced edge %d-%d not recovered", i, j)
		}
		assertValidTiling(t, 0, d, want)
	}
	if !flipped {
		t.Fatal("no forced edge was initially absent; the recovery path was not exercised")
	}
}

// TestForceEdgeThroughVertexBails covers the defensive path: a segment with a
// vertex strictly on it (the precondition forbids this — the caller splits first)
// crosses no edge properly, so convexCrossing finds nothing and forceEdge bails
// without altering the mesh, rather than spinning.
func TestForceEdgeThroughVertexBails(t *testing.T) {
	face := tri([3]float64{0, 0, 0}, [3]float64{10, 0, 0}, [3]float64{0, 10, 0})
	d := newDelaunayInTriangle(face)
	d.Insert(pt([3]float64{5, 0, 0})) // on the bottom edge, strictly between corners 0 and 1
	n := len(d.tris)
	d.forceEdge(0, 1)
	if len(d.tris) != n {
		t.Fatalf("forceEdge through an intermediate vertex changed the mesh (%d -> %d)", n, len(d.tris))
	}
	if _, _, ok := d.convexCrossing(0, 1); ok {
		t.Fatal("convexCrossing found a flippable crossing for a boundary segment")
	}
}

// TestForceEdgeNoOps guards the trivial cases: forcing a vertex to itself, and
// forcing an edge that already exists, both leave the mesh unchanged.
func TestForceEdgeNoOpsDelaunay(t *testing.T) {
	face := tri([3]float64{0, 0, 0}, [3]float64{8, 0, 0}, [3]float64{0, 8, 0})
	d := newDelaunayInTriangle(face)
	d.Insert(pt([3]float64{2, 2, 0}))
	n := len(d.tris)
	d.forceEdge(0, 0)              // same vertex
	d.forceEdge(0, d.tris[0].v[1]) // an existing edge
	if len(d.tris) != n {
		t.Fatalf("no-op forceEdge changed the triangle count (%d -> %d)", n, len(d.tris))
	}
}
