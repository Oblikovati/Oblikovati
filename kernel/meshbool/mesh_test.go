// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import (
	"math/big"
	"testing"
)

// crossingSquares builds mesh A (a 4x4 square in z=0) and mesh B (a taller square
// in the x=2 plane). They meet along the segment (2,0,0)-(2,4,0): A is split at
// x=2, B at z=0, and both must gain that segment as shared edges.
func crossingSquares() (a, b [][3]Point) {
	a = [][3]Point{
		tri([3]float64{0, 0, 0}, [3]float64{4, 0, 0}, [3]float64{4, 4, 0}),
		tri([3]float64{0, 0, 0}, [3]float64{4, 4, 0}, [3]float64{0, 4, 0}),
	}
	b = [][3]Point{
		tri([3]float64{2, -1, -1}, [3]float64{2, 5, -1}, [3]float64{2, 5, 1}),
		tri([3]float64{2, -1, -1}, [3]float64{2, 5, 1}, [3]float64{2, -1, 1}),
	}
	return a, b
}

func TestCoRefineCrossingSquares(t *testing.T) {
	t.Parallel()
	a, b := crossingSquares()
	aOut, bOut := CoRefine(a, b)

	// Each mesh still tiles its original area exactly (no gaps/overlaps).
	assertMeshArea(t, "A", aOut, 2, big.NewRat(32, 1)) // 2*4*4
	assertMeshArea(t, "B", bOut, 0, big.NewRat(24, 1)) // 2*6*2

	// The intersection endpoints and midpoint are shared vertices in BOTH meshes.
	seg := [2]Point{pt([3]float64{2, 0, 0}), pt([3]float64{2, 4, 0})}
	for _, v := range []Point{seg[0], seg[1], pt([3]float64{2, 2, 0})} {
		if !meshHasVertex(aOut, v) {
			t.Fatalf("A is not split at the intersection vertex %v", v.Round())
		}
		if !meshHasVertex(bOut, v) {
			t.Fatalf("B is not split at the intersection vertex %v", v.Round())
		}
	}

	// Neither refined mesh has a triangle straddling the intersection line.
	assertNoEdgeCrosses(t, "A", aOut, seg, 2)
	assertNoEdgeCrosses(t, "B", bOut, seg, 0)
}

func TestCoRefineCoplanarOverlap(t *testing.T) {
	t.Parallel()
	// Two triangles in z=0 that overlap; the shared region is triangle
	// (1,1)-(3,1)-(1,3) and both faces must be imprinted with its boundary.
	a := [][3]Point{tri([3]float64{0, 0, 0}, [3]float64{4, 0, 0}, [3]float64{0, 4, 0})}
	b := [][3]Point{tri([3]float64{1, 1, 0}, [3]float64{5, 1, 0}, [3]float64{1, 5, 0})}
	aOut, bOut := CoRefine(a, b)

	assertMeshArea(t, "A", aOut, 2, big.NewRat(16, 1)) // 2 * (½·4·4)
	assertMeshArea(t, "B", bOut, 2, big.NewRat(16, 1))

	overlap := []Point{pt([3]float64{1, 1, 0}), pt([3]float64{3, 1, 0}), pt([3]float64{1, 3, 0})}
	for _, v := range overlap {
		if !meshHasVertex(aOut, v) || !meshHasVertex(bOut, v) {
			t.Fatalf("overlap corner %v is not a shared vertex of both meshes", v.Round())
		}
	}
	for i := range overlap {
		edge := [2]Point{overlap[i], overlap[(i+1)%len(overlap)]}
		assertNoEdgeCrosses(t, "A", aOut, edge, 2)
		assertNoEdgeCrosses(t, "B", bOut, edge, 2)
	}
}

func TestCoRefineDisjointUnchanged(t *testing.T) {
	t.Parallel()
	a := [][3]Point{tri([3]float64{0, 0, 0}, [3]float64{4, 0, 0}, [3]float64{0, 4, 0})}
	b := [][3]Point{tri([3]float64{0, 0, 9}, [3]float64{4, 0, 9}, [3]float64{0, 4, 9})}
	aOut, bOut := CoRefine(a, b)
	if len(aOut) != 1 || len(bOut) != 1 {
		t.Fatalf("disjoint meshes were altered: |aOut|=%d |bOut|=%d, want 1 and 1", len(aOut), len(bOut))
	}
}

// --- helpers ---

func assertMeshArea(t *testing.T, name string, tris [][3]Point, axis int, want *big.Rat) {
	t.Helper()
	sum := new(big.Rat)
	for _, tt := range tris {
		d := orient2Val(tt[0], tt[1], tt[2], axis)
		if d.Sign() <= 0 {
			t.Fatalf("mesh %s: degenerate or clockwise triangle", name)
		}
		sum.Add(sum, d)
	}
	if sum.Cmp(want) != 0 {
		t.Fatalf("mesh %s: area (2x) = %s, want %s", name, sum.RatString(), want.RatString())
	}
}

func assertNoEdgeCrosses(t *testing.T, name string, tris [][3]Point, seg [2]Point, axis int) {
	t.Helper()
	for _, tt := range tris {
		for e := range 3 {
			p, q := tt[e], tt[(e+1)%3]
			if segmentsProperlyCross(p, q, seg[0], seg[1], axis) {
				t.Fatalf("mesh %s: triangle edge (%v,%v) crosses the intersection", name, p.Round(), q.Round())
			}
		}
	}
}
