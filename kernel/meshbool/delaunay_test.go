// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import (
	"math/big"
	"math/rand"
	"testing"
)

// TestDelaunayInsertInvariants inserts random interior and edge points into several
// faces and checks the full contract after every build: the triangles tile the face
// exactly (area conserved), every triangle is non-degenerate and CCW, the adjacency
// is symmetric, and the triangulation is Delaunay (no in-circle-illegal edge). An
// adjacency or flip bug breaks at least one of these.
func TestDelaunayInsertInvariants(t *testing.T) {
	faces := [][3]Point{
		tri([3]float64{0, 0, 0}, [3]float64{12, 0, 0}, [3]float64{0, 12, 0}),
		tri([3]float64{0, 0, 0}, [3]float64{0, 6, 0}, [3]float64{0, 0, 6}),
		tri([3]float64{0, 0, 0}, [3]float64{6, 0, 3}, [3]float64{1, 9, 2}),
	}
	r := rand.New(rand.NewSource(0x2b2b))
	for fi, face := range faces {
		d := newDelaunayInTriangle(face)
		want := new(big.Rat).Abs(orient2Val(face[0], face[1], face[2], d.axis))
		var inserted []Point
		for k := 0; k < 150; k++ {
			p := randBary(r, face, k%5 == 0) // one in five on a face edge
			d.Insert(p)
			inserted = append(inserted, p)
		}
		assertDelaunayMesh(t, fi, d, want)
		for _, p := range inserted {
			if !hasVertex(d, p) {
				t.Fatalf("face %d: inserted point %v is not a vertex", fi, p.Round())
			}
		}
	}
}

// TestDelaunaySharedEdgeSplit exercises the four-triangle shared-edge split: insert
// the face centre (creating internal edges), then the midpoint of an internal edge.
func TestDelaunaySharedEdgeSplit(t *testing.T) {
	face := tri([3]float64{0, 0, 0}, [3]float64{6, 0, 0}, [3]float64{0, 6, 0})
	d := newDelaunayInTriangle(face)
	d.Insert(pt([3]float64{2, 2, 0})) // centre → internal edges to the corners
	d.Insert(pt([3]float64{1, 1, 0})) // midpoint of the internal edge (0,0)-(2,2)
	want := new(big.Rat).Abs(orient2Val(face[0], face[1], face[2], d.axis))
	assertDelaunayMesh(t, 0, d, want)
	if !hasVertex(d, pt([3]float64{1, 1, 0})) {
		t.Fatal("internal-edge midpoint was not inserted")
	}
}

func TestDelaunayEdgeCases(t *testing.T) {
	face := tri([3]float64{0, 0, 0}, [3]float64{6, 0, 0}, [3]float64{0, 6, 0})
	d := newDelaunayInTriangle(face)
	d.Insert(pt([3]float64{1, 1, 0}))

	if len(d.triangles()) != len(d.tris) {
		t.Fatalf("triangles() returned %d, want %d", len(d.triangles()), len(d.tris))
	}
	before := len(d.tris)
	d.Insert(pt([3]float64{100, 100, 0})) // outside the face → located nowhere
	if len(d.tris) != before {
		t.Fatal("inserting a point outside the face changed the mesh")
	}
	if d.edgeSlot(0, 999, 998) != -1 {
		t.Fatal("edgeSlot of a non-edge must be -1")
	}
}

func TestNewDelaunayNormalizesCW(t *testing.T) {
	cw := tri([3]float64{0, 0, 0}, [3]float64{0, 6, 0}, [3]float64{6, 0, 0}) // CW in xy
	d := newDelaunayInTriangle(cw)
	tr := d.tris[0]
	if orient2(d.verts[tr.v[0]], d.verts[tr.v[1]], d.verts[tr.v[2]], d.axis) <= 0 {
		t.Fatal("newDelaunayInTriangle did not normalize a clockwise face to CCW")
	}
}

// --- invariant checks ---

