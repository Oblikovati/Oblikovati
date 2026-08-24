// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import (
	"math/big"
	"math/rand"
	"testing"
)

// TestInsertPointAreaConserved is the core invariant: after any sequence of point
// insertions the sub-triangles exactly tile the face (sum of signed projected
// areas equals the face's), every sub-triangle is non-degenerate and identically
// oriented, and every inserted interior/edge point became a vertex. Exact area
// conservation catches any overlap or gap. Runs on an axis-aligned and a tilted
// face so planeAxis and all three projections are exercised.
func TestInsertPointAreaConserved(t *testing.T) {
	faces := [][3]Point{
		tri([3]float64{0, 0, 0}, [3]float64{12, 0, 0}, [3]float64{0, 12, 0}), // z=0 → drop z
		tri([3]float64{0, 0, 0}, [3]float64{0, 6, 0}, [3]float64{0, 0, 6}),   // x=0 → drop x
		tri([3]float64{0, 0, 0}, [3]float64{6, 0, 0}, [3]float64{0, 0, 6}),   // y=0 → drop y
		tri([3]float64{0, 0, 0}, [3]float64{6, 0, 3}, [3]float64{1, 9, 2}),   // tilted
	}
	r := rand.New(rand.NewSource(0x1c07))
	for fi, face := range faces {
		trg := NewTriangulation(face)
		faceDet := orient2Val(face[0], face[1], face[2], trg.axis)
		var inserted []Point
		for k := 0; k < 200; k++ {
			p := randBary(r, face, k%4 == 0) // one in four lands on an edge
			trg.InsertPoint(p)
			inserted = append(inserted, p)
		}
		if sum := totalDet(trg); sum.Cmp(faceDet) != 0 {
			t.Fatalf("face %d: area not conserved: sum=%s faceDet=%s", fi, sum.RatString(), faceDet.RatString())
		}
		wantSign := faceDet.Sign()
		for i, tt := range trg.tris {
			d := orient2Val(trg.verts[tt[0]], trg.verts[tt[1]], trg.verts[tt[2]], trg.axis)
			if d.Sign() == 0 {
				t.Fatalf("face %d: degenerate sub-triangle %d", fi, i)
			}
			if d.Sign() != wantSign {
				t.Fatalf("face %d: sub-triangle %d has opposite orientation", fi, i)
			}
		}
		for _, p := range inserted {
			if trg.indexOf(p) < 0 {
				t.Fatalf("face %d: inserted point %v never became a vertex", fi, p.Round())
			}
		}
	}
}

func TestInsertPointNoOps(t *testing.T) {
	face := tri([3]float64{0, 0, 0}, [3]float64{10, 0, 0}, [3]float64{0, 10, 0})
	trg := NewTriangulation(face)
	nv, nt := len(trg.verts), len(trg.tris)

	trg.InsertPoint(face[0]) // duplicate of a corner
	if len(trg.verts) != nv || len(trg.tris) != nt {
		t.Fatalf("duplicate insert changed the mesh: verts %d->%d tris %d->%d", nv, len(trg.verts), nt, len(trg.tris))
	}
	trg.InsertPoint(pt([3]float64{100, 100, 0})) // outside the face
	if len(trg.verts) != nv || len(trg.tris) != nt {
		t.Fatalf("outside insert changed the mesh (orphan vertex left behind)")
	}
}

func TestInsertPointOnBoundaryEdgeSplits(t *testing.T) {
	face := tri([3]float64{0, 0, 0}, [3]float64{10, 0, 0}, [3]float64{0, 10, 0})
	trg := NewTriangulation(face)
	trg.InsertPoint(pt([3]float64{5, 0, 0})) // midpoint of edge (0,1)
	if len(trg.tris) != 2 {
		t.Fatalf("boundary-edge insert: got %d triangles, want 2", len(trg.tris))
	}
	if sum := totalDet(trg); sum.Cmp(orient2Val(face[0], face[1], face[2], trg.axis)) != 0 {
		t.Fatal("boundary-edge insert did not conserve area")
	}
}

func TestTrianglesAndClassify(t *testing.T) {
	face := tri([3]float64{0, 0, 0}, [3]float64{6, 0, 0}, [3]float64{0, 6, 0})
	trg := NewTriangulation(face)
	trg.InsertPoint(pt([3]float64{1, 1, 0})) // interior → 3 triangles

	got := trg.Triangles()
	if len(got) != 3 {
		t.Fatalf("Triangles: got %d, want 3", len(got))
	}
	for _, tt := range got { // every emitted triangle is a real, non-degenerate face
		if orient2Val(tt[0], tt[1], tt[2], trg.axis).Sign() == 0 {
			t.Fatal("Triangles emitted a degenerate triangle")
		}
	}

	// classify: a corner is a vertex coincidence (two zeros) → locOutside branch.
	if loc, _ := trg.classify(trg.tris[0], face[0]); loc != locOutside {
		t.Fatalf("classify at a shared corner: got %d, want locOutside", loc)
	}
}

// --- helpers ---

// totalDet sums the signed projected determinants of all sub-triangles.
func totalDet(tr *Triangulation) *big.Rat {
	sum := new(big.Rat)
	for _, t := range tr.tris {
		sum.Add(sum, orient2Val(tr.verts[t[0]], tr.verts[t[1]], tr.verts[t[2]], tr.axis))
	}
	return sum
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