func assertDelaunayMesh(t *testing.T, fi int, d *delaunayMesh, wantArea *big.Rat) {
	t.Helper()
	assertValidTiling(t, fi, d, wantArea)
	assertDelaunayProperty(t, fi, d)
}

// assertValidTiling checks area conservation, non-degenerate CCW triangles, and
// symmetric adjacency — the invariants that hold even for a CONSTRAINED (non-
// Delaunay) triangulation.
func assertValidTiling(t *testing.T, fi int, d *delaunayMesh, wantArea *big.Rat) {
	t.Helper()
	sum := new(big.Rat)
	for i, tr := range d.tris {
		det := orient2Val(d.verts[tr.v[0]], d.verts[tr.v[1]], d.verts[tr.v[2]], d.axis)
		if det.Sign() <= 0 {
			t.Fatalf("face %d: triangle %d is degenerate or clockwise", fi, i)
		}
		sum.Add(sum, det)
	}
	if sum.Cmp(wantArea) != 0 {
		t.Fatalf("face %d: area not conserved: sum=%s want=%s", fi, sum.RatString(), wantArea.RatString())
	}
	assertAdjacencySymmetric(t, fi, d)
}

func assertAdjacencySymmetric(t *testing.T, fi int, d *delaunayMesh) {
	t.Helper()
	for ti, tr := range d.tris {
		for e := 0; e < 3; e++ {
			tj := tr.adj[e]
			if tj < 0 {
				continue
			}
			a, b := tr.v[e], tr.v[(e+1)%3]
			if se := d.edgeSlot(tj, b, a); se < 0 || d.tris[tj].adj[se] != ti {
				t.Fatalf("face %d: adjacency broken at triangle %d edge %d (neighbour %d)", fi, ti, e, tj)
			}
		}
	}
}

func assertDelaunayProperty(t *testing.T, fi int, d *delaunayMesh) {
	t.Helper()
	for ti, tr := range d.tris {
		for e := 0; e < 3; e++ {
			tj := tr.adj[e]
			if tj < 0 {
				continue
			}
			a, b := tr.v[e], tr.v[(e+1)%3]
			p := tr.v[(e+2)%3]
			q := d.oppositeVertex(tj, a, b)
			if inCircleSign(d.verts[a], d.verts[b], d.verts[p], d.verts[q], d.axis) > 0 {
				t.Fatalf("face %d: non-Delaunay edge at triangle %d", fi, ti)
			}
		}
	}
}

func hasVertex(d *delaunayMesh, p Point) bool {
	for _, v := range d.verts {
		if v.Equal(p) {
			return true
		}
	}
	return false
}

// randBary returns an exact point inside face (or on an edge when onEdge) from
// integer barycentric weights, so it lies exactly in the face plane.
func randBary(r *rand.Rand, face [3]Point, onEdge bool) Point {
	wi, wj, wk := int64(r.Intn(9)+1), int64(r.Intn(9)+1), int64(r.Intn(9)+1)
	if onEdge {
		switch r.Intn(3) {
		case 0:
			wi = 0
		case 1:
			wj = 0
		default:
			wk = 0
		}
	}
	return baryPoint(face, wi, wj, wk)
}

func baryPoint(face [3]Point, wi, wj, wk int64) Point {
	den := big.NewRat(wi+wj+wk, 1)
	mix := func(a, b, c *big.Rat) *big.Rat {
		s := new(big.Rat).Mul(a, big.NewRat(wi, 1))
		s.Add(s, new(big.Rat).Mul(b, big.NewRat(wj, 1)))
		s.Add(s, new(big.Rat).Mul(c, big.NewRat(wk, 1)))
		return s.Quo(s, den)
	}
	return Point{
		X: mix(face[0].X, face[1].X, face[2].X),
		Y: mix(face[0].Y, face[1].Y, face[2].Y),
		Z: mix(face[0].Z, face[1].Z, face[2].Z),
	}
}
